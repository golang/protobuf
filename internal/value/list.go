// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value

import (
	"reflect"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

func ListOf(p interface{}, c Converter) interface {
	pref.List
	Unwrapper
} {
	// TODO: Validate that p is a *[]T?
	rv := reflect.ValueOf(p)
	return listReflect{rv, c}
}

type listReflect struct {
	v    reflect.Value // *[]T
	conv Converter
}

func (ls listReflect) Len() int {
	if ls.v.IsNil() {
		return 0
	}
	return ls.v.Elem().Len()
}
func (ls listReflect) Get(i int) pref.Value {
	return ls.conv.PBValueOf(ls.v.Elem().Index(i))
}
func (ls listReflect) Set(i int, v pref.Value) {
	ls.v.Elem().Index(i).Set(ls.conv.GoValueOf(v))
}
func (ls listReflect) Append(v pref.Value) {
	ls.v.Elem().Set(reflect.Append(ls.v.Elem(), ls.conv.GoValueOf(v)))
}
func (ls listReflect) Truncate(i int) {
	ls.v.Elem().Set(ls.v.Elem().Slice(0, i))
}
func (ls listReflect) NewMessage() pref.Message {
	return ls.conv.MessageType.New()
}
func (ls listReflect) ProtoUnwrap() interface{} {
	return ls.v.Interface()
}
