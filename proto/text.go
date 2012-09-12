// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 Google Inc.  All rights reserved.
// http://code.google.com/p/goprotobuf/
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package proto

// Functions for writing the text protocol buffer format.

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
)

// textWriter is an io.Writer that tracks its indentation level.
type textWriter struct {
	ind      int
	complete bool // if the current position is a complete line
	compact  bool // whether to write out as a one-liner
	writer   io.Writer

	c [1]byte // scratch
}

func (w *textWriter) Write(p []byte) (n int, err error) {
	n, err = len(p), nil

	frags := strings.Split(string(p), "\n")
	if w.compact {
		w.writer.Write([]byte(strings.Join(frags, " ")))
		return
	}

	for i, frag := range frags {
		if w.complete {
			for j := 0; j < w.ind; j++ {
				w.writer.Write([]byte{' ', ' '})
			}
			w.complete = false
		}

		w.writer.Write([]byte(frag))
		if i+1 < len(frags) {
			w.writer.Write([]byte{'\n'})
		}
	}
	w.complete = len(frags[len(frags)-1]) == 0

	return
}

func (w *textWriter) WriteByte(c byte) error {
	w.c[0] = c
	_, err := w.Write(w.c[:])
	return err
}

func (w *textWriter) indent() { w.ind++ }

func (w *textWriter) unindent() {
	if w.ind == 0 {
		log.Printf("proto: textWriter unindented too far")
		return
	}
	w.ind--
}

func writeName(w *textWriter, props *Properties) {
	io.WriteString(w, props.OrigName)
	if props.Wire != "group" {
		w.WriteByte(':')
	}
}

var (
	messageSetType = reflect.TypeOf((*MessageSet)(nil)).Elem()
)

// raw is the interface satisfied by RawMessage.
type raw interface {
	Bytes() []byte
}

func writeStruct(w *textWriter, sv reflect.Value) {
	if sv.Type() == messageSetType {
		writeMessageSet(w, sv.Addr().Interface().(*MessageSet))
		return
	}

	st := sv.Type()
	sprops := GetProperties(st)
	for i := 0; i < sv.NumField(); i++ {
		fv := sv.Field(i)
		if name := st.Field(i).Name; strings.HasPrefix(name, "XXX_") {
			// There's only two XXX_ fields:
			//   XXX_unrecognized []byte
			//   XXX_extensions   map[int32]proto.Extension
			// The first is handled here;
			// the second is handled at the bottom of this function.
			if name == "XXX_unrecognized" && !fv.IsNil() {
				writeUnknownStruct(w, fv.Interface().([]byte))
			}
			continue
		}
		props := sprops.Prop[i]
		if fv.Kind() == reflect.Ptr && fv.IsNil() {
			// Field not filled in. This could be an optional field or
			// a required field that wasn't filled in. Either way, there
			// isn't anything we can show for it.
			continue
		}
		if fv.Kind() == reflect.Slice && fv.IsNil() {
			// Repeated field that is empty, or a bytes field that is unused.
			continue
		}

		if props.Repeated && fv.Kind() == reflect.Slice {
			// Repeated field.
			for j := 0; j < fv.Len(); j++ {
				writeName(w, props)
				if !w.compact {
					w.WriteByte(' ')
				}
				writeAny(w, fv.Index(j), props)
				w.WriteByte('\n')
			}
			continue
		}

		writeName(w, props)
		if !w.compact {
			w.WriteByte(' ')
		}
		if b, ok := fv.Interface().(raw); ok {
			writeRaw(w, b.Bytes())
			continue
		}
		if props.Enum != "" && tryWriteEnum(w, props.Enum, fv) {
			// Enum written.
		} else {
			writeAny(w, fv, props)
		}
		w.WriteByte('\n')
	}

	// Extensions (the XXX_extensions field).
	pv := sv.Addr()
	if pv.Type().Implements(extendableProtoType) {
		writeExtensions(w, pv)
	}
}

// writeRaw writes an uninterpreted raw message.
func writeRaw(w *textWriter, b []byte) {
	w.WriteByte('<')
	if !w.compact {
		w.WriteByte('\n')
	}
	w.indent()
	writeUnknownStruct(w, b)
	w.unindent()
	w.WriteByte('>')
}

