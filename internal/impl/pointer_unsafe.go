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

// IsValid reports whether the offset is valid.
func (f offset) IsValid() bool { return f != invalidOffset }

// invalidOffset is an invalid field offset.
var invalidOffset = ^offset(0)

// zeroOffset is a noop when calling pointer.Apply.
var zeroOffset = offset(0)

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

func (p pointer) Bool() *bool                     { return (*bool)(p.p) }
func (p pointer) BoolPtr() **bool                 { return (**bool)(p.p) }
func (p pointer) BoolSlice() *[]bool              { return (*[]bool)(p.p) }
func (p pointer) Int32() *int32                   { return (*int32)(p.p) }
func (p pointer) Int32Ptr() **int32               { return (**int32)(p.p) }
func (p pointer) Int32Slice() *[]int32            { return (*[]int32)(p.p) }
func (p pointer) Int64() *int64                   { return (*int64)(p.p) }
func (p pointer) Int64Ptr() **int64               { return (**int64)(p.p) }
func (p pointer) Int64Slice() *[]int64            { return (*[]int64)(p.p) }
func (p pointer) Uint32() *uint32                 { return (*uint32)(p.p) }
func (p pointer) Uint32Ptr() **uint32             { return (**uint32)(p.p) }
func (p pointer) Uint32Slice() *[]uint32          { return (*[]uint32)(p.p) }
func (p pointer) Uint64() *uint64                 { return (*uint64)(p.p) }
func (p pointer) Uint64Ptr() **uint64             { return (**uint64)(p.p) }
func (p pointer) Uint64Slice() *[]uint64          { return (*[]uint64)(p.p) }
func (p pointer) Float32() *float32               { return (*float32)(p.p) }
func (p pointer) Float32Ptr() **float32           { return (**float32)(p.p) }
func (p pointer) Float32Slice() *[]float32        { return (*[]float32)(p.p) }
func (p pointer) Float64() *float64               { return (*float64)(p.p) }
func (p pointer) Float64Ptr() **float64           { return (**float64)(p.p) }
func (p pointer) Float64Slice() *[]float64        { return (*[]float64)(p.p) }
func (p pointer) String() *string                 { return (*string)(p.p) }
func (p pointer) StringPtr() **string             { return (**string)(p.p) }
func (p pointer) StringSlice() *[]string          { return (*[]string)(p.p) }
func (p pointer) Bytes() *[]byte                  { return (*[]byte)(p.p) }
func (p pointer) BytesSlice() *[][]byte           { return (*[][]byte)(p.p) }
func (p pointer) Extensions() *legacyExtensionMap { return (*legacyExtensionMap)(p.p) }

func (p pointer) Elem() pointer {
	return pointer{p: *(*unsafe.Pointer)(p.p)}
}

// PointerSlice loads []*T from p as a []pointer.
// The value returned is aliased with the original slice.
// This behavior differs from the implementation in pointer_reflect.go.
func (p pointer) PointerSlice() []pointer {
	// Super-tricky - p should point to a []*T where T is a
	// message type. We load it as []pointer.
	return *(*[]pointer)(p.p)
}
