// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sync"

	pvalue "google.golang.org/protobuf/internal/value"
	"google.golang.org/protobuf/reflect/protoreflect"
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

	// Slow-path: initialize EnumDescriptor from the raw descriptor.
	ev := reflect.Zero(t).Interface()
	if _, ok := ev.(pref.Enum); ok {
		panic(fmt.Sprintf("%v already implements proto.Enum", t))
	}
	edV1, ok := ev.(enumV1)
	if !ok {
		panic(fmt.Sprintf("enum %v is no longer supported; please regenerate", t))
	}
	b, idxs := edV1.EnumDescriptor()

	var ed pref.EnumDescriptor
	if len(idxs) == 1 {
		ed = legacyLoadFileDesc(b).Enums().Get(idxs[0])
	} else {
		md := legacyLoadFileDesc(b).Messages().Get(idxs[0])
		for _, i := range idxs[1 : len(idxs)-1] {
			md = md.Messages().Get(i)
		}
		ed = md.Enums().Get(idxs[len(idxs)-1])
	}
	if ed, ok := legacyEnumDescCache.LoadOrStore(t, ed); ok {
		return ed.(protoreflect.EnumDescriptor)
	}
	return ed
}