// tryWriteEnum attempts to write an enum value as a symbolic constant.
// If the enum is unregistered, nothing is written and false is returned.
func tryWriteEnum(w *textWriter, enum string, v reflect.Value) bool {
	v = reflect.Indirect(v)
	if v.Type().Kind() != reflect.Int32 {
		return false
	}
	m, ok := enumNameMaps[enum]
	if !ok {
		return false
	}
	str, ok := m[int32(v.Int())]
	if !ok {
		return false
	}
	fmt.Fprintf(w, str)
	return true
}

// writeAny writes an arbitrary field.
func writeAny(w *textWriter, v reflect.Value, props *Properties) {
	v = reflect.Indirect(v)

	// We don't attempt to serialise every possible value type; only those
	// that can occur in protocol buffers, plus a few extra that were easy.
	switch v.Kind() {
	case reflect.Slice:
		// Should only be a []byte; repeated fields are handled in writeStruct.
		writeString(w, string(v.Interface().([]byte)))
	case reflect.String:
		writeString(w, v.String())
	case reflect.Struct:
		// Required/optional group/message.
		var bra, ket byte = '<', '>'
		if props != nil && props.Wire == "group" {
			bra, ket = '{', '}'
		}
		w.WriteByte(bra)
		if !w.compact {
			w.WriteByte('\n')
		}
		w.indent()
		writeStruct(w, v)
		w.unindent()
		w.WriteByte(ket)
	default:
		fmt.Fprint(w, v.Interface())
	}
}

// equivalent to C's isprint.
func isprint(c byte) bool {
	return c >= 0x20 && c < 0x7f
}

// writeString writes a string in the protocol buffer text format.
// It is similar to strconv.Quote except we don't use Go escape sequences,
// we treat the string as a byte sequence, and we use octal escapes.
// These differences are to maintain interoperability with the other
// languages' implementations of the text format.
func writeString(w *textWriter, s string) {
	w.WriteByte('"')

	// Loop over the bytes, not the runes.
	for i := 0; i < len(s); i++ {
		// Divergence from C++: we don't escape apostrophes.
		// There's no need to escape them, and the C++ parser
		// copes with a naked apostrophe.
		switch c := s[i]; c {
		case '\n':
			w.Write([]byte{'\\', 'n'})
		case '\r':
			w.Write([]byte{'\\', 'r'})
		case '\t':
			w.Write([]byte{'\\', 't'})
		case '"':
			w.Write([]byte{'\\', '"'})
		case '\\':
			w.Write([]byte{'\\', '\\'})
		default:
			if isprint(c) {
				w.WriteByte(c)
			} else {
				fmt.Fprintf(w, "\\%03o", c)
			}
		}
	}

	w.WriteByte('"')
}

func writeMessageSet(w *textWriter, ms *MessageSet) {
	for _, item := range ms.Item {
		id := *item.TypeId
		if msd, ok := messageSetMap[id]; ok {
			// Known message set type.
			fmt.Fprintf(w, "[%s]: <\n", msd.name)
			w.indent()

			pb := reflect.New(msd.t.Elem())
			if err := Unmarshal(item.Message, pb.Interface().(Message)); err != nil {
				fmt.Fprintf(w, "/* bad message: %v */\n", err)
			} else {
				writeStruct(w, pb.Elem())
			}
		} else {
			// Unknown type.
			fmt.Fprintf(w, "[%d]: <\n", id)
			w.indent()
			writeUnknownStruct(w, item.Message)
		}
		w.unindent()
		w.Write([]byte(">\n"))
	}
}

