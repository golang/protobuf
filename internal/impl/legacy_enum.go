// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"math"
	"reflect"
	"sync"

	ptype "google.golang.org/protobuf/internal/prototype"
	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/prototype"
)

// legacyWrapEnum wraps v as a protoreflect.Enum,
// where v must be a int32 kind and not implement the v2 API already.
func legacyWrapEnum(v reflect.Value) pref.Enum {
	et := legacyLoadEnumType(v.Type())
	return et.New(pref.EnumNumber(v.Int()))
}

var legacyEnumTypeCache sync.Map // map[reflect.Type]protoreflect.EnumType

// legacyLoadEnumType dynamically loads a protoreflect.EnumType for t,
// where t must be an int32 kind and not implement the v2 API already.
func legacyLoadEnumType(t reflect.Type) pref.EnumType {
	// Fast-path: check if a EnumType is cached for this concrete type.
	if et, ok := legacyEnumTypeCache.Load(t); ok {
		return et.(pref.EnumType)
	}

	// Slow-path: derive enum descriptor and initialize EnumType.
	var et pref.EnumType
	var m sync.Map // map[protoreflect.EnumNumber]proto.Enum
	ed := LegacyLoadEnumDesc(t)
	et = &prototype.Enum{
		EnumDescriptor: ed,
		NewEnum: func(n pref.EnumNumber) pref.Enum {
			if e, ok := m.Load(n); ok {
				return e.(pref.Enum)
			}
			e := &legacyEnumWrapper{num: n, pbTyp: et, goTyp: t}
			m.Store(n, e)
			return e
		},
	}
	if et, ok := legacyEnumTypeCache.LoadOrStore(t, et); ok {
		return et.(pref.EnumType)
	}
	return et
}

type legacyEnumWrapper struct {
	num   pref.EnumNumber
	pbTyp pref.EnumType
	goTyp reflect.Type
}

// TODO: Remove this.
func (e *legacyEnumWrapper) Type() pref.EnumType {
	return e.pbTyp
}
func (e *legacyEnumWrapper) Descriptor() pref.EnumDescriptor {
	return e.pbTyp.Descriptor()
}
func (e *legacyEnumWrapper) Number() pref.EnumNumber {
	return e.num
}
func (e *legacyEnumWrapper) ProtoReflect() pref.Enum {
	return e
}
func (e *legacyEnumWrapper) ProtoUnwrap() interface{} {
	v := reflect.New(e.goTyp).Elem()
	v.SetInt(int64(e.num))
	return v.Interface()
}

var (
	_ pref.Enum        = (*legacyEnumWrapper)(nil)
	_ pvalue.Unwrapper = (*legacyEnumWrapper)(nil)
)

var legacyEnumDescCache sync.Map // map[reflect.Type]protoreflect.EnumDescriptor

var legacyEnumNumberType = reflect.TypeOf(pref.EnumNumber(0))

// LegacyLoadEnumDesc returns an EnumDescriptor derived from the Go type,
// which must be an int32 kind and not implement the v2 API already.
//
// This is exported for testing purposes.
func LegacyLoadEnumDesc(t reflect.Type) pref.EnumDescriptor {
	// Fast-path: check if an EnumDescriptor is cached for this concrete type.
	if ed, ok := legacyEnumDescCache.Load(t); ok {
		return ed.(pref.EnumDescriptor)
	}

	// Slow-path: initialize EnumDescriptor from the proto descriptor.
	if t.Kind() != reflect.Int32 || t.PkgPath() == "" {
		panic(fmt.Sprintf("got %v, want named int32 kind", t))
	}
	if t == legacyEnumNumberType {
		panic(fmt.Sprintf("cannot be %v", t))
	}

	// Derive the enum descriptor from the raw descriptor proto.
	e := new(ptype.StandaloneEnum)
	ev := reflect.Zero(t).Interface()
	if _, ok := ev.(pref.Enum); ok {
		panic(fmt.Sprintf("%v already implements proto.Enum", t))
	}
	if ed, ok := ev.(enumV1); ok {
		b, idxs := ed.EnumDescriptor()
		fd := legacyLoadFileDesc(b)

		// Derive syntax.
		switch fd.GetSyntax() {
		case "proto2", "":
			e.Syntax = pref.Proto2
		case "proto3":
			e.Syntax = pref.Proto3
		}

		// Derive the full name and correct enum descriptor.
		var ed *legacyEnumDescriptorProto
		e.FullName = pref.FullName(fd.GetPackage())
		if len(idxs) == 1 {
			ed = fd.EnumType[idxs[0]]
			e.FullName = e.FullName.Append(pref.Name(ed.GetName()))
		} else {
			md := fd.MessageType[idxs[0]]
			e.FullName = e.FullName.Append(pref.Name(md.GetName()))
			for _, i := range idxs[1 : len(idxs)-1] {
				md = md.NestedType[i]
				e.FullName = e.FullName.Append(pref.Name(md.GetName()))
			}
			ed = md.EnumType[idxs[len(idxs)-1]]
			e.FullName = e.FullName.Append(pref.Name(ed.GetName()))
		}

		// Derive the enum values.
		for _, vd := range ed.Value {
			e.Values = append(e.Values, ptype.EnumValue{
				Name:   pref.Name(vd.GetName()),
				Number: pref.EnumNumber(vd.GetNumber()),
			})
		}
	} else {
		// If the type does not implement enumV1, then there is no reliable
		// way to derive the original protobuf type information.
		// We are unable to use the global enum registry since it is
		// unfortunately keyed by the full name, which we do not know.
		// Furthermore, some generated enums register with a fork of
		// golang/protobuf so the enum may not even be found in the registry.
		//
		// Instead, create a bogus enum descriptor to ensure that
		// most operations continue to work. For example, prototext and protojson
		// will be unable to parse a message with an enum value by name.
		e.Syntax = pref.Proto2
		e.FullName = legacyDeriveFullName(t)
		e.Values = []ptype.EnumValue{{Name: "INVALID", Number: math.MinInt32}}
	}

	ed, err := ptype.NewEnum(e)
	if err != nil {
		panic(err)
	}
	if ed, ok := legacyEnumDescCache.LoadOrStore(t, ed); ok {
		return ed.(pref.EnumDescriptor)
	}
	return ed
}
