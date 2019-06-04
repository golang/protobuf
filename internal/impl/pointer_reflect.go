// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build purego appengine

package impl

import (
	"fmt"
	"reflect"
)

// offset represents the offset to a struct field, accessible from a pointer.
// The offset is the field index into a struct.
type offset []int

// offsetOf returns a field offset for the struct field.
func offsetOf(f reflect.StructField) offset {
	if len(f.Index) != 1 {
		panic("embedded structs are not supported")
	}
	return f.Index
}

// IsValid reports whether the offset is valid.
func (f offset) IsValid() bool { return f != nil }

// invalidOffset is an invalid field offset.
var invalidOffset = offset(nil)

// zeroOffset is a noop when calling pointer.Apply.
var zeroOffset = offset([]int{0})

// pointer is an abstract representation of a pointer to a struct or field.
type pointer struct{ v reflect.Value }

// pointerOfValue returns v as a pointer.
func pointerOfValue(v reflect.Value) pointer {
	return pointer{v: v}
}

// pointerOfIface returns the pointer portion of an interface.
func pointerOfIface(v interface{}) pointer {
	return pointer{v: reflect.ValueOf(v)}
}

// IsNil reports whether the pointer is nil.
func (p pointer) IsNil() bool {
	return p.v.IsNil()
}

// Apply adds an offset to the pointer to derive a new pointer
// to a specified field. The current pointer must be pointing at a struct.
func (p pointer) Apply(f offset) pointer {
	// TODO: Handle unexported fields in an API that hides XXX fields?
	return pointer{v: p.v.Elem().FieldByIndex(f).Addr()}
}

// AsValueOf treats p as a pointer to an object of type t and returns the value.
// It is equivalent to reflect.ValueOf(p.AsIfaceOf(t))
func (p pointer) AsValueOf(t reflect.Type) reflect.Value {
	if got := p.v.Type().Elem(); got != t {
		panic(fmt.Sprintf("invalid type: got %v, want %v", got, t))
	}
	return p.v
}

// AsIfaceOf treats p as a pointer to an object of type t and returns the value.
// It is equivalent to p.AsValueOf(t).Interface()
func (p pointer) AsIfaceOf(t reflect.Type) interface{} {
	return p.AsValueOf(t).Interface()
}

func (p pointer) Bool() *bool              { return p.v.Interface().(*bool) }
func (p pointer) BoolPtr() **bool          { return p.v.Interface().(**bool) }
func (p pointer) BoolSlice() *[]bool       { return p.v.Interface().(*[]bool) }
func (p pointer) Int32() *int32            { return p.v.Interface().(*int32) }
func (p pointer) Int32Ptr() **int32        { return p.v.Interface().(**int32) }
func (p pointer) Int32Slice() *[]int32     { return p.v.Interface().(*[]int32) }
func (p pointer) Int64() *int64            { return p.v.Interface().(*int64) }
func (p pointer) Int64Ptr() **int64        { return p.v.Interface().(**int64) }
func (p pointer) Int64Slice() *[]int64     { return p.v.Interface().(*[]int64) }
func (p pointer) Uint32() *uint32          { return p.v.Interface().(*uint32) }
func (p pointer) Uint32Ptr() **uint32      { return p.v.Interface().(**uint32) }
func (p pointer) Uint32Slice() *[]uint32   { return p.v.Interface().(*[]uint32) }
func (p pointer) Uint64() *uint64          { return p.v.Interface().(*uint64) }
func (p pointer) Uint64Ptr() **uint64      { return p.v.Interface().(**uint64) }
func (p pointer) Uint64Slice() *[]uint64   { return p.v.Interface().(*[]uint64) }
func (p pointer) Float32() *float32        { return p.v.Interface().(*float32) }
func (p pointer) Float32Ptr() **float32    { return p.v.Interface().(**float32) }
func (p pointer) Float32Slice() *[]float32 { return p.v.Interface().(*[]float32) }
func (p pointer) Float64() *float64        { return p.v.Interface().(*float64) }
func (p pointer) Float64Ptr() **float64    { return p.v.Interface().(**float64) }
func (p pointer) Float64Slice() *[]float64 { return p.v.Interface().(*[]float64) }
func (p pointer) String() *string          { return p.v.Interface().(*string) }
func (p pointer) StringPtr() **string      { return p.v.Interface().(**string) }
func (p pointer) StringSlice() *[]string   { return p.v.Interface().(*[]string) }
func (p pointer) Bytes() *[]byte           { return p.v.Interface().(*[]byte) }
func (p pointer) BytesSlice() *[][]byte    { return p.v.Interface().(*[][]byte) }
func (p pointer) Extensions() *map[int32]ExtensionField {
	return p.v.Interface().(*map[int32]ExtensionField)
}

func (p pointer) Elem() pointer {
	return pointer{v: p.v.Elem()}
}

// PointerSlice copies []*T from p as a new []pointer.
// This behavior differs from the implementation in pointer_unsafe.go.
func (p pointer) PointerSlice() []pointer {
	// TODO: reconsider this
	if p.v.IsNil() {
		return nil
	}
	n := p.v.Elem().Len()
	s := make([]pointer, n)
	for i := 0; i < n; i++ {
		s[i] = pointer{v: p.v.Elem().Index(i)}
	}
	return s
}
