// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dynamicpb creates protocol buffer messages using runtime type information.
package dynamicpb

import (
	"math"

	"google.golang.org/protobuf/internal/errors"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

// A Message is a dynamically constructed protocol buffer message.
//
// Message implements the proto.Message interface, and may be used with all
// standard proto package functions such as Marshal, Unmarshal, and so forth.
//
// Message also implements the protoreflect.Message interface. See the protoreflect
// package documentation for that interface for how to get and set fields and
// otherwise interact with the contents of a Message.
//
// Reflection API functions which construct messages, such as NewMessage,
// return new dynamic messages of the appropriate type. Functions which take
// messages, such as Set for a message-value field, will accept any message
// with a compatible type.
//
// Operations which modify a Message are not safe for concurrent use.
type Message struct {
	desc    pref.MessageDescriptor
	known   map[pref.FieldNumber]pref.Value
	ext     map[pref.FieldNumber]pref.FieldDescriptor
	unknown pref.RawFields
}

// New creates a new message with the provided descriptor.
func New(desc pref.MessageDescriptor) *Message {
	return &Message{
		desc:  desc,
		known: make(map[pref.FieldNumber]pref.Value),
		ext:   make(map[pref.FieldNumber]pref.FieldDescriptor),
	}
}

// ProtoReflect implements the protoreflect.ProtoMessage interface.
func (m *Message) ProtoReflect() pref.Message {
	return m
}

// String returns a string representation of a message.
func (m *Message) String() string {
	return protoimpl.X.MessageStringOf(m)
}

// Descriptor returns the message descriptor.
func (m *Message) Descriptor() pref.MessageDescriptor {
	return m.desc
}

// New returns a newly allocated empty message with the same descriptor.
// See protoreflect.Message for details.
func (m *Message) New() pref.Message {
	return New(m.desc)
}

// Interface returns the message.
// See protoreflect.Message for details.
func (m *Message) Interface() pref.ProtoMessage {
	return m
}

// Len returns the number of populated fields.
// See protoreflect.Message for details.
func (m *Message) Len() int {
	count := 0
	for num, v := range m.known {
		if m.ext[num] != nil {
			count++
			continue
		}
		if isSet(m.desc.Fields().ByNumber(num), v) {
			count++
		}
	}
	return count
}

// Range visits every populated field in undefined order.
// See protoreflect.Message for details.
func (m *Message) Range(f func(pref.FieldDescriptor, pref.Value) bool) {
	for num, v := range m.known {
		fd := m.ext[num]
		if fd == nil {
			fd = m.desc.Fields().ByNumber(num)
			if !isSet(fd, v) {
				continue
			}
		}
		if !f(fd, v) {
			return
		}
	}
}

// Range reports whether a field is populated.
// See protoreflect.Message for details.
func (m *Message) Has(fd pref.FieldDescriptor) bool {
	m.checkField(fd)
	if fd.IsExtension() {
		return m.ext[fd.Number()] == fd
	}
	v, ok := m.known[fd.Number()]
	if !ok {
		return false
	}
	return isSet(fd, v)
}

// Clear clears a field.
// See protoreflect.Message for details.
func (m *Message) Clear(fd pref.FieldDescriptor) {
	m.checkField(fd)
	num := fd.Number()
	delete(m.known, num)
	delete(m.ext, num)
}

// Get returns the value of a field.
// See protoreflect.Message for details.
func (m *Message) Get(fd pref.FieldDescriptor) pref.Value {
	m.checkField(fd)
	num := fd.Number()
	if v, ok := m.known[num]; ok {
		if !fd.IsExtension() || fd == m.ext[num] {
			return v
		}
	}
	switch {
	case fd.IsMap():
		return pref.ValueOf(&dynamicMap{desc: fd})
	case fd.Cardinality() == pref.Repeated:
		return pref.ValueOf(emptyList{desc: fd})
	case fd.Kind() == pref.BytesKind:
		return pref.ValueOf(append([]byte(nil), fd.Default().Bytes()...))
	default:
		return fd.Default()
	}
}

// Mutable returns a mutable reference to a repeated, map, or message field.
// See protoreflect.Message for details.
func (m *Message) Mutable(fd pref.FieldDescriptor) pref.Value {
	m.checkField(fd)
	num := fd.Number()
	if v, ok := m.known[num]; ok {
		if !fd.IsExtension() || fd == m.ext[num] {
			return v
		}
	}
	if !fd.IsMap() && !fd.IsList() && fd.Message() == nil {
		panic(errors.New("%v: getting mutable reference to non-composite type", fd.FullName()))
	}
	m.clearOtherOneofFields(fd)
	switch {
	case fd.IsExtension():
		m.known[num] = fd.(pref.ExtensionType).New()
		m.ext[num] = fd
	case fd.IsMap():
		m.known[num] = pref.ValueOf(&dynamicMap{
			desc: fd,
			mapv: make(map[interface{}]pref.Value),
		})
	case fd.IsList():
		m.known[num] = pref.ValueOf(&dynamicList{desc: fd})
	case fd.Message() != nil:
		m.known[num] = pref.ValueOf(m.NewMessage(fd))
	}
	return m.known[num]
}

// Set stores a value in a field.
// See protoreflect.Message for details.
func (m *Message) Set(fd pref.FieldDescriptor, v pref.Value) {
	m.checkField(fd)
	switch {
	case fd.IsExtension():
		// Call InterfaceOf just to let the extension typecheck the value.
		_ = fd.(pref.ExtensionType).InterfaceOf(v)
		m.ext[fd.Number()] = fd
	case fd.IsMap():
		if mapv, ok := v.Interface().(*dynamicMap); !ok || mapv.desc != fd {
			panic(errors.New("%v: assigning invalid type %T", fd.FullName(), v.Interface()))
		}
	case fd.IsList():
		if list, ok := v.Interface().(*dynamicList); !ok || list.desc != fd {
			panic(errors.New("%v: assigning invalid type %T", fd.FullName(), v.Interface()))
		}
	default:
		typecheckSingular(fd, v)
	}
	m.clearOtherOneofFields(fd)
	m.known[fd.Number()] = v
}

func (m *Message) clearOtherOneofFields(fd pref.FieldDescriptor) {
	od := fd.ContainingOneof()
	if od == nil {
		return
	}
	num := fd.Number()
	for i := 0; i < od.Fields().Len(); i++ {
		if n := od.Fields().Get(i).Number(); n != num {
			delete(m.known, n)
		}
	}
}

// NewMessage returns a newly-allocated message assignable to a field.
// See protoreflect.Message for details.
func (m *Message) NewMessage(fd pref.FieldDescriptor) pref.Message {
	m.checkField(fd)
	md := fd.Message()
	if fd.Cardinality() == pref.Repeated || md == nil {
		panic(errors.New("%v: field is not of non-repeated message type", fd.FullName()))
	}
	return New(md).ProtoReflect()
}

// WhichOneof reports which field in a oneof is populated, returning nil if none are populated.
// See protoreflect.Message for details.
func (m *Message) WhichOneof(od pref.OneofDescriptor) pref.FieldDescriptor {
	for i := 0; i < od.Fields().Len(); i++ {
		fd := od.Fields().Get(i)
		if m.Has(fd) {
			return fd
		}
	}
	return nil
}

// GetUnknown returns the raw unknown fields.
// See protoreflect.Message for details.
func (m *Message) GetUnknown() pref.RawFields {
	return m.unknown
}

// SetUnknown sets the raw unknown fields.
// See protoreflect.Message for details.
func (m *Message) SetUnknown(r pref.RawFields) {
	m.unknown = r
}

func (m *Message) checkField(fd pref.FieldDescriptor) {
	if fd.IsExtension() && fd.ContainingMessage().FullName() == m.desc.FullName() {
		if _, ok := fd.(pref.ExtensionType); !ok {
			panic(errors.New("%v: extension field descriptor does not implement ExtensionType", fd.FullName()))
		}
		return
	}
	if fd.Parent() == m.desc {
		return
	}
	fields := m.desc.Fields()
	index := fd.Index()
	if index >= fields.Len() || fields.Get(index) != fd {
		panic(errors.New("%v: field descriptor does not belong to this message", fd.FullName()))
	}
}

type emptyList struct {
	desc pref.FieldDescriptor
}

func (x emptyList) Len() int                { return 0 }
func (x emptyList) Get(n int) pref.Value    { panic(errors.New("out of range")) }
func (x emptyList) Set(n int, v pref.Value) { panic(errors.New("modification of immutable list")) }
func (x emptyList) Append(v pref.Value)     { panic(errors.New("modification of immutable list")) }
func (x emptyList) Truncate(n int)          { panic(errors.New("modification of immutable list")) }
func (x emptyList) NewMessage() pref.Message {
	md := x.desc.Message()
	if md == nil {
		panic(errors.New("list is not of message type"))
	}
	return New(md).ProtoReflect()
}

type dynamicList struct {
	desc pref.FieldDescriptor
	list []pref.Value
}

func (x *dynamicList) Len() int {
	return len(x.list)
}

func (x *dynamicList) Get(n int) pref.Value {
	return x.list[n]
}

func (x *dynamicList) Set(n int, v pref.Value) {
	typecheckSingular(x.desc, v)
	x.list[n] = v
}

func (x *dynamicList) Append(v pref.Value) {
	typecheckSingular(x.desc, v)
	x.list = append(x.list, v)
}

func (x *dynamicList) Truncate(n int) {
	// Zero truncated elements to avoid keeping data live.
	for i := n; i < len(x.list); i++ {
		x.list[i] = pref.Value{}
	}
	x.list = x.list[:n]
}

func (x *dynamicList) NewMessage() pref.Message {
	md := x.desc.Message()
	if md == nil {
		panic(errors.New("list is not of message type"))
	}
	return New(md).ProtoReflect()
}

type dynamicMap struct {
	desc pref.FieldDescriptor
	mapv map[interface{}]pref.Value
}

func (x *dynamicMap) Get(k pref.MapKey) pref.Value { return x.mapv[k.Interface()] }
func (x *dynamicMap) Set(k pref.MapKey, v pref.Value) {
	typecheckSingular(x.desc.MapKey(), k.Value())
	typecheckSingular(x.desc.MapValue(), v)
	x.mapv[k.Interface()] = v
}
func (x *dynamicMap) Has(k pref.MapKey) bool { return x.Get(k).IsValid() }
func (x *dynamicMap) Clear(k pref.MapKey)    { delete(x.mapv, k.Interface()) }
func (x *dynamicMap) Len() int               { return len(x.mapv) }
func (x *dynamicMap) NewMessage() pref.Message {
	md := x.desc.MapValue().Message()
	if md == nil {
		panic(errors.New("map value is not of message type"))
	}
	return New(md).ProtoReflect()
}
func (x *dynamicMap) Range(f func(pref.MapKey, pref.Value) bool) {
	for k, v := range x.mapv {
		if !f(pref.ValueOf(k).MapKey(), v) {
			return
		}
	}
}

func isSet(fd pref.FieldDescriptor, v pref.Value) bool {
	switch {
	case fd.IsMap():
		return v.Map().Len() > 0
	case fd.IsList():
		return v.List().Len() > 0
	case fd.ContainingOneof() != nil:
		return true
	case fd.Syntax() == pref.Proto3:
		switch fd.Kind() {
		case pref.BoolKind:
			return v.Bool()
		case pref.EnumKind:
			return v.Enum() != 0
		case pref.Int32Kind, pref.Sint32Kind, pref.Int64Kind, pref.Sint64Kind, pref.Sfixed32Kind, pref.Sfixed64Kind:
			return v.Int() != 0
		case pref.Uint32Kind, pref.Uint64Kind, pref.Fixed32Kind, pref.Fixed64Kind:
			return v.Uint() != 0
		case pref.FloatKind, pref.DoubleKind:
			return v.Float() != 0 || math.Signbit(v.Float())
		case pref.StringKind:
			return v.String() != ""
		case pref.BytesKind:
			return len(v.Bytes()) > 0
		}
	}
	return true
}

func typecheckSingular(fd pref.FieldDescriptor, v pref.Value) {
	vi := v.Interface()
	var ok bool
	switch fd.Kind() {
	case pref.BoolKind:
		_, ok = vi.(bool)
	case pref.EnumKind:
		// We could check against the valid set of enum values, but do not.
		_, ok = vi.(pref.EnumNumber)
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		_, ok = vi.(int32)
	case pref.Uint32Kind, pref.Fixed32Kind:
		_, ok = vi.(uint32)
	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		_, ok = vi.(int64)
	case pref.Uint64Kind, pref.Fixed64Kind:
		_, ok = vi.(uint64)
	case pref.FloatKind:
		_, ok = vi.(float32)
	case pref.DoubleKind:
		_, ok = vi.(float64)
	case pref.StringKind:
		_, ok = vi.(string)
	case pref.BytesKind:
		_, ok = vi.([]byte)
	case pref.MessageKind, pref.GroupKind:
		var m pref.Message
		m, ok = vi.(pref.Message)
		if ok && m.Descriptor().FullName() != fd.Message().FullName() {
			panic(errors.New("%v: assigning invalid message type %v", fd.FullName(), m.Descriptor().FullName()))
		}
	}
	if !ok {
		panic(errors.New("%v: assigning invalid type %T", fd.FullName(), v.Interface()))
	}
}
