// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// TODO: Remove this file.

// TODO: Remove this.
func (m *messageReflectWrapper) Type() pref.MessageType {
	return m.mi.PBType
}

// TODO: Remove this.
func (m *messageReflectWrapper) KnownFields() pref.KnownFields {
	m.mi.init()
	return (*knownFields)(m)
}

// TODO: Remove this.
func (m *messageReflectWrapper) UnknownFields() pref.UnknownFields {
	m.mi.init()
	return m.mi.unknownFields((*messageDataType)(m))
}

// TODO: Remove this.
type knownFields messageDataType

func (fs *knownFields) Len() (cnt int) {
	for _, fi := range fs.mi.fields {
		if fi.has(fs.p) {
			cnt++
		}
	}
	return cnt + fs.extensionFields().Len()
}
func (fs *knownFields) Has(n pref.FieldNumber) bool {
	if fi := fs.mi.fields[n]; fi != nil {
		return fi.has(fs.p)
	}
	return fs.extensionFields().Has(n)
}
func (fs *knownFields) Get(n pref.FieldNumber) pref.Value {
	if fi := fs.mi.fields[n]; fi != nil {
		if !fi.has(fs.p) && isComposite(fi.fieldDesc) {
			if fi.newMessage != nil {
				return pref.Value{}
			}
			if !fs.p.IsNil() {
				return fi.mutable(fs.p)
			}
		}
		return fi.get(fs.p)
	}
	return fs.extensionFields().Get(n)
}
func (fs *knownFields) Set(n pref.FieldNumber, v pref.Value) {
	if fi := fs.mi.fields[n]; fi != nil {
		fi.set(fs.p, v)
		return
	}
	if fs.mi.PBType.Descriptor().ExtensionRanges().Has(n) {
		fs.extensionFields().Set(n, v)
		return
	}
	panic(fmt.Sprintf("invalid field: %d", n))
}
func (fs *knownFields) Clear(n pref.FieldNumber) {
	if fi := fs.mi.fields[n]; fi != nil {
		fi.clear(fs.p)
		return
	}
	if fs.mi.PBType.Descriptor().ExtensionRanges().Has(n) {
		fs.extensionFields().Clear(n)
		return
	}
}
func (fs *knownFields) WhichOneof(s pref.Name) pref.FieldNumber {
	if oi := fs.mi.oneofs[s]; oi != nil {
		return oi.which(fs.p)
	}
	return 0
}
func (fs *knownFields) Range(f func(pref.FieldNumber, pref.Value) bool) {
	for n, fi := range fs.mi.fields {
		if fi.has(fs.p) {
			if !f(n, fi.get(fs.p)) {
				return
			}
		}
	}
	fs.extensionFields().Range(f)
}
func (fs *knownFields) NewMessage(n pref.FieldNumber) pref.Message {
	if fi := fs.mi.fields[n]; fi != nil {
		return fi.newMessage()
	}
	if fs.mi.PBType.Descriptor().ExtensionRanges().Has(n) {
		return fs.extensionFields().NewMessage(n)
	}
	panic(fmt.Sprintf("invalid field: %d", n))
}
func (fs *knownFields) ExtensionTypes() pref.ExtensionFieldTypes {
	return fs.extensionFields().ExtensionTypes()
}
func (fs *knownFields) extensionFields() pref.KnownFields {
	return fs.mi.extensionFields((*messageDataType)(fs))
}

// TODO: Remove this.
type emptyUnknownFields struct{}

func (emptyUnknownFields) Len() int                                          { return 0 }
func (emptyUnknownFields) Get(pref.FieldNumber) pref.RawFields               { return nil }
func (emptyUnknownFields) Set(pref.FieldNumber, pref.RawFields)              { return } // noop
func (emptyUnknownFields) Range(func(pref.FieldNumber, pref.RawFields) bool) { return }
func (emptyUnknownFields) IsSupported() bool                                 { return false }

// TODO: Remove this.
type emptyExtensionFields struct{}

func (emptyExtensionFields) Len() int                                      { return 0 }
func (emptyExtensionFields) Has(pref.FieldNumber) bool                     { return false }
func (emptyExtensionFields) Get(pref.FieldNumber) pref.Value               { return pref.Value{} }
func (emptyExtensionFields) Set(pref.FieldNumber, pref.Value)              { panic("extensions not supported") }
func (emptyExtensionFields) Clear(pref.FieldNumber)                        { return } // noop
func (emptyExtensionFields) WhichOneof(pref.Name) pref.FieldNumber         { return 0 }
func (emptyExtensionFields) Range(func(pref.FieldNumber, pref.Value) bool) { return }
func (emptyExtensionFields) NewMessage(pref.FieldNumber) pref.Message {
	panic("extensions not supported")
}
func (emptyExtensionFields) ExtensionTypes() pref.ExtensionFieldTypes { return emptyExtensionTypes{} }

// TODO: Remove this.
type emptyExtensionTypes struct{}

func (emptyExtensionTypes) Len() int                                     { return 0 }
func (emptyExtensionTypes) Register(pref.ExtensionType)                  { panic("extensions not supported") }
func (emptyExtensionTypes) Remove(pref.ExtensionType)                    { return } // noop
func (emptyExtensionTypes) ByNumber(pref.FieldNumber) pref.ExtensionType { return nil }
func (emptyExtensionTypes) ByName(pref.FullName) pref.ExtensionType      { return nil }
func (emptyExtensionTypes) Range(func(pref.ExtensionType) bool)          { return }
