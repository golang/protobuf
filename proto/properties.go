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
	"unsafe"
)

const debug bool = false

// Constants that identify the encoding of a value on the wire.
const (
	WireVarint     = 0
	WireFixed64    = 1
	WireBytes      = 2
	WireStartGroup = 3
	WireEndGroup   = 4
	WireFixed32    = 5
)

const startSize = 10 // initial slice/string sizes

// Encoders are defined in encoder.go
// An encoder outputs the full representation of a field, including its
// tag and encoder type.
type encoder func(p *Buffer, prop *Properties, base uintptr) os.Error

// A valueEncoder encodes a single integer in a particular encoding.
type valueEncoder func(o *Buffer, x uint64) os.Error

// Decoders are defined in decode.go
// A decoder creates a value from its wire representation.
// Unrecognized subelements are saved in unrec.
type decoder func(p *Buffer, prop *Properties, base uintptr, sbase uintptr) os.Error

// A valueDecoder decodes a single integer in a particular encoding.
type valueDecoder func(o *Buffer) (x uint64, err os.Error)

// StructProperties represents properties for all the fields of a struct.
type StructProperties struct {
	Prop      []*Properties  // properties for each field
	reqCount  int            // required count
	tags      map[int]int    // map from proto tag to struct field number
	origNames map[string]int // map from original name to struct field number
	order     []int          // list of struct field numbers in tag order
	nscratch  uintptr        // size of scratch space
}

// Implement the sorting interface so we can sort the fields in tag order, as recommended by the spec.
// See encoder.go, (*Buffer).enc_struct.

func (sp *StructProperties) Len() int { return len(sp.order) }
func (sp *StructProperties) Less(i, j int) bool {
	return sp.Prop[sp.order[i]].Tag < sp.Prop[sp.order[j]].Tag
}
func (sp *StructProperties) Swap(i, j int) { sp.order[i], sp.order[j] = sp.order[j], sp.order[i] }

// Properties represents the protocol-specific behavior of a single struct field.
type Properties struct {
	Name       string // name of the field, for error messages
	OrigName   string // original name before protocol compiler (always set)
	Wire       string
	WireType   int
	Tag        int
	Required   bool
	Optional   bool
	Repeated   bool
	Packed     bool   // relevant for repeated primitives only
	Enum       string // set for enum types only
	Default    string // default value
	def_uint64 uint64

	enc     encoder
	valEnc  valueEncoder // set for bool and numeric types only
	offset  uintptr
	tagcode []byte // encoding of EncodeVarint((Tag<<3)|WireType)
	tagbuf  [8]byte
	stype   reflect.Type

	dec     decoder
	valDec  valueDecoder // set for bool and numeric types only
	scratch uintptr
	sizeof  uintptr // calculations of scratch space
	alignof uintptr

	// If this is a packable field, this will be the decoder for the packed version of the field.
	packedDec decoder
}

// String formats the properties in the protobuf struct field tag style.
func (p *Properties) String() string {
	s := p.Wire
	s = ","
	s += strconv.Itoa(p.Tag)
	if p.Required {
		s += ",req"
	}
	if p.Optional {
		s += ",opt"
	}
	if p.Repeated {
		s += ",rep"
	}
	if p.Packed {
		s += ",packed"
	}
	if p.OrigName != p.Name {
		s += ",name=" + p.OrigName
	}
	if len(p.Enum) > 0 {
		s += ",enum=" + p.Enum
	}
	if len(p.Default) > 0 {
		s += ",def=" + p.Default
	}
	return s
}

