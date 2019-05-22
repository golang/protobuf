// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sync"

	"google.golang.org/protobuf/internal/descfmt"
	ptag "google.golang.org/protobuf/internal/encoding/tag"
	ptype "google.golang.org/protobuf/internal/prototype"
	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/reflect/prototype"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// legacyExtensionDescKey is a comparable version of protoiface.ExtensionDescV1
// suitable for use as a key in a map.
type legacyExtensionDescKey struct {
	typeV2        pref.ExtensionType
	extendedType  reflect.Type
	extensionType reflect.Type
	field         int32
	name          string
	tag           string
	filename      string
}

func legacyExtensionDescKeyOf(d *piface.ExtensionDescV1) legacyExtensionDescKey {
	return legacyExtensionDescKey{
		d.Type,
		reflect.TypeOf(d.ExtendedType),
		reflect.TypeOf(d.ExtensionType),
		d.Field, d.Name, d.Tag, d.Filename,
	}
}

var (
	legacyExtensionTypeCache sync.Map // map[legacyExtensionDescKey]protoreflect.ExtensionType
	legacyExtensionDescCache sync.Map // map[protoreflect.ExtensionType]*protoiface.ExtensionDescV1
)

// legacyExtensionDescFromType converts a v2 protoreflect.ExtensionType to a
// protoiface.ExtensionDescV1. The returned ExtensionDesc must not be mutated.
func legacyExtensionDescFromType(xt pref.ExtensionType) *piface.ExtensionDescV1 {
	// Fast-path: check whether an extension desc is already nested within.
	if xt, ok := xt.(interface {
		ProtoLegacyExtensionDesc() *piface.ExtensionDescV1
	}); ok {
		if d := xt.ProtoLegacyExtensionDesc(); d != nil {
			return d
		}
	}

	// Fast-path: check the cache for whether this ExtensionType has already
	// been converted to a legacy descriptor.
	if d, ok := legacyExtensionDescCache.Load(xt); ok {
		return d.(*piface.ExtensionDescV1)
	}

	// Determine the parent type if possible.
	var parent piface.MessageV1
	messageName := xt.Descriptor().ContainingMessage().FullName()
	if mt, _ := preg.GlobalTypes.FindMessageByName(messageName); mt != nil {
		// Create a new parent message and unwrap it if possible.
		mv := mt.New().Interface()
		t := reflect.TypeOf(mv)
		if mv, ok := mv.(pvalue.Unwrapper); ok {
			t = reflect.TypeOf(mv.ProtoUnwrap())
		}

		// Check whether the message implements the legacy v1 Message interface.
		mz := reflect.Zero(t).Interface()
		if mz, ok := mz.(piface.MessageV1); ok {
			parent = mz
		}
	}

	// Determine the v1 extension type, which is unfortunately not the same as
	// the v2 ExtensionType.GoType.
	extType := xt.GoType()
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
	if xt.Descriptor().Kind() == pref.EnumKind {
		// Derive Go type name.
		t := extType
		if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
			t = t.Elem()
		}
		enumName = t.Name()

		// Derive the proto package name.
		// For legacy enums, obtain the proto package from the raw descriptor.
		var protoPkg string
		if fd := xt.Descriptor().Enum().ParentFile(); fd != nil {
			protoPkg = string(fd.Package())
		}
		if ed, ok := reflect.Zero(t).Interface().(enumV1); ok && protoPkg == "" {
			b, _ := ed.EnumDescriptor()
			protoPkg = legacyLoadFileDesc(b).GetPackage()
		}

		if protoPkg != "" {
			enumName = protoPkg + "." + enumName
		}
	}

	// Derive the proto file that the extension was declared within.
	var filename string
	if fd := xt.Descriptor().ParentFile(); fd != nil {
		filename = fd.Path()
	}

	// Construct and return a ExtensionDescV1.
	d := &piface.ExtensionDescV1{
		Type:          xt,
		ExtendedType:  parent,
		ExtensionType: reflect.Zero(extType).Interface(),
		Field:         int32(xt.Descriptor().Number()),
		Name:          string(xt.Descriptor().FullName()),
		Tag:           ptag.Marshal(xt.Descriptor(), enumName),
		Filename:      filename,
	}
	if d, ok := legacyExtensionDescCache.LoadOrStore(xt, d); ok {
		return d.(*piface.ExtensionDescV1)
	}
	return d
}

