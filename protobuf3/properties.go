// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2016 Mist Systems. All rights reserved.
//
// This code is derived from earlier code which was itself:
//
// Copyright 2010 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
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

package protobuf3

/*
 * Routines for encoding data into the wire format for protocol buffers.
 */

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

// compile with true to get some debug msgs when working on this file
const debug bool = false

// XXXHack enables a backwards compatibility hack to match the canonical golang.go/protobuf error behavior for fields whose names start with XXX_
// This isn't needed unless you are dealing with old protobuf v2 generated types like some unit tests do
var XXXHack = false

// MakeFieldName is a pointer to a function which returns what should be the name of field f in the protobuf definition of type t.
// You can replace this with your own function before calling AsProtobuf[Full]() to control the field names yourself.
var MakeFieldName func(f string, t reflect.Type) string = MakeLowercaseFieldName

// MakeTypeName is a pointer to a function which returns what should be the name of the protobuf message of type t, which is the type
// of a field named f.
var MakeTypeName func(t reflect.Type, f string) string = MakeUppercaseTypeName

var MakePackageName func(pkgpath string) string = MakeSamePackageName

// AsProtobuf3er is the interface which returns the protobuf v3 type equivalent to what the MarshalProtobuf3() method
// encodes. This is optional, but useful when using AsProtobufFull() against types implementing Marshaler.
type AsProtobuf3er interface {
	AsProtobuf3() (name string, definition string)
}

// maxLen is the maximum length possible for a byte array. On a 64-bit target this is (1<<50)-1. On a 32-bit target it is (1<<31)-1
// The tricky part is figuring out in a constant what flavor of target we are on. I could sure use a ?: here. It would be more
// clear than using &^uint(0) to truncate (or not) the upper 32 bits of a constant.
const maxLen = int((1 << (31 + (((50-31)<<32)&uint64(^uint(0)))>>32)) - 1) // experiments with go1.7 on amd64 show any larger size causes the compiler to error

// Constants that identify the encoding of a value on the wire.
const (
	WireVarint     = WireType(0)
	WireFixed64    = WireType(1)
	WireBytes      = WireType(2)
	WireStartGroup = WireType(3) // legacy from protobuf v2. Groups are not used in protobuf v3
	WireEndGroup   = WireType(4) // legacy...
	WireFixed32    = WireType(5)
)

type WireType byte

// mapping from WireType to string
var wireTypeNames = []string{WireVarint: "varint", WireFixed64: "fixed64", WireBytes: "bytes", WireStartGroup: "start-group", WireEndGroup: "end-group", WireFixed32: "fixed32"}

func (wt WireType) String() string {
	if int(wt) < len(wireTypeNames) {
		return wireTypeNames[wt]
	}
	return fmt.Sprintf("WireType(%d)", byte(wt))
}

// Encoders are defined in encode.go
// An encoder outputs the full representation of a field, including its
// tag and encoder type.
type encoder func(p *Buffer, prop *Properties, base unsafe.Pointer)

// A valueEncoder encodes a single integer in a particular encoding.
type valueEncoder func(o *Buffer, x uint64)

// Decoders are defined in decode.go
// A decoder creates a value from its wire representation.
// Unrecognized subelements are saved in unrec.
type decoder func(p *Buffer, prop *Properties, base unsafe.Pointer) error

// A valueDecoder decodes a single integer in a particular encoding.
type valueDecoder func(o *Buffer) (x uint64, err error)

// StructProperties represents properties for all the fields of a struct.
type StructProperties struct {
	props []Properties // properties for each field encoded in protobuf, ordered by tag id
}

// Implement the sorting interface so we can sort the fields in tag order, as recommended by the spec.
// See encode.go, (*Buffer).enc_struct.
func (sp *StructProperties) Len() int { return len(sp.props) }
func (sp *StructProperties) Less(i, j int) bool {
	return sp.props[i].Tag < sp.props[j].Tag
}
func (sp *StructProperties) Swap(i, j int) { sp.props[i], sp.props[j] = sp.props[j], sp.props[i] }

// returns the properties into protobuf v3 format, suitable for feeding back into the protobuf compiler.
func (sp *StructProperties) asProtobuf(t reflect.Type, tname string) string {
	lines := []string{fmt.Sprintf("message %s {", tname)}
	for i := range sp.props {
		pp := &sp.props[i]
		if pp.Wire != "-" {
			lines = append(lines, fmt.Sprintf("  %s %s = %d;", pp.asProtobuf, pp.protobufFieldName(t), pp.Tag))
		}
	}
	lines = append(lines, "}")
	return strings.Join(lines, "\n")
}

// return the name of this field in protobuf
func (p *Properties) protobufFieldName(struct_type reflect.Type) string {
	// the "name=" tag overrides any computed field name. That lets us automate any manual fixup of names we might need.
	for _, t := range strings.Split(p.Wire, ",") {
		if strings.HasPrefix(t, "name=") {
			return t[5:]
		}
	}

	return MakeFieldName(p.Name, struct_type)
}

