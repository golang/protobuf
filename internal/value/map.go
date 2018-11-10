// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value

import (
	"reflect"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

func MapOf(p interface{}, kc, kv Converter) pref.Map {
	// TODO: Validate that p is a *map[K]V?
	rv := reflect.ValueOf(p).Elem()
	return mapReflect{rv, kc, kv}
}

type mapReflect struct {
	v       reflect.Value // addressable map[K]V
	keyConv Converter
	valConv Converter
}

func (ms mapReflect) Len() int {
	return ms.v.Len()
}
func (ms mapReflect) Has(k pref.MapKey) bool {
	rk := ms.keyConv.GoValueOf(k.Value())
	rv := ms.v.MapIndex(rk)
	return rv.IsValid()
}
func (ms mapReflect) Get(k pref.MapKey) pref.Value {
	rk := ms.keyConv.GoValueOf(k.Value())
	rv := ms.v.MapIndex(rk)
	if !rv.IsValid() {
		return pref.Value{}
	}
	return ms.valConv.PBValueOf(rv)
}
func (ms mapReflect) Set(k pref.MapKey, v pref.Value) {
	if ms.v.IsNil() {
		ms.v.Set(reflect.MakeMap(ms.v.Type()))
	}
	rk := ms.keyConv.GoValueOf(k.Value())
	rv := ms.valConv.GoValueOf(v)
	ms.v.SetMapIndex(rk, rv)
}
func (ms mapReflect) Clear(k pref.MapKey) {
	rk := ms.keyConv.GoValueOf(k.Value())
	ms.v.SetMapIndex(rk, reflect.Value{})
}
func (ms mapReflect) Mutable(k pref.MapKey) pref.Mutable {
	// Mutable is only valid for messages and panics for other kinds.
	if ms.v.IsNil() {
		ms.v.Set(reflect.MakeMap(ms.v.Type()))
	}
	rk := ms.keyConv.GoValueOf(k.Value())
	rv := ms.v.MapIndex(rk)
	if !rv.IsValid() || rv.IsNil() {
		// TODO: Is checking for nil proper behavior for custom messages?
		pv := pref.ValueOf(ms.valConv.MessageType.New().ProtoReflect())
		rv = ms.valConv.GoValueOf(pv)
		ms.v.SetMapIndex(rk, rv)
	}
	return rv.Interface().(pref.Message)
}
func (ms mapReflect) Range(f func(pref.MapKey, pref.Value) bool) {
	for _, k := range ms.v.MapKeys() {
		if v := ms.v.MapIndex(k); v.IsValid() {
			pk := ms.keyConv.PBValueOf(k).MapKey()
			pv := ms.valConv.PBValueOf(v)
			if !f(pk, pv) {
				return
			}
		}
	}
}
func (ms mapReflect) Unwrap() interface{} {
	return ms.v.Interface()
}
func (ms mapReflect) ProtoMutable() {}

var (
	_ pref.Map  = mapReflect{}
	_ Unwrapper = mapReflect{}
)