// legacyExtensionTypeFromDesc converts a protoiface.ExtensionDescV1 to a
// v2 protoreflect.ExtensionType. The returned descriptor type takes ownership
// of the input extension desc. The input must not be mutated so long as the
// returned type is still in use.
func legacyExtensionTypeFromDesc(d *piface.ExtensionDescV1) pref.ExtensionType {
	// Fast-path: check whether an extension type is already nested within.
	if d.Type != nil {
		return d.Type
	}

	// Fast-path: check the cache for whether this ExtensionType has already
	// been converted from a legacy descriptor.
	dk := legacyExtensionDescKeyOf(d)
	if t, ok := legacyExtensionTypeCache.Load(dk); ok {
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
	var ed pref.EnumDescriptor
	var md pref.MessageDescriptor
	switch f.Kind {
	case pref.EnumKind:
		if e, ok := reflect.Zero(t).Interface().(pref.Enum); ok {
			ed = e.Descriptor()
		} else {
			ed = LegacyLoadEnumDesc(t)
		}
	case pref.MessageKind, pref.GroupKind:
		if m, ok := reflect.Zero(t).Interface().(pref.ProtoMessage); ok {
			md = m.ProtoReflect().Descriptor()
		} else {
			md = LegacyLoadMessageDesc(t)
		}
	}
	xd, err := ptype.NewExtension(&ptype.StandaloneExtension{
		FullName:     pref.FullName(d.Name),
		Number:       pref.FieldNumber(d.Field),
		Cardinality:  f.Cardinality,
		Kind:         f.Kind,
		Default:      f.Default,
		Options:      f.Options,
		EnumType:     ed,
		MessageType:  md,
		ExtendedType: Export{}.MessageDescriptorOf(d.ExtendedType),
	})
	if err != nil {
		panic(err)
	}
	xt := LegacyExtensionTypeOf(xd, t)

	// Cache the conversion for both directions.
	legacyExtensionDescCache.LoadOrStore(xt, d)
	if xt, ok := legacyExtensionTypeCache.LoadOrStore(dk, xt); ok {
		return xt.(pref.ExtensionType)
	}
	return xt
}

// LegacyExtensionTypeOf returns a protoreflect.ExtensionType where the
// element type of the field is t. The type t must be provided if the field
// is an enum or message.
//
// This is exported for testing purposes.
func LegacyExtensionTypeOf(xd pref.ExtensionDescriptor, t reflect.Type) pref.ExtensionType {
	var conv pvalue.Converter
	var isLegacy bool
	xt := &prototype.Extension{ExtensionDescriptor: xd}
	switch xd.Kind() {
	case pref.EnumKind:
		conv, isLegacy = newConverter(t, xd.Kind())
		xt.NewEnum = conv.NewEnum
	case pref.MessageKind, pref.GroupKind:
		conv, isLegacy = newConverter(t, xd.Kind())
		xt.NewMessage = conv.NewMessage
	default:
		// Extension types for non-enums and non-messages are simple.
		return &prototype.Extension{ExtensionDescriptor: xd}
	}
	if !isLegacy {
		return xt
	}

	// Wrap ExtensionType such that GoType presents the legacy Go type.
	xt2 := &legacyExtensionType{ExtensionType: xt}
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

type legacyExtensionType struct {
	pref.ExtensionType
	typ         reflect.Type
	new         func() pref.Value
	valueOf     func(interface{}) pref.Value
	interfaceOf func(pref.Value) interface{}
}

func (x *legacyExtensionType) GoType() reflect.Type                 { return x.typ }
func (x *legacyExtensionType) New() pref.Value                      { return x.new() }
func (x *legacyExtensionType) ValueOf(v interface{}) pref.Value     { return x.valueOf(v) }
func (x *legacyExtensionType) InterfaceOf(v pref.Value) interface{} { return x.interfaceOf(v) }
func (x *legacyExtensionType) Format(s fmt.State, r rune)           { descfmt.FormatDesc(s, r, x.Descriptor()) }
