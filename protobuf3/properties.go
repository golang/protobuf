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
	"log"
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
	WireVarint     = 0
	WireFixed64    = 1
	WireBytes      = 2
	WireStartGroup = 3
	WireEndGroup   = 4
	WireFixed32    = 5
)

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

// tagMap is an optimization over map[int]int for typical protocol buffer
// use-cases. Encoded protocol buffers are often in tag order with small tag
// numbers.
type tagMap struct {
	fastTags []int
	slowTags map[int]int
}

// tagMapFastLimit is the upper bound on the tag number that will be stored in
// the tagMap slice rather than its map.
const tagMapFastLimit = 1024

func (p *tagMap) get(t int) (int, bool) {
	if t > 0 && t < tagMapFastLimit {
		if t >= len(p.fastTags) {
			return 0, false
		}
		fi := p.fastTags[t]
		return fi, fi >= 0
	}
	fi, ok := p.slowTags[t]
	return fi, ok
}

func (p *tagMap) put(t int, fi int) {
	if t > 0 && t < tagMapFastLimit {
		for len(p.fastTags) < t+1 {
			p.fastTags = append(p.fastTags, -1)
		}
		p.fastTags[t] = fi
		return
	}
	if p.slowTags == nil {
		p.slowTags = make(map[int]int)
	}
	p.slowTags[t] = fi
}

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
	Tag      int
	Repeated bool

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
	s = ","
	s += strconv.Itoa(p.Tag)
	s += ",opt" // all protobuf v3 fields are optional
	if p.Repeated {
		s += ",rep"
	}
	s += ",packed" // we pack all fields
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
	case "fixed32":
		p.valEnc = (*Buffer).EncodeFixed32
	case "fixed64":
		p.valEnc = (*Buffer).EncodeFixed64
	case "zigzag32":
		p.valEnc = (*Buffer).EncodeZigzag32
	case "zigzag64":
		p.valEnc = (*Buffer).EncodeZigzag64
	case "bytes":
		// no numeric converter for non-numeric types
	case "skip":
		// used to mark fields which should be skipped by the protobuf encoder
		return true, nil
	default:
		return false, fmt.Errorf("protobuf3: tag of %q has unknown wire type: %q", p.Name, s)
	}

	var err error
	p.Tag, err = strconv.Atoi(fields[1])
	if err != nil {
		return false, fmt.Errorf("protobuf3: tag id of %q invalid: %s: %s", p.Name, s, err.Error())
	}

	for i := 2; i < len(fields); i++ {
		f := fields[i]
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
func (p *Properties) setEnc(typ reflect.Type, f *reflect.StructField, lockGetProp bool) {
	p.enc = nil

	switch t1 := typ; t1.Kind() {
	default:
		fmt.Fprintf(os.Stderr, "proto: no coders for %v\n", t1)

	// proto3 scalar types

	case reflect.Bool:
		p.enc = (*Buffer).enc_proto3_bool
	case reflect.Int32:
		p.enc = (*Buffer).enc_proto3_int32
	case reflect.Uint32:
		p.enc = (*Buffer).enc_proto3_uint32
	case reflect.Int64, reflect.Uint64:
		p.enc = (*Buffer).enc_proto3_int64
	case reflect.Float32:
		p.enc = (*Buffer).enc_proto3_uint32 // can just treat them as bits
	case reflect.Float64:
		p.enc = (*Buffer).enc_proto3_int64 // can just treat them as bits
	case reflect.String:
		p.enc = (*Buffer).enc_proto3_string

	case reflect.Struct:
		p.stype = t1
		p.isMarshaler = isMarshaler(t1)
		p.enc = (*Buffer).enc_struct_message

	case reflect.Ptr:
		switch t2 := t1.Elem(); t2.Kind() {
		default:
			fmt.Fprintf(os.Stderr, "proto: no encoder function for %v -> %v\n", t1, t2)
			break
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
			p.stype = t1.Elem()
			p.isMarshaler = isMarshaler(t1)
			p.enc = (*Buffer).enc_struct_message
		}

	case reflect.Slice:
		switch t2 := t1.Elem(); t2.Kind() {
		default:
			logNoSliceEnc(t1, t2)
			break
		case reflect.Bool:
			p.enc = (*Buffer).enc_slice_packed_bool
		case reflect.Int32:
			p.enc = (*Buffer).enc_slice_packed_int32
		case reflect.Uint32:
			p.enc = (*Buffer).enc_slice_packed_uint32
		case reflect.Int64, reflect.Uint64:
			p.enc = (*Buffer).enc_slice_packed_int64
		case reflect.Uint8:
			p.enc = (*Buffer).enc_proto3_slice_byte
		case reflect.Float32, reflect.Float64:
			switch t2.Bits() {
			case 32:
				// can just treat them as bits
				p.enc = (*Buffer).enc_slice_packed_uint32
			case 64:
				// can just treat them as bits
				p.enc = (*Buffer).enc_slice_packed_int64
			default:
				logNoSliceEnc(t1, t2)
				break
			}
		case reflect.String:
			p.enc = (*Buffer).enc_slice_string
		case reflect.Ptr:
			switch t3 := t2.Elem(); t3.Kind() {
			default:
				fmt.Fprintf(os.Stderr, "proto: no ptr oenc for %T -> %T -> %T\n", t1, t2, t3)
				break
			case reflect.Struct:
				p.stype = t2.Elem()
				p.isMarshaler = isMarshaler(t2)
				p.enc = (*Buffer).enc_slice_struct_message
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
		p.mkeyprop.init(reflect.PtrTo(p.mtype.Key()), "Key", f.Tag.Get("protobuf_key"), nil, lockGetProp)
		p.mvalprop = &Properties{}
		vtype := p.mtype.Elem()
		if vtype.Kind() != reflect.Ptr && vtype.Kind() != reflect.Slice {
			// The value type is not a message (*T) or bytes ([]byte),
			// so we need encoders for the pointer to this type.
			vtype = reflect.PtrTo(vtype)
		}
		p.mvalprop.init(vtype, "Value", f.Tag.Get("protobuf_val"), nil, lockGetProp)
	}

	// precalculate tag code
	wire := WireBytes
	x := uint32(p.Tag)<<3 | uint32(wire)
	i := 0
	for i = 0; x > 127; i++ {
		p.tagbuf[i] = 0x80 | uint8(x&0x7F)
		x >>= 7
	}
	p.tagbuf[i] = uint8(x)
	p.tagcode = p.tagbuf[0 : i+1]

	if p.stype != nil {
		if lockGetProp {
			p.sprop = GetProperties(p.stype)
		} else {
			p.sprop = getPropertiesLocked(p.stype)
		}
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
func (p *Properties) Init(typ reflect.Type, name, tag string, f *reflect.StructField) {
	p.init(typ, name, tag, f, true)
}

func (p *Properties) init(typ reflect.Type, name, tag string, f *reflect.StructField, lockGetProp bool) (bool, error) {
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

	p.setEnc(typ, f, lockGetProp)

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

		skip, err := p.init(f.Type, name, f.Tag.Get("protobuf"), &f, false)
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

// Return the Properties object for the x[0]'th field of the structure.
func propByIndex(t reflect.Type, x []int) *Properties {
	if len(x) != 1 {
		fmt.Fprintf(os.Stderr, "proto: field index dimension %d (not 1) for type %s\n", len(x), t)
		return nil
	}
	prop := GetProperties(t)
	return &prop.Prop[x[0]]
}

// Get the address and type of a pointer to a struct from an interface.
func getbase(pb Message) (t reflect.Type, b structPointer, err error) {
	if pb == nil {
		err = ErrNil
		return
	}
	// get the reflect type of the pointer to the struct.
	t = reflect.TypeOf(pb)
	if t.Kind() != reflect.Ptr {
		// create the pointer we need
		t = reflect.PtrTo(t)
		value := reflect.ValueOf(pb)
		// hmmm. stuck.
		_ = value
	} else {
		// get the address of the struct.
		value := reflect.ValueOf(pb)
		b = toStructPointer(value)
	}
	return
}

// A global registry of enum types.
// The generated code will register the generated maps by calling RegisterEnum.

var enumValueMaps = make(map[string]map[string]int32)

// RegisterEnum is called from the generated code to install the enum descriptor
// maps into the global table to aid parsing text format protocol buffers.
func RegisterEnum(typeName string, unusedNameMap map[int32]string, valueMap map[string]int32) {
	if _, ok := enumValueMaps[typeName]; ok {
		panic("proto: duplicate enum registered: " + typeName)
	}
	enumValueMaps[typeName] = valueMap
}

// EnumValueMap returns the mapping from names to integers of the
// enum type enumType, or a nil if not found.
func EnumValueMap(enumType string) map[string]int32 {
	return enumValueMaps[enumType]
}

// A registry of all linked message types.
// The string is a fully-qualified proto name ("pkg.Message").
var (
	protoTypes    = make(map[string]reflect.Type)
	revProtoTypes = make(map[reflect.Type]string)
)

// RegisterType is called from generated code and maps from the fully qualified
// proto name to the type (pointer to struct) of the protocol buffer.
func RegisterType(x Message, name string) {
	if _, ok := protoTypes[name]; ok {
		// TODO: Some day, make this a panic.
		log.Printf("proto: duplicate proto type registered: %s", name)
		return
	}
	t := reflect.TypeOf(x)
	protoTypes[name] = t
	revProtoTypes[t] = name
}

// MessageName returns the fully-qualified proto name for the given message type.
func MessageName(x Message) string {
	return revProtoTypes[reflect.TypeOf(x)]
}

// MessageType returns the message type (pointer to struct) for a named message.
func MessageType(name string) reflect.Type { return protoTypes[name] }

// A registry of all linked proto files.
var (
	protoFiles = make(map[string][]byte) // file name => fileDescriptor
)

// RegisterFile is called from generated code and maps from the
// full file name of a .proto file to its compressed FileDescriptorProto.
func RegisterFile(filename string, fileDescriptor []byte) {
	protoFiles[filename] = fileDescriptor
}

// FileDescriptor returns the compressed FileDescriptorProto for a .proto file.
func FileDescriptor(filename string) []byte { return protoFiles[filename] }
