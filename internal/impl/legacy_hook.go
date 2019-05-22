// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"

	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// TODO: Add a default LegacyWrapper that panics with a more helpful message?
var legacyWrapper LegacyWrapper

// RegisterLegacyWrapper registers a set of constructor functions that are
// called when a legacy enum or message is encountered that does not natively
// support the protobuf reflection APIs.
func RegisterLegacyWrapper(w LegacyWrapper) {
	legacyWrapper = w
}

// LegacyWrapper is a set of wrapper methods that wraps legacy v1 Go types
// to implement the v2 reflection APIs.
type LegacyWrapper interface {
	NewConverter(reflect.Type, pref.Kind) pvalue.Converter

	EnumOf(interface{}) pref.Enum
	EnumTypeOf(interface{}) pref.EnumType
	EnumDescriptorOf(interface{}) pref.EnumDescriptor

	MessageOf(interface{}) pref.Message
	MessageTypeOf(interface{}) pref.MessageType
	MessageDescriptorOf(interface{}) pref.MessageDescriptor

	// TODO: Remove these eventually.
	// See the TODOs in internal/impl/legacy_extension.go.
	ExtensionDescFromType(pref.ExtensionType) *piface.ExtensionDescV1
	ExtensionTypeFromDesc(*piface.ExtensionDescV1) pref.ExtensionType
}
