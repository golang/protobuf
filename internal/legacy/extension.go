// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"fmt"
	"reflect"

	papi "github.com/golang/protobuf/protoapi"
	ptag "github.com/golang/protobuf/v2/internal/encoding/tag"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
)

func legacyExtensionDescFromType(t pref.ExtensionType) *papi.ExtensionDesc {
	if t, ok := t.(dualExtensionType); ok {
		return t.desc
	}

	// Determine the parent type if possible.
	var parent papi.Message
	if mt, ok := t.ExtendedType().(pref.MessageType); ok {
		// Create a new parent message and unwrap it if possible.
		mv := mt.New()
		t := reflect.TypeOf(mv)
		if mv, ok := mv.(pvalue.Unwrapper); ok {
			t = reflect.TypeOf(mv.ProtoUnwrap())
		}

		// Check whether the message implements the legacy v1 Message interface.
		mz := reflect.Zero(t).Interface()
		if mz, ok := mz.(papi.Message); ok {
			parent = mz
		}
	}

	// Determine the v1 extension type, which is unfortunately not the same as
	// the v2 ExtensionType.GoType.
	extType := t.GoType()
	switch extType.Kind() {
	case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		extType = reflect.PtrTo(extType) // T -> *T for singular scalar fields
	case reflect.Ptr:
		if extType.Elem().Kind() == reflect.Slice {
			extType = extType.Elem() // *[]T -> []T for repeated fields
		}
	}

	// Reconstruct the legacy enum full name, which is an odd mixture of the
	// proto package name with the Go type name.
	var enumName string
	if t.Kind() == pref.EnumKind {
		// Derive Go type name.
		// For legacy enums, unwrap the wrapper to get the underlying Go type.
		et := t.EnumType().(pref.EnumType)
		var ev interface{} = et.New(0)
		if u, ok := ev.(pvalue.Unwrapper); ok {
			ev = u.ProtoUnwrap()
		}
		enumName = reflect.TypeOf(ev).Name()

		// Derive the proto package name.
		// For legacy enums, obtain the proto package from the raw descriptor.
		var protoPkg string
		if fd := parentFileDescriptor(et); fd != nil {
			protoPkg = string(fd.Package())
		}
		if ed, ok := ev.(legacyEnum); ok && protoPkg == "" {
			b, _ := ed.EnumDescriptor()
			protoPkg = legacyLoadFileDesc(b).GetPackage()
		}

		if protoPkg != "" {
			enumName = protoPkg + "." + enumName
		}
	}

	// Derive the proto file that the extension was declared within.
	var filename string
	if fd := parentFileDescriptor(t); fd != nil {
		filename = fd.Path()
	}

	// Construct and return a v1 ExtensionDesc.
	return &papi.ExtensionDesc{
		Type:          t,
		ExtendedType:  parent,
		ExtensionType: reflect.Zero(extType).Interface(),
		Field:         int32(t.Number()),
		Name:          string(t.FullName()),
		Tag:           ptag.Marshal(t, enumName),
		Filename:      filename,
	}
}

func legacyExtensionTypeFromDesc(d *papi.ExtensionDesc) pref.ExtensionType {
	if d.Type != nil {
		return dualExtensionType{d.Type, d}
	}

	// Derive basic field information from the struct tag.
	t := reflect.TypeOf(d.ExtensionType)
	isOptional := t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct
	isRepeated := t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
	if isOptional || isRepeated {
		t = t.Elem()
	}
	f := ptag.Unmarshal(d.Tag, t)

	// Construct a v2 ExtensionType.
	conv := pvalue.NewLegacyConverter(t, f.Kind, Export{})
	xd, err := ptype.NewExtension(&ptype.StandaloneExtension{
		FullName:     pref.FullName(d.Name),
		Number:       pref.FieldNumber(d.Field),
		Cardinality:  f.Cardinality,
		Kind:         f.Kind,
		Default:      f.Default,
		Options:      f.Options,
		EnumType:     conv.EnumType,
		MessageType:  conv.MessageType,
		ExtendedType: Export{}.MessageTypeOf(d.ExtendedType),
	})
	if err != nil {
		panic(err)
	}
	xt := pimpl.Export{}.ExtensionTypeOf(xd, reflect.Zero(t).Interface())
	return dualExtensionType{xt, d}
}

type dualExtensionType struct {
	pref.ExtensionType
	desc *papi.ExtensionDesc
}

// TODO: Provide custom stringer for dualExtensionType.

// legacyExtensionTypeOf returns a protoreflect.ExtensionType where the GoType
// is the underlying v1 Go type instead of the wrapper types used to present
// v1 Go types as if they satisfied the v2 API.
//
// This function is only valid if xd.Kind is an enum or message.
func legacyExtensionTypeOf(xd pref.ExtensionDescriptor, t reflect.Type) pref.ExtensionType {
	// Step 1: Create an ExtensionType where GoType is the wrapper type.
	conv := pvalue.NewLegacyConverter(t, xd.Kind(), Export{})
	xt := ptype.GoExtension(xd, conv.EnumType, conv.MessageType)

	// Step 2: Wrap ExtensionType such that GoType presents the legacy Go type.
	xt2 := &legacyExtensionType{ExtensionType: xt}
	if xd.Cardinality() != pref.Repeated {
		xt2.typ = t
		xt2.new = func() interface{} {
			return xt.New().(pvalue.Unwrapper).ProtoUnwrap()
		}
		xt2.valueOf = func(v interface{}) pref.Value {
			if reflect.TypeOf(v) != xt2.typ {
				panic(fmt.Sprintf("invalid type: got %T, want %v", v, xt2.typ))
			}
			if xd.Kind() == pref.EnumKind {
				return xt.ValueOf(Export{}.EnumOf(v))
			} else {
				return xt.ValueOf(Export{}.MessageOf(v))
			}
		}
		xt2.interfaceOf = func(v pref.Value) interface{} {
			return xt.InterfaceOf(v).(pvalue.Unwrapper).ProtoUnwrap()
		}
	} else {
		xt2.typ = reflect.PtrTo(reflect.SliceOf(t))
		xt2.new = func() interface{} {
			return reflect.New(xt2.typ.Elem()).Interface()
		}
		xt2.valueOf = func(v interface{}) pref.Value {
			if reflect.TypeOf(v) != xt2.typ {
				panic(fmt.Sprintf("invalid type: got %T, want %v", v, xt2.typ))
			}
			return pref.ValueOf(pvalue.ListOf(v, conv))
		}
		xt2.interfaceOf = func(pv pref.Value) interface{} {
			v := pv.List().(pvalue.Unwrapper).ProtoUnwrap()
			if reflect.TypeOf(v) != xt2.typ {
				panic(fmt.Sprintf("invalid type: got %T, want %v", v, xt2.typ))
			}
			return v
		}
	}
	return xt2
}

type legacyExtensionType struct {
	pref.ExtensionType
	typ         reflect.Type
	new         func() interface{}
	valueOf     func(interface{}) pref.Value
	interfaceOf func(pref.Value) interface{}
}

func (x *legacyExtensionType) GoType() reflect.Type                 { return x.typ }
func (x *legacyExtensionType) New() interface{}                     { return x.new() }
func (x *legacyExtensionType) ValueOf(v interface{}) pref.Value     { return x.valueOf(v) }
func (x *legacyExtensionType) InterfaceOf(v pref.Value) interface{} { return x.interfaceOf(v) }

// TODO: Provide custom stringer with the new GoType.
