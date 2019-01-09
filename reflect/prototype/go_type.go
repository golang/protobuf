// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/protobuf/v2/internal/typefmt"
	"github.com/golang/protobuf/v2/internal/value"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// GoEnum creates a new protoreflect.EnumType by combining the provided
// protoreflect.EnumDescriptor with the provided constructor function.
func GoEnum(ed protoreflect.EnumDescriptor, fn func(protoreflect.EnumType, protoreflect.EnumNumber) protoreflect.Enum) protoreflect.EnumType {
	if ed.IsPlaceholder() {
		panic("enum descriptor must not be a placeholder")
	}
	return &goEnum{EnumDescriptor: ed, new: fn}
}

type goEnum struct {
	protoreflect.EnumDescriptor
	new func(protoreflect.EnumType, protoreflect.EnumNumber) protoreflect.Enum

	once sync.Once
	typ  reflect.Type
}

func (t *goEnum) GoType() reflect.Type {
	t.New(0) // initialize t.typ
	return t.typ
}
func (t *goEnum) New(n protoreflect.EnumNumber) protoreflect.Enum {
	e := t.new(t, n)
	t.once.Do(func() { t.typ = reflect.TypeOf(e) })
	if t.typ != reflect.TypeOf(e) {
		panic(fmt.Sprintf("mismatching types for enum: got %T, want %v", e, t.typ))
	}
	return e
}
func (t *goEnum) Format(s fmt.State, r rune) {
	typefmt.FormatDesc(s, r, t)
}

// GoMessage creates a new protoreflect.MessageType by combining the provided
// protoreflect.MessageDescriptor with the provided constructor function.
func GoMessage(md protoreflect.MessageDescriptor, fn func(protoreflect.MessageType) protoreflect.Message) protoreflect.MessageType {
	if md.IsPlaceholder() {
		panic("message descriptor must not be a placeholder")
	}
	// NOTE: Avoid calling fn in the constructor since fn itself may depend on
	// this function returning (for cyclic message dependencies).
	return &goMessage{MessageDescriptor: md, new: fn}
}

type goMessage struct {
	protoreflect.MessageDescriptor
	new func(protoreflect.MessageType) protoreflect.Message

	once sync.Once
	typ  reflect.Type
}

func (t *goMessage) GoType() reflect.Type {
	t.New() // initialize t.typ
	return t.typ
}
func (t *goMessage) New() protoreflect.Message {
	m := t.new(t)
	mi := m.Interface()
	t.once.Do(func() { t.typ = reflect.TypeOf(mi) })
	if t.typ != reflect.TypeOf(mi) {
		panic(fmt.Sprintf("mismatching types for message: got %T, want %v", mi, t.typ))
	}
	return m
}
func (t *goMessage) Format(s fmt.State, r rune) {
	typefmt.FormatDesc(s, r, t)
}

