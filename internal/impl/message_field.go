// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/v2/internal/flags"
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
	conv := matchGoTypePBKind(ot.Field(0).Type, fd.Kind())
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
					return conv.toPB(rv)
				}
				return fd.Default()
			}
			rv = rv.Elem().Elem().Field(0)
			return conv.toPB(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				rv.Set(reflect.New(ot))
			}
			rv = rv.Elem().Elem().Field(0)
			rv.Set(conv.toGo(v))
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
				pv := pref.ValueOf(conv.newMessage())
				rv.Set(conv.toGo(pv))
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
	keyConv := matchGoTypePBKind(ft.Key(), fd.MessageType().Fields().ByNumber(1).Kind())
	valConv := matchGoTypePBKind(ft.Elem(), fd.MessageType().Fields().ByNumber(2).Kind())
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return rv.Len() > 0
		},
		get: func(p pointer) pref.Value {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return pref.ValueOf(mapReflect{rv, keyConv, valConv})
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(v.Map().(mapReflect).v)
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return mapReflect{rv, keyConv, valConv}
		},
	}
}

type mapReflect struct {
	v       reflect.Value // addressable map[K]V
	keyConv converter
	valConv converter
}

func (ms mapReflect) Len() int {
	return ms.v.Len()
}
func (ms mapReflect) Has(k pref.MapKey) bool {
	rk := ms.keyConv.toGo(k.Value())
	rv := ms.v.MapIndex(rk)
	return rv.IsValid()
}
func (ms mapReflect) Get(k pref.MapKey) pref.Value {
	rk := ms.keyConv.toGo(k.Value())
	rv := ms.v.MapIndex(rk)
	if !rv.IsValid() {
		return pref.Value{}
	}
	return ms.valConv.toPB(rv)
}
func (ms mapReflect) Set(k pref.MapKey, v pref.Value) {
	if ms.v.IsNil() {
		ms.v.Set(reflect.MakeMap(ms.v.Type()))
	}
	rk := ms.keyConv.toGo(k.Value())
	rv := ms.valConv.toGo(v)
	ms.v.SetMapIndex(rk, rv)
}
func (ms mapReflect) Clear(k pref.MapKey) {
	rk := ms.keyConv.toGo(k.Value())
	ms.v.SetMapIndex(rk, reflect.Value{})
}
func (ms mapReflect) Mutable(k pref.MapKey) pref.Mutable {
	// Mutable is only valid for messages and panics for other kinds.
	if ms.v.IsNil() {
		ms.v.Set(reflect.MakeMap(ms.v.Type()))
	}
	rk := ms.keyConv.toGo(k.Value())
	rv := ms.v.MapIndex(rk)
	if !rv.IsValid() || rv.IsNil() {
		pv := pref.ValueOf(ms.valConv.newMessage())
		rv = ms.valConv.toGo(pv)
		ms.v.SetMapIndex(rk, rv)
	}
	return rv.Interface().(pref.Message)
}
func (ms mapReflect) Range(f func(pref.MapKey, pref.Value) bool) {
	for _, k := range ms.v.MapKeys() {
		if v := ms.v.MapIndex(k); v.IsValid() {
			pk := ms.keyConv.toPB(k).MapKey()
			pv := ms.valConv.toPB(v)
			if !f(pk, pv) {
				return
			}
		}
	}
}
func (ms mapReflect) Unwrap() interface{} { // TODO: unexport?
	return ms.v.Interface()
}
func (ms mapReflect) ProtoMutable() {}

var _ pref.Map = mapReflect{}

func fieldInfoForVector(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid type: got %v, want slice kind", ft))
	}
	conv := matchGoTypePBKind(ft.Elem(), fd.Kind())
	fieldOffset := offsetOf(fs)
	// TODO: Implement unsafe fast path?
	return fieldInfo{
		has: func(p pointer) bool {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return rv.Len() > 0
		},
		get: func(p pointer) pref.Value {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return pref.ValueOf(vectorReflect{rv, conv})
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(v.Vector().(vectorReflect).v)
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			return vectorReflect{rv, conv}
		},
	}
}

type vectorReflect struct {
	v    reflect.Value // addressable []T
	conv converter
}

