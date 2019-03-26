// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build use_golang_protobuf_v1

package proto

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Constants that identify the encoding of a value on the wire.
const (
	WireVarint     = 0
	WireFixed32    = 5
	WireFixed64    = 1
	WireBytes      = 2
	WireStartGroup = 3
	WireEndGroup   = 4
)

// StructProperties represents properties for all the fields of a struct.
type StructProperties struct {
	Prop []*Properties // properties for each field

	// OneofTypes contains information about the oneof fields in this message.
	// It is keyed by the original name of a field.
	OneofTypes map[string]*OneofProperties
}

// OneofProperties represents information about a specific field in a oneof.
type OneofProperties struct {
	Type  reflect.Type // pointer to generated struct type for this oneof field
	Field int          // struct field number of the containing oneof in the message
	Prop  *Properties
}

func (sp *StructProperties) Len() int           { return len(sp.Prop) }
func (sp *StructProperties) Less(i, j int) bool { return false }
func (sp *StructProperties) Swap(i, j int)      { return }

// Properties represents the protocol-specific behavior of a single struct field.
type Properties struct {
	Name     string // name of the field, for error messages
	OrigName string // original name before protocol compiler (always set)
	JSONName string // name to use for JSON; determined by protoc
	Wire     string
	WireType int
	Tag      int
	Required bool
	Optional bool
	Repeated bool
	Packed   bool   // relevant for repeated primitives only
	Enum     string // set for enum types only
	Proto3   bool   // whether this is known to be a proto3 field
	Oneof    bool   // whether this is a oneof field

	Default    string // default value
	HasDefault bool   // whether an explicit default was provided

	MapKeyProp *Properties // set for map types only
	MapValProp *Properties // set for map types only
}

// String formats the properties in the protobuf struct field tag style.
func (p *Properties) String() string {
	s := p.Wire
	s += "," + strconv.Itoa(p.Tag)
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
	s += ",name=" + p.OrigName
	if p.JSONName != "" {
		s += ",json=" + p.JSONName
	}
	if p.Proto3 {
		s += ",proto3"
	}
	if p.Oneof {
		s += ",oneof"
	}
	if len(p.Enum) > 0 {
		s += ",enum=" + p.Enum
	}
	if p.HasDefault {
		s += ",def=" + p.Default
	}
	return s
}

// Parse populates p by parsing a string in the protobuf struct field tag style.
func (p *Properties) Parse(tag string) {
	// For example: "bytes,49,opt,name=foo,def=hello!"
	for len(tag) > 0 {
		i := strings.IndexByte(tag, ',')
		if i < 0 {
			i = len(tag)
		}
		switch s := tag[:i]; {
		case strings.HasPrefix(s, "name="):
			p.OrigName = s[len("name="):]
		case strings.HasPrefix(s, "json="):
			p.JSONName = s[len("json="):]
		case strings.HasPrefix(s, "enum="):
			p.Enum = s[len("enum="):]
		case strings.Trim(s, "0123456789") == "":
			n, _ := strconv.ParseUint(s, 10, 32)
			p.Tag = int(n)
		case s == "opt":
			p.Optional = true
		case s == "req":
			p.Required = true
		case s == "rep":
			p.Repeated = true
		case s == "varint" || s == "zigzag32" || s == "zigzag64":
			p.Wire = s
			p.WireType = WireVarint
		case s == "fixed32":
			p.Wire = s
			p.WireType = WireFixed32
		case s == "fixed64":
			p.Wire = s
			p.WireType = WireFixed64
		case s == "bytes":
			p.Wire = s
			p.WireType = WireBytes
		case s == "group":
			p.Wire = s
			p.WireType = WireStartGroup
		case s == "packed":
			p.Packed = true
		case s == "proto3":
			p.Proto3 = true
		case s == "oneof":
			p.Oneof = true
		case strings.HasPrefix(s, "def="):
			// The default tag is special in that everything afterwards is the
			// default regardless of the presence of commas.
			p.HasDefault = true
			p.Default, i = tag[len("def="):], len(tag)
		}
		tag = strings.TrimPrefix(tag[i:], ",")
	}
}

// Init populates the properties from a protocol buffer struct tag.
func (p *Properties) Init(typ reflect.Type, name, tag string, f *reflect.StructField) {
	p.init(typ, name, tag, f)
}

func (p *Properties) init(typ reflect.Type, name, tag string, f *reflect.StructField) {
	p.Name = name
	p.OrigName = name
	if tag == "" {
		return
	}
	p.Parse(tag)

	if typ != nil && typ.Kind() == reflect.Map {
		p.MapKeyProp = new(Properties)
		p.MapKeyProp.init(nil, "Key", f.Tag.Get("protobuf_key"), nil)
		p.MapValProp = new(Properties)
		p.MapValProp.init(nil, "Value", f.Tag.Get("protobuf_val"), nil)
	}
}

var propertiesCache sync.Map // map[reflect.Type]*StructProperties

// GetProperties returns the list of properties for the type represented by t.
// t must represent a generated struct type of a protocol message.
func GetProperties(t reflect.Type) *StructProperties {
	if p, ok := propertiesCache.Load(t); ok {
		return p.(*StructProperties)
	}
	p, _ := propertiesCache.LoadOrStore(t, newProperties(t))
	return p.(*StructProperties)
}

func newProperties(t reflect.Type) *StructProperties {
	if t.Kind() != reflect.Struct {
		panic("proto: type must have kind struct")
	}

	prop := new(StructProperties)

	// Construct a list of properties for each field in the struct.
	for i := 0; i < t.NumField(); i++ {
		p := new(Properties)
		f := t.Field(i)
		p.init(f.Type, f.Name, f.Tag.Get("protobuf"), &f)

		if name := f.Tag.Get("protobuf_oneof"); name != "" {
			p.OrigName = name
		}
		prop.Prop = append(prop.Prop, p)
	}

	// Construct a mapping of oneof field names to properties.
	var oneofWrappers []interface{}
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofFuncs"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[3].Interface().([]interface{})
	}
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofWrappers"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0].Interface().([]interface{})
	}
	if len(oneofWrappers) > 0 {
		prop.OneofTypes = make(map[string]*OneofProperties)
		for _, wrapper := range oneofWrappers {
			p := &OneofProperties{
				Type: reflect.ValueOf(wrapper).Type(), // *T
				Prop: new(Properties),
			}
			f := p.Type.Elem().Field(0)
			p.Prop.Name = f.Name
			p.Prop.Parse(f.Tag.Get("protobuf"))

			// Determine the struct field that contains this oneof.
			// Each wrapper is assignable to exactly one parent field.
			for i := 0; i < t.NumField(); i++ {
				if p.Type.AssignableTo(t.Field(i).Type) {
					p.Field = i
					break
				}
			}
			prop.OneofTypes[p.Prop.OrigName] = p
		}
	}

	return prop
}
