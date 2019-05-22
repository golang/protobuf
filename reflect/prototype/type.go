// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prototype provides constructors for protoreflect.EnumType,
// protoreflect.MessageType, and protoreflect.ExtensionType.
package prototype

import (
	"fmt"
	"reflect"
	"sync"

	"google.golang.org/protobuf/internal/descfmt"
	"google.golang.org/protobuf/internal/value"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Enum is a protoreflect.EnumType which combines a
// protoreflect.EnumDescriptor with a constructor function.
//
// Both EnumDescriptor and NewEnum must be populated.
// Once constructed, the exported fields must not be modified.
type Enum struct {
	protoreflect.EnumDescriptor

	// NewEnum constructs a new protoreflect.Enum representing the provided
	// enum number. The returned Go type must be identical for every call.
	NewEnum func(protoreflect.EnumNumber) protoreflect.Enum

	once   sync.Once
	goType reflect.Type
}

func (t *Enum) New(n protoreflect.EnumNumber) protoreflect.Enum {
	e := t.NewEnum(n)
	t.once.Do(func() {
		t.goType = reflect.TypeOf(e)
		if e.Descriptor() != t.Descriptor() {
			panic(fmt.Sprintf("mismatching enum descriptor: got %v, want %v", e.Descriptor(), t.Descriptor()))
		}
		if e.Descriptor().IsPlaceholder() {
			panic("enum descriptor must not be a placeholder")
		}
	})
	if t.goType != reflect.TypeOf(e) {
		panic(fmt.Sprintf("mismatching types for enum: got %T, want %v", e, t.goType))
	}
	return e
}

func (t *Enum) GoType() reflect.Type {
	t.New(0) // initialize t.typ
	return t.goType
}

func (t *Enum) Descriptor() protoreflect.EnumDescriptor {
	return t.EnumDescriptor
}

func (t *Enum) Format(s fmt.State, r rune) {
	descfmt.FormatDesc(s, r, t)
}

// Message is a protoreflect.MessageType which combines a
// protoreflect.MessageDescriptor with a constructor function.
//
// Both MessageDescriptor and NewMessage must be populated.
// Once constructed, the exported fields must not be modified.
type Message struct {
	protoreflect.MessageDescriptor

	// NewMessage constructs an empty, newly allocated protoreflect.Message.
	// The returned Go type must be identical for every call.
	NewMessage func() protoreflect.Message

	once   sync.Once
	goType reflect.Type
}

func (t *Message) New() protoreflect.Message {
	m := t.NewMessage()
	mi := m.Interface()
	t.once.Do(func() {
		t.goType = reflect.TypeOf(mi)
		if m.Descriptor() != t.Descriptor() {
			panic(fmt.Sprintf("mismatching message descriptor: got %v, want %v", m.Descriptor(), t.Descriptor()))
		}
		if m.Descriptor().IsPlaceholder() {
			panic("message descriptor must not be a placeholder")
		}
	})
	if t.goType != reflect.TypeOf(mi) {
		panic(fmt.Sprintf("mismatching types for message: got %T, want %v", mi, t.goType))
	}
	return m
}

func (t *Message) GoType() reflect.Type {
	t.New() // initialize t.goType
	return t.goType
}

func (t *Message) Descriptor() protoreflect.MessageDescriptor {
	return t.MessageDescriptor
}

func (t *Message) Format(s fmt.State, r rune) {
	descfmt.FormatDesc(s, r, t)
}

// Extension is a protoreflect.ExtensionType which combines a
// protoreflect.ExtensionDescriptor with a constructor function.
//
// ExtensionDescriptor must be populated, while NewEnum or NewMessage must
// populated depending on the kind of the extension field.
// Once constructed, the exported fields must not be modified.
type Extension struct {
	protoreflect.ExtensionDescriptor

	// NewEnum constructs a new enum (see Enum.NewEnum).
	// This must be populated if and only if ExtensionDescriptor.Kind
	// is a protoreflect.EnumKind.
	NewEnum func(protoreflect.EnumNumber) protoreflect.Enum

	// NewMessage constructs a new message (see Enum.NewMessage).
	// This must be populated if and only if ExtensionDescriptor.Kind
	// is a protoreflect.MessageKind or protoreflect.GroupKind.
	NewMessage func() protoreflect.Message

	// TODO: Allow users to manually set new, valueOf, and interfaceOf.
	// This allows users to implement custom composite types (e.g., List) or
	// custom Go types for primitives (e.g., int32).

	once        sync.Once
	goType      reflect.Type
	new         func() protoreflect.Value
	valueOf     func(v interface{}) protoreflect.Value
	interfaceOf func(v protoreflect.Value) interface{}
}

func (t *Extension) New() protoreflect.Value {
	t.lazyInit()
	pv := t.new()
	v := t.interfaceOf(pv)
	if reflect.TypeOf(v) != t.goType {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, t.goType))
	}
	return pv
}

func (t *Extension) ValueOf(v interface{}) protoreflect.Value {
	t.lazyInit()
	if reflect.TypeOf(v) != t.goType {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, t.goType))
	}
	return t.valueOf(v)
}

