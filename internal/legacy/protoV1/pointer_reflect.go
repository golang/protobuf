// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build purego

package protoV1

import (
	"reflect"
	"sync"

	"github.com/golang/protobuf/protoapi"
)

// A field identifies a field in a struct, accessible from a pointer.
// In this implementation, a field is identified by the sequence of field indices
// passed to reflect's FieldByIndex.
type field []int

// toField returns a field equivalent to the given reflect field.
func toField(f *reflect.StructField) field {
	return f.Index
}

// invalidField is an invalid field identifier.
var invalidField = field(nil)

// zeroField is a noop when calling pointer.offset.
var zeroField = field([]int{})

// IsValid reports whether the field identifier is valid.
func (f field) IsValid() bool { return f != nil }

// The pointer type is for the table-driven decoder.
// The implementation here uses a reflect.Value of pointer type to
// create a generic pointer. In pointer_unsafe.go we use unsafe
// instead of reflect to implement the same (but faster) interface.
type pointer struct {
	v reflect.Value
}

// toPointer converts an interface of pointer type to a pointer
// that points to the same target.
func toPointer(i *Message) pointer {
	return pointer{v: reflect.ValueOf(*i)}
}

// valToPointer converts v to a pointer.  v must be of pointer type.
func valToPointer(v reflect.Value) pointer {
	return pointer{v: v}
}

// offset converts from a pointer to a structure to a pointer to
// one of its fields.
func (p pointer) offset(f field) pointer {
	return pointer{v: p.v.Elem().FieldByIndex(f).Addr()}
}

func (p pointer) isNil() bool {
	return p.v.IsNil()
}

// grow updates the slice s in place to make it one element longer.
// s must be addressable.
// Returns the (addressable) new element.
func grow(s reflect.Value) reflect.Value {
	n, m := s.Len(), s.Cap()
	if n < m {
		s.SetLen(n + 1)
	} else {
		s.Set(reflect.Append(s, reflect.Zero(s.Type().Elem())))
	}
	return s.Index(n)
}

func (p pointer) toInt64() *int64 {
	return p.v.Interface().(*int64)
}
func (p pointer) toInt64Ptr() **int64 {
	return p.v.Interface().(**int64)
}
func (p pointer) toInt64Slice() *[]int64 {
	return p.v.Interface().(*[]int64)
}

var int32ptr = reflect.TypeOf((*int32)(nil))

func (p pointer) toInt32() *int32 {
	return p.v.Convert(int32ptr).Interface().(*int32)
}

// The toInt32Ptr/Slice methods don't work because of enums.
// Instead, we must use set/get methods for the int32ptr/slice case.
/*
	func (p pointer) toInt32Ptr() **int32 {
		return p.v.Interface().(**int32)
}
	func (p pointer) toInt32Slice() *[]int32 {
		return p.v.Interface().(*[]int32)
}
*/
func (p pointer) setInt32Ptr(v int32) {
	// Allocate value in a *int32. Possibly convert that to a *enum.
	// Then assign it to a **int32 or **enum.
	// Note: we can convert *int32 to *enum, but we can't convert
	// **int32 to **enum!
	p.v.Elem().Set(reflect.ValueOf(&v).Convert(p.v.Type().Elem()))
}

func (p pointer) appendInt32Slice(v int32) {
	grow(p.v.Elem()).SetInt(int64(v))
}

func (p pointer) toUint64() *uint64 {
	return p.v.Interface().(*uint64)
}
func (p pointer) toUint64Ptr() **uint64 {
	return p.v.Interface().(**uint64)
}
func (p pointer) toUint64Slice() *[]uint64 {
	return p.v.Interface().(*[]uint64)
}
func (p pointer) toUint32() *uint32 {
	return p.v.Interface().(*uint32)
}
func (p pointer) toUint32Ptr() **uint32 {
	return p.v.Interface().(**uint32)
}
func (p pointer) toUint32Slice() *[]uint32 {
	return p.v.Interface().(*[]uint32)
}
func (p pointer) toBool() *bool {
	return p.v.Interface().(*bool)
}
func (p pointer) toBoolPtr() **bool {
	return p.v.Interface().(**bool)
}
func (p pointer) toBoolSlice() *[]bool {
	return p.v.Interface().(*[]bool)
}
func (p pointer) toFloat64() *float64 {
	return p.v.Interface().(*float64)
}
func (p pointer) toFloat64Ptr() **float64 {
	return p.v.Interface().(**float64)
}
func (p pointer) toFloat64Slice() *[]float64 {
	return p.v.Interface().(*[]float64)
}
func (p pointer) toFloat32() *float32 {
	return p.v.Interface().(*float32)
}
func (p pointer) toFloat32Ptr() **float32 {
	return p.v.Interface().(**float32)
}
func (p pointer) toFloat32Slice() *[]float32 {
	return p.v.Interface().(*[]float32)
}
func (p pointer) toString() *string {
	return p.v.Interface().(*string)
}
func (p pointer) toStringPtr() **string {
	return p.v.Interface().(**string)
}
func (p pointer) toStringSlice() *[]string {
	return p.v.Interface().(*[]string)
}
func (p pointer) toBytes() *[]byte {
	return p.v.Interface().(*[]byte)
}
func (p pointer) toBytesSlice() *[][]byte {
	return p.v.Interface().(*[][]byte)
}
func (p pointer) toExtensions() *protoapi.XXX_InternalExtensions {
	return p.v.Interface().(*protoapi.XXX_InternalExtensions)
}
func (p pointer) toOldExtensions() *map[int32]protoapi.ExtensionField {
	return p.v.Interface().(*map[int32]protoapi.ExtensionField)
}
func (p pointer) getPointer() pointer {
	return pointer{v: p.v.Elem()}
}
func (p pointer) setPointer(q pointer) {
	p.v.Elem().Set(q.v)
}
func (p pointer) appendPointer(q pointer) {
	grow(p.v.Elem()).Set(q.v)
}

func (p pointer) asPointerTo(t reflect.Type) reflect.Value {
	// TODO: check that p.v.Type().Elem() == t?
	return p.v
}

func atomicLoadUnmarshalInfo(p **unmarshalInfo) *unmarshalInfo {
	atomicLock.Lock()
	defer atomicLock.Unlock()
	return *p
}
func atomicStoreUnmarshalInfo(p **unmarshalInfo, v *unmarshalInfo) {
	atomicLock.Lock()
	defer atomicLock.Unlock()
	*p = v
}

var atomicLock sync.Mutex
