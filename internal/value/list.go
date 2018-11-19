// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value

import (
	"reflect"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

func ListOf(p interface{}, c Converter) pref.List {
	// TODO: Validate that p is a *[]T?
	rv := reflect.ValueOf(p).Elem()
	return listReflect{rv, c}
}

type listReflect struct {
	v    reflect.Value // addressable []T
	conv Converter
}

func (ls listReflect) Len() int {
	return ls.v.Len()
}
func (ls listReflect) Get(i int) pref.Value {
	return ls.conv.PBValueOf(ls.v.Index(i))
}
func (ls listReflect) Set(i int, v pref.Value) {
	ls.v.Index(i).Set(ls.conv.GoValueOf(v))
}
func (ls listReflect) Append(v pref.Value) {
	ls.v.Set(reflect.Append(ls.v, ls.conv.GoValueOf(v)))
}
func (ls listReflect) Mutable(i int) pref.Mutable {
	// Mutable is only valid for messages and panics for other kinds.
	return ls.conv.PBValueOf(ls.v.Index(i)).Message()
}
func (ls listReflect) MutableAppend() pref.Mutable {
	// MutableAppend is only valid for messages and panics for other kinds.
	pv := pref.ValueOf(ls.conv.MessageType.New().ProtoReflect())
	ls.v.Set(reflect.Append(ls.v, ls.conv.GoValueOf(pv)))
	return pv.Message()
}
func (ls listReflect) Truncate(i int) {
	ls.v.Set(ls.v.Slice(0, i))
}
func (ls listReflect) Unwrap() interface{} {
	return ls.v.Addr().Interface()
}
func (ls listReflect) ProtoMutable() {}

var (
	_ pref.List = listReflect{}
	_ Unwrapper = listReflect{}
)
