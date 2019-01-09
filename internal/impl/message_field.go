// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/v2/internal/flags"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

type fieldInfo struct {
	// TODO: specialize marshal and unmarshal functions?

	has        func(pointer) bool
	get        func(pointer) pref.Value
	set        func(pointer, pref.Value)
	clear      func(pointer)
	newMessage func() pref.Message
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
	conv := pvalue.NewLegacyConverter(ot.Field(0).Type, fd.Kind(), legacyWrapper)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		// NOTE: The logic below intentionally assumes that oneof fields are
		// well-formatted. That is, the oneof interface never contains a
		// typed nil pointer to one of the wrapper structs.

		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return false
			}
			return true
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return defaultValueOf(fd)
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return defaultValueOf(fd)
			}
			rv = rv.Elem().Elem().Field(0)
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				rv.Set(reflect.New(ot))
			}
			rv = rv.Elem().Elem().Field(0)
			rv.Set(conv.GoValueOf(v))
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return
			}
			rv.Set(reflect.Zero(rv.Type()))
		},
		newMessage: func() pref.Message {
			// This is only valid for messages and panics for other kinds.
			return conv.MessageType.New()
		},
	}
}

func fieldInfoForMap(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Map {
		panic(fmt.Sprintf("invalid type: got %v, want map kind", ft))
	}
	keyConv := pvalue.NewLegacyConverter(ft.Key(), fd.MessageType().Fields().ByNumber(1).Kind(), legacyWrapper)
	valConv := pvalue.NewLegacyConverter(ft.Elem(), fd.MessageType().Fields().ByNumber(2).Kind(), legacyWrapper)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return rv.Len() > 0
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				v := reflect.Zero(reflect.PtrTo(fs.Type)).Interface()
				return pref.ValueOf(pvalue.MapOf(v, keyConv, valConv))
			}
			v := p.Apply(fieldOffset).AsIfaceOf(fs.Type)
			return pref.ValueOf(pvalue.MapOf(v, keyConv, valConv))
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.ValueOf(v.Map().(pvalue.Unwrapper).ProtoUnwrap()).Elem())
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
	}
}

func fieldInfoForList(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid type: got %v, want slice kind", ft))
	}
	conv := pvalue.NewLegacyConverter(ft.Elem(), fd.Kind(), legacyWrapper)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return rv.Len() > 0
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				v := reflect.Zero(reflect.PtrTo(fs.Type)).Interface()
				return pref.ValueOf(pvalue.ListOf(v, conv))
			}
			v := p.Apply(fieldOffset).AsIfaceOf(fs.Type)
			return pref.ValueOf(pvalue.ListOf(v, conv))
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.ValueOf(v.List().(pvalue.Unwrapper).ProtoUnwrap()).Elem())
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
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
	conv := pvalue.NewLegacyConverter(ft, fd.Kind(), legacyWrapper)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if nullable {
				return !rv.IsNil()
			}
			switch rv.Kind() {
			case reflect.Bool:
				return rv.Bool()
			case reflect.Int32, reflect.Int64:
				return rv.Int() != 0
			case reflect.Uint32, reflect.Uint64:
				return rv.Uint() != 0
			case reflect.Float32, reflect.Float64:
				return rv.Float() != 0
			case reflect.String, reflect.Slice:
				return rv.Len() > 0
			default:
				panic(fmt.Sprintf("invalid type: %v", rv.Type())) // should never happen
			}
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return defaultValueOf(fd)
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if nullable {
				if rv.IsNil() {
					return defaultValueOf(fd)
				}
				if rv.Kind() == reflect.Ptr {
					rv = rv.Elem()
				}
			}
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
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
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
	}
}

func fieldInfoForMessage(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	conv := pvalue.NewLegacyConverter(ft, fd.Kind(), legacyWrapper)
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return !rv.IsNil()
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return pref.Value{}
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				return pref.Value{}
			}
			return conv.PBValueOf(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(conv.GoValueOf(v))
			if rv.IsNil() {
				panic("invalid nil pointer")
			}
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		newMessage: func() pref.Message {
			return conv.MessageType.New()
		},
	}
}

// defaultValueOf returns the default value for the field.
func defaultValueOf(fd pref.FieldDescriptor) pref.Value {
	if fd == nil {
		return pref.Value{}
	}
	pv := fd.Default() // invalid Value for messages and repeated fields
	if fd.Kind() == pref.BytesKind && pv.IsValid() && len(pv.Bytes()) > 0 {
		return pref.ValueOf(append([]byte(nil), pv.Bytes()...)) // copy default bytes for safety
	}
	return pv
}