// Parse populates p by parsing a string in the protobuf struct field tag style.
func (p *Properties) Parse(s string) {
	// "bytes,49,opt,def=hello!,name=foo"
	fields := strings.Split(s, ",") // breaks def=, but handled below.
	if len(fields) < 2 {
		fmt.Fprintf(os.Stderr, "proto: tag has too few fields: %q\n", s)
		return
	}

	p.Wire = fields[0]
	switch p.Wire {
	case "varint":
		p.WireType = WireVarint
		p.valEnc = (*Buffer).EncodeVarint
		p.valDec = (*Buffer).DecodeVarint
	case "fixed32":
		p.WireType = WireFixed32
		p.valEnc = (*Buffer).EncodeFixed32
		p.valDec = (*Buffer).DecodeFixed32
	case "fixed64":
		p.WireType = WireFixed64
		p.valEnc = (*Buffer).EncodeFixed64
		p.valDec = (*Buffer).DecodeFixed64
	case "zigzag32":
		p.WireType = WireVarint
		p.valEnc = (*Buffer).EncodeZigzag32
		p.valDec = (*Buffer).DecodeZigzag32
	case "zigzag64":
		p.WireType = WireVarint
		p.valEnc = (*Buffer).EncodeZigzag64
		p.valDec = (*Buffer).DecodeZigzag64
	case "bytes", "group":
		p.WireType = WireBytes
		// no numeric converter for non-numeric types
	default:
		fmt.Fprintf(os.Stderr, "proto: tag has unknown wire type: %q\n", s)
		return
	}

	var err os.Error
	p.Tag, err = strconv.Atoi(fields[1])
	if err != nil {
		return
	}

	for i := 2; i < len(fields); i++ {
		f := fields[i]
		switch {
		case f == "req":
			p.Required = true
		case f == "opt":
			p.Optional = true
		case f == "rep":
			p.Repeated = true
		case f == "packed":
			p.Packed = true
		case len(f) >= 5 && f[0:5] == "name=":
			p.OrigName = f[5:len(f)]
		case len(f) >= 5 && f[0:5] == "enum=":
			p.Enum = f[5:len(f)]
		case len(f) >= 4 && f[0:4] == "def=":
			p.Default = f[4:len(f)] // rest of string
			if i+1 < len(fields) {
				// Commas aren't escaped, and def is always last.
				p.Default += "," + strings.Join(fields[i+1:len(fields)], ",")
				break
			}
		}
	}
}

func logNoSliceEnc(t1, t2 reflect.Type) {
	fmt.Fprintf(os.Stderr, "proto: no slice oenc for %T = []%T\n", t1, t2)
}