// MakeLowercaseFieldName returns a reasonable lowercase field name
func MakeLowercaseFieldName(f string, t reflect.Type) string {
	// To make people who use other languages happy it would be nice if our field names were like most and were lowercase.
	// (In addition, since we use the name of fields with anonymous types as the name of the anonmymous types, we need to
	// alter those fields (or the type's name) so there isn't a collision.)
	// Converting "XxxYYzz" to "xxx_yyy_zz" seems to be reasonable for most fields names.
	// If the name already has any '_' it then I just lowercase it without inserting any more.

	if strings.IndexRune(f, '_') >= 0 {
		return strings.ToLower(f)
	}

	buf := make([]byte, 2*len(f)+4) // 2x is enough for every 2nd rune to be a '_'. +4 is enough room for anything EncodeRune() might emit
	j := 0
	prev_was_upper := true // initial condition happens to prevent the 1st rune (which is almost certainly uppercase) from getting prefixed with _
	for _, r := range f {
		if unicode.IsUpper(r) {
			// lowercase r, and prepend a '_' if this is a good place to break up the name
			if !prev_was_upper {
				buf[j] = '_'
				j++
			}
			r = unicode.ToLower(r)
			prev_was_upper = true
		} else if unicode.IsLower(r) {
			prev_was_upper = false
		} // else leave prev_was_upper alone. This rule handles some edge condition names better ("L2TP" for instance, which otherwise would be named "l2_tp")
		j += utf8.EncodeRune(buf[j:], r)
	}

	return string(buf[:j])

	// PS I tried doing things like lowercasing and inserting a '_' before each group of uppercase chars.
	// It didn't do well with field names our software was using. Yet
}

// returns the type expressed in protobuf v3 format, suitable for feeding back into the protobuf compiler.
func AsProtobuf(t reflect.Type) string {
	// dig down through any pointer types
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	prop, err := GetProperties(t)
	if err != nil {
		return "# " + err.Error() // cause an error in the protobuf compiler
	}
	return prop.asProtobuf(t, t.Name())
}

// given the full path of the package of the 1st type passed to AsProtobufFull(), return
// the last component of the package path to be used as the package name.
func MakeSamePackageName(pkgpath string) string {
	slash := strings.LastIndexByte(pkgpath, '/')
	pkg := pkgpath
	if slash >= 0 {
		pkg = pkgpath[slash+1:]
	}
	return pkg
}

// returns the type expressed in protobuf v3 format, including all dependent types and imports
func AsProtobufFull(t reflect.Type, more ...reflect.Type) string {
	// dig down through any pointer types on the first type, since we'll use that one to determine the package
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	todo := make(map[reflect.Type]struct{})
	discovered := make(map[reflect.Type]struct{})

	pkgpath := t.PkgPath()

	headers := []string{
		fmt.Sprintf("// protobuf definitions generated by protobuf3.AsProtobufFull(%s.%s)", pkgpath, t.Name()),
		"",
		`syntax = "proto3";`,
		"",
	}
	var body []string

	if pkgpath != "" {
		headers = append(headers, fmt.Sprintf("package %s;", MakePackageName(pkgpath)))
	} // else the type is synthesized and lacks a path; humans need to deal with the output (after all they caused this)

	// place all the arguments in the todo table to start things off
	todo[t] = struct{}{}
	for _, t := range more {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		todo[t] = struct{}{}
	}

	// and lather/rinse/repeat until we've discovered all the types
	for len(todo) != 0 {
		for t := range todo {
			// move t from todo to discovered
			delete(todo, t)
			discovered[t] = struct{}{}

			// add to todo any new, non-anonymous types used by struct t's fields
			p, err := GetProperties(t)
			if err != nil {
				body = append(body, "# "+err.Error()) // cause an error in the protobuf compiler
				continue
			}
			for i := range p.props {
				pp := &p.props[i]
				tt := pp.Subtype()
				if tt != nil {
					if _, ok := discovered[tt]; !ok {
						// it's a new type of field
						switch {
						case pp.isMarshaler:
							// we can't recurse further into a custom type
							discovered[tt] = struct{}{}
						case tt.Kind() == reflect.Struct:
							switch tt {
							case time_Time_type, time_Duration_type:
								// the timestamp and duration types get defined by an import of timestamp.proto
								discovered[tt] = struct{}{}
							default:
								// put this new type in the todo table if it isn't already there
								// (the duplicate insert when it is already present is a no-op)
								todo[tt] = struct{}{}
							}
						}
					}
				}
			}

			// and we must break since todo has possibly been altered
			break
		}
	}

	// now that the types we need have all been discovered, sort their names and generate the .proto source
	// the reason we do this in 2 passes is so that the output is consistent from run to run, and diff'able
	// across runs with incremental differences.

	ordered := make(Types, 0, len(discovered))
	for t := range discovered {
		if t.Name() != "" { // skip anonymous types
			ordered = append(ordered, t)
		}
	}
	sort.Sort(ordered)

	for _, t := range ordered {
		// generate type t's protobuf definition

		switch {
		case t == time_Time_type:
			// the timestamp type gets defined by an import
			headers = append(headers, `import "google/protobuf/timestamp.proto";`)

		case t == time_Duration_type:
			// the duration type gets defined by an import
			headers = append(headers, `import "google/protobuf/duration.proto";`)

		case isMarshaler(reflect.PtrTo(t)):
			// we can't define a custom type automatically. see if it can tell us, and otherwise remind the human to do it.
			it := reflect.New(t).Interface()
			if aper, ok := it.(AsProtobuf3er); ok {
				_, definition := aper.AsProtobuf3()
				if definition != "" {
					body = append(body, "") // put a blank line between each message definition
					body = append(body, definition)
				} // else the type doesn't need any additional definition (its name was sufficient)
			} else {
				headers = append(headers, fmt.Sprintf("// TODO supply the definition of message %s", t.Name()))
			}

		default:
			// save t's definition
			body = append(body, "") // put a blank line between each message definition
			body = append(body, AsProtobuf(t))
		}
	}

	headers = append(headers, "")

	return strings.Join(append(headers, body...), "\n")
}

