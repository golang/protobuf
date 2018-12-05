// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !purego

package protoV1

import (
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/golang/protobuf/protoapi"
)

// A field identifies a field in a struct, accessible from a pointer.
// In this implementation, a field is identified by its byte offset from the start of the struct.
type field uintptr

// toField returns a field equivalent to the given reflect field.
func toField(f *reflect.StructField) field {
	return field(f.Offset)
}

// invalidField is an invalid field identifier.
const invalidField = ^field(0)

// zeroField is a noop when calling pointer.offset.
const zeroField = field(0)

// IsValid reports whether the field identifier is valid.
func (f field) IsValid() bool {
	return f != invalidField
}

// The pointer type below is for the new table-driven encoder/decoder.
// The implementation here uses unsafe.Pointer to create a generic pointer.
// In pointer_reflect.go we use reflect instead of unsafe to implement
// the same (but slower) interface.
type pointer struct {
	p unsafe.Pointer
}

// toPointer converts an interface of pointer type to a pointer
// that points to the same target.
func toPointer(i *Message) pointer {
	// Super-tricky - read pointer out of data word of interface value.
	// Saves ~25ns over the equivalent:
	// return valToPointer(reflect.ValueOf(*i))
	return pointer{p: (*[2]unsafe.Pointer)(unsafe.Pointer(i))[1]}
}

// valToPointer converts v to a pointer. v must be of pointer type.
func valToPointer(v reflect.Value) pointer {
	return pointer{p: unsafe.Pointer(v.Pointer())}
}

// offset converts from a pointer to a structure to a pointer to
// one of its fields.
func (p pointer) offset(f field) pointer {
	// For safety, we should panic if !f.IsValid, however calling panic causes
	// this to no longer be inlineable, which is a serious performance cost.
	/*
		if !f.IsValid() {
			panic("invalid field")
		}
	*/
	return pointer{p: unsafe.Pointer(uintptr(p.p) + uintptr(f))}
}

func (p pointer) isNil() bool {
	return p.p == nil
}

func (p pointer) toInt64() *int64 {
	return (*int64)(p.p)
}
func (p pointer) toInt64Ptr() **int64 {
	return (**int64)(p.p)
}
func (p pointer) toInt64Slice() *[]int64 {
	return (*[]int64)(p.p)
}
func (p pointer) toInt32() *int32 {
	return (*int32)(p.p)
}

// See pointer_reflect.go for why toInt32Ptr/Slice doesn't exist.
/*
	func (p pointer) toInt32Ptr() **int32 {
		return (**int32)(p.p)
	}
	func (p pointer) toInt32Slice() *[]int32 {
		return (*[]int32)(p.p)
	}
*/
func (p pointer) setInt32Ptr(v int32) {
	*(**int32)(p.p) = &v
}

// TODO: Can we get rid of appendInt32Slice and use setInt32Slice instead?
func (p pointer) appendInt32Slice(v int32) {
	s := (*[]int32)(p.p)
	*s = append(*s, v)
}

func (p pointer) toUint64() *uint64 {
	return (*uint64)(p.p)
}
func (p pointer) toUint64Ptr() **uint64 {
	return (**uint64)(p.p)
}
func (p pointer) toUint64Slice() *[]uint64 {
	return (*[]uint64)(p.p)
}
func (p pointer) toUint32() *uint32 {
	return (*uint32)(p.p)
}
func (p pointer) toUint32Ptr() **uint32 {
	return (**uint32)(p.p)
}
func (p pointer) toUint32Slice() *[]uint32 {
	return (*[]uint32)(p.p)
}
func (p pointer) toBool() *bool {
	return (*bool)(p.p)
}
func (p pointer) toBoolPtr() **bool {
	return (**bool)(p.p)
}
func (p pointer) toBoolSlice() *[]bool {
	return (*[]bool)(p.p)
}
func (p pointer) toFloat64() *float64 {
	return (*float64)(p.p)
}
func (p pointer) toFloat64Ptr() **float64 {
	return (**float64)(p.p)
}
func (p pointer) toFloat64Slice() *[]float64 {
	return (*[]float64)(p.p)
}
func (p pointer) toFloat32() *float32 {
	return (*float32)(p.p)
}
func (p pointer) toFloat32Ptr() **float32 {
	return (**float32)(p.p)
}
func (p pointer) toFloat32Slice() *[]float32 {
	return (*[]float32)(p.p)
}
func (p pointer) toString() *string {
	return (*string)(p.p)
}
func (p pointer) toStringPtr() **string {
	return (**string)(p.p)
}
func (p pointer) toStringSlice() *[]string {
	return (*[]string)(p.p)
}
func (p pointer) toBytes() *[]byte {
	return (*[]byte)(p.p)
}
func (p pointer) toBytesSlice() *[][]byte {
	return (*[][]byte)(p.p)
}
func (p pointer) toExtensions() *protoapi.XXX_InternalExtensions {
	return (*protoapi.XXX_InternalExtensions)(p.p)
}
func (p pointer) toOldExtensions() *map[int32]protoapi.ExtensionField {
	return (*map[int32]protoapi.ExtensionField)(p.p)
}

// getPointer loads the pointer at p and returns it.
func (p pointer) getPointer() pointer {
	return pointer{p: *(*unsafe.Pointer)(p.p)}
}

// setPointer stores the pointer q at p.
func (p pointer) setPointer(q pointer) {
	*(*unsafe.Pointer)(p.p) = q.p
}

// append q to the slice pointed to by p.
func (p pointer) appendPointer(q pointer) {
	s := (*[]unsafe.Pointer)(p.p)
	*s = append(*s, q.p)
}

// asPointerTo returns a reflect.Value that is a pointer to an
// object of type t stored at p.
func (p pointer) asPointerTo(t reflect.Type) reflect.Value {
	return reflect.NewAt(t, p.p)
}

func atomicLoadUnmarshalInfo(p **unmarshalInfo) *unmarshalInfo {
	return (*unmarshalInfo)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(p))))
}
func atomicStoreUnmarshalInfo(p **unmarshalInfo, v *unmarshalInfo) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(p)), unsafe.Pointer(v))
}
