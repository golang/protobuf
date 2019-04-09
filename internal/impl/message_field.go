// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"math"
	"reflect"

	"google.golang.org/protobuf/internal/encoding/wire"
	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

type fieldInfo struct {
	fieldDesc pref.FieldDescriptor

	// These fields are used for protobuf reflection support.
	has        func(pointer) bool
	clear      func(pointer)
	get        func(pointer) pref.Value
	set        func(pointer, pref.Value)
	mutable    func(pointer) pref.Value
	newMessage func() pref.Message

	// These fields are used for fast-path functions.
	funcs      pointerCoderFuncs // fast-path per-field functions
	num        pref.FieldNumber  // field number
	offset     offset            // struct field offset
	wiretag    uint64            // field tag (number + wire type)
	tagsize    int               // size of the varint-encoded tag
	isPointer  bool              // true if IsNil may be called on the struct field
	isRequired bool              // true if field is required
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
	conv, _ := newConverter(ot.Field(0).Type, fd.Kind())
	var frozenEmpty pref.Value
	if conv.NewMessage != nil {
		frozenEmpty = pref.ValueOf(frozenMessage{conv.NewMessage()})
	}

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs)
	return fieldInfo{
		// NOTE: The logic below intentionally assumes that oneof fields are
		// well-formatted. That is, the oneof interface never contains a
		// typed nil pointer to one of the wrapper structs.

		fieldDesc: fd,
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
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				return
			}
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				if frozenEmpty.IsValid() {
					return frozenEmpty
				}
				return defaultValueOf(fd)
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				if frozenEmpty.IsValid() {
					return frozenEmpty
				}
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
		mutable: func(p pointer) pref.Value {
			if conv.NewMessage == nil {
				panic("invalid Mutable on field with non-composite type")
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() || rv.Elem().Type().Elem() != ot {
				rv.Set(reflect.New(ot))
			}
			rv = rv.Elem().Elem().Field(0)
			if rv.IsNil() {
				rv.Set(conv.GoValueOf(pref.ValueOf(conv.NewMessage())))
			}
			return conv.PBValueOf(rv)
		},
		newMessage: conv.NewMessage,
		offset:     fieldOffset,
		isPointer:  true,
	}
}

func fieldInfoForMap(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Map {
		panic(fmt.Sprintf("invalid type: got %v, want map kind", ft))
	}
	keyConv, _ := newConverter(ft.Key(), fd.MapKey().Kind())
	valConv, _ := newConverter(ft.Elem(), fd.MapValue().Kind())
	wiretag := wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])
	frozenEmpty := pref.ValueOf(frozenMap{
		pvalue.MapOf(reflect.Zero(reflect.PtrTo(fs.Type)).Interface(), keyConv, valConv),
	})

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs)
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return rv.Len() > 0
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return frozenEmpty
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				return frozenEmpty
			}
			return pref.ValueOf(pvalue.MapOf(rv.Addr().Interface(), keyConv, valConv))
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.ValueOf(v.Map().(pvalue.Unwrapper).ProtoUnwrap()).Elem())
		},
		mutable: func(p pointer) pref.Value {
			v := p.Apply(fieldOffset).AsIfaceOf(fs.Type)
			return pref.ValueOf(pvalue.MapOf(v, keyConv, valConv))
		},
		funcs:     encoderFuncsForMap(fd, ft),
		offset:    fieldOffset,
		wiretag:   wiretag,
		tagsize:   wire.SizeVarint(wiretag),
		isPointer: true,
	}
}