func (t *Extension) InterfaceOf(v protoreflect.Value) interface{} {
	t.lazyInit()
	vi := t.interfaceOf(v)
	if reflect.TypeOf(vi) != t.goType {
		panic(fmt.Sprintf("invalid type: got %T, want %v", vi, t.goType))
	}
	return vi
}

// GoType is the type of the extension field.
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
func (t *Extension) GoType() reflect.Type {
	t.lazyInit()
	return t.goType
}

func (t *Extension) Descriptor() protoreflect.ExtensionDescriptor {
	return t.ExtensionDescriptor
}

func (t *Extension) Format(s fmt.State, r rune) {
	descfmt.FormatDesc(s, r, t)
}

func (t *Extension) lazyInit() {
	t.once.Do(func() {
		switch t.Kind() {
		case protoreflect.EnumKind:
			if t.NewEnum == nil || t.NewMessage != nil {
				panic("NewEnum alone must be set")
			}
			e := t.NewEnum(0)
			if e.Descriptor() != t.Enum() {
				panic(fmt.Sprintf("mismatching enum descriptor: got %v, want %v", e.Descriptor(), t.Enum()))
			}
			t.goType = reflect.TypeOf(e)
		case protoreflect.MessageKind, protoreflect.GroupKind:
			if t.NewEnum != nil || t.NewMessage == nil {
				panic("NewMessage alone must be set")
			}
			m := t.NewMessage()
			if m.Descriptor() != t.Message() {
				panic(fmt.Sprintf("mismatching message descriptor: got %v, want %v", m.Descriptor(), t.Message()))
			}
			t.goType = reflect.TypeOf(m.Interface())
		default:
			if t.NewEnum != nil || t.NewMessage != nil {
				panic("neither NewEnum nor NewMessage may be set")
			}
			t.goType = goTypeForPBKind[t.Kind()]
		}

		switch t.Cardinality() {
		case protoreflect.Optional:
			switch t.Kind() {
			case protoreflect.EnumKind:
				t.new = func() protoreflect.Value {
					return t.Default()
				}
				t.valueOf = func(v interface{}) protoreflect.Value {
					ev := v.(protoreflect.Enum)
					return protoreflect.ValueOf(ev.Number())
				}
				t.interfaceOf = func(v protoreflect.Value) interface{} {
					return t.NewEnum(v.Enum())
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				t.new = func() protoreflect.Value {
					return protoreflect.ValueOf(t.NewMessage())
				}
				t.valueOf = func(v interface{}) protoreflect.Value {
					mv := v.(protoreflect.ProtoMessage).ProtoReflect()
					return protoreflect.ValueOf(mv)
				}
				t.interfaceOf = func(v protoreflect.Value) interface{} {
					return v.Message().Interface()
				}
			default:
				t.new = func() protoreflect.Value {
					v := t.Default()
					if t.Kind() == protoreflect.BytesKind {
						// Copy default bytes to avoid aliasing the original.
						v = protoreflect.ValueOf(append([]byte(nil), v.Bytes()...))
					}
					return v
				}
				t.valueOf = func(v interface{}) protoreflect.Value {
					return protoreflect.ValueOf(v)
				}
				t.interfaceOf = func(v protoreflect.Value) interface{} {
					return v.Interface()
				}
			}
		case protoreflect.Repeated:
			var conv value.Converter
			elemType := t.goType
			switch t.Kind() {
			case protoreflect.EnumKind:
				conv = value.Converter{
					PBValueOf: func(v reflect.Value) protoreflect.Value {
						if v.Type() != elemType {
							panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), elemType))
						}
						e := v.Interface().(protoreflect.Enum)
						return protoreflect.ValueOf(e.Number())
					},
					GoValueOf: func(v protoreflect.Value) reflect.Value {
						rv := reflect.ValueOf(t.NewEnum(v.Enum()))
						if rv.Type() != elemType {
							panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), elemType))
						}
						return rv
					},
					NewEnum: t.NewEnum,
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				conv = value.Converter{
					PBValueOf: func(v reflect.Value) protoreflect.Value {
						if v.Type() != elemType {
							panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), elemType))
						}
						m := v.Interface().(protoreflect.ProtoMessage).ProtoReflect()
						return protoreflect.ValueOf(m)
					},
					GoValueOf: func(v protoreflect.Value) reflect.Value {
						rv := reflect.ValueOf(v.Message().Interface())
						if rv.Type() != elemType {
							panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), elemType))
						}
						return rv
					},
					NewMessage: t.NewMessage,
				}
			default:
				conv = value.NewConverter(elemType, t.Kind())
			}

			t.goType = reflect.PtrTo(reflect.SliceOf(elemType))
			t.new = func() protoreflect.Value {
				v := reflect.New(t.goType.Elem()).Interface()
				return protoreflect.ValueOf(value.ListOf(v, conv))
			}
			t.valueOf = func(v interface{}) protoreflect.Value {
				return protoreflect.ValueOf(value.ListOf(v, conv))
			}
			t.interfaceOf = func(v protoreflect.Value) interface{} {
				return v.List().(value.Unwrapper).ProtoUnwrap()
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

var (
	_ protoreflect.EnumType      = (*Enum)(nil)
	_ protoreflect.MessageType   = (*Message)(nil)
	_ protoreflect.ExtensionType = (*Extension)(nil)
)
