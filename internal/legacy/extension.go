// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"fmt"
	"reflect"
	"sync"

	papi "github.com/golang/protobuf/protoapi"
	ptag "github.com/golang/protobuf/v2/internal/encoding/tag"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	pfmt "github.com/golang/protobuf/v2/internal/typefmt"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
)

// extensionDescKey is a comparable version of protoapi.ExtensionDesc
// suitable for use as a key in a map.
type extensionDescKey struct {
	typeV2        pref.ExtensionType
	extendedType  reflect.Type
	extensionType reflect.Type
	field         int32
	name          string
	tag           string
	filename      string
}

func extensionDescKeyOf(d *papi.ExtensionDesc) extensionDescKey {
	return extensionDescKey{
		d.Type,
		reflect.TypeOf(d.ExtendedType),
		reflect.TypeOf(d.ExtensionType),
		d.Field, d.Name, d.Tag, d.Filename,
	}
}

var (
	extensionTypeCache sync.Map // map[extensionDescKey]protoreflect.ExtensionType
	extensionDescCache sync.Map // map[protoreflect.ExtensionType]*protoapi.ExtensionDesc
)

// extensionDescFromType converts a v2 protoreflect.ExtensionType to a
// v1 protoapi.ExtensionDesc. The returned ExtensionDesc must not be mutated.
func extensionDescFromType(t pref.ExtensionType) *papi.ExtensionDesc {
	// Fast-path: check the cache for whether this ExtensionType has already
	// been converted to a legacy descriptor.
	if d, ok := extensionDescCache.Load(t); ok {
		return d.(*papi.ExtensionDesc)
	}

	// Determine the parent type if possible.
	var parent papi.Message
	if mt, ok := t.ExtendedType().(pref.MessageType); ok {
		// Create a new parent message and unwrap it if possible.
		mv := mt.New().Interface()
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
		if ed, ok := ev.(enumV1); ok && protoPkg == "" {
			b, _ := ed.EnumDescriptor()
			protoPkg = loadFileDesc(b).GetPackage()
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
	d := &papi.ExtensionDesc{
		Type:          t,
		ExtendedType:  parent,
		ExtensionType: reflect.Zero(extType).Interface(),
		Field:         int32(t.Number()),
		Name:          string(t.FullName()),
		Tag:           ptag.Marshal(t, enumName),
		Filename:      filename,
	}
	extensionDescCache.Store(t, d)
	return d
}

// extensionTypeFromDesc converts a v1 protoapi.ExtensionDesc to a
// v2 protoreflect.ExtensionType. The returned descriptor type takes ownership
// of the input extension desc. The input must not be mutated so long as the
// returned type is still in use.
func extensionTypeFromDesc(d *papi.ExtensionDesc) pref.ExtensionType {
	// Fast-path: check whether an extension type is already nested within.
	if d.Type != nil {
		// Cache descriptor for future extensionDescFromType operation.
		// This assumes that there is only one legacy protoapi.ExtensionDesc
		// that wraps any given specific protoreflect.ExtensionType.
		extensionDescCache.LoadOrStore(d.Type, d)
		return d.Type
	}

	// Fast-path: check the cache for whether this ExtensionType has already
	// been converted from a legacy descriptor.
	dk := extensionDescKeyOf(d)
	if t, ok := extensionTypeCache.Load(dk); ok {
		return t.(pref.ExtensionType)
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
		ExtendedType: pimpl.Export{}.MessageTypeOf(d.ExtendedType),
	})
	if err != nil {
		panic(err)
	}
	var zv interface{}
	switch xd.Kind() {
	case pref.EnumKind, pref.MessageKind, pref.GroupKind:
		zv = reflect.Zero(t).Interface()
	}
	xt := pimpl.Export{}.ExtensionTypeOf(xd, zv)

	// Cache the conversion for both directions.
	extensionDescCache.Store(xt, d)
	extensionTypeCache.Store(dk, xt)
	return xt
}

// extensionTypeOf returns a protoreflect.ExtensionType where the GoType
// is the underlying v1 Go type instead of the wrapper types used to present
// v1 Go types as if they satisfied the v2 API.
//
// This function is only valid if xd.Kind is an enum or message.
func extensionTypeOf(xd pref.ExtensionDescriptor, t reflect.Type) pref.ExtensionType {
	// Step 1: Create an ExtensionType where GoType is the wrapper type.
	conv := pvalue.NewLegacyConverter(t, xd.Kind(), Export{})
	xt := ptype.GoExtension(xd, conv.EnumType, conv.MessageType)

	// Step 2: Wrap ExtensionType such that GoType presents the legacy Go type.
	xt2 := &extensionType{ExtensionType: xt}
	if xd.Cardinality() != pref.Repeated {
		xt2.typ = t
		xt2.new = func() pref.Value {
			return xt.New()
		}
		xt2.valueOf = func(v interface{}) pref.Value {
			if reflect.TypeOf(v) != xt2.typ {
				panic(fmt.Sprintf("invalid type: got %T, want %v", v, xt2.typ))
			}
			if xd.Kind() == pref.EnumKind {
				return xt.ValueOf(Export{}.EnumOf(v))
			} else {
				return xt.ValueOf(Export{}.MessageOf(v).Interface())
			}
		}
		xt2.interfaceOf = func(v pref.Value) interface{} {
			return xt.InterfaceOf(v).(pvalue.Unwrapper).ProtoUnwrap()
		}
	} else {
		xt2.typ = reflect.PtrTo(reflect.SliceOf(t))
		xt2.new = func() pref.Value {
			v := reflect.New(xt2.typ.Elem()).Interface()
			return pref.ValueOf(pvalue.ListOf(v, conv))
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

type extensionType struct {
	pref.ExtensionType
	typ         reflect.Type
	new         func() pref.Value
	valueOf     func(interface{}) pref.Value
	interfaceOf func(pref.Value) interface{}
}

func (x *extensionType) GoType() reflect.Type                 { return x.typ }
func (x *extensionType) New() pref.Value                      { return x.new() }
func (x *extensionType) ValueOf(v interface{}) pref.Value     { return x.valueOf(v) }
func (x *extensionType) InterfaceOf(v pref.Value) interface{} { return x.interfaceOf(v) }
func (x *extensionType) Format(s fmt.State, r rune)           { pfmt.FormatDesc(s, r, x) }
