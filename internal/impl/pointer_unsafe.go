// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !purego,!appengine

package impl

import (
	"reflect"
	"unsafe"
)

// offset represents the offset to a struct field, accessible from a pointer.
// The offset is the byte offset to the field from the start of the struct.
type offset uintptr

// offsetOf returns a field offset for the struct field.
func offsetOf(f reflect.StructField) offset {
	return offset(f.Offset)
}

// pointer is a pointer to a message struct or field.
type pointer struct{ p unsafe.Pointer }

// pointerOfValue returns v as a pointer.
func pointerOfValue(v reflect.Value) pointer {
	return pointer{p: unsafe.Pointer(v.Pointer())}
}

// pointerOfIface returns the pointer portion of an interface.
func pointerOfIface(v interface{}) pointer {
	type ifaceHeader struct {
		Type unsafe.Pointer
		Data unsafe.Pointer
	}
	return pointer{p: (*ifaceHeader)(unsafe.Pointer(&v)).Data}
}

// IsNil reports whether the pointer is nil.
func (p pointer) IsNil() bool {
	return p.p == nil
}

// Apply adds an offset to the pointer to derive a new pointer
// to a specified field. The pointer must be valid and pointing at a struct.
func (p pointer) Apply(f offset) pointer {
	if p.IsNil() {
		panic("invalid nil pointer")
	}
	return pointer{p: unsafe.Pointer(uintptr(p.p) + uintptr(f))}
}

// AsValueOf treats p as a pointer to an object of type t and returns the value.
// It is equivalent to reflect.ValueOf(p.AsIfaceOf(t))
func (p pointer) AsValueOf(t reflect.Type) reflect.Value {
	return reflect.NewAt(t, p.p)
}

// AsIfaceOf treats p as a pointer to an object of type t and returns the value.
// It is equivalent to p.AsValueOf(t).Interface()
func (p pointer) AsIfaceOf(t reflect.Type) interface{} {
	// TODO: Use tricky unsafe magic to directly create ifaceHeader.
	return p.AsValueOf(t).Interface()
}
