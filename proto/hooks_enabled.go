// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !use_golang_protobuf_v1

package proto

import (
	"github.com/golang/protobuf/internal/proto"
)

var (
	// Hooks for lib.go.
	setDefaultsAlt = proto.SetDefaults

	// Hooks for discard.go.
	discardUnknownAlt = proto.DiscardUnknown

	// Hooks for registry.go.
	registerEnumAlt         = proto.RegisterEnum
	enumValueMapAlt         = proto.EnumValueMap
	registerTypeAlt         = proto.RegisterType
	registerMapTypeAlt      = proto.RegisterMapType
	messageNameAlt          = proto.MessageName
	messageTypeAlt          = proto.MessageType
	registerFileAlt         = proto.RegisterFile
	fileDescriptorAlt       = proto.FileDescriptor
	registerExtensionAlt    = proto.RegisterExtension
	registeredExtensionsAlt = proto.RegisteredExtensions

	// Hooks for text.go
	marshalTextAlt       = proto.MarshalText
	marshalTextStringAlt = proto.MarshalTextString
	compactTextAlt       = proto.CompactText
	compactTextStringAlt = proto.CompactTextString

	// Hooks for text_parser.go
	unmarshalTextAlt = proto.UnmarshalText
)

// Hooks for lib.go.
type RequiredNotSetError = proto.RequiredNotSetError

// Hooks for text.go
type TextMarshaler = proto.TextMarshaler
