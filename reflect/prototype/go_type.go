// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/protobuf/v2/internal/value"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// GoEnum is a constructor for a protoreflect.EnumType.
type GoEnum struct {
	protoreflect.EnumDescriptor

	// New returns a concrete proto.Enum value with the given enum number.
	// The constructor must return the same concrete type for each invocation.
	New func(protoreflect.EnumType, protoreflect.EnumNumber) protoreflect.ProtoEnum

	once   sync.Once
	goType reflect.Type
}
type goEnum struct{ *GoEnum }

// NewGoEnum creates a new protoreflect.EnumType.
//
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields.
func NewGoEnum(t *GoEnum) protoreflect.EnumType {
	if t.IsPlaceholder() {
		panic("enum descriptor must not be a placeholder")
	}
	if t.New == nil {
		panic("invalid nil constructor for enum kind")
	}
	return goEnum{t}
}
func (p goEnum) GoNew(n protoreflect.EnumNumber) protoreflect.ProtoEnum {
	e := p.New(p, n)
	p.once.Do(func() { p.goType = reflect.TypeOf(e) })
	if p.goType != reflect.TypeOf(e) {
		panic(fmt.Sprintf("mismatching types for enum: got %T, want %v", e, p.goType))
	}
	return e
}
func (p goEnum) GoType() reflect.Type {
	p.once.Do(func() { p.goType = reflect.TypeOf(p.New(p, 0)) })
	return p.goType
}

// GoMessage is a constructor for a protoreflect.MessageType.
type GoMessage struct {
	protoreflect.MessageDescriptor

	// New returns a new empty proto.Message instance.
	// The constructor must return the same concrete type for each invocation.
	New func(protoreflect.MessageType) protoreflect.ProtoMessage

	once   sync.Once
	goType reflect.Type
}
type goMessage struct{ *GoMessage }

// NewGoMessage creates a new protoreflect.MessageType.
//
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields.
func NewGoMessage(t *GoMessage) protoreflect.MessageType {
	if t.IsPlaceholder() {
		panic("message descriptor must not be a placeholder")
	}
	if t.New == nil {
		panic("invalid nil constructor for message kind")
	}
	return goMessage{t}
}
func (p goMessage) GoNew() protoreflect.ProtoMessage {
	m := p.New(p)
	p.once.Do(func() { p.goType = reflect.TypeOf(m) })
	if p.goType != reflect.TypeOf(m) {
		panic(fmt.Sprintf("mismatching types for message: got %T, want %v", m, p.goType))
	}
	return m
}
func (p goMessage) GoType() reflect.Type {
	p.once.Do(func() { p.goType = reflect.TypeOf(p.New(p)) })
	return p.goType
}

// GoExtension is a constructor for a protoreflect.ExtensionType.
type GoExtension struct {
	protoreflect.ExtensionDescriptor

	// NewEnum returns a concrete proto.Enum value with the given enum number.
	// The constructor must be provided if protoreflect.ExtensionDescriptor.Kind
	// is protoreflect.EnumKind.
	//
	// The returned enum must represent an protoreflect.EnumDescriptor
	// that matches protoreflect.ExtensionDescriptor.EnumType.
	NewEnum func(protoreflect.EnumNumber) protoreflect.ProtoEnum

	// NewMessage returns a new empty proto.Message instance.
	// The constructor must be provided if protoreflect.ExtensionDescriptor.Kind
	// is protoreflect.MessageKind or protoreflect.GroupKind.
	//
	// The returned message must represent an protoreflect.MessageDescriptor
	// that matches protoreflect.ExtensionDescriptor.MessageType.
	NewMessage func() protoreflect.ProtoMessage

	// TODO: Separate NewEnum and NewMessage constructors make it possible for
	// users to provide a constructor that returns a Go type does not match
	// the corresponding protobuf descriptor in ExtensionDescriptor.
	// Checking for correctness is hard since descriptors are not comparable.
	//
	// An alternative API is for ExtensionDescriptor.{EnumType,MessageType}
	// to document that it must implement protoreflect.{EnumType,MessageType}.

	// TODO: Support custom Go types? If so, the user needs to provide their
	// own New, ValueOf, and InterfaceOf adapters.

	once        sync.Once
	new         func() interface{}
	goType      reflect.Type
	valueOf     func(v interface{}) protoreflect.Value
	interfaceOf func(v protoreflect.Value) interface{}
}
type goExtension struct{ *GoExtension }