func (vs vectorReflect) Len() int {
	return vs.v.Len()
}
func (vs vectorReflect) Get(i int) pref.Value {
	return vs.conv.toPB(vs.v.Index(i))
}
func (vs vectorReflect) Set(i int, v pref.Value) {
	vs.v.Index(i).Set(vs.conv.toGo(v))
}
func (vs vectorReflect) Append(v pref.Value) {
	vs.v.Set(reflect.Append(vs.v, vs.conv.toGo(v)))
}
func (vs vectorReflect) Mutable(i int) pref.Mutable {
	// Mutable is only valid for messages and panics for other kinds.
	rv := vs.v.Index(i)
	if rv.IsNil() {
		pv := pref.ValueOf(vs.conv.newMessage())
		rv.Set(vs.conv.toGo(pv))
	}
	return rv.Interface().(pref.Message)
}
func (vs vectorReflect) MutableAppend() pref.Mutable {
	// MutableAppend is only valid for messages and panics for other kinds.
	pv := pref.ValueOf(vs.conv.newMessage())
	vs.v.Set(reflect.Append(vs.v, vs.conv.toGo(pv)))
	return vs.v.Index(vs.Len() - 1).Interface().(pref.Message)
}
func (vs vectorReflect) Truncate(i int) {
	vs.v.Set(vs.v.Slice(0, i))
}
func (vs vectorReflect) Unwrap() interface{} { // TODO: unexport?
	return vs.v.Interface()
}
func (vs vectorReflect) ProtoMutable() {}

var _ pref.Vector = vectorReflect{}

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
	conv := matchGoTypePBKind(ft, fd.Kind())
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
			return conv.toPB(rv)
		},
		set: func(p pointer, v pref.Value) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if nullable && rv.Kind() == reflect.Ptr {
				if rv.IsNil() {
					rv.Set(reflect.New(ft))
				}
				rv = rv.Elem()
			}
			rv.Set(conv.toGo(v))
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
	conv := matchGoTypePBKind(ft, fd.Kind())
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
			return conv.toPB(rv)
		},
		set: func(p pointer, v pref.Value) {
			// TODO: Similarly, is it valid to set this to a typed nil pointer?
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(conv.toGo(v))
		},
		clear: func(p pointer) {
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		mutable: func(p pointer) pref.Mutable {
			// Mutable is only valid for messages and panics for other kinds.
			rv := p.apply(fieldOffset).asType(fs.Type).Elem()
			if rv.IsNil() {
				pv := pref.ValueOf(conv.newMessage())
				rv.Set(conv.toGo(pv))
			}
			return conv.toPB(rv).Message()
		},
	}
}

// messageV1 is the protoV1.Message interface.
type messageV1 interface {
	Reset()
	String() string
	ProtoMessage()
}

var (
	boolType    = reflect.TypeOf(bool(false))
	int32Type   = reflect.TypeOf(int32(0))
	int64Type   = reflect.TypeOf(int64(0))
	uint32Type  = reflect.TypeOf(uint32(0))
	uint64Type  = reflect.TypeOf(uint64(0))
	float32Type = reflect.TypeOf(float32(0))
	float64Type = reflect.TypeOf(float64(0))
	stringType  = reflect.TypeOf(string(""))
	bytesType   = reflect.TypeOf([]byte(nil))

	enumIfaceV2    = reflect.TypeOf((*pref.ProtoEnum)(nil)).Elem()
	messageIfaceV1 = reflect.TypeOf((*messageV1)(nil)).Elem()
	messageIfaceV2 = reflect.TypeOf((*pref.ProtoMessage)(nil)).Elem()

	byteType = reflect.TypeOf(byte(0))
)

