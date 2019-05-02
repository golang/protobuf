// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package irregular

import (
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

func (m *message) KnownFields() pref.KnownFields     { return (*known)(m) }
func (m *message) UnknownFields() pref.UnknownFields { return (*unknown)(m) }

type known IrregularMessage

func (m *known) Len() int {
	if m.set {
		return 1
	}
	return 0
}

func (m *known) Has(num pref.FieldNumber) bool {
	switch num {
	case fieldS:
		return m.set
	}
	return false
}

func (m *known) Get(num pref.FieldNumber) pref.Value {
	switch num {
	case fieldS:
		return pref.ValueOf(m.value)
	}
	return pref.Value{}
}

func (m *known) Set(num pref.FieldNumber, v pref.Value) {
	switch num {
	case fieldS:
		m.value = v.String()
	default:
		panic("unknown field")
	}
}

func (m *known) Clear(num pref.FieldNumber) {
	switch num {
	case fieldS:
		m.value = ""
		m.set = false
	default:
		panic("unknown field")
	}
}

func (m *known) WhichOneof(name pref.Name) pref.FieldNumber {
	return 0
}

func (m *known) Range(f func(pref.FieldNumber, pref.Value) bool) {
	if m.set {
		f(fieldS, pref.ValueOf(m.value))
	}
}

func (m *known) NewMessage(num pref.FieldNumber) pref.Message {
	panic("not a message field")
}

func (m *known) ExtensionTypes() pref.ExtensionFieldTypes {
	return (*exttypes)(m)
}

const fieldS = pref.FieldNumber(1)

type unknown IrregularMessage

func (m *unknown) Len() int                                          { return 0 }
func (m *unknown) Get(pref.FieldNumber) pref.RawFields               { return nil }
func (m *unknown) Set(pref.FieldNumber, pref.RawFields)              {}
func (m *unknown) Range(func(pref.FieldNumber, pref.RawFields) bool) {}
func (m *unknown) IsSupported() bool                                 { return false }

type exttypes IrregularMessage

func (m *exttypes) Len() int                                     { return 0 }
func (m *exttypes) Register(pref.ExtensionType)                  { panic("not extendable") }
func (m *exttypes) Remove(pref.ExtensionType)                    {}
func (m *exttypes) ByNumber(pref.FieldNumber) pref.ExtensionType { return nil }
func (m *exttypes) ByName(pref.FullName) pref.ExtensionType      { return nil }
func (m *exttypes) Range(func(pref.ExtensionType) bool)          {}