// Initialize the fields for encoding and decoding.
func (p *Properties) setEncAndDec(typ reflect.Type) {
	var vbool bool
	var vbyte byte
	var vint32 int32
	var vint64 int64
	var vfloat32 float32
	var vfloat64 float64
	var vstring string
	var vslice []byte

	p.enc = nil
	p.dec = nil

	switch t1 := typ; t1.Kind() {
	default:
		fmt.Fprintf(os.Stderr, "proto: no coders for %T\n", t1)
		break

	case reflect.Ptr:
		switch t2 := t1.Elem(); t2.Kind() {
		default:
			fmt.Fprintf(os.Stderr, "proto: no encoder function for %T -> %T\n", t1, t2)
			break
		case reflect.Bool:
			p.enc = (*Buffer).enc_bool
			p.dec = (*Buffer).dec_bool
			p.alignof = unsafe.Alignof(vbool)
			p.sizeof = unsafe.Sizeof(vbool)
		case reflect.Int32, reflect.Uint32:
			p.enc = (*Buffer).enc_int32
			p.dec = (*Buffer).dec_int32
			p.alignof = unsafe.Alignof(vint32)
			p.sizeof = unsafe.Sizeof(vint32)
		case reflect.Int64, reflect.Uint64:
			p.enc = (*Buffer).enc_int64
			p.dec = (*Buffer).dec_int64
			p.alignof = unsafe.Alignof(vint64)
			p.sizeof = unsafe.Sizeof(vint64)
		case reflect.Float32:
			p.enc = (*Buffer).enc_int32 // can just treat them as bits
			p.dec = (*Buffer).dec_int32
			p.alignof = unsafe.Alignof(vfloat32)
			p.sizeof = unsafe.Sizeof(vfloat32)
		case reflect.Float64:
			p.enc = (*Buffer).enc_int64 // can just treat them as bits
			p.dec = (*Buffer).dec_int64
			p.alignof = unsafe.Alignof(vfloat64)
			p.sizeof = unsafe.Sizeof(vfloat64)
		case reflect.String:
			p.enc = (*Buffer).enc_string
			p.dec = (*Buffer).dec_string
			p.alignof = unsafe.Alignof(vstring)
			p.sizeof = unsafe.Sizeof(vstring) + startSize*unsafe.Sizeof(vbyte)
		case reflect.Struct:
			p.stype = t1
			if p.Wire == "bytes" {
				p.enc = (*Buffer).enc_struct_message
				p.dec = (*Buffer).dec_struct_message
			} else {
				p.enc = (*Buffer).enc_struct_group
				p.dec = (*Buffer).dec_struct_group
			}
		}

	case reflect.Slice:
		switch t2 := t1.Elem(); t2.Kind() {
		default:
			logNoSliceEnc(t1, t2)
			break
		case reflect.Bool:
			if p.Packed {
				p.enc = (*Buffer).enc_slice_packed_bool
			} else {
				p.enc = (*Buffer).enc_slice_bool
			}
			p.dec = (*Buffer).dec_slice_bool
			p.packedDec = (*Buffer).dec_slice_packed_bool
			p.alignof = unsafe.Alignof(vbool)
			p.sizeof = startSize * unsafe.Sizeof(vbool)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			switch t2.Bits() {
			case 32:
				if p.Packed {
					p.enc = (*Buffer).enc_slice_packed_int32
				} else {
					p.enc = (*Buffer).enc_slice_int32
				}
				p.dec = (*Buffer).dec_slice_int32
				p.packedDec = (*Buffer).dec_slice_packed_int32
				p.alignof = unsafe.Alignof(vint32)
				p.sizeof = startSize * unsafe.Sizeof(vint32)
			case 64:
				if p.Packed {
					p.enc = (*Buffer).enc_slice_packed_int64
				} else {
					p.enc = (*Buffer).enc_slice_int64
				}
				p.dec = (*Buffer).dec_slice_int64
				p.packedDec = (*Buffer).dec_slice_packed_int64
				p.alignof = unsafe.Alignof(vint64)
				p.sizeof = startSize * unsafe.Sizeof(vint64)
			case 8:
				if t2.Kind() == reflect.Uint8 {
					p.enc = (*Buffer).enc_slice_byte
					p.dec = (*Buffer).dec_slice_byte
					p.alignof = unsafe.Alignof(vbyte)
					p.sizeof = startSize * unsafe.Sizeof(vbyte)
				}
			default:
				logNoSliceEnc(t1, t2)
				break
			}
		case reflect.Float32, reflect.Float64:
			switch t2.Bits() {
			case 32:
				// can just treat them as bits
				if p.Packed {
					p.enc = (*Buffer).enc_slice_packed_int32
				} else {
					p.enc = (*Buffer).enc_slice_int32
				}
				p.dec = (*Buffer).dec_slice_int32
				p.packedDec = (*Buffer).dec_slice_packed_int32
				p.alignof = unsafe.Alignof(vfloat32)
				p.sizeof = startSize * unsafe.Sizeof(vfloat32)
			case 64:
				// can just treat them as bits
				if p.Packed {
					p.enc = (*Buffer).enc_slice_packed_int64
				} else {
					p.enc = (*Buffer).enc_slice_int64
				}
				p.dec = (*Buffer).dec_slice_int64
				p.packedDec = (*Buffer).dec_slice_packed_int64
				p.alignof = unsafe.Alignof(vfloat64)
				p.sizeof = startSize * unsafe.Sizeof(vfloat64)
			default:
				logNoSliceEnc(t1, t2)
				break
			}
		case reflect.String:
			p.enc = (*Buffer).enc_slice_string
			p.dec = (*Buffer).dec_slice_string
			p.alignof = unsafe.Alignof(vstring)
			p.sizeof = startSize * unsafe.Sizeof(vstring)
		case reflect.Ptr:
			switch t3 := t2.Elem(); t3.Kind() {
			default:
				fmt.Fprintf(os.Stderr, "proto: no ptr oenc for %T -> %T -> %T\n", t1, t2, t3)
				break
			case reflect.Struct:
				p.stype = t2
				p.enc = (*Buffer).enc_slice_struct_group
				p.dec = (*Buffer).dec_slice_struct_group
				if p.Wire == "bytes" {
					p.enc = (*Buffer).enc_slice_struct_message
					p.dec = (*Buffer).dec_slice_struct_message
				}
				p.alignof = unsafe.Alignof(vslice)
				p.sizeof = startSize * unsafe.Sizeof(vslice)
			}
		case reflect.Slice:
			switch t2.Elem().Kind() {
			default:
				fmt.Fprintf(os.Stderr, "proto: no slice elem oenc for %T -> %T -> %T\n", t1, t2, t2.Elem())
				break
			case reflect.Uint8:
				p.enc = (*Buffer).enc_slice_slice_byte
				p.dec = (*Buffer).dec_slice_slice_byte
				p.alignof = unsafe.Alignof(vslice)
				p.sizeof = startSize * unsafe.Sizeof(vslice)
			}
		}
	}

	// precalculate tag code
	wire := p.WireType
	if p.Packed {
		wire = WireBytes
	}
	x := uint32(p.Tag)<<3 | uint32(wire)
	i := 0
	for i = 0; x > 127; i++ {
		p.tagbuf[i] = 0x80 | uint8(x&0x7F)
		x >>= 7
	}
	p.tagbuf[i] = uint8(x)
	p.tagcode = p.tagbuf[0 : i+1]
}

// Init populates the properties from a protocol buffer struct tag.
func (p *Properties) Init(typ reflect.Type, name, tag string, offset uintptr) {
	// "bytes,49,opt,def=hello!"
	p.Name = name
	p.OrigName = name
	p.offset = offset

	if tag == "" {
		return
	}
	p.Parse(tag)
	p.setEncAndDec(typ)
}