// matchGoTypePBKind matches a Go type with the protobuf kind.
//
// This matcher deliberately supports a wider range of Go types than what
// protoc-gen-go historically generated to be able to automatically wrap some
// v1 messages generated by other forks of protoc-gen-go.
func matchGoTypePBKind(t reflect.Type, k pref.Kind) converter {
	switch k {
	case pref.BoolKind:
		if t.Kind() == reflect.Bool {
			return makeScalarConverter(t, boolType)
		}
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		if t.Kind() == reflect.Int32 {
			return makeScalarConverter(t, int32Type)
		}
	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		if t.Kind() == reflect.Int64 {
			return makeScalarConverter(t, int64Type)
		}
	case pref.Uint32Kind, pref.Fixed32Kind:
		if t.Kind() == reflect.Uint32 {
			return makeScalarConverter(t, uint32Type)
		}
	case pref.Uint64Kind, pref.Fixed64Kind:
		if t.Kind() == reflect.Uint64 {
			return makeScalarConverter(t, uint64Type)
		}
	case pref.FloatKind:
		if t.Kind() == reflect.Float32 {
			return makeScalarConverter(t, float32Type)
		}
	case pref.DoubleKind:
		if t.Kind() == reflect.Float64 {
			return makeScalarConverter(t, float64Type)
		}
	case pref.StringKind:
		if t.Kind() == reflect.String || (t.Kind() == reflect.Slice && t.Elem() == byteType) {
			return makeScalarConverter(t, stringType)
		}
	case pref.BytesKind:
		if t.Kind() == reflect.String || (t.Kind() == reflect.Slice && t.Elem() == byteType) {
			return makeScalarConverter(t, bytesType)
		}
	case pref.EnumKind:
		// Handle v2 enums, which must satisfy the proto.Enum interface.
		if t.Kind() != reflect.Ptr && t.Implements(enumIfaceV2) {
			et := reflect.Zero(t).Interface().(pref.ProtoEnum).ProtoReflect().Type()
			return converter{
				toPB: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					e := v.Interface().(pref.ProtoEnum)
					return pref.ValueOf(e.ProtoReflect().Number())
				},
				toGo: func(v pref.Value) reflect.Value {
					rv := reflect.ValueOf(et.GoNew(v.Enum()))
					if rv.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), t))
					}
					return rv
				},
			}
		}

		// Handle v1 enums, which we identify as simply a named int32 type.
		if t.Kind() == reflect.Int32 && t.PkgPath() != "" {
			return converter{
				toPB: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					return pref.ValueOf(pref.EnumNumber(v.Int()))
				},
				toGo: func(v pref.Value) reflect.Value {
					return reflect.ValueOf(v.Enum()).Convert(t)
				},
			}
		}
	case pref.MessageKind, pref.GroupKind:
		// Handle v2 messages, which must satisfy the proto.Message interface.
		if t.Kind() == reflect.Ptr && t.Implements(messageIfaceV2) {
			mt := reflect.Zero(t).Interface().(pref.ProtoMessage).ProtoReflect().Type()
			return converter{
				toPB: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					return pref.ValueOf(v.Interface())
				},
				toGo: func(v pref.Value) reflect.Value {
					rv := reflect.ValueOf(v.Message())
					if rv.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), t))
					}
					return rv
				},
				newMessage: func() pref.Message {
					return mt.GoNew().ProtoReflect()
				},
			}
		}

		// Handle v1 messages, which we need to wrap as a v2 message.
		if t.Kind() == reflect.Ptr && t.Implements(messageIfaceV1) {
			return converter{
				toPB: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					return pref.ValueOf(wrapLegacyMessage(v))
				},
				toGo: func(v pref.Value) reflect.Value {
					type unwrapper interface{ Unwrap() interface{} }
					rv := reflect.ValueOf(v.Message().(unwrapper).Unwrap())
					if rv.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), t))
					}
					return rv
				},
				newMessage: func() pref.Message {
					return wrapLegacyMessage(reflect.New(t.Elem()))
				},
			}
		}
	}
	panic(fmt.Sprintf("invalid Go type %v for protobuf kind %v", t, k))
}

// converter provides functions for converting to/from Go reflect.Value types
// and protobuf protoreflect.Value types.
type converter struct {
	toPB       func(reflect.Value) pref.Value
	toGo       func(pref.Value) reflect.Value
	newMessage func() pref.Message
}

func makeScalarConverter(goType, pbType reflect.Type) converter {
	return converter{
		toPB: func(v reflect.Value) pref.Value {
			if v.Type() != goType {
				panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), goType))
			}
			if goType.Kind() == reflect.String && pbType.Kind() == reflect.Slice && v.Len() == 0 {
				return pref.ValueOf([]byte(nil)) // ensure empty string is []byte(nil)
			}
			return pref.ValueOf(v.Convert(pbType).Interface())
		},
		toGo: func(v pref.Value) reflect.Value {
			rv := reflect.ValueOf(v.Interface())
			if rv.Type() != pbType {
				panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), pbType))
			}
			if pbType.Kind() == reflect.String && goType.Kind() == reflect.Slice && rv.Len() == 0 {
				return reflect.Zero(goType) // ensure empty string is []byte(nil)
			}
			return rv.Convert(goType)
		},
	}
}
