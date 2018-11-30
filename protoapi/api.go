// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protoapi contains the set of types referenced by generated messages.
//
// WARNING: This package should only ever be imported by generated messages.
// The compatibility agreement covers nothing except for functionality needed
// to keep existing generated messages operational.
package protoapi

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// TODO: How to handle Registration during the v1 to v2 switchover?

type (
	Message interface {
		Reset()
		String() string
		ProtoMessage()
	}

	ExtensionRange struct {
		Start, End int32 // both inclusive
	}

	ExtensionDesc struct {
		// Type is the descriptor type for the extension field using the v2 API.
		// If populated, the information in this field takes precedence over
		// all other fields in ExtensionDesc.
		Type protoreflect.ExtensionType

		// ExtendedType is a typed nil-pointer to the parent message type that
		// is being extended. It is possible for this to be unpopulated in v2
		// since the message may no longer implement the v1 Message interface.
		//
		// Deprecated: Use Type.ExtendedType instead.
		ExtendedType Message

		// ExtensionType is zero value of the extension type.
		//
		// For historical reasons, reflect.TypeOf(ExtensionType) and Type.GoType
		// may not be identical:
		//	* for scalars (except []byte), where ExtensionType uses *T,
		//	while Type.GoType uses T.
		//	* for repeated fields, where ExtensionType uses []T,
		//	while Type.GoType uses *[]T.
		//
		// Deprecated: Use Type.GoType instead.
		ExtensionType interface{}

		// Field is the field number of the extension.
		//
		// Deprecated: Use Type.Number instead.
		Field int32 // field number

		// Name is the fully qualified name of extension.
		//
		// Deprecated: Use Type.FullName instead.
		Name string

		// Tag is the protobuf struct tag used in the v1 API.
		//
		// Deprecated: Do not use.
		Tag string

		// Filename is the proto filename in which the extension is defined.
		//
		// Deprecated: Use Type.Parent to ascend to the top-most parent and use
		// protoreflect.FileDescriptor.Path.
		Filename string
	}

	ExtensionFields        extensionFields
	ExtensionField         extensionField
	XXX_InternalExtensions extensionSyncMap
)

// ExtensionFieldsOf returns an ExtensionFields abstraction over various
// internal representations of extension fields.
func ExtensionFieldsOf(p interface{}) ExtensionFields {
	switch p := p.(type) {
	case *map[int32]ExtensionField:
		return (*extensionMap)(p)
	case *XXX_InternalExtensions:
		return (*extensionSyncMap)(p)
	default:
		panic(fmt.Sprintf("invalid extension fields type: %T", p))
	}
}

type extensionFields interface {
	Len() int
	Has(protoreflect.FieldNumber) bool
	Get(protoreflect.FieldNumber) ExtensionField
	Set(protoreflect.FieldNumber, ExtensionField)
	Clear(protoreflect.FieldNumber)
	Range(f func(protoreflect.FieldNumber, ExtensionField) bool)

	// HasInit and Locker are used by v1 GetExtension to provide
	// an artificial degree of concurrent safety.
	HasInit() bool
	sync.Locker
}

type extensionField struct {
	// When an extension is stored in a message using SetExtension
	// only desc and value are set. When the message is marshaled
	// Raw will be set to the encoded form of the message.
	//
	// When a message is unmarshaled and contains extensions, each
	// extension will have only Raw set. When such an extension is
	// accessed using GetExtension (or GetExtensions) desc and value
	// will be set.
	Desc *ExtensionDesc // TODO: switch to protoreflect.ExtensionType

	// Value is a concrete value for the extension field. Let the type of
	// Desc.ExtensionType be the "API type" and the type of Value be the
	// "storage type". The API type and storage type are the same except:
	//	* for scalars (except []byte), where the API type uses *T,
	//	while the storage type uses T.
	//	* for repeated fields, where the API type uses []T,
	//	while the storage type uses *[]T.
	//
	// The reason for the divergence is so that the storage type more naturally
	// matches what is expected of when retrieving the values through the
	// protobuf reflection APIs.
	//
	// The Value may only be populated if Desc is also populated.
	Value interface{} // TODO: switch to protoreflect.Value

	// Raw is the raw encoded bytes for the extension field.
	// It is possible for Raw to be populated irrespective of whether the
	// other fields are populated.
	Raw []byte // TODO: switch to protoreflect.RawFields
}

type extensionSyncMap struct {
	p *struct {
		mu sync.Mutex
		m  extensionMap
	}
}

func (m extensionSyncMap) Len() int {
	if m.p == nil {
		return 0
	}
	return m.p.m.Len()
}
func (m extensionSyncMap) Has(n protoreflect.FieldNumber) bool {
	if m.p == nil {
		return false
	}
	return m.p.m.Has(n)
}
func (m extensionSyncMap) Get(n protoreflect.FieldNumber) ExtensionField {
	if m.p == nil {
		return ExtensionField{}
	}
	return m.p.m.Get(n)
}
func (m *extensionSyncMap) Set(n protoreflect.FieldNumber, x ExtensionField) {
	if m.p == nil {
		m.p = new(struct {
			mu sync.Mutex
			m  extensionMap
		})
	}
	m.p.m.Set(n, x)
}
func (m extensionSyncMap) Clear(n protoreflect.FieldNumber) {
	if m.p == nil {
		return
	}
	m.p.m.Clear(n)
}
func (m extensionSyncMap) Range(f func(protoreflect.FieldNumber, ExtensionField) bool) {
	if m.p == nil {
		return
	}
	m.p.m.Range(f)
}

func (m extensionSyncMap) HasInit() bool {
	return m.p != nil
}
func (m extensionSyncMap) Lock() {
	m.p.mu.Lock()
}
func (m extensionSyncMap) Unlock() {
	m.p.mu.Unlock()
}

type extensionMap map[int32]ExtensionField

func (m extensionMap) Len() int {
	return len(m)
}
func (m extensionMap) Has(n protoreflect.FieldNumber) bool {
	_, ok := m[int32(n)]
	return ok
}
func (m extensionMap) Get(n protoreflect.FieldNumber) ExtensionField {
	return m[int32(n)]
}
func (m *extensionMap) Set(n protoreflect.FieldNumber, x ExtensionField) {
	if *m == nil {
		*m = make(map[int32]ExtensionField)
	}
	(*m)[int32(n)] = x
}
func (m *extensionMap) Clear(n protoreflect.FieldNumber) {
	delete(*m, int32(n))
}
func (m extensionMap) Range(f func(protoreflect.FieldNumber, ExtensionField) bool) {
	for n, x := range m {
		if !f(protoreflect.FieldNumber(n), x) {
			return
		}
	}
}

var globalLock sync.Mutex

func (m extensionMap) HasInit() bool {
	return m != nil
}
func (m extensionMap) Lock() {
	if !m.HasInit() {
		panic("cannot lock an uninitialized map")
	}
	globalLock.Lock()
}
func (m extensionMap) Unlock() {
	globalLock.Unlock()
}
