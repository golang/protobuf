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

// Functions for writing the Text protocol buffer format.
// TODO:
//	- Message sets, groups.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// An io.Writer wrapper that tracks its indentation level.
type textWriter struct {
	indent_level int
	complete     bool // if the current position is a complete line
	compact      bool // whether to write out as a one-liner
	writer       io.Writer
}

func (w *textWriter) Write(p []byte) (n int, err os.Error) {
	n, err = len(p), nil

	frags := strings.Split(string(p), "\n", -1)
	if w.compact {
		w.writer.Write([]byte(strings.Join(frags, " ")))
		return
	}

	for i := 0; i < len(frags); i++ {
		if w.complete {
			for j := 0; j < w.indent_level; j++ {
				w.writer.Write([]byte{' ', ' '})
			}
			w.complete = false
		}

		w.writer.Write([]byte(frags[i]))
		if i+1 < len(frags) {
			w.writer.Write([]byte{'\n'})
		}
	}
	w.complete = len(frags[len(frags)-1]) == 0

	return
}

func (w *textWriter) indent() { w.indent_level++ }

func (w *textWriter) unindent() {
	if w.indent_level == 0 {
		fmt.Fprintln(os.Stderr, "proto: textWriter unindented too far!")
	} else {
		w.indent_level--
	}
}

func writeStruct(w *textWriter, sv reflect.Value) {
	st := sv.Type()
	sprops := GetProperties(st)
	for i := 0; i < sv.NumField(); i++ {
		if strings.HasPrefix(st.Field(i).Name, "XXX_") {
			continue
		}
		props := sprops.Prop[i]
		fv := sv.Field(i)
		if pv := fv; pv.Kind() == reflect.Ptr && pv.IsNil() {
			// Field not filled in. This could be an optional field or
			// a required field that wasn't filled in. Either way, there
			// isn't anything we can show for it.
			continue
		}
		if av := fv; av.Kind() == reflect.Slice && av.IsNil() {
			// Repeated field that is empty, or a bytes field that is unused.
			continue
		}

		if props.Repeated {
			if av := fv; av.Kind() == reflect.Slice {
				// Repeated field.
				for j := 0; j < av.Len(); j++ {
					fmt.Fprintf(w, "%v:", props.OrigName)
					if !w.compact {
						w.Write([]byte{' '})
					}
					writeAny(w, av.Index(j))
					fmt.Fprint(w, "\n")
				}
				continue
			}
		}

		fmt.Fprintf(w, "%v:", props.OrigName)
		if !w.compact {
			w.Write([]byte{' '})
		}
		if len(props.Enum) == 0 || !tryWriteEnum(w, props.Enum, fv) {
			writeAny(w, fv)
		}
		fmt.Fprint(w, "\n")
	}
}

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

func writeAny(w *textWriter, v reflect.Value) {
	v = reflect.Indirect(v)

	// We don't attempt to serialise every possible value type; only those
	// that can occur in protocol buffers, plus a few extra that were easy.
	switch val := v; val.Kind() {
	case reflect.Slice:
		// Should only be a []byte; repeated fields are handled in writeStruct.
		// TODO: Handle other cases more cleanly.
		bytes := make([]byte, val.Len())
		for i := 0; i < val.Len(); i++ {
			bytes[i] = byte(val.Index(i).Uint())
		}
		// TODO: Should be strconv.QuoteC, which doesn't exist yet
		fmt.Fprint(w, strconv.Quote(string(bytes)))
	case reflect.String:
		// TODO: Should be strconv.QuoteC, which doesn't exist yet
		fmt.Fprint(w, strconv.Quote(val.String()))
	case reflect.Struct:
		// Required/optional group/message.
		// TODO: groups use { } instead of < >, and no colon.
		if !w.compact {
			fmt.Fprint(w, "<\n")
		} else {
			fmt.Fprint(w, "<")
		}
		w.indent()
		writeStruct(w, val)
		w.unindent()
		fmt.Fprint(w, ">")
	default:
		fmt.Fprint(w, val.Interface())
	}
}

func marshalText(w io.Writer, pb interface{}, compact bool) {
	if pb == nil {
		w.Write([]byte("<nil>"))
		return
	}
	aw := new(textWriter)
	aw.writer = w
	aw.complete = true
	aw.compact = compact

	v := reflect.NewValue(pb)
	// We should normally be passed a struct, or a pointer to a struct,
	// and we don't want the outer < and > in that case.
	v = reflect.Indirect(v)
	if sv := v; sv.Kind() == reflect.Struct {
		writeStruct(aw, sv)
	} else {
		writeAny(aw, v)
	}
}

// MarshalText writes a given protobuffer in Text format.
// Non-protobuffers can also be written, but their formatting is not guaranteed.
func MarshalText(w io.Writer, pb interface{}) { marshalText(w, pb, false) }

// CompactText writes a given protobuffer in compact Text format (one line).
// Non-protobuffers can also be written, but their formatting is not guaranteed.
func CompactText(w io.Writer, pb interface{}) { marshalText(w, pb, true) }

// CompactTextString is the same as CompactText, but returns the string directly.
func CompactTextString(pb interface{}) string {
	buf := new(bytes.Buffer)
	marshalText(buf, pb, true)
	return buf.String()
}