func fieldInfoForList(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	if ft.Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid type: got %v, want slice kind", ft))
	}
	conv, _ := newConverter(ft.Elem(), fd.Kind())
	var wiretag uint64
	if !fd.IsPacked() {
		wiretag = wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])
	} else {
		wiretag = wire.EncodeTag(fd.Number(), wire.BytesType)
	}
	frozenEmpty := pref.ValueOf(frozenList{
		pvalue.ListOf(reflect.Zero(reflect.PtrTo(fs.Type)).Interface(), conv),
	})

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs)
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return rv.Len() > 0
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return frozenEmpty
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.Len() == 0 {
				return frozenEmpty
			}
			return pref.ValueOf(pvalue.ListOf(rv.Addr().Interface(), conv))
		},
		set: func(p pointer, v pref.Value) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.ValueOf(v.List().(pvalue.Unwrapper).ProtoUnwrap()).Elem())
		},
		mutable: func(p pointer) pref.Value {
			v := p.Apply(fieldOffset).AsIfaceOf(fs.Type)
			return pref.ValueOf(pvalue.ListOf(v, conv))
		},
		funcs:     fieldCoder(fd, ft),
		offset:    fieldOffset,
		wiretag:   wiretag,
		tagsize:   wire.SizeVarint(wiretag),
		isPointer: true,
	}
}

var emptyBytes = reflect.ValueOf([]byte{})

func fieldInfoForScalar(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	funcs := fieldCoder(fd, ft)
	nullable := fd.Syntax() == pref.Proto2
	if nullable {
		if ft.Kind() != reflect.Ptr && ft.Kind() != reflect.Slice {
			panic(fmt.Sprintf("invalid type: got %v, want pointer", ft))
		}
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
	}
	conv, _ := newConverter(ft, fd.Kind())
	wiretag := wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs)
	return fieldInfo{
		fieldDesc: fd,
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
				return rv.Float() != 0 || math.Signbit(rv.Float())
			case reflect.String, reflect.Slice:
				return rv.Len() > 0
			default:
				panic(fmt.Sprintf("invalid type: %v", rv.Type())) // should never happen
			}
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
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
		funcs:      funcs,
		offset:     fieldOffset,
		isPointer:  nullable,
		isRequired: fd.Cardinality() == pref.Required,
		wiretag:    wiretag,
		tagsize:    wire.SizeVarint(wiretag),
	}
}

func fieldInfoForMessage(fd pref.FieldDescriptor, fs reflect.StructField) fieldInfo {
	ft := fs.Type
	conv, _ := newConverter(ft, fd.Kind())
	wiretag := wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])
	frozenEmpty := pref.ValueOf(frozenMessage{conv.NewMessage()})

	// TODO: Implement unsafe fast path?
	fieldOffset := offsetOf(fs)
	return fieldInfo{
		fieldDesc: fd,
		has: func(p pointer) bool {
			if p.IsNil() {
				return false
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			return !rv.IsNil()
		},
		clear: func(p pointer) {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			rv.Set(reflect.Zero(rv.Type()))
		},
		get: func(p pointer) pref.Value {
			if p.IsNil() {
				return frozenEmpty
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				return frozenEmpty
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
		mutable: func(p pointer) pref.Value {
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				rv.Set(conv.GoValueOf(pref.ValueOf(conv.NewMessage())))
			}
			return conv.PBValueOf(rv)
		},
		newMessage: conv.NewMessage,
		funcs:      fieldCoder(fd, ft),
		offset:     fieldOffset,
		isPointer:  true,
		isRequired: fd.Cardinality() == pref.Required,
		wiretag:    wiretag,
		tagsize:    wire.SizeVarint(wiretag),
	}
}

type oneofInfo struct {
	oneofDesc pref.OneofDescriptor
	which     func(pointer) pref.FieldNumber
}

func makeOneofInfo(od pref.OneofDescriptor, fs reflect.StructField, wrappersByType map[reflect.Type]pref.FieldNumber) *oneofInfo {
	fieldOffset := offsetOf(fs)
	return &oneofInfo{
		oneofDesc: od,
		which: func(p pointer) pref.FieldNumber {
			if p.IsNil() {
				return 0
			}
			rv := p.Apply(fieldOffset).AsValueOf(fs.Type).Elem()
			if rv.IsNil() {
				return 0
			}
			return wrappersByType[rv.Elem().Type().Elem()]
		},
	}
}

var (
	enumIfaceV2    = reflect.TypeOf((*pref.Enum)(nil)).Elem()
	messageIfaceV1 = reflect.TypeOf((*piface.MessageV1)(nil)).Elem()
	messageIfaceV2 = reflect.TypeOf((*pref.ProtoMessage)(nil)).Elem()
)

func newConverter(t reflect.Type, k pref.Kind) (conv pvalue.Converter, isLegacy bool) {
	switch k {
	case pref.EnumKind:
		if t.Kind() == reflect.Int32 && !t.Implements(enumIfaceV2) {
			return pvalue.Converter{
				PBValueOf: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					return pref.ValueOf(pref.EnumNumber(v.Int()))
				},
				GoValueOf: func(v pref.Value) reflect.Value {
					return reflect.ValueOf(v.Enum()).Convert(t)
				},
				NewEnum: func(n pref.EnumNumber) pref.Enum {
					return legacyWrapEnum(reflect.ValueOf(n).Convert(t))
				},
			}, true
		}
	case pref.MessageKind, pref.GroupKind:
		if t.Kind() == reflect.Ptr && t.Implements(messageIfaceV1) && !t.Implements(messageIfaceV2) {
			return pvalue.Converter{
				PBValueOf: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					return pref.ValueOf(Export{}.MessageOf(v.Interface()))
				},
				GoValueOf: func(v pref.Value) reflect.Value {
					rv := reflect.ValueOf(v.Message().(pvalue.Unwrapper).ProtoUnwrap())
					if rv.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), t))
					}
					return rv
				},
				NewMessage: func() pref.Message {
					return legacyWrapMessage(reflect.New(t.Elem())).ProtoReflect()
				},
			}, true
		}
	}
	return pvalue.NewConverter(t, k), false
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