type Types []reflect.Type

func (ts Types) Len() int           { return len(ts) }
func (ts Types) Swap(i, j int)      { ts[i], ts[j] = ts[j], ts[i] }
func (ts Types) Less(i, j int) bool { return ts[i].Name() < ts[j].Name() } // sort types by their names

// Properties represents the protocol-specific behavior of a single struct field.
type Properties struct {
	Name       string // name of the field, for error messages
	Wire       string
	asProtobuf string // protobuf v3 type for this field (or something equivalent, since we can't figure it out perfectly from the Go field type and tags)
	Tag        uint32
	WireType   WireType

	enc         encoder
	valEnc      valueEncoder      // set for bool and numeric types only
	offset      uintptr           // byte offset of this field within the struct
	tagcode     string            // encoding of EncodeVarint((Tag<<3)|WireType), stored in a string for efficiency
	stype       reflect.Type      // set for struct types only
	sprop       *StructProperties // set for struct types only
	isMarshaler bool

	mtype    reflect.Type // set for map types only
	mkeyprop *Properties  // set for map types only
	mvalprop *Properties  // set for map types only

	length int // set for array types only

	dec    decoder
	valDec valueDecoder // set for bool and numeric types only
}

// String formats the properties in the protobuf struct field tag style.
func (p *Properties) String() string {
	if p.stype != nil {
		return fmt.Sprintf("%s %s (%s)", p.Wire, p.Name, p.stype.Name())
	}
	if p.mtype != nil {
		return fmt.Sprintf("%s %s (%s)", p.Wire, p.Name, p.mtype.Name())
	}
	return fmt.Sprintf("%s %s", p.Wire, p.Name)
}

// returns the inner type, or nil
func (p *Properties) Subtype() reflect.Type {
	return p.stype
}

// IntEncoder enumerates the different ways of encoding integers in Protobuf v3
type IntEncoder int

const (
	UnknownEncoder IntEncoder = iota // make the zero-value be different from any valid value so I can tell it is not set
	VarintEncoder
	Fixed32Encoder
	Fixed64Encoder
	Zigzag32Encoder
	Zigzag64Encoder
)

// Parse populates p by parsing a string in the protobuf struct field tag style.
func (p *Properties) Parse(s string) (IntEncoder, bool, error) {
	p.Wire = s

	// "bytes,49,rep,..."
	fields := strings.Split(s, ",")

	if len(fields) < 2 {
		if len(fields) > 0 && fields[0] == "-" {
			// `protobuf="-"` is used to mark fields which should be skipped by the protobuf encoder (this is same mark as is used by the std encoding/json package)
			return 0, true, nil
		}
		return 0, true, fmt.Errorf("protobuf3: tag of %q has too few fields: %q", p.Name, s)
	}

	var enc IntEncoder
	switch fields[0] {
	case "varint":
		p.valEnc = (*Buffer).EncodeVarint
		p.valDec = (*Buffer).DecodeVarint
		p.WireType = WireVarint
		enc = VarintEncoder
	case "fixed32":
		p.valEnc = (*Buffer).EncodeFixed32
		p.valDec = (*Buffer).DecodeFixed32
		p.WireType = WireFixed32
		enc = Fixed32Encoder
	case "fixed64":
		p.valEnc = (*Buffer).EncodeFixed64
		p.valDec = (*Buffer).DecodeFixed64
		p.WireType = WireFixed64
		enc = Fixed64Encoder
	case "zigzag32":
		p.valEnc = (*Buffer).EncodeZigzag32
		p.valDec = (*Buffer).DecodeZigzag32
		p.WireType = WireVarint
		enc = Zigzag32Encoder
	case "zigzag64":
		p.valEnc = (*Buffer).EncodeZigzag64
		p.valDec = (*Buffer).DecodeZigzag64
		p.WireType = WireVarint
		enc = Zigzag64Encoder
	case "bytes":
		// no numeric converter for non-numeric types
		p.WireType = WireBytes
	default:
		return 0, false, fmt.Errorf("protobuf3: tag of %q has unknown wire type: %q", p.Name, s)
	}

	tag, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, false, fmt.Errorf("protobuf3: tag id of %q invalid: %s: %s", p.Name, s, err.Error())
	}
	if tag <= 0 { // catch any negative or 0 values
		return 0, false, fmt.Errorf("protobuf3: tag id of %q out of range: %s", p.Name, s)
	}
	p.Tag = uint32(tag)

	// and we don't care about any other fields
	// (if you don't mark slices/arrays/maps with ",rep" that's your own problem; this encoder always repeats those types)

	return enc, false, nil
}