var (
	mutex         sync.Mutex
	propertiesMap = make(map[reflect.Type]*StructProperties)
)

// GetProperties returns the list of properties for the type represented by t.
func GetProperties(t reflect.Type) *StructProperties {
	mutex.Lock()
	if prop, ok := propertiesMap[t]; ok {
		mutex.Unlock()
		stats.Chit++
		return prop
	}
	stats.Cmiss++

	prop := new(StructProperties)

	// build properties
	prop.Prop = make([]*Properties, t.NumField())
	prop.order = make([]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		p := new(Properties)
		p.Init(f.Type, f.Name, f.Tag.Get("protobuf"), f.Offset)
		if f.Name == "XXX_extensions" { // special case
			var vmap map[int32][]byte
			p.enc = (*Buffer).enc_map
			p.dec = nil // not needed
			p.alignof = unsafe.Alignof(vmap)
			p.sizeof = unsafe.Sizeof(vmap)
		}
		prop.Prop[i] = p
		prop.order[i] = i
		if debug {
			print(i, " ", f.Name, " ", t.String(), " ")
			if p.Tag > 0 {
				print(p.String())
			}
			print("\n")
		}
		if p.enc == nil && !strings.HasPrefix(f.Name, "XXX_") {
			fmt.Fprintln(os.Stderr, "proto: no encoder for", f.Name, f.Type.String(), "[GetProperties]")
		}
	}

	// Re-order prop.order.
	sort.Sort(prop)

	// build required counts
	// build scratch offsets
	// build tags
	reqCount := 0
	scratch := uintptr(0)
	prop.tags = make(map[int]int)
	prop.origNames = make(map[string]int)
	for i, p := range prop.Prop {
		if p.Required {
			reqCount++
		}
		scratch = align(scratch, p.alignof)
		p.scratch = scratch
		scratch += p.sizeof
		prop.tags[p.Tag] = i
		prop.origNames[p.OrigName] = i
	}
	prop.reqCount = reqCount
	prop.nscratch = scratch

	propertiesMap[t] = prop
	mutex.Unlock()
	return prop
}

// Alignment of the data in the scratch area.  It doesn't have to be
// exact, just conservative.  Returns the first number >= o that divides s.
func align(o uintptr, s uintptr) uintptr {
	if s != 0 {
		for o%uintptr(s) != 0 {
			o++
		}
	}
	return o
}

// Return the field index of the named field.
// Returns nil if there is no such field.
func fieldIndex(t reflect.Type, name string) []int {
	if field, ok := t.FieldByName(name); ok {
		return field.Index
	}
	return nil
}

// Return the Properties object for the x[0]'th field of the structure.
func propByIndex(t reflect.Type, x []int) *Properties {
	if len(x) != 1 {
		fmt.Fprintf(os.Stderr, "proto: field index dimension %d (not 1) for type %s\n", len(x), t)
		return nil
	}
	prop := GetProperties(t)
	return prop.Prop[x[0]]
}

// Get the address and type of a pointer to a struct from an interface.
// unsafe.Reflect can do this, but does multiple mallocs.
func getbase(pb interface{}) (t reflect.Type, b uintptr, err os.Error) {
	// get pointer
	x := *(*[2]uintptr)(unsafe.Pointer(&pb))
	b = x[1]
	if b == 0 {
		err = ErrNil
		return
	}

	// get the reflect type of the struct.
	t = reflect.TypeOf(pb)
	return
}

// Allocate the aux space containing all the decoded data.  The structure
// handed into Unmarshal is filled with pointers to this newly allocated
// data.
func getsbase(prop *StructProperties) uintptr {
	var vbyteptr *byte
	if prop.nscratch == 0 {
		return 0
	}

	// allocate the decode space as pointers
	// so that the GC will scan it for pointers
	n := uintptr(unsafe.Sizeof(vbyteptr))
	b := make([]*byte, (prop.nscratch+n-1)/n)
	sbase := uintptr(unsafe.Pointer(&b[0]))
	return sbase
}

// A global registry of enum types.
// The generated code will register the generated maps by calling RegisterEnum.

var enumNameMaps = make(map[string]map[int32]string)
var enumValueMaps = make(map[string]map[string]int32)

// RegisterEnum is called from the generated code to install the enum descriptor
// maps into the global table to aid parsing ASCII protocol buffers.
func RegisterEnum(typeName string, nameMap map[int32]string, valueMap map[string]int32) {
	if _, ok := enumNameMaps[typeName]; ok {
		panic("proto: duplicate enum registered: " + typeName)
	}
	enumNameMaps[typeName] = nameMap
	enumValueMaps[typeName] = valueMap
}