// frozenValueOf returns a frozen version of any composite value.
func frozenValueOf(v pref.Value) pref.Value {
	switch v := v.Interface().(type) {
	case pref.Message:
		if _, ok := v.(frozenMessage); !ok {
			return pref.ValueOf(frozenMessage{v})
		}
	case pref.List:
		if _, ok := v.(frozenList); !ok {
			return pref.ValueOf(frozenList{v})
		}
	case pref.Map:
		if _, ok := v.(frozenMap); !ok {
			return pref.ValueOf(frozenMap{v})
		}
	}
	return v
}

type frozenMessage struct{ pref.Message }

func (m frozenMessage) ProtoReflect() pref.Message   { return m }
func (m frozenMessage) Interface() pref.ProtoMessage { return m }
func (m frozenMessage) Range(f func(pref.FieldDescriptor, pref.Value) bool) {
	m.Message.Range(func(fd pref.FieldDescriptor, v pref.Value) bool {
		return f(fd, frozenValueOf(v))
	})
}
func (m frozenMessage) Get(fd pref.FieldDescriptor) pref.Value {
	v := m.Message.Get(fd)
	return frozenValueOf(v)
}
func (frozenMessage) Clear(pref.FieldDescriptor)              { panic("invalid on read-only Message") }
func (frozenMessage) Set(pref.FieldDescriptor, pref.Value)    { panic("invalid on read-only Message") }
func (frozenMessage) Mutable(pref.FieldDescriptor) pref.Value { panic("invalid on read-only Message") }
func (frozenMessage) SetUnknown(pref.RawFields)               { panic("invalid on read-only Message") }

type frozenList struct{ pref.List }

func (ls frozenList) Get(i int) pref.Value {
	v := ls.List.Get(i)
	return frozenValueOf(v)
}
func (frozenList) Set(i int, v pref.Value) { panic("invalid on read-only List") }
func (frozenList) Append(v pref.Value)     { panic("invalid on read-only List") }
func (frozenList) Truncate(i int)          { panic("invalid on read-only List") }

type frozenMap struct{ pref.Map }

func (ms frozenMap) Get(k pref.MapKey) pref.Value {
	v := ms.Map.Get(k)
	return frozenValueOf(v)
}
func (ms frozenMap) Range(f func(pref.MapKey, pref.Value) bool) {
	ms.Map.Range(func(k pref.MapKey, v pref.Value) bool {
		return f(k, frozenValueOf(v))
	})
}
func (frozenMap) Set(k pref.MapKey, v pref.Value) { panic("invalid n read-only Map") }
func (frozenMap) Clear(k pref.MapKey)             { panic("invalid on read-only Map") }