// Initialize the fields for encoding and decoding.
func (p *Properties) setEncAndDec(t1 reflect.Type, f *reflect.StructField, int_encoder IntEncoder) error {
	var err error
	p.enc = nil
	p.dec = nil
	wire := p.WireType

	// since so many cases need it, decode int_encoder into a  string now
	var int32_encoder_txt, uint32_encoder_txt,
		int64_encoder_txt, uint64_encoder_txt string
	switch int_encoder {
	case VarintEncoder:
		uint32_encoder_txt = "uint32"
		int32_encoder_txt = uint32_encoder_txt[1:] // strip the 'u' off
		uint64_encoder_txt = "uint64"
		int64_encoder_txt = uint64_encoder_txt[1:] // strip the 'u' off
	case Fixed32Encoder:
		int32_encoder_txt = "sfixed32"
		uint32_encoder_txt = int32_encoder_txt[1:] // strip the 's' off
	case Fixed64Encoder:
		int64_encoder_txt = "sfixed64"
		uint64_encoder_txt = int64_encoder_txt[1:] // strip the 's' off
	case Zigzag32Encoder:
		int32_encoder_txt = "sint32"
	case Zigzag64Encoder:
		int64_encoder_txt = "sint64"
	}

	// can t1 marshal itself?
	if isMarshaler(reflect.PtrTo(t1)) {
		p.isMarshaler = true
		p.stype = t1
		p.enc = (*Buffer).enc_marshaler
		p.dec = (*Buffer).dec_marshaler
		p.asProtobuf = p.stypeAsProtobuf()
	} else {
		switch t1.Kind() {
		default:
			return fmt.Errorf("protobuf3: no encoder/decoder for type %s", t1.Name())

		// proto3 scalar types

		case reflect.Bool:
			p.enc = (*Buffer).enc_bool
			p.dec = (*Buffer).dec_bool
			p.asProtobuf = "bool"
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Int:
			p.enc = (*Buffer).enc_int
			p.dec = (*Buffer).dec_int
			p.asProtobuf = int32_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Uint:
			p.enc = (*Buffer).enc_uint
			p.dec = (*Buffer).dec_int // signness doesn't matter when decoding. either the top bit is set or it isn't
			p.asProtobuf = uint32_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Int8:
			p.enc = (*Buffer).enc_int8
			p.dec = (*Buffer).dec_int8
			p.asProtobuf = int32_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Uint8:
			p.enc = (*Buffer).enc_uint8
			p.dec = (*Buffer).dec_int8
			p.asProtobuf = uint32_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Int16:
			p.enc = (*Buffer).enc_int16
			p.dec = (*Buffer).dec_int16
			p.asProtobuf = int32_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Uint16:
			p.enc = (*Buffer).enc_uint16
			p.dec = (*Buffer).dec_int16
			p.asProtobuf = uint32_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Int32:
			p.enc = (*Buffer).enc_int32
			p.dec = (*Buffer).dec_int32
			p.asProtobuf = int32_encoder_txt
			if p.valEnc == nil { // note it is safe, though peculiar, for an int32 to have a wiretype of fixed64
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Uint32:
			p.enc = (*Buffer).enc_uint32
			p.dec = (*Buffer).dec_int32
			p.asProtobuf = uint32_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Int64:
			// this might be a time.Duration, or it might be an ordinary int64
			// if the caller wants a time.Duration to be encoded as a protobuf Duration then the
			// wiretype must be WireBytes. Otherwise they'll get the int64 encoding they've selected.
			if p.WireType == WireBytes && t1 == time_Duration_type {
				p.enc = (*Buffer).enc_time_Duration
				p.dec = (*Buffer).dec_time_Duration
				p.asProtobuf = "google.protobuf.Duration"
			} else {
				p.enc = (*Buffer).enc_int64
				p.dec = (*Buffer).dec_int64
				p.asProtobuf = int64_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			}
		case reflect.Uint64:
			p.enc = (*Buffer).enc_int64
			p.dec = (*Buffer).dec_int64
			p.asProtobuf = uint64_encoder_txt
			if p.valEnc == nil {
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Float32:
			p.enc = (*Buffer).enc_uint32 // can just treat them as bits
			p.dec = (*Buffer).dec_int32
			p.asProtobuf = "float"
			if p.valEnc == nil || wire != WireFixed32 { // the way we encode and decode float32 at the moment means we can only support fixed32
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.Float64:
			p.enc = (*Buffer).enc_int64 // can just treat them as bits
			p.dec = (*Buffer).dec_int64
			p.asProtobuf = "double"
			if p.valEnc == nil || wire != WireFixed64 { // the way we encode and decode float32 at the moment means we can only support fixed64
				return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
			}
		case reflect.String:
			p.enc = (*Buffer).enc_string
			p.dec = (*Buffer).dec_string
			p.asProtobuf = "string"

		case reflect.Struct:
			p.stype = t1
			p.sprop, err = getPropertiesLocked(t1)
			if err != nil {
				return err
			}
			p.asProtobuf = p.stypeAsProtobuf()
			switch t1 {
			case time_Time_type:
				p.enc = (*Buffer).enc_struct_message // time.Time encodes as a struct with 1 (made up) field
				p.dec = (*Buffer).dec_time_Time      // but it decodes with a custom function
			default:
				p.enc = (*Buffer).enc_struct_message
				p.dec = (*Buffer).dec_struct_message
			}

		case reflect.Ptr:
			t2 := t1.Elem()
			// can the target of the pointer marshal itself?
			if isMarshaler(t1) {
				p.stype = t2
				p.isMarshaler = true
				p.enc = (*Buffer).enc_ptr_marshaler
				p.dec = (*Buffer).dec_ptr_marshaler
				p.asProtobuf = p.stypeAsProtobuf()
				break
			}

			switch t2.Kind() {
			default:
				fmt.Fprintf(os.Stderr, "protobuf3: no encoder function for %s -> %s\n", t1, t2.Name())
				break
			case reflect.Bool:
				p.enc = (*Buffer).enc_ptr_bool
				p.dec = (*Buffer).dec_ptr_bool
				p.asProtobuf = "bool"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int32:
				p.enc = (*Buffer).enc_ptr_int32
				p.dec = (*Buffer).dec_ptr_int32
				p.asProtobuf = int32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Uint32:
				p.enc = (*Buffer).enc_ptr_uint32
				p.dec = (*Buffer).dec_ptr_int32
				p.asProtobuf = uint32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int64:
				if p.WireType == WireBytes && t2 == time_Duration_type {
					p.enc = (*Buffer).enc_ptr_time_Duration
					p.dec = (*Buffer).dec_ptr_time_Duration
					p.asProtobuf = "google.protobuf.Duration"
				} else {
					p.enc = (*Buffer).enc_ptr_int64
					p.dec = (*Buffer).dec_ptr_int64
					p.asProtobuf = int64_encoder_txt
					if p.valEnc == nil {
						return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
					}
				}
			case reflect.Uint64:
				p.enc = (*Buffer).enc_ptr_int64
				p.dec = (*Buffer).dec_ptr_int64
				p.asProtobuf = uint64_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Float32:
				p.enc = (*Buffer).enc_ptr_uint32 // can just treat them as bits
				p.dec = (*Buffer).dec_ptr_int32
				p.asProtobuf = "float"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Float64:
				p.enc = (*Buffer).enc_ptr_int64 // can just treat them as bits
				p.dec = (*Buffer).dec_ptr_int64
				p.asProtobuf = "double"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.String:
				p.enc = (*Buffer).enc_ptr_string
				p.dec = (*Buffer).dec_ptr_string
				p.asProtobuf = "string"
			case reflect.Struct:
				p.stype = t2
				p.sprop, err = getPropertiesLocked(t2)
				if err != nil {
					return err
				}
				p.asProtobuf = p.stypeAsProtobuf()
				p.enc = (*Buffer).enc_ptr_struct_message
				switch {
				case t2 == time_Time_type:
					p.dec = (*Buffer).dec_ptr_time_Time
				default:
					p.dec = (*Buffer).dec_ptr_struct_message
				}

				// what about *Array types? Fill them in when we need them.
			}

		case reflect.Slice:
			// can elements of the slice marshal themselves?
			t2 := t1.Elem()
			if isMarshaler(reflect.PtrTo(t2)) {
				p.isMarshaler = true
				p.stype = t2
				p.enc = (*Buffer).enc_slice_marshaler
				p.dec = (*Buffer).dec_slice_marshaler
				p.asProtobuf = "repeated " + p.stypeAsProtobuf()
				break
			}

			switch t2.Kind() {
			default:
				fmt.Fprintf(os.Stderr, "protobuf3: no slice encoder for %s = []%s\n", t1.Name(), t2.Name())
				break
			case reflect.Bool:
				p.enc = (*Buffer).enc_slice_packed_bool
				p.dec = (*Buffer).dec_slice_packed_bool
				wire = WireBytes // packed=true is implied in protobuf v3
				p.asProtobuf = "repeated bool"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int:
				p.enc = (*Buffer).enc_slice_packed_int
				p.dec = (*Buffer).dec_slice_packed_int
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + int32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Uint:
				p.enc = (*Buffer).enc_slice_packed_uint
				p.dec = (*Buffer).dec_slice_packed_int
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + uint32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int8:
				p.enc = (*Buffer).enc_slice_packed_int8
				p.dec = (*Buffer).dec_slice_packed_int8
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + int32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Uint8:
				p.enc = (*Buffer).enc_slice_byte
				p.dec = (*Buffer).dec_slice_byte
				wire = WireBytes // packed=true... even for integers
				p.asProtobuf = "bytes"
			case reflect.Int16:
				p.enc = (*Buffer).enc_slice_packed_int16
				p.dec = (*Buffer).dec_slice_packed_int16
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + int32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Uint16:
				p.enc = (*Buffer).enc_slice_packed_uint16
				p.dec = (*Buffer).dec_slice_packed_int16
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + uint32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int32:
				p.enc = (*Buffer).enc_slice_packed_int32
				p.dec = (*Buffer).dec_slice_packed_int32
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + int32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Uint32:
				p.enc = (*Buffer).enc_slice_packed_uint32
				p.dec = (*Buffer).dec_slice_packed_int32
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + uint32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int64:
				if p.WireType == WireBytes && t2 == time_Duration_type {
					p.enc = (*Buffer).enc_slice_time_Duration
					p.dec = (*Buffer).dec_slice_time_Duration
					p.asProtobuf = "repeated google.protobuf.Duration"
				} else {
					p.enc = (*Buffer).enc_slice_packed_int64
					p.dec = (*Buffer).dec_slice_packed_int64
					wire = WireBytes // packed=true...
					p.asProtobuf = "repeated " + int64_encoder_txt
					if p.valEnc == nil {
						return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
					}
				}
			case reflect.Uint64:
				p.enc = (*Buffer).enc_slice_packed_int64
				p.dec = (*Buffer).dec_slice_packed_int64
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + int64_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Float32:
				// can just treat them as bits
				p.enc = (*Buffer).enc_slice_packed_uint32
				p.dec = (*Buffer).dec_slice_packed_int32
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated float"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Float64:
				// can just treat them as bits
				p.enc = (*Buffer).enc_slice_packed_int64
				p.dec = (*Buffer).dec_slice_packed_int64
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated double"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.String:
				p.enc = (*Buffer).enc_slice_string
				p.dec = (*Buffer).dec_slice_string
				p.asProtobuf = "repeated string"
			case reflect.Struct:
				p.stype = t2
				p.sprop, err = getPropertiesLocked(t2)
				if err != nil {
					return err
				}
				p.isMarshaler = isMarshaler(reflect.PtrTo(t2))
				p.enc = (*Buffer).enc_slice_struct_message
				p.dec = (*Buffer).dec_slice_struct_message
				p.asProtobuf = "repeated " + p.stypeAsProtobuf()
			case reflect.Ptr:
				switch t3 := t2.Elem(); t3.Kind() {
				default:
					fmt.Fprintf(os.Stderr, "protobuf3: no ptr encoder for %s -> %s -> %s\n", t1.Name(), t2.Name(), t3.Name())
					break
				case reflect.Struct:
					p.stype = t3
					p.sprop, err = getPropertiesLocked(t3)
					if err != nil {
						return err
					}
					p.isMarshaler = isMarshaler(t2)
					p.enc = (*Buffer).enc_slice_ptr_struct_message
					p.dec = (*Buffer).dec_slice_ptr_struct_message
					p.asProtobuf = "repeated " + p.stypeAsProtobuf()
				}
			case reflect.Slice:
				switch t2.Elem().Kind() {
				default:
					fmt.Fprintf(os.Stderr, "protobuf3: no slice elem encoder for %s -> %s -> %s\n", t1.Name(), t2.Name(), t2.Elem().Name())
					break
				case reflect.Uint8:
					p.enc = (*Buffer).enc_slice_slice_byte
					p.dec = (*Buffer).dec_slice_slice_byte
					p.asProtobuf = "repeated bytes"
				}
			}

		case reflect.Array:
			p.length = t1.Len()

			if p.length == 0 {
				// save checking the array length at encode-time by doing it now
				// a zero-length array will always encode as nothing
				p.enc = (*Buffer).enc_nothing
				// and a zero-length array need not have any decoder
				p.dec = nil
				break
			}

			t2 := t1.Elem()
			if isMarshaler(reflect.PtrTo(t2)) {
				// elements of the array can marshal themselves
				p.isMarshaler = true
				p.stype = t2
				p.enc = (*Buffer).enc_array_marshaler
				p.dec = (*Buffer).dec_array_marshaler
				p.asProtobuf = "repeated " + p.stypeAsProtobuf()
				break
			}

			switch t2.Kind() {
			default:
				fmt.Fprintf(os.Stderr, "protobuf3: no array encoder for %s = %s\n", t1.Name(), t2.Name())
				break
			case reflect.Bool:
				p.enc = (*Buffer).enc_array_packed_bool
				p.dec = (*Buffer).dec_array_packed_bool
				wire = WireBytes // packed=true is implied in protobuf v3
				p.asProtobuf = "repeated bool"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int32:
				p.enc = (*Buffer).enc_array_packed_int32
				p.dec = (*Buffer).dec_array_packed_int32
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + int32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Uint32:
				p.enc = (*Buffer).enc_array_packed_uint32
				p.dec = (*Buffer).dec_array_packed_int32
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + uint32_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Int64:
				if p.WireType == WireBytes && t2 == time_Duration_type {
					p.enc = (*Buffer).enc_array_time_Duration
					p.dec = (*Buffer).dec_array_time_Duration
					p.asProtobuf = "repeated google.protobuf.Duration"
				} else {
					p.enc = (*Buffer).enc_array_packed_int64
					p.dec = (*Buffer).dec_array_packed_int64
					wire = WireBytes // packed=true...
					p.asProtobuf = "repeated " + int64_encoder_txt
					if p.valEnc == nil {
						return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
					}
				}
			case reflect.Uint64:
				p.enc = (*Buffer).enc_array_packed_int64
				p.dec = (*Buffer).dec_array_packed_int64
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated " + uint64_encoder_txt
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Uint8:
				p.enc = (*Buffer).enc_array_byte
				p.dec = (*Buffer).dec_array_byte
				p.asProtobuf = "bytes"
			case reflect.Float32:
				// can just treat them as bits
				p.enc = (*Buffer).enc_array_packed_uint32
				p.dec = (*Buffer).dec_array_packed_int32
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated float"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.Float64:
				// can just treat them as bits
				p.enc = (*Buffer).enc_array_packed_int64
				p.dec = (*Buffer).dec_array_packed_int64
				wire = WireBytes // packed=true...
				p.asProtobuf = "repeated double"
				if p.valEnc == nil {
					return fmt.Errorf("protobuf3: %q %s cannot have wiretype %s", f.Name, t1, wire)
				}
			case reflect.String:
				p.enc = (*Buffer).enc_array_string
				p.dec = (*Buffer).dec_array_string
				p.asProtobuf = "repeated string"
			case reflect.Struct:
				p.stype = t2
				p.sprop, err = getPropertiesLocked(t2)
				if err != nil {
					return err
				}
				p.enc = (*Buffer).enc_array_struct_message
				p.dec = (*Buffer).dec_array_struct_message
				p.asProtobuf = "repeated " + p.stypeAsProtobuf()
			case reflect.Ptr:
				switch t3 := t2.Elem(); t3.Kind() {
				default:
					fmt.Fprintf(os.Stderr, "protobuf3: no ptr encoder for %s -> %s -> %s\n", t1.Name(), t2.Name(), t3.Name())
					break
				case reflect.Struct:
					p.stype = t3
					p.sprop, err = getPropertiesLocked(t3)
					if err != nil {
						return err
					}
					p.isMarshaler = isMarshaler(t2)
					p.enc = (*Buffer).enc_array_ptr_struct_message
					p.dec = (*Buffer).dec_array_ptr_struct_message
					p.asProtobuf = "repeated " + p.stypeAsProtobuf()
				}
			}

		case reflect.Map:
			p.enc = (*Buffer).enc_new_map
			p.dec = (*Buffer).dec_new_map

			p.mtype = t1
			p.mkeyprop = &Properties{}
			p.mkeyprop.init(reflect.PtrTo(p.mtype.Key()), "Key", f.Tag.Get("protobuf_key"), nil)
			p.mvalprop = &Properties{}

			vtype := p.mtype.Elem()
			if vtype.Kind() != reflect.Ptr && vtype.Kind() != reflect.Slice {
				// The value type is not a message (*T) or bytes ([]byte),
				// so we need encoders for the pointer to this type.
				vtype = reflect.PtrTo(vtype)
			}
			p.mvalprop.init(vtype, "Value", f.Tag.Get("protobuf_val"), nil)
			p.asProtobuf = fmt.Sprintf("map<%s, %s>", p.mkeyprop.asProtobuf, p.mvalprop.asProtobuf)
		}
	}

	// precalculate tag code
	x := p.Tag<<3 | uint32(wire)
	i := 0
	var tagbuf [8]byte
	for i = 0; x > 127; i++ {
		tagbuf[i] = 0x80 | uint8(x&0x7F)
		x >>= 7
	}
	tagbuf[i] = uint8(x)
	p.tagcode = string(tagbuf[0 : i+1])

	return nil
}

// using p.Name, p.stype and p.sprop, figure out the right name for the type of field p.
// if the name of the type is known, use that. Otherwise build a nested type and use it.
func (p *Properties) stypeAsProtobuf() string {
	// special case for time.Time and time.Duration (any other future special cases)
	switch p.sprop {
	case time_Time_sprop:
		return "google.protobuf.Timestamp"
		// note: there is no time.Duration case here because only struct types set .stype, and time.Duration is an int64
	}

	var name string

	// if the stype implements AsProtobuf3er and returns a type name, use that
	if reflect.PtrTo(p.stype).Implements(asprotobuffer3Type) {
		it := reflect.NewAt(p.stype, nil).Interface()
		if aper, ok := it.(AsProtobuf3er); ok {
			name, _ = aper.AsProtobuf3() // note AsProtobuf3() might return name "" anyway
		}
	}

	if name == "" {
		name = MakeTypeName(p.stype, p.Name)
	}

	if p.stype.Name() == "" {
		// p.stype is an anonymous type. define it inline with the enclosing message
		// we want the type definition to preceed the type's name, so that in the end it
		// formats something like:
		//   message Outer {
		//    `message Inner { ... }
		//     Inner' inner = 1;
		//   }
		// where the section in `' is the string we need to generate.
		lines := []string{p.sprop.asProtobuf(p.stype, name)}
		lines = append(lines, name)
		str := strings.Join(lines, "\n")
		// indent str two spaces to the right. we have to do this as a search step rather than as part of Join()
		// because the strings lines are already multi-line strings. (The other solutions are to indent as a
		// reformatting step at the end, or to store Properties.asProtobuf as []string and never loose the LFs.
		// The latter makes asProtobuf expensive for all the simple types. Reformatting needs to work on all fields.
		// So the "nasty" approach here is, AFAICS, for the best.
		name = strings.Replace(str, "\n", "\n  ", -1)
	}

	return name
}

// MakeUppercaseTypeName makes an uppercase message type name for type t, which is the type of a field named f.
// Since the field is visible to us it is public, and thus it is uppercase. And since the type is similarly visible
// it is almost certainly uppercased too. So there isn't much to do except pick whichever is appropriate.
func MakeUppercaseTypeName(t reflect.Type, f string) string {
	// if the Go type is named, a good start is to use the name of the go type
	// (even if it is in a different package than the enclosing type? that can cause collisions.
	//  for now the humans can sort those out after protoc errors on the duplicate records)
	n := t.Name()
	if n != "" {
		return n
	}

	// the struct has no typename. It is an anonymous type in Go. The equivalent in Protobuf is
	// a a nested type. It would be nice to use the name of the field as the name of the type,
	// since the name of the field ought to be unique within the enclosing struct type. However
	// protoc 3.0.2 cannot handle a field and a type having the same name. So we need to make up
	// a reasonable name for this type. I didn't like the result of appending "_msg" or other
	// 'uniquifier' to p.Name. So instead I've done the non-Go thing and made fields be lowercase,
	// thus reserving uppercase names for types, and thus avoiding any collisions.
	return f
}

var (
	marshalerType      = reflect.TypeOf((*Marshaler)(nil)).Elem()
	asprotobuffer3Type = reflect.TypeOf((*AsProtobuf3er)(nil)).Elem()
)

// isMarshaler reports whether type t implements Marshaler.
func isMarshaler(t reflect.Type) bool {
	return t.Implements(marshalerType)
}

// Init populates the properties from a protocol buffer struct tag.
func (p *Properties) init(typ reflect.Type, name, tag string, f *reflect.StructField) (bool, error) {
	// "bytes,49,opt,def=hello!"

	// fields without a protobuf tag are an error
	if tag == "" {
		// backwards compatibility HACK. canonical golang.org/protobuf ignores errors on fields with names that start with XXX_
		// we must do the same to pass their unit tests
		if XXXHack && strings.HasPrefix(name, "XXX_") {
			return true, nil
		}
		err := fmt.Errorf("protobuf3: %s (%s) lacks a protobuf tag. Tag it, or mark it with `protobuf:\"-\"` if it isn't intended to be marshaled to/from protobuf", name, typ.String())
		fmt.Fprintln(os.Stderr, err) // print the error too
		return true, err
	}

	p.Name = name
	if f != nil {
		p.offset = f.Offset
	}

	intencoder, skip, err := p.Parse(tag)
	if skip || err != nil {
		return skip, err
	}

	return false, p.setEncAndDec(typ, f, intencoder)
}

var (
	propertiesMu  sync.RWMutex
	propertiesMap = make(map[reflect.Type]*StructProperties)
)

// synthesize a StructProperties for time.Time which will encode it
// to the same as the standard protobuf3 Timestamp type.
var time_Time_type = reflect.TypeOf(time.Time{})
var time_Time_sprop = &StructProperties{
	props: []Properties{
		// we need just one made-up field with a .enc() method which we've hooked into
		Properties{
			Name:     "time.Time",
			WireType: WireBytes,
			enc:      (*Buffer).enc_time_Time,
			// note: .dec isn't used
		},
	},
}

// similarly for time.Duration ... standard protobuf3 Duration type. Note that because
// go time.Duration isn't a struct (it's a int64) there isn't a time_Duration_sprop at all.
var time_Duration_type = reflect.TypeOf(time.Duration(0))

func init() {
	propertiesMap[time_Time_type] = time_Time_sprop
}

// GetProperties returns the list of properties for the type represented by t.
// t must represent a generated struct type of a protocol message.
func GetProperties(t reflect.Type) (*StructProperties, error) {
	k := t.Kind()
	// accept a pointer-to-struct as well (but just one level)
	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	if k != reflect.Struct {
		panic("protobuf3: type must have kind struct")
	}

	// Most calls to GetProperties in a long-running program will be
	// retrieving details for types we have seen before.
	propertiesMu.RLock()
	sprop, ok := propertiesMap[t]
	propertiesMu.RUnlock()
	if ok {
		return sprop, nil
	}

	propertiesMu.Lock()
	sprop, err := getPropertiesLocked(t)
	propertiesMu.Unlock()
	return sprop, err
}

// getPropertiesLocked requires that propertiesMu is held.
func getPropertiesLocked(t reflect.Type) (*StructProperties, error) {
	if prop, ok := propertiesMap[t]; ok {
		return prop, nil
	}

	prop := new(StructProperties)

	// in case of recursion, add ourselves to propertiesMap now. we'll remove ourselves if we error
	propertiesMap[t] = prop

	// build properties
	nf := t.NumField()
	prop.props = make([]Properties, 0, nf)

	for i := 0; i < nf; i++ {
		f := t.Field(i)
		name := f.Name
		if name == "" && f.Anonymous {
			// use the type's name for embedded fields, like go does
			name = f.Type.Name()
			if name == "" {
				// use the type's unamed type
				name = f.Type.String()
			}
		}
		if name == "" {
			// unnamed embedded field types have no simple name
			name = "<unnamed field>"
		}

		tag := f.Tag.Get("protobuf")

		if tag == "embedded" && f.Anonymous {
			// field f is embedded in type t and has the special `protobuf:"embedded"` tag. Get f's fields and then merge them into t's
			fprop, err := getPropertiesLocked(f.Type)
			if err != nil {
				err := fmt.Errorf("protobuf3: Error preparing field %q of type %q: %v", name, t.Name(), err)
				fmt.Fprintln(os.Stderr, err) // print the error too
				delete(propertiesMap, t)
				return nil, err
			}

			// merge fprop's fields into prop
			for ii, p := range fprop.props {
				// fixup the field property as we copy them
				p.offset += f.Offset

				prop.props = append(prop.props, p)

				if debug {
					print(i, ".", ii, " ", name, " ", t.String(), " ")
					if p.Tag > 0 {
						print(p.String())
					}
					print("\n")
				}
			}

			continue
		}

		prop.props = append(prop.props, Properties{})
		p := &prop.props[len(prop.props)-1]

		skip, err := p.init(f.Type, name, tag, &f)
		if err != nil {
			err := fmt.Errorf("protobuf3: Error preparing field %q of type %q: %v", name, t.Name(), err)
			fmt.Fprintln(os.Stderr, err) // print the error too
			delete(propertiesMap, t)
			return nil, err
		}
		if skip {
			// silently skip this field. It's not part of the protobuf encoding of this struct
			prop.props = prop.props[:len(prop.props)-1] // remove it from properties
			continue
		}

		if debug {
			print(i, " ", name, " ", t.String(), " ")
			if p.Tag > 0 {
				print(p.String())
			}
			print("\n")
		}

		if p.enc == nil || p.dec == nil {
			tname := t.Name()
			if tname == "" {
				tname = "<anonymous struct>"
			}
			err := fmt.Errorf("protobuf3: Error no encoder or decoder for field %q.%q of type %q", tname, name, f.Type.Name())
			fmt.Fprintln(os.Stderr, err) // print the error too
			delete(propertiesMap, t)
			return nil, err
		}
	}

	// sort prop.props by tag, so we naturally encode in tag order as suggested by protobuf documentation
	sort.Sort(prop)
	if debug {
		for i := range prop.props {
			p := &prop.props[i]
			print("| ", t.Name(), ".", p.Name, "  ", p.WireType.String(), ",", p.Tag, "  offset=", p.offset, "\n")
		}
	}

	// now that they are sorted, sanity check for duplicate tags, since some of us are hand editing the tags
	prev_tag := uint32(0)
	for i := range prop.props {
		p := &prop.props[i]
		if prev_tag == p.Tag {
			err := fmt.Errorf("protobuf3: Error duplicate tag id %d assigned to %s.%s", p.Tag, t.String(), p.Name)
			fmt.Fprintln(os.Stderr, err) // print the error too
			delete(propertiesMap, t)
			return nil, err
		}
		prev_tag = p.Tag
	}

	return prop, nil
}
