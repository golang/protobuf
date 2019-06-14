// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package irregular

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protodesc"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type IrregularMessage struct {
	set   bool
	value string
}

func (m *IrregularMessage) ProtoReflect() pref.Message { return (*message)(m) }

type message IrregularMessage

func (m *message) Descriptor() pref.MessageDescriptor { return descriptor.Messages().Get(0) }
func (m *message) Type() pref.MessageType             { return nil }
func (m *message) KnownFields() pref.KnownFields      { return (*known)(m) }
func (m *message) UnknownFields() pref.UnknownFields  { return (*unknown)(m) }
func (m *message) New() pref.Message                  { return &message{} }
func (m *message) Interface() pref.ProtoMessage       { return (*IrregularMessage)(m) }

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

const fieldS = pref.FieldNumber(1)

var descriptor = func() pref.FileDescriptor {
	p := &descriptorpb.FileDescriptorProto{}
	if err := prototext.Unmarshal([]byte(descriptorText), p); err != nil {
		panic(err)
	}
	file, err := protodesc.NewFile(p, nil)
	if err != nil {
		panic(err)
	}
	return file
}()

func file_irregular_irregular_proto_init() { _ = descriptor }

const descriptorText = `
  name: "internal/testprotos/irregular/irregular.proto"
  package: "goproto.proto.thirdparty"
  message_type {
    name: "IrregularMessage"
    field {
      name: "s"
      number: 1
      label: LABEL_OPTIONAL
      type: TYPE_STRING
      json_name: "s"
    }
  }
  options {
    go_package: "google.golang.org/protobuf/internal/testprotos/irregular"
  }
`
