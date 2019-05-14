// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"fmt"
	"math"
	"reflect"
	"sync"

	ptype "google.golang.org/protobuf/internal/prototype"
	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// wrapEnum wraps v as a protoreflect.Enum,
// where v must be a int32 kind and not implement the v2 API already.
func wrapEnum(v reflect.Value) pref.Enum {
	et := loadEnumType(v.Type())
	return et.New(pref.EnumNumber(v.Int()))
}

var enumTypeCache sync.Map // map[reflect.Type]protoreflect.EnumType

// loadEnumType dynamically loads a protoreflect.EnumType for t,
// where t must be an int32 kind and not implement the v2 API already.
func loadEnumType(t reflect.Type) pref.EnumType {
	// Fast-path: check if a EnumType is cached for this concrete type.
	if et, ok := enumTypeCache.Load(t); ok {
		return et.(pref.EnumType)
	}

	// Slow-path: derive enum descriptor and initialize EnumType.
	var m sync.Map // map[protoreflect.EnumNumber]proto.Enum
	ed := LoadEnumDesc(t)
	et := ptype.GoEnum(ed, func(et pref.EnumType, n pref.EnumNumber) pref.Enum {
		if e, ok := m.Load(n); ok {
			return e.(pref.Enum)
		}
		e := &enumWrapper{num: n, pbTyp: et, goTyp: t}
		m.Store(n, e)
		return e
	})
	if et, ok := enumTypeCache.LoadOrStore(t, et); ok {
		return et.(pref.EnumType)
	}
	return et
}

type enumWrapper struct {
	num   pref.EnumNumber
	pbTyp pref.EnumType
	goTyp reflect.Type
}

// TODO: Remove this.
func (e *enumWrapper) Type() pref.EnumType {
	return e.pbTyp
}
func (e *enumWrapper) Descriptor() pref.EnumDescriptor {
	return e.pbTyp.Descriptor()
}
func (e *enumWrapper) Number() pref.EnumNumber {
	return e.num
}
func (e *enumWrapper) ProtoReflect() pref.Enum {
	return e
}
func (e *enumWrapper) ProtoUnwrap() interface{} {
	v := reflect.New(e.goTyp).Elem()
	v.SetInt(int64(e.num))
	return v.Interface()
}

var (
	_ pref.Enum        = (*enumWrapper)(nil)
	_ pvalue.Unwrapper = (*enumWrapper)(nil)
)

var enumDescCache sync.Map // map[reflect.Type]protoreflect.EnumDescriptor

var enumNumberType = reflect.TypeOf(pref.EnumNumber(0))

// LoadEnumDesc returns an EnumDescriptor derived from the Go type,
// which must be an int32 kind and not implement the v2 API already.
//
// This is exported for testing purposes.
func LoadEnumDesc(t reflect.Type) pref.EnumDescriptor {
	// Fast-path: check if an EnumDescriptor is cached for this concrete type.
	if ed, ok := enumDescCache.Load(t); ok {
		return ed.(pref.EnumDescriptor)
	}

	// Slow-path: initialize EnumDescriptor from the proto descriptor.
	if t.Kind() != reflect.Int32 || t.PkgPath() == "" {
		panic(fmt.Sprintf("got %v, want named int32 kind", t))
	}
	if t == enumNumberType {
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
		fd := loadFileDesc(b)

		// Derive syntax.
		switch fd.GetSyntax() {
		case "proto2", "":
			e.Syntax = pref.Proto2
		case "proto3":
			e.Syntax = pref.Proto3
		}

		// Derive the full name and correct enum descriptor.
		var ed *enumDescriptorProto
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
		// most operations continue to work. For example, textpb and jsonpb
		// will be unable to parse a message with an enum value by name.
		e.Syntax = pref.Proto2
		e.FullName = deriveFullName(t)
		e.Values = []ptype.EnumValue{{Name: "INVALID", Number: math.MinInt32}}
	}

	ed, err := ptype.NewEnum(e)
	if err != nil {
		panic(err)
	}
	if ed, ok := enumDescCache.LoadOrStore(t, ed); ok {
		return ed.(pref.EnumDescriptor)
	}
	return ed
}