func writeUnknownStruct(w *textWriter, data []byte) {
	if !w.compact {
		fmt.Fprintf(w, "/* %d unknown bytes */\n", len(data))
	}
	b := NewBuffer(data)
	for b.index < len(b.buf) {
		x, err := b.DecodeVarint()
		if err != nil {
			fmt.Fprintf(w, "/* %v */\n", err)
			return
		}
		wire, tag := x&7, x>>3
		if wire == WireEndGroup {
			w.unindent()
			w.Write([]byte("}\n"))
			continue
		}
		fmt.Fprintf(w, "tag%d", tag)
		if wire != WireStartGroup {
			w.WriteByte(':')
		}
		if !w.compact || wire == WireStartGroup {
			w.WriteByte(' ')
		}
		switch wire {
		case WireBytes:
			buf, err := b.DecodeRawBytes(false)
			if err == nil {
				fmt.Fprintf(w, "%q", buf)
			} else {
				fmt.Fprintf(w, "/* %v */", err)
			}
		case WireFixed32:
			x, err := b.DecodeFixed32()
			writeUnknownInt(w, x, err)
		case WireFixed64:
			x, err := b.DecodeFixed64()
			writeUnknownInt(w, x, err)
		case WireStartGroup:
			fmt.Fprint(w, "{")
			w.indent()
		case WireVarint:
			x, err := b.DecodeVarint()
			writeUnknownInt(w, x, err)
		default:
			fmt.Fprintf(w, "/* unknown wire type %d */", wire)
		}
		w.WriteByte('\n')
	}
}

func writeUnknownInt(w *textWriter, x uint64, err error) {
	if err == nil {
		fmt.Fprint(w, x)
	} else {
		fmt.Fprintf(w, "/* %v */", err)
	}
}

type int32Slice []int32

func (s int32Slice) Len() int           { return len(s) }
func (s int32Slice) Less(i, j int) bool { return s[i] < s[j] }
func (s int32Slice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// writeExtensions writes all the extensions in pv.
// pv is assumed to be a pointer to a protocol message struct that is extendable.
func writeExtensions(w *textWriter, pv reflect.Value) {
	emap := extensionMaps[pv.Type().Elem()]
	ep := pv.Interface().(extendableProto)

	// Order the extensions by ID.
	// This isn't strictly necessary, but it will give us
	// canonical output, which will also make testing easier.
	m := ep.ExtensionMap()
	ids := make([]int32, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Sort(int32Slice(ids))

	for _, extNum := range ids {
		ext := m[extNum]
		var desc *ExtensionDesc
		if emap != nil {
			desc = emap[extNum]
		}
		if desc == nil {
			// Unknown extension.
			writeUnknownStruct(w, ext.enc)
			continue
		}

		pb, err := GetExtension(ep, desc)
		if err != nil {
			fmt.Fprintln(os.Stderr, "proto: failed getting extension: ", err)
			continue
		}

		// Repeated extensions will appear as a slice.
		if !desc.repeated() {
			writeExtension(w, desc.Name, pb)
		} else {
			v := reflect.ValueOf(pb)
			for i := 0; i < v.Len(); i++ {
				writeExtension(w, desc.Name, v.Index(i).Interface())
			}
		}
	}
}

func writeExtension(w *textWriter, name string, pb interface{}) {
	fmt.Fprintf(w, "[%s]:", name)
	if !w.compact {
		w.WriteByte(' ')
	}
	writeAny(w, reflect.ValueOf(pb), nil)
	w.WriteByte('\n')
}

func marshalText(w io.Writer, pb Message, compact bool) {
	if pb == nil {
		w.Write([]byte("<nil>"))
		return
	}
	aw := new(textWriter)
	aw.writer = w
	aw.complete = true
	aw.compact = compact

	// Dereference the received pointer so we don't have outer < and >.
	v := reflect.Indirect(reflect.ValueOf(pb))
	writeStruct(aw, v)
}

// MarshalText writes a given protocol buffer in text format.
func MarshalText(w io.Writer, pb Message) { marshalText(w, pb, false) }

// MarshalTextString is the same as MarshalText, but returns the string directly.
func MarshalTextString(pb Message) string {
	var buf bytes.Buffer
	marshalText(&buf, pb, false)
	return buf.String()
}

// CompactText writes a given protocol buffer in compact text format (one line).
func CompactText(w io.Writer, pb Message) { marshalText(w, pb, true) }

// CompactTextString is the same as CompactText, but returns the string directly.
func CompactTextString(pb Message) string {
	var buf bytes.Buffer
	marshalText(&buf, pb, true)
	return buf.String()
}
