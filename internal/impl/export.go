// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
)

// Export is a zero-length named type that exists only to export a set of
// functions that we do not want to appear in godoc.
type Export struct{}

// EnumOf returns the protoreflect.Enum interface over e.
// If e already implements proto.Enum, then it directly calls the
// ProtoReflect method, otherwise it wraps the v1 enum to implement
// the v2 reflective interface.
func (Export) EnumOf(e interface{}) pref.Enum {
	if ev, ok := e.(pref.Enum); ok {
		return ev
	}
	return legacyWrapper.EnumOf(e)
}

// EnumTypeOf returns the protoreflect.EnumType for e.
// If e already implements proto.Enum, then it obtains the type by directly
// calling the ProtoReflect.Type method, otherwise it derives an enum type
// from the v1 named int32 type.
func (Export) EnumTypeOf(e interface{}) pref.EnumType {
	if ev, ok := e.(pref.Enum); ok {
		return ev.Type()
	}
	return legacyWrapper.EnumTypeOf(e)
}

// MessageOf returns the protoreflect.Message interface over m.
// If m already implements proto.Message, then it directly calls the
// ProtoReflect method, otherwise it wraps the v1 message to implement
// the v2 reflective interface.
func (Export) MessageOf(m interface{}) pref.Message {
	if mv, ok := m.(pref.ProtoMessage); ok {
		return mv.ProtoReflect()
	}
	return legacyWrapper.MessageOf(m)
}

// MessageTypeOf returns the protoreflect.MessageType for m.
// If m already implements proto.Message, then it obtains the type by directly
// calling the ProtoReflect.Type method, otherwise it derives a message type
// from the v1 message struct.
func (Export) MessageTypeOf(m interface{}) pref.MessageType {
	if mv, ok := m.(pref.ProtoMessage); ok {
		return mv.ProtoReflect().Type()
	}
	return legacyWrapper.MessageTypeOf(m)
}

// ExtensionTypeOf returns a protoreflect.ExtensionType where the type of the
// field is t. The type t must be provided if the field is an enum or message.
// If t already implements proto.Enum or proto.Message, then this returns
// an extension type by directly calling prototype.GoExtension.
// Otherwise, it derives an extension type by wrapping the enum or message
// using EnumOf or MessageOf.
func (Export) ExtensionTypeOf(d pref.ExtensionDescriptor, t interface{}) pref.ExtensionType {
	switch t := t.(type) {
	case nil:
		return ptype.GoExtension(d, nil, nil)
	case pref.Enum:
		return ptype.GoExtension(d, t.Type(), nil)
	case pref.ProtoMessage:
		return ptype.GoExtension(d, nil, t.ProtoReflect().Type())
	}
	return legacyWrapper.ExtensionTypeOf(d, t)
}