// GoExtension creates a new protoreflect.ExtensionType.
//
// An enum type must be provided for enum extension fields if
// ExtensionDescriptor.EnumType does not implement protoreflect.EnumType,
// in which case it replaces the original enum in ExtensionDescriptor.
//
// Similarly, a message type must be provided for message extension fields if
// ExtensionDescriptor.MessageType does not implement protoreflect.MessageType,
// in which case it replaces the original message in ExtensionDescriptor.
//
// The Go type is currently determined automatically.
// The type is T for scalars and *[]T for lists (maps are not allowed).
// The type T is determined as follows:
//
//	+------------+-------------------------------------+
//	| Go type    | Protobuf kind                       |
//	+------------+-------------------------------------+
//	| bool       | BoolKind                            |
//	| int32      | Int32Kind, Sint32Kind, Sfixed32Kind |
//	| int64      | Int64Kind, Sint64Kind, Sfixed64Kind |
//	| uint32     | Uint32Kind, Fixed32Kind             |
//	| uint64     | Uint64Kind, Fixed64Kind             |
//	| float32    | FloatKind                           |
//	| float64    | DoubleKind                          |
//	| string     | StringKind                          |
//	| []byte     | BytesKind                           |
//	| E          | EnumKind                            |
//	| M          | MessageKind, GroupKind              |
//	+------------+-------------------------------------+
//
// The type E is the concrete enum type returned by NewEnum,
// which is often, but not required to be, a named int32 type.
// The type M is the concrete message type returned by NewMessage,
// which is often, but not required to be, a pointer to a named struct type.
func GoExtension(xd protoreflect.ExtensionDescriptor, et protoreflect.EnumType, mt protoreflect.MessageType) protoreflect.ExtensionType {
	if xd.ExtendedType() == nil {
		panic("field descriptor does not extend a message")
	}
	switch xd.Kind() {
	case protoreflect.EnumKind:
		if et2, ok := xd.EnumType().(protoreflect.EnumType); ok && et == nil {
			et = et2
		}
		if et == nil {
			panic("enum type not provided for enum kind")
		}
		if mt != nil {
			panic("message type provided for enum kind")
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if mt2, ok := xd.MessageType().(protoreflect.MessageType); ok && mt == nil {
			mt = mt2
		}
		if et != nil {
			panic("enum type provided for message kind")
		}
		if mt == nil {
			panic("message type not provided for message kind")
		}
	default:
		if et != nil || mt != nil {
			panic(fmt.Sprintf("enum or message type provided for %v kind", xd.Kind()))
		}
	}
	return &goExtension{ExtensionDescriptor: xd, enumType: et, messageType: mt}
}

type goExtension struct {
	protoreflect.ExtensionDescriptor
	enumType    protoreflect.EnumType
	messageType protoreflect.MessageType

	once        sync.Once
	typ         reflect.Type
	new         func() protoreflect.Value
	valueOf     func(v interface{}) protoreflect.Value
	interfaceOf func(v protoreflect.Value) interface{}
}

func (t *goExtension) EnumType() protoreflect.EnumDescriptor {
	return t.enumType
}
func (t *goExtension) MessageType() protoreflect.MessageDescriptor {
	return t.messageType
}
func (t *goExtension) GoType() reflect.Type {
	t.lazyInit()
	return t.typ
}
func (t *goExtension) New() protoreflect.Value {
	t.lazyInit()
	pv := t.new()
	v := t.interfaceOf(pv)
	if reflect.TypeOf(v) != t.typ {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, t.typ))
	}
	return pv
}
func (t *goExtension) ValueOf(v interface{}) protoreflect.Value {
	t.lazyInit()
	if reflect.TypeOf(v) != t.typ {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, t.typ))
	}
	return t.valueOf(v)
}
func (t *goExtension) InterfaceOf(pv protoreflect.Value) interface{} {
	t.lazyInit()
	v := t.interfaceOf(pv)
	if reflect.TypeOf(v) != t.typ {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, t.typ))
	}
	return v
}
func (t *goExtension) Format(s fmt.State, r rune) {
	typefmt.FormatDesc(s, r, t)
}
func (t *goExtension) lazyInit() {
	t.once.Do(func() {
		switch t.Cardinality() {
		case protoreflect.Optional:
			switch t.Kind() {
			case protoreflect.EnumKind:
				t.typ = t.enumType.GoType()
				t.new = func() protoreflect.Value {
					return t.Default()
				}
				t.valueOf = func(v interface{}) protoreflect.Value {
					ev := v.(protoreflect.Enum)
					return protoreflect.ValueOf(ev.Number())
				}
				t.interfaceOf = func(pv protoreflect.Value) interface{} {
					return t.enumType.New(pv.Enum())
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				t.typ = t.messageType.GoType()
				t.new = func() protoreflect.Value {
					return protoreflect.ValueOf(t.messageType.New())
				}
				t.valueOf = func(v interface{}) protoreflect.Value {
					mv := v.(protoreflect.ProtoMessage).ProtoReflect()
					return protoreflect.ValueOf(mv)
				}
				t.interfaceOf = func(pv protoreflect.Value) interface{} {
					return pv.Message().Interface()
				}
			default:
				t.typ = goTypeForPBKind[t.Kind()]
				t.new = func() protoreflect.Value {
					return t.Default()
				}
				t.valueOf = func(v interface{}) protoreflect.Value {
					return protoreflect.ValueOf(v)
				}
				t.interfaceOf = func(pv protoreflect.Value) interface{} {
					return pv.Interface()
				}
			}
		case protoreflect.Repeated:
			var typ reflect.Type
			switch t.Kind() {
			case protoreflect.EnumKind:
				typ = t.enumType.GoType()
			case protoreflect.MessageKind, protoreflect.GroupKind:
				typ = t.messageType.GoType()
			default:
				typ = goTypeForPBKind[t.Kind()]
			}
			c := value.NewConverter(typ, t.Kind())
			t.typ = reflect.PtrTo(reflect.SliceOf(typ))
			t.new = func() protoreflect.Value {
				v := reflect.New(t.typ.Elem()).Interface()
				return protoreflect.ValueOf(value.ListOf(v, c))
			}
			t.valueOf = func(v interface{}) protoreflect.Value {
				return protoreflect.ValueOf(value.ListOf(v, c))
			}
			t.interfaceOf = func(pv protoreflect.Value) interface{} {
				return pv.List().(value.Unwrapper).ProtoUnwrap()
			}
		default:
			panic(fmt.Sprintf("invalid cardinality: %v", t.Cardinality()))
		}
	})
}

var goTypeForPBKind = map[protoreflect.Kind]reflect.Type{
	protoreflect.BoolKind:     reflect.TypeOf(bool(false)),
	protoreflect.Int32Kind:    reflect.TypeOf(int32(0)),
	protoreflect.Sint32Kind:   reflect.TypeOf(int32(0)),
	protoreflect.Sfixed32Kind: reflect.TypeOf(int32(0)),
	protoreflect.Int64Kind:    reflect.TypeOf(int64(0)),
	protoreflect.Sint64Kind:   reflect.TypeOf(int64(0)),
	protoreflect.Sfixed64Kind: reflect.TypeOf(int64(0)),
	protoreflect.Uint32Kind:   reflect.TypeOf(uint32(0)),
	protoreflect.Fixed32Kind:  reflect.TypeOf(uint32(0)),
	protoreflect.Uint64Kind:   reflect.TypeOf(uint64(0)),
	protoreflect.Fixed64Kind:  reflect.TypeOf(uint64(0)),
	protoreflect.FloatKind:    reflect.TypeOf(float32(0)),
	protoreflect.DoubleKind:   reflect.TypeOf(float64(0)),
	protoreflect.StringKind:   reflect.TypeOf(string("")),
	protoreflect.BytesKind:    reflect.TypeOf([]byte(nil)),
}
