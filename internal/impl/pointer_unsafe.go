// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !purego

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

// apply adds an offset to the pointer to derive a new pointer
// to a specified field. The current pointer must be pointing at a struct.
func (p pointer) apply(f offset) pointer {
	return pointer{p: unsafe.Pointer(uintptr(p.p) + uintptr(f))}
}

// asType treats p as a pointer to an object of type t and returns the value.
func (p pointer) asType(t reflect.Type) reflect.Value {
	return reflect.NewAt(t, p.p)
}
