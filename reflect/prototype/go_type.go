// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"
	"reflect"
	"sync"

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

// TODO: add extension support.
