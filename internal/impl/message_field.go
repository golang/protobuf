// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/v2/internal/flags"
	"github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

type fieldInfo struct {
	// TODO: specialize marshal and unmarshal functions?

	has     func(pointer) bool
	get     func(pointer) pref.Value
	set     func(pointer, pref.Value)
	clear   func(pointer)
	mutable func(pointer) pref.Mutable
}

func fieldInfoForWeak(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	if !flags.Proto1Legacy {
		panic("weak fields not supported")
	}
	// TODO: support weak fields.
	panic(fmt.Sprintf("invalid field: %v", fd))
}

func fieldInfoForOneof(fd pref.FieldDescriptor, fs reflect.StructField, ot reflect.Type) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Interface {
		panic(fmt.Sprintf("invalid type: got %v, want interface kind", ft))
	}
	if ot.Kind() != reflect.Struct {
		panic(fmt.Sprintf("invalid type: got %v, want struct kind", ot))
	}
	if !reflect.PtrTo(ot).Implements(ft) {
		panic(fmt.Sprintf("invalid type: %v does not implement %v", ot, ft))
	}
	conv := value.NewLegacyConverter(ot.Field(0).Type, fd.Kind(), wrapLegacyMessage)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		// NOTE: The logic below intentionally assumes that oneof fields are
		// well-formatted. That is, the oneof interface never contains a
		// typed nil pointer to one of the wrapper structs.

		has: func(p pointer) bool {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return false
			}
			return true
		},
		get: func(p pointer) pref.Value {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				if fd.Kind() == pref.MessageKind || fd.Kind() == pref.GroupKind {
					// TODO: Should this return an invalid protoreflect.Value?
					rv = reflect.Zero(ot.Field(0).Type)
					return conv.PBValueOf(rv)
				}
				return fd.Default()
			}
			rv = rv.Elem().Elem().Field(0)
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				rv.Set(reflect.New(ot))
			}
			rv = rv.Elem().Elem().Field(0)
			rv.Set(conv.GoValueOf(v))
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return
			}
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			// Mutable is only valid for messages and panics for other kinds.
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				rv.Set(reflect.New(ot))
			}
			rv = rv.Elem().Elem().Field(0)
			if rv.IsNil() {
				pv := pref.ValueOf(conv.NewMessage())
				rv.Set(conv.GoValueOf(pv))
			}
			return rv.Interface().(pref.Message)
		},
	}
}

func fieldInfoForMap(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Map {
		panic(fmt.Sprintf("invalid type: got %v, want map kind", ft))
	}
	keyConv := value.NewLegacyConverter(ft.Key(), fd.MessageType().Fields().ByNumber(1).Kind(), wrapLegacyMessage)
	valConv := value.NewLegacyConverter(ft.Elem(), fd.MessageType().Fields().ByNumber(2).Kind(), wrapLegacyMessage)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return rv.Len() > 0
		},
		get: func(p pointer) pref.Value {
			v := p.apply(fieldOffset).asType(fs.Type).Interface()
			return pref.ValueOf(value.MapOf(v, keyConv, valConv))
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.ValueOf(v.Map().(value.Unwrapper).Unwrap()))
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			v := p.apply(fieldOffset).asType(fs.Type).Interface()
			return value.MapOf(v, keyConv, valConv)
		},
	}
}

func fieldInfoForVector(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid type: got %v, want slice kind", ft))
	}
	conv := value.NewLegacyConverter(ft.Elem(), fd.Kind(), wrapLegacyMessage)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return rv.Len() > 0
		},
		get: func(p pointer) pref.Value {
			v := p.apply(fieldOffset).asType(fs.Type).Interface()
			return pref.ValueOf(value.VectorOf(v, conv))
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.ValueOf(v.Vector().(value.Unwrapper).Unwrap()))
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			v := p.apply(fieldOffset).asType(fs.Type).Interface()
			return value.VectorOf(v, conv)
		},
	}
}

var emptyBytes = reflect.ValueOf([]byte{})

func fieldInfoForScalar(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	nullable := fd.Syntax() == pref.Proto2
	if nullable {
		if ft.Kind() != reflect.Ptr && ft.Kind() != reflect.Slice {
			panic(fmt.Sprintf("invalid type: got %v, want pointer", ft))
		}
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
	}
	conv := value.NewLegacyConverter(ft, fd.Kind(), wrapLegacyMessage)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if nullable {
				return !rv.IsNil()
			}
			switch rv.Kind() {
			case reflect.Bool:
				return rv.Bool()
			case reflect.Int32, reflect.Int64:
				return rv.Int() > 0
			case reflect.Uint32, reflect.Uint64:
				return rv.Uint() > 0
			case reflect.Float32, reflect.Float64:
				return rv.Float() > 0
			case reflect.String, reflect.Slice:
				return rv.Len() > 0
			default:
				panic(fmt.Sprintf("invalid type: %v", rv.Type())) // should never happen
			}
		},
		get: func(p pointer) pref.Value {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if nullable {
				if rv.IsNil() {
					pv := fd.Default()
					if fd.Kind() == pref.BytesKind && len(pv.Bytes()) > 0 {
						return pref.ValueOf(append([]byte(nil), pv.Bytes()...)) // copy default bytes for safety
					}
					return pv
				}
				if rv.Kind() == reflect.Ptr {
					rv = rv.Elem()
				}
			}
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if nullable && rv.Kind() == reflect.Ptr {
				if rv.IsNil() {
					rv.Set(reflect.New(ft))
				}
				rv = rv.Elem()
			}
			rv.Set(conv.GoValueOf(v))
			if nullable && rv.Kind() == reflect.Slice && rv.IsNil() {
				rv.Set(emptyBytes)
			}
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			panic("invalid mutable call")
		},
	}
}

func fieldInfoForMessage(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	conv := value.NewLegacyConverter(ft, fd.Kind(), wrapLegacyMessage)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return !rv.IsNil()
		},
		get: func(p pointer) pref.Value {
			// TODO: If rv.IsNil(), should this return a typed-nil pointer or
			// an invalid protoreflect.Value?
			//
			// Returning a typed nil pointer assumes that such values
			// are valid for all possible custom message types,
			// which may not be case for dynamic messages.
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			// TODO: Similarly, is it valid to set this to a typed nil pointer?
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(conv.GoValueOf(v))
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			// Mutable is only valid for messages and panics for other kinds.
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() {
				pv := pref.ValueOf(conv.NewMessage())
				rv.Set(conv.GoValueOf(pv))
			}
			return conv.PBValueOf(rv).Message()
		},
	}
}
