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
)

// TODO: How to handle Registration during the v1 to v2 switchover?
// TODO: Should a v2 ExtensionType be added to ExtensionDesc or ExtensionField?

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
		// TODO: Document that ExtendedType is unpopulated for v2 messages that
		// do not implement the v1 Message interface.

		ExtendedType  Message     // nil pointer to the type that is being extended
		ExtensionType interface{} // nil pointer to the extension type
		Field         int32       // field number
		Name          string      // fully-qualified name of extension, for text formatting
		Tag           string      // protobuf tag style
		Filename      string      // name of the file in which the extension is defined
	}

	ExtensionFields    extensionFields
	ExtensionField     extensionField
	InternalExtensions extensionSyncMap
)

// ExtensionFieldsOf returns an ExtensionFields abstraction over various
// internal representations of extension fields.
func ExtensionFieldsOf(p interface{}) ExtensionFields {
	switch p := p.(type) {
	case *map[int32]ExtensionField:
		return (*extensionMap)(p)
	case *InternalExtensions:
		return (*extensionSyncMap)(p)
	default:
		panic(fmt.Sprintf("invalid extension fields type: %T", p))
	}
}

type extensionFields interface {
	Len() int
	Has(int32) bool
	Get(int32) ExtensionField
	Set(int32, ExtensionField)
	Clear(int32)
	Range(f func(int32, ExtensionField) bool)

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
	Desc *ExtensionDesc

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
	Value interface{}

	// Raw is the raw encoded bytes for the extension field.
	// It is possible for Raw to be populated irrespective of whether the
	// other fields are populated.
	Raw []byte
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
func (m extensionSyncMap) Has(n int32) bool {
	if m.p == nil {
		return false
	}
	return m.p.m.Has(n)
}
func (m extensionSyncMap) Get(n int32) ExtensionField {
	if m.p == nil {
		return ExtensionField{}
	}
	return m.p.m.Get(n)
}
func (m *extensionSyncMap) Set(n int32, x ExtensionField) {
	if m.p == nil {
		m.p = new(struct {
			mu sync.Mutex
			m  extensionMap
		})
	}
	m.p.m.Set(n, x)
}
func (m extensionSyncMap) Clear(n int32) {
	if m.p == nil {
		return
	}
	m.p.m.Clear(n)
}
func (m extensionSyncMap) Range(f func(int32, ExtensionField) bool) {
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
func (m extensionMap) Has(n int32) bool {
	_, ok := m[n]
	return ok
}
func (m extensionMap) Get(n int32) ExtensionField {
	return m[n]
}
func (m *extensionMap) Set(n int32, x ExtensionField) {
	if *m == nil {
		*m = make(map[int32]ExtensionField)
	}
	(*m)[n] = x
}
func (m *extensionMap) Clear(n int32) {
	delete(*m, n)
}
func (m extensionMap) Range(f func(int32, ExtensionField) bool) {
	for n, x := range m {
		if !f(n, x) {
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
	globalLock.Lock()
}
