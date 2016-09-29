// Go support for Protocol Buffers - Google's data interchange format
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
)

const debug bool = false

// Constants that identify the encoding of a value on the wire.
const (
	WireVarint     = WireType(0)
	WireFixed64    = WireType(1)
	WireBytes      = WireType(2)
	WireStartGroup = WireType(3)
	WireEndGroup   = WireType(4)
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

const startSize = 10 // initial slice/string sizes

// Encoders are defined in encode.go
// An encoder outputs the full representation of a field, including its
// tag and encoder type.
type encoder func(p *Buffer, prop *Properties, base structPointer) error

// A valueEncoder encodes a single integer in a particular encoding.
type valueEncoder func(o *Buffer, x uint64)

// Decoders are defined in decode.go
// A decoder creates a value from its wire representation.
// Unrecognized subelements are saved in unrec.
type decoder func(p *Buffer, prop *Properties, base structPointer) error

// A valueDecoder decodes a single integer in a particular encoding.
type valueDecoder func(o *Buffer) (x uint64, err error)

// StructProperties represents properties for all the fields of a struct.
type StructProperties struct {
	Prop  []Properties // properties for each field, indexed by reflection's field number. Fields which are not encoded in protobuf have incomplete Properties
	order []int        // list of struct field numbers in tag order, indexed 0 to N-1 by the number of fields to encode in protobuf. value indexes into .Prop[]
}

// Implement the sorting interface so we can sort the fields in tag order, as recommended by the spec.
// See encode.go, (*Buffer).enc_struct.

func (sp *StructProperties) Len() int { return len(sp.order) }
func (sp *StructProperties) Less(i, j int) bool {
	return sp.Prop[sp.order[i]].Tag < sp.Prop[sp.order[j]].Tag
}
func (sp *StructProperties) Swap(i, j int) { sp.order[i], sp.order[j] = sp.order[j], sp.order[i] }

// Properties represents the protocol-specific behavior of a single struct field.
type Properties struct {
	Name     string // name of the field, for error messages
	Wire     string
	Tag      uint32
	Repeated bool
	WireType WireType

	enc         encoder
	valEnc      valueEncoder // set for bool and numeric types only
	field       field
	tagcode     []byte // encoding of EncodeVarint((Tag<<3)|WireType)
	tagbuf      [8]byte
	stype       reflect.Type      // set for struct types only
	sprop       *StructProperties // set for struct types only
	isMarshaler bool

	mtype    reflect.Type // set for map types only
	mkeyprop *Properties  // set for map types only
	mvalprop *Properties  // set for map types only
}

// String formats the properties in the protobuf struct field tag style.
func (p *Properties) String() string {
	s := p.Wire
	s += ","
	s += strconv.FormatUint(uint64(p.Tag), 10)
	s += ",opt" // all protobuf v3 fields are optional
	if p.Repeated {
		s += ",rep"
	}
	s += ",proto3"

	return s
}

// Parse populates p by parsing a string in the protobuf struct field tag style.
func (p *Properties) Parse(s string) (bool, error) {
	// "bytes,49,rep,..."
	fields := strings.Split(s, ",")
	if len(fields) < 1 {
		return true, fmt.Errorf("protobuf3: tag of %q has too few fields: %q", p.Name, s)
	}

	p.Wire = fields[0]
	switch p.Wire {
	case "varint":
		p.valEnc = (*Buffer).EncodeVarint
		p.WireType = WireVarint
	case "fixed32":
		p.valEnc = (*Buffer).EncodeFixed32
		p.WireType = WireFixed32
	case "fixed64":
		p.valEnc = (*Buffer).EncodeFixed64
		p.WireType = WireFixed64
	case "zigzag32":
		p.valEnc = (*Buffer).EncodeZigzag32
		p.WireType = WireVarint
	case "zigzag64":
		p.valEnc = (*Buffer).EncodeZigzag64
	case "bytes":
		// no numeric converter for non-numeric types
		p.WireType = WireBytes
	case "skip":
		// used to mark fields which should be skipped by the protobuf encoder
		return true, nil
	default:
		return false, fmt.Errorf("protobuf3: tag of %q has unknown wire type: %q", p.Name, s)
	}

	tag, err := strconv.Atoi(fields[1])
	if err != nil {
		return false, fmt.Errorf("protobuf3: tag id of %q invalid: %s: %s", p.Name, s, err.Error())
	}
	if tag <= 0 { // catch any negative or 0 values
		return false, fmt.Errorf("protobuf3: tag id of %q out of range: %s", p.Name, s)
	}
	p.Tag = uint32(tag)

	for _, f := range fields[2:] {
		switch f {
		case "rep":
			p.Repeated = true
		}
	}

	return false, nil
}

func logNoSliceEnc(t1, t2 reflect.Type) {
	fmt.Fprintf(os.Stderr, "proto: no slice oenc for %T = []%T\n", t1, t2)
}

var protoMessageType = reflect.TypeOf((*Message)(nil)).Elem()

// Initialize the fields for encoding and decoding.
func (p *Properties) setEnc(typ reflect.Type, f *reflect.StructField) {
	p.enc = nil
	wire := p.WireType

	switch t1 := typ; t1.Kind() {
	default:
		fmt.Fprintf(os.Stderr, "proto: no coders for %v\n", t1)

	// proto3 scalar types

	case reflect.Bool:
		p.enc = (*Buffer).enc_bool
	case reflect.Int32:
		p.enc = (*Buffer).enc_int32
	case reflect.Uint32:
		p.enc = (*Buffer).enc_uint32
	case reflect.Int64, reflect.Uint64:
		p.enc = (*Buffer).enc_int64
	case reflect.Float32:
		p.enc = (*Buffer).enc_uint32 // can just treat them as bits
	case reflect.Float64:
		p.enc = (*Buffer).enc_int64 // can just treat them as bits
	case reflect.String:
		p.enc = (*Buffer).enc_string

	case reflect.Struct:
		p.stype = t1
		p.isMarshaler = isMarshaler(reflect.PtrTo(t1))
		p.enc = (*Buffer).enc_struct_message

	case reflect.Ptr:
		switch t2 := t1.Elem(); t2.Kind() {
		default:
			fmt.Fprintf(os.Stderr, "proto: no encoder function for %v -> %v\n", t1, t2)
			break
		case reflect.Bool:
			p.enc = (*Buffer).enc_ptr_bool
		case reflect.Int32:
			p.enc = (*Buffer).enc_ptr_int32
		case reflect.Uint32:
			p.enc = (*Buffer).enc_ptr_uint32
		case reflect.Int64, reflect.Uint64:
			p.enc = (*Buffer).enc_ptr_int64
		case reflect.Float32:
			p.enc = (*Buffer).enc_ptr_uint32 // can just treat them as bits
		case reflect.Float64:
			p.enc = (*Buffer).enc_ptr_int64 // can just treat them as bits
		case reflect.String:
			p.enc = (*Buffer).enc_ptr_string
		case reflect.Struct:
			p.stype = t2
			p.isMarshaler = isMarshaler(t1)
			p.enc = (*Buffer).enc_ptr_struct_message
		}

	case reflect.Slice:
		switch t2 := t1.Elem(); t2.Kind() {
		default:
			logNoSliceEnc(t1, t2)
			break
		case reflect.Bool:
			p.enc = (*Buffer).enc_slice_packed_bool
			wire = WireBytes // packed=true is implied in protobuf v3
		case reflect.Int32:
			p.enc = (*Buffer).enc_slice_packed_int32
			wire = WireBytes // packed=true...
		case reflect.Uint32:
			p.enc = (*Buffer).enc_slice_packed_uint32
			wire = WireBytes // packed=true...
		case reflect.Int64, reflect.Uint64:
			p.enc = (*Buffer).enc_slice_packed_int64
			wire = WireBytes // packed=true...
		case reflect.Uint8:
			p.enc = (*Buffer).enc_slice_byte
		case reflect.Float32:
			// can just treat them as bits
			p.enc = (*Buffer).enc_slice_packed_uint32
			wire = WireBytes // packed=true...
		case reflect.Float64:
			// can just treat them as bits
			p.enc = (*Buffer).enc_slice_packed_int64
			wire = WireBytes // packed=true...
		case reflect.String:
			p.enc = (*Buffer).enc_slice_string
		case reflect.Struct:
			p.stype = t2
			p.isMarshaler = isMarshaler(reflect.PtrTo(t2))
			p.enc = (*Buffer).enc_slice_struct_message
		case reflect.Ptr:
			switch t3 := t2.Elem(); t3.Kind() {
			default:
				fmt.Fprintf(os.Stderr, "proto: no ptr oenc for %T -> %T -> %T\n", t1, t2, t3)
				break
			case reflect.Struct:
				p.stype = t3
				p.isMarshaler = isMarshaler(t2)
				p.enc = (*Buffer).enc_slice_ptr_struct_message
			}
		case reflect.Slice:
			switch t2.Elem().Kind() {
			default:
				fmt.Fprintf(os.Stderr, "proto: no slice elem oenc for %T -> %T -> %T\n", t1, t2, t2.Elem())
				break
			case reflect.Uint8:
				p.enc = (*Buffer).enc_slice_slice_byte
			}
		}

	case reflect.Map:
		p.enc = (*Buffer).enc_new_map

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
	}

	// precalculate tag code
	x := p.Tag<<3 | uint32(wire)
	i := 0
	for i = 0; x > 127; i++ {
		p.tagbuf[i] = 0x80 | uint8(x&0x7F)
		x >>= 7
	}
	p.tagbuf[i] = uint8(x)
	p.tagcode = p.tagbuf[0 : i+1]

	if p.stype != nil {
		p.sprop = getPropertiesLocked(p.stype)
	}
}

var (
	marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
)

// isMarshaler reports whether type t implements Marshaler.
func isMarshaler(t reflect.Type) bool {
	return t.Implements(marshalerType)
}

// Init populates the properties from a protocol buffer struct tag.
func (p *Properties) init(typ reflect.Type, name, tag string, f *reflect.StructField) (bool, error) {
	// "bytes,49,opt,def=hello!"

	// skip fields without protobuf tags
	if tag == "" {
		return true, nil
	}

	p.Name = name
	if f != nil {
		p.field = toField(f)
	}

	skip, err := p.Parse(tag)
	if skip || err != nil {
		return skip, err
	}

	p.setEnc(typ, f)

	return false, nil
}

var (
	propertiesMu  sync.RWMutex
	propertiesMap = make(map[reflect.Type]*StructProperties)
)

// GetProperties returns the list of properties for the type represented by t.
// t must represent a generated struct type of a protocol message.
func GetProperties(t reflect.Type) *StructProperties {
	k := t.Kind()
	// accept a pointer-to-struct as well (but just one level)
	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	if k != reflect.Struct {
		panic("proto: type must have kind struct")
	}

	// Most calls to GetProperties in a long-running program will be
	// retrieving details for types we have seen before.
	propertiesMu.RLock()
	sprop, ok := propertiesMap[t]
	propertiesMu.RUnlock()
	if ok {
		return sprop
	}

	propertiesMu.Lock()
	sprop = getPropertiesLocked(t)
	propertiesMu.Unlock()
	return sprop
}

// getPropertiesLocked requires that propertiesMu is held.
func getPropertiesLocked(t reflect.Type) *StructProperties {
	if prop, ok := propertiesMap[t]; ok {
		return prop
	}

	prop := new(StructProperties)
	// in case of recursive protos, fill this in now.
	propertiesMap[t] = prop

	// build properties
	nf := t.NumField()
	prop.Prop = make([]Properties, nf)
	prop.order = make([]int, nf)

	j := 0
	for i := 0; i < nf; i++ {
		f := t.Field(i)
		p := &prop.Prop[i]
		name := f.Name

		skip, err := p.init(f.Type, name, f.Tag.Get("protobuf"), &f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing field %q of type %q: %v\n", name, t.Name(), err)
			continue
		}
		if skip {
			// silently skip this field. It's not part of the protobuf encoding of this struct
			continue
		}

		prop.order[j] = i
		j++

		if debug {
			print(i, " ", f.Name, " ", t.String(), " ")
			if p.Tag > 0 {
				print(p.String())
			}
			print("\n")
		}

		if p.enc == nil {
			fmt.Fprintln(os.Stderr, "proto: no encoder for", f.Name, f.Type.String(), "[GetProperties]")
		}
	}

	// slice off any unused indexes
	prop.order = prop.order[:j]

	// Re-order prop.order.
	sort.Sort(prop)

	return prop
}
