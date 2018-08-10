// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prototype provides builders to construct protobuf types that
// implement the interfaces defined in the protoreflect package.
//
// Protobuf types can either be constructed as standalone types
// (e.g., StandaloneMessage), or together as a batch of types in a single
// proto file (e.g., File). When creating standalone types, additional
// information must be provided such as the full type name and the proto syntax.
// When creating an entire file, the syntax and full name is derived from
// the parent type.
package prototype

import (
	"google.golang.org/proto/reflect/protoreflect"
)

// Every struct has a "meta" struct embedded within it as a pointer.
// The meta type provides additional data structures for efficient lookup on
// certain methods (e.g., ByName) or derived information that can be
// derived from the parent (e.g., FullName). The meta type is lazily allocated
// and initialized. This architectural approach keeps the literal representation
// smaller, which then keeps the generated code size smaller.

// TODO: Support initializing File from a google.protobuf.FileDescriptor?

// TODO: Instead of a top-down construction approach where internal references
// to message types use placeholder types, we could add a Reference method
// on Message and Enum that creates a MessageDescriptor or EnumDescriptor
// reference that only becomes valid after NewFile.
// However, that API approach is more error prone, as it causes more memory
// aliasing and provides more opportunity for misuse.
// Also, it requires that NewFile at least eagerly initialize all
// messages and enums list types. We can always add that API in the future.

// File is a constructor for protoreflect.FileDescriptor.
type File struct {
	Syntax  protoreflect.Syntax
	Path    string
	Package protoreflect.FullName
	Imports []protoreflect.FileImport

	Messages   []Message
	Enums      []Enum
	Extensions []Extension
	Services   []Service

	*fileMeta
}

// NewFile creates a new protoreflect.FileDescriptor from the provided value.
// The file must represent a valid proto file according to protobuf semantics.
//
// Fields that reference an enum or message that is being declared within the
// same File can be represented using a placeholder descriptor. NewFile will
// automatically resolve the placeholder to point to the concrete type.
//
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields. The input must not contain slices that are
// sub-slices of each other.
func NewFile(t *File) (protoreflect.FileDescriptor, error) {
	// TODO: Provide an unverified make that avoids validating the file.
	// This is useful for generated code since we know that protoc-gen-go
	// already validated the protobuf types.
	ft := newFile(t)
	if err := validateFile(ft); err != nil {
		return nil, err
	}
	return ft, nil
}

// Message is a constructor for protoreflect.MessageDescriptor.
type Message struct {
	Name            protoreflect.Name
	IsMapEntry      bool
	Fields          []Field
	Oneofs          []Oneof
	ExtensionRanges [][2]protoreflect.FieldNumber

	Messages   []Message
	Enums      []Enum
	Extensions []Extension

	*messageMeta
}

// Field is a constructor for protoreflect.FieldDescriptor.
type Field struct {
	Name        protoreflect.Name
	Number      protoreflect.FieldNumber
	Cardinality protoreflect.Cardinality
	Kind        protoreflect.Kind
	JSONName    string
	IsPacked    bool
	IsWeak      bool
	Default     protoreflect.Value
	OneofName   protoreflect.Name
	MessageType protoreflect.MessageDescriptor
	EnumType    protoreflect.EnumDescriptor

	*fieldMeta
}

// Oneof is a constructor for protoreflect.OneofDescriptor.
type Oneof struct {
	Name protoreflect.Name

	*oneofMeta
}

// Extension is a constructor for protoreflect.ExtensionDescriptor.
type Extension struct {
	Name         protoreflect.Name
	Number       protoreflect.FieldNumber
	Cardinality  protoreflect.Cardinality
	Kind         protoreflect.Kind
	IsPacked     bool
	Default      protoreflect.Value
	MessageType  protoreflect.MessageDescriptor
	EnumType     protoreflect.EnumDescriptor
	ExtendedType protoreflect.MessageDescriptor

	*extensionMeta
}

// Enum is a constructor for protoreflect.EnumDescriptor.
type Enum struct {
	Name   protoreflect.Name
	Values []EnumValue

	*enumMeta
}

// EnumValue is a constructor for protoreflect.EnumValueDescriptor.
type EnumValue struct {
	Name   protoreflect.Name
	Number protoreflect.EnumNumber

	*enumValueMeta
}

// Service is a constructor for protoreflect.ServiceDescriptor.
type Service struct {
	Name    protoreflect.Name
	Methods []Method

	*serviceMeta
}

// Method is a constructor for protoreflect.MethodDescriptor.
type Method struct {
	Name              protoreflect.Name
	InputType         protoreflect.MessageDescriptor
	OutputType        protoreflect.MessageDescriptor
	IsStreamingClient bool
	IsStreamingServer bool

	*methodMeta
}
