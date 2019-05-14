// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protoiface contains types referenced by generated messages.
//
// WARNING: This package should only ever be imported by generated messages.
// The compatibility agreement covers nothing except for functionality needed
// to keep existing generated messages operational.
package protoiface

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

type MessageV1 interface {
	Reset()
	String() string
	ProtoMessage()
}

type ExtensionRangeV1 struct {
	Start, End int32 // both inclusive
}

type ExtensionDescV1 struct {
	// Type is the descriptor type for the extension field using the v2 API.
	// If populated, the information in this field takes precedence over
	// all other fields in ExtensionDescV1.
	//
	// TODO: Delete this and make this whole struct implement ExtensionDescV1.
	Type protoreflect.ExtensionType

	// ExtendedType is a typed nil-pointer to the parent message type that
	// is being extended. It is possible for this to be unpopulated in v2
	// since the message may no longer implement the MessageV1 interface.
	//
	// Deprecated: Use Type.ExtendedType instead.
	ExtendedType MessageV1

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
	Field int32

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
