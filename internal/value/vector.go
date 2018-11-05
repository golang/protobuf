// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package value

import (
	"reflect"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

func VectorOf(p interface{}, c Converter) pref.Vector {
	// TODO: Validate that p is a *[]T?
	rv := reflect.ValueOf(p).Elem()
	return vectorReflect{rv, c}
}

type vectorReflect struct {
	v    reflect.Value // addressable []T
	conv Converter
}

func (vs vectorReflect) Len() int {
	return vs.v.Len()
}
func (vs vectorReflect) Get(i int) pref.Value {
	return vs.conv.PBValueOf(vs.v.Index(i))
}
func (vs vectorReflect) Set(i int, v pref.Value) {
	vs.v.Index(i).Set(vs.conv.GoValueOf(v))
}
func (vs vectorReflect) Append(v pref.Value) {
	vs.v.Set(reflect.Append(vs.v, vs.conv.GoValueOf(v)))
}
func (vs vectorReflect) Mutable(i int) pref.Mutable {
	// Mutable is only valid for messages and panics for other kinds.
	rv := vs.v.Index(i)
	if rv.IsNil() {
		// TODO: Is checking for nil proper behavior for custom messages?
		pv := pref.ValueOf(vs.conv.NewMessage())
		rv.Set(vs.conv.GoValueOf(pv))
	}
	return rv.Interface().(pref.Message)
}
func (vs vectorReflect) MutableAppend() pref.Mutable {
	// MutableAppend is only valid for messages and panics for other kinds.
	pv := pref.ValueOf(vs.conv.NewMessage())
	vs.v.Set(reflect.Append(vs.v, vs.conv.GoValueOf(pv)))
	return vs.v.Index(vs.Len() - 1).Interface().(pref.Message)
}
func (vs vectorReflect) Truncate(i int) {
	vs.v.Set(vs.v.Slice(0, i))
}
func (vs vectorReflect) Unwrap() interface{} {
	return vs.v.Interface()
}
func (vs vectorReflect) ProtoMutable() {}

var (
	_ pref.Vector = vectorReflect{}
	_ Unwrapper   = vectorReflect{}
)