// NewGoExtension creates a new protoreflect.ExtensionType.
//
// The Go type is currently determined automatically (although custom Go types
// may be supported in the future). The type is T for scalars and
// *[]T for vectors. Maps are not valid in extension fields.
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
//
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields.
func NewGoExtension(t *GoExtension) protoreflect.ExtensionType {
	if t.ExtendedType() == nil {
		panic("field descriptor does not extend a message")
	}
	switch t.Kind() {
	case protoreflect.EnumKind:
		if t.NewEnum == nil {
			panic("enum constructor not provided for enum kind")
		}
		if t.NewMessage != nil {
			panic("message constructor provided for enum kind")
		}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if t.NewMessage == nil {
			panic("message constructor not provided for message kind")
		}
		if t.NewEnum != nil {
			panic("enum constructor provided for message kind")
		}
	default:
		if t.NewMessage != nil || t.NewEnum != nil {
			panic(fmt.Sprintf("enum or message constructor provided for %v kind", t.Kind()))
		}
	}
	return goExtension{t}
}
func (p goExtension) GoNew() interface{} {
	p.lazyInit()
	v := p.new()
	if reflect.TypeOf(v) != p.goType {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, p.goType))
	}
	return v
}
func (p goExtension) GoType() reflect.Type {
	p.lazyInit()
	return p.goType
}
func (p goExtension) ValueOf(v interface{}) protoreflect.Value {
	p.lazyInit()
	if reflect.TypeOf(v) != p.goType {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, p.goType))
	}
	return p.valueOf(v)
}
func (p goExtension) InterfaceOf(pv protoreflect.Value) interface{} {
	p.lazyInit()
	v := p.interfaceOf(pv)
	if reflect.TypeOf(v) != p.goType {
		panic(fmt.Sprintf("invalid type: got %T, want %v", v, p.goType))
	}
	return v
}
func (p goExtension) lazyInit() {
	p.once.Do(func() {
		switch p.Cardinality() {
		case protoreflect.Optional:
			switch p.Kind() {
			case protoreflect.EnumKind:
				p.goType = reflect.TypeOf(p.NewEnum(0))
				p.new = func() interface{} { return p.NewEnum(p.Default().Enum()) }
				p.valueOf = func(v interface{}) protoreflect.Value {
					ev := v.(protoreflect.ProtoEnum).ProtoReflect()
					return protoreflect.ValueOf(ev.Number())
				}
				p.interfaceOf = func(pv protoreflect.Value) interface{} {
					return p.NewEnum(pv.Enum())
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				p.goType = reflect.TypeOf(p.NewMessage())
				p.new = func() interface{} { return p.NewMessage() }
				p.valueOf = func(v interface{}) protoreflect.Value {
					return protoreflect.ValueOf(v)
				}
				p.interfaceOf = func(pv protoreflect.Value) interface{} {
					return pv.Message().Interface()
				}
			default:
				p.goType = goTypeForPBKind[p.Kind()]
				p.new = func() interface{} { return p.Default().Interface() }
				p.valueOf = func(v interface{}) protoreflect.Value {
					return protoreflect.ValueOf(v)
				}
				p.interfaceOf = func(pv protoreflect.Value) interface{} {
					v := pv.Interface()
					return v
				}
			}
		case protoreflect.Repeated:
			var goType reflect.Type
			switch p.Kind() {
			case protoreflect.EnumKind:
				goType = reflect.TypeOf(p.NewEnum(p.Default().Enum()))
			case protoreflect.MessageKind, protoreflect.GroupKind:
				goType = reflect.TypeOf(p.NewMessage())
			default:
				goType = goTypeForPBKind[p.Kind()]
			}
			c := value.NewConverter(goType, p.Kind())
			p.goType = reflect.PtrTo(reflect.SliceOf(goType))
			p.new = func() interface{} { return reflect.New(p.goType.Elem()).Interface() }
			p.valueOf = func(v interface{}) protoreflect.Value {
				return protoreflect.ValueOf(value.VectorOf(v, c))
			}
			p.interfaceOf = func(v protoreflect.Value) interface{} {
				// TODO: Can we assume that Vector implementations know how
				// to unwrap themselves?
				// Should this be part of the public API in protoreflect?
				return v.Vector().(value.Unwrapper).Unwrap()
			}
		default:
			panic(fmt.Sprintf("invalid cardinality: %v", p.Cardinality()))
		}
	})
}

var goTypeForPBKind = map[protoreflect.Kind]reflect.Type{
	protoreflect.BoolKind:     reflect.TypeOf(bool(false)),
	protoreflect.Int32Kind:    reflect.TypeOf(int32(0)),
	protoreflect.Sint32Kind:   reflect.TypeOf(int32(0)),
	protoreflect.Fixed32Kind:  reflect.TypeOf(int32(0)),
	protoreflect.Int64Kind:    reflect.TypeOf(int64(0)),
	protoreflect.Sint64Kind:   reflect.TypeOf(int64(0)),
	protoreflect.Fixed64Kind:  reflect.TypeOf(int64(0)),
	protoreflect.Uint32Kind:   reflect.TypeOf(uint32(0)),
	protoreflect.Sfixed32Kind: reflect.TypeOf(uint32(0)),
	protoreflect.Uint64Kind:   reflect.TypeOf(uint64(0)),
	protoreflect.Sfixed64Kind: reflect.TypeOf(uint64(0)),
	protoreflect.FloatKind:    reflect.TypeOf(float32(0)),
	protoreflect.DoubleKind:   reflect.TypeOf(float64(0)),
	protoreflect.StringKind:   reflect.TypeOf(string("")),
	protoreflect.BytesKind:    reflect.TypeOf([]byte(nil)),
}
