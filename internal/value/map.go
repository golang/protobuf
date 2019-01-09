// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value

import (
	"reflect"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

func MapOf(p interface{}, kc, kv Converter) interface {
	pref.Map
	Unwrapper
} {
	// TODO: Validate that p is a *map[K]V?
	rv := reflect.ValueOf(p)
	return mapReflect{rv, kc, kv}
}

type mapReflect struct {
	v       reflect.Value // *map[K]V
	keyConv Converter
	valConv Converter
}

func (ms mapReflect) Len() int {
	if ms.v.IsNil() {
		return 0
	}
	return ms.v.Elem().Len()
}
func (ms mapReflect) Has(k pref.MapKey) bool {
	if ms.v.IsNil() {
		return false
	}
	rk := ms.keyConv.GoValueOf(k.Value())
	rv := ms.v.Elem().MapIndex(rk)
	return rv.IsValid()
}
func (ms mapReflect) Get(k pref.MapKey) pref.Value {
	if ms.v.IsNil() {
		return pref.Value{}
	}
	rk := ms.keyConv.GoValueOf(k.Value())
	rv := ms.v.Elem().MapIndex(rk)
	if !rv.IsValid() {
		return pref.Value{}
	}
	return ms.valConv.PBValueOf(rv)
}
func (ms mapReflect) Set(k pref.MapKey, v pref.Value) {
	if ms.v.Elem().IsNil() {
		ms.v.Elem().Set(reflect.MakeMap(ms.v.Elem().Type()))
	}
	rk := ms.keyConv.GoValueOf(k.Value())
	rv := ms.valConv.GoValueOf(v)
	ms.v.Elem().SetMapIndex(rk, rv)
}
func (ms mapReflect) Clear(k pref.MapKey) {
	rk := ms.keyConv.GoValueOf(k.Value())
	ms.v.Elem().SetMapIndex(rk, reflect.Value{})
}
func (ms mapReflect) Range(f func(pref.MapKey, pref.Value) bool) {
	if ms.v.IsNil() {
		return
	}
	for _, k := range ms.v.Elem().MapKeys() {
		if v := ms.v.Elem().MapIndex(k); v.IsValid() {
			pk := ms.keyConv.PBValueOf(k).MapKey()
			pv := ms.valConv.PBValueOf(v)
			if !f(pk, pv) {
				return
			}
		}
	}
}
func (ms mapReflect) NewMessage() pref.Message {
	return ms.valConv.MessageType.New()
}
func (ms mapReflect) ProtoUnwrap() interface{} {
	return ms.v.Interface()
}
