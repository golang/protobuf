// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// pointerCoderFuncs is a set of pointer encoding functions.
type pointerCoderFuncs struct {
	size    func(p pointer, tagsize int, opts marshalOptions) int
	marshal func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error)
}

// ifaceCoderFuncs is a set of interface{} encoding functions.
type ifaceCoderFuncs struct {
	size    func(ival interface{}, tagsize int, opts marshalOptions) int
	marshal func(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error)
}

// fieldCoder returns pointer functions for a field, used for operating on
// struct fields.
func fieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	switch {
	case fd.Cardinality() == pref.Repeated && !fd.IsPacked():
		// Repeated fields (not packed).
		if ft.Kind() != reflect.Slice {
			break
		}
		ft := ft.Elem()
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolSlice
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumSlice
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32Slice
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32Slice
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32Slice
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64Slice
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64Slice
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64Slice
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32Slice
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32Slice
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatSlice
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64Slice
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64Slice
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoubleSlice
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String && fd.Syntax() == pref.Proto3 {
				return coderStringSliceValidateUTF8
			}
			if ft.Kind() == reflect.String {
				return coderStringSlice
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesSlice
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringSlice
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesSlice
			}
		case pref.MessageKind:
			return makeMessageSliceFieldCoder(fd, ft)
		case pref.GroupKind:
			return makeGroupSliceFieldCoder(fd, ft)
		}
	case fd.Cardinality() == pref.Repeated && fd.IsPacked():
		// Packed repeated fields.
		//
		// Only repeated fields of primitive numeric types
		// (Varint, Fixed32, or Fixed64 wire type) can be packed.
		if ft.Kind() != reflect.Slice {
			break
		}
		ft := ft.Elem()
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolPackedSlice
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumPackedSlice
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32PackedSlice
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32PackedSlice
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32PackedSlice
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64PackedSlice
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64PackedSlice
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64PackedSlice
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32PackedSlice
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32PackedSlice
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatPackedSlice
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64PackedSlice
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64PackedSlice
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoublePackedSlice
			}
		}
	case fd.Kind() == pref.MessageKind:
		return makeMessageFieldCoder(fd, ft)
	case fd.Kind() == pref.GroupKind:
		return makeGroupFieldCoder(fd, ft)
	case fd.Syntax() == pref.Proto3 && fd.ContainingOneof() == nil:
		// Populated oneof fields always encode even if set to the zero value,
		// which normally are not encoded in proto3.
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolNoZero
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumNoZero
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32NoZero
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32NoZero
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32NoZero
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64NoZero
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64NoZero
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64NoZero
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32NoZero
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32NoZero
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatNoZero
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64NoZero
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64NoZero
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoubleNoZero
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String {
				return coderStringNoZeroValidateUTF8
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesNoZero
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringNoZero
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesNoZero
			}
		}
	case ft.Kind() == reflect.Ptr:
		ft := ft.Elem()
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolPtr
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumPtr
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32Ptr
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32Ptr
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32Ptr
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64Ptr
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64Ptr
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64Ptr
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32Ptr
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32Ptr
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatPtr
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64Ptr
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64Ptr
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoublePtr
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String {
				return coderStringPtr
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringPtr
			}
		}
	default:
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBool
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnum
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloat
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDouble
			}
		case pref.StringKind:
			if fd.Syntax() == pref.Proto3 && ft.Kind() == reflect.String {
				return coderStringValidateUTF8
			}
			if ft.Kind() == reflect.String {
				return coderString
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytes
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderString
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytes
			}
		}
	}
	panic(fmt.Errorf("invalid type: no encoder for %v %v %v/%v", fd.FullName(), fd.Cardinality(), fd.Kind(), ft))
}

// encoderFuncsForValue returns interface{} value functions for a field, used for
// extension values and map encoding.
func encoderFuncsForValue(fd pref.FieldDescriptor, ft reflect.Type) ifaceCoderFuncs {
	switch {
	case fd.Cardinality() == pref.Repeated && !fd.IsPacked():
		if ft.Kind() != reflect.Ptr || ft.Elem().Kind() != reflect.Slice {
			break
		}
		ft := ft.Elem().Elem()
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolSliceIface
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumSliceIface
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32SliceIface
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32SliceIface
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32SliceIface
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64SliceIface
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64SliceIface
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64SliceIface
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32SliceIface
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32SliceIface
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatSliceIface
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64SliceIface
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64SliceIface
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoubleSliceIface
			}
		case pref.StringKind:
			if ft.Kind() == reflect.String {
				return coderStringSliceIface
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesSliceIface
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringSliceIface
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesSliceIface
			}
		case pref.MessageKind:
			return coderMessageSliceIface
		case pref.GroupKind:
			return coderGroupSliceIface
		}
	case fd.Cardinality() == pref.Repeated && fd.IsPacked():
	default:
		switch fd.Kind() {
		case pref.BoolKind:
			if ft.Kind() == reflect.Bool {
				return coderBoolIface
			}
		case pref.EnumKind:
			if ft.Kind() == reflect.Int32 {
				return coderEnumIface
			}
		case pref.Int32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderInt32Iface
			}
		case pref.Sint32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSint32Iface
			}
		case pref.Uint32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderUint32Iface
			}
		case pref.Int64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderInt64Iface
			}
		case pref.Sint64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSint64Iface
			}
		case pref.Uint64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderUint64Iface
			}
		case pref.Sfixed32Kind:
			if ft.Kind() == reflect.Int32 {
				return coderSfixed32Iface
			}
		case pref.Fixed32Kind:
			if ft.Kind() == reflect.Uint32 {
				return coderFixed32Iface
			}
		case pref.FloatKind:
			if ft.Kind() == reflect.Float32 {
				return coderFloatIface
			}
		case pref.Sfixed64Kind:
			if ft.Kind() == reflect.Int64 {
				return coderSfixed64Iface
			}
		case pref.Fixed64Kind:
			if ft.Kind() == reflect.Uint64 {
				return coderFixed64Iface
			}
		case pref.DoubleKind:
			if ft.Kind() == reflect.Float64 {
				return coderDoubleIface
			}
		case pref.StringKind:
			if fd.Syntax() == pref.Proto3 && ft.Kind() == reflect.String {
				return coderStringIfaceValidateUTF8
			}
			if ft.Kind() == reflect.String {
				return coderStringIface
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesIface
			}
		case pref.BytesKind:
			if ft.Kind() == reflect.String {
				return coderStringIface
			}
			if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Uint8 {
				return coderBytesIface
			}
		case pref.MessageKind:
			return coderMessageIface
		case pref.GroupKind:
			return coderGroupIface
		}
	}
	panic(fmt.Errorf("invalid type: no encoder for %v %v %v/%v", fd.FullName(), fd.Cardinality(), fd.Kind(), ft))
}
