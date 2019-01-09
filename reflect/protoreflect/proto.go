// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protoreflect provides interfaces to dynamically manipulate messages.
//
// This package includes type descriptors which describe the structure of
// types defined in proto source files, and value interfaces which provide the
// ability to examine and manipulate the contents of messages.
//
// Type Descriptors
//
// The type descriptors (e.g., MessageDescriptor or EnumDescriptor)
// are immutable objects that represent protobuf type information.
// They are wrappers around the messages declared in descriptor.proto.
//
// The Message and Enum interfaces provide a Type method which returns the
// appropriate descriptor type for a value.
//
// Value Interfaces
//
// The protoreflect.Message type is a reflective view of a message instance.
// This type provides the ability to manipulate the fields of a message.
//
// To convert a proto.Message to a protoreflect.Message, use the
// former's ProtoReflect method.
package protoreflect

import (
	"regexp"
	"strings"

	"github.com/golang/protobuf/v2/internal/encoding/wire"
	"github.com/golang/protobuf/v2/internal/pragma"
)

type doNotImplement pragma.DoNotImplement

// ProtoMessage is the top-level interface that all proto messages implement.
// This is declared in the protoreflect package to avoid a cyclic dependency;
// use the proto.Message type instead, which aliases this type.
type ProtoMessage interface{ ProtoReflect() Message }

// Syntax is the language version of the proto file.
type Syntax syntax

type syntax int8 // keep exact type opaque as the int type may change

const (
	Proto2 Syntax = 2
	Proto3 Syntax = 3
)

// IsValid reports whether the syntax is valid.
func (s Syntax) IsValid() bool {
	switch s {
	case Proto2, Proto3:
		return true
	default:
		return false
	}
}

// String returns s as a proto source identifier.
func (s Syntax) String() string {
	switch s {
	case Proto2:
		return "proto2"
	case Proto3:
		return "proto3"
	default:
		return "<unknown>"
	}
}

// GoString returns s as a Go source identifier.
func (s Syntax) GoString() string {
	switch s {
	case Proto2:
		return "Proto2"
	case Proto3:
		return "Proto3"
	default:
		return "<unknown>"
	}
}

// Cardinality determines whether a field is optional, required, or repeated.
type Cardinality cardinality

type cardinality int8 // keep exact type opaque as the int type may change

// Constants as defined by the google.protobuf.Cardinality enumeration.
const (
	Optional Cardinality = 1 // appears zero or one times
	Required Cardinality = 2 // appears exactly one time; invalid with Proto3
	Repeated Cardinality = 3 // appears zero or more times
)

// IsValid reports whether the cardinality is valid.
func (c Cardinality) IsValid() bool {
	switch c {
	case Optional, Required, Repeated:
		return true
	default:
		return false
	}
}

// String returns c as a proto source identifier.
func (c Cardinality) String() string {
	switch c {
	case Optional:
		return "optional"
	case Required:
		return "required"
	case Repeated:
		return "repeated"
	default:
		return "<unknown>"
	}
}

// GoString returns c as a Go source identifier.
func (c Cardinality) GoString() string {
	switch c {
	case Optional:
		return "Optional"
	case Required:
		return "Required"
	case Repeated:
		return "Repeated"
	default:
		return "<unknown>"
	}
}

// Kind indicates the basic proto kind of a field.
type Kind kind

type kind int8 // keep exact type opaque as the int type may change

// Constants as defined by the google.protobuf.Field.Kind enumeration.
const (
	BoolKind     Kind = 8
	EnumKind     Kind = 14
	Int32Kind    Kind = 5
	Sint32Kind   Kind = 17
	Uint32Kind   Kind = 13
	Int64Kind    Kind = 3
	Sint64Kind   Kind = 18
	Uint64Kind   Kind = 4
	Sfixed32Kind Kind = 15
	Fixed32Kind  Kind = 7
	FloatKind    Kind = 2
	Sfixed64Kind Kind = 16
	Fixed64Kind  Kind = 6
	DoubleKind   Kind = 1
	StringKind   Kind = 9
	BytesKind    Kind = 12
	MessageKind  Kind = 11
	GroupKind    Kind = 10
)

// IsValid reports whether the kind is valid.
func (k Kind) IsValid() bool {
	switch k {
	case BoolKind, EnumKind,
		Int32Kind, Sint32Kind, Uint32Kind,
		Int64Kind, Sint64Kind, Uint64Kind,
		Sfixed32Kind, Fixed32Kind, FloatKind,
		Sfixed64Kind, Fixed64Kind, DoubleKind,
		StringKind, BytesKind, MessageKind, GroupKind:
		return true
	default:
		return false
	}
}

// String returns k as a proto source identifier.
func (k Kind) String() string {
	switch k {
	case BoolKind:
		return "bool"
	case EnumKind:
		return "enum"
	case Int32Kind:
		return "int32"
	case Sint32Kind:
		return "sint32"
	case Uint32Kind:
		return "uint32"
	case Int64Kind:
		return "int64"
	case Sint64Kind:
		return "sint64"
	case Uint64Kind:
		return "uint64"
	case Sfixed32Kind:
		return "sfixed32"
	case Fixed32Kind:
		return "fixed32"
	case FloatKind:
		return "float"
	case Sfixed64Kind:
		return "sfixed64"
	case Fixed64Kind:
		return "fixed64"
	case DoubleKind:
		return "double"
	case StringKind:
		return "string"
	case BytesKind:
		return "bytes"
	case MessageKind:
		return "message"
	case GroupKind:
		return "group"
	default:
		return "<unknown>"
	}
}

// GoString returns k as a Go source identifier.
func (k Kind) GoString() string {
	switch k {
	case BoolKind:
		return "BoolKind"
	case EnumKind:
		return "EnumKind"
	case Int32Kind:
		return "Int32Kind"
	case Sint32Kind:
		return "Sint32Kind"
	case Uint32Kind:
		return "Uint32Kind"
	case Int64Kind:
		return "Int64Kind"
	case Sint64Kind:
		return "Sint64Kind"
	case Uint64Kind:
		return "Uint64Kind"
	case Sfixed32Kind:
		return "Sfixed32Kind"
	case Fixed32Kind:
		return "Fixed32Kind"
	case FloatKind:
		return "FloatKind"
	case Sfixed64Kind:
		return "Sfixed64Kind"
	case Fixed64Kind:
		return "Fixed64Kind"
	case DoubleKind:
		return "DoubleKind"
	case StringKind:
		return "StringKind"
	case BytesKind:
		return "BytesKind"
	case MessageKind:
		return "MessageKind"
	case GroupKind:
		return "GroupKind"
	default:
		return "<unknown>"
	}
}

// FieldNumber is the field number in a message.
type FieldNumber = wire.Number

// FieldNumbers represent a list of field numbers.
type FieldNumbers interface {
	// Len reports the number of fields in the list.
	Len() int
	// Get returns the ith field number. It panics if out of bounds.
	Get(i int) FieldNumber
	// Has reports whether n is within the list of fields.
	Has(n FieldNumber) bool

	doNotImplement
}

// FieldRanges represent a list of field number ranges.
type FieldRanges interface {
	// Len reports the number of ranges in the list.
	Len() int
	// Get returns the ith range. It panics if out of bounds.
	Get(i int) [2]FieldNumber // start inclusive; end exclusive
	// Has reports whether n is within any of the ranges.
	Has(n FieldNumber) bool

	doNotImplement
}

// EnumNumber is the numeric value for an enum.
type EnumNumber int32

// EnumRanges represent a list of enum number ranges.
type EnumRanges interface {
	// Len reports the number of ranges in the list.
	Len() int
	// Get returns the ith range. It panics if out of bounds.
	Get(i int) [2]EnumNumber // start inclusive; end inclusive
	// Has reports whether n is within any of the ranges.
	Has(n EnumNumber) bool

	doNotImplement
}

var (
	regexName     = regexp.MustCompile(`^[_a-zA-Z][_a-zA-Z0-9]*$`)
	regexFullName = regexp.MustCompile(`^[_a-zA-Z][_a-zA-Z0-9]*(\.[_a-zA-Z][_a-zA-Z0-9]*)*$`)
)

// Name is the short name for a proto declaration. This is not the name
// as used in Go source code, which might not be identical to the proto name.
type Name string // e.g., "Kind"

// IsValid reports whether n is a syntactically valid name.
// An empty name is invalid.
func (n Name) IsValid() bool {
	return regexName.MatchString(string(n))
}

// Names represent a list of names.
type Names interface {
	// Len reports the number of names in the list.
	Len() int
	// Get returns the ith name. It panics if out of bounds.
	Get(i int) Name
	// Has reports whether s matches any names in the list.
	Has(s Name) bool

	doNotImplement
}

// FullName is a qualified name that uniquely identifies a proto declaration.
// A qualified name is the concatenation of the proto package along with the
// fully-declared name (i.e., name of parent preceding the name of the child),
// with a '.' delimiter placed between each Name.
//
// This should not have any leading or trailing dots.
type FullName string // e.g., "google.protobuf.Field.Kind"

// IsValid reports whether n is a syntactically valid full name.
// An empty full name is invalid.
func (n FullName) IsValid() bool {
	return regexFullName.MatchString(string(n))
}

// Name returns the short name, which is the last identifier segment.
// A single segment FullName is the Name itself.
func (n FullName) Name() Name {
	if i := strings.LastIndexByte(string(n), '.'); i >= 0 {
		return Name(n[i+1:])
	}
	return Name(n)
}

// Parent returns the full name with the trailing identifier removed.
// A single segment FullName has no parent.
func (n FullName) Parent() FullName {
	if i := strings.LastIndexByte(string(n), '.'); i >= 0 {
		return n[:i]
	}
	return ""
}

// Append returns the qualified name appended with the provided short name.
//
// Invariant: n == n.Parent().Append(n.Name()) // assuming n is valid
func (n FullName) Append(s Name) FullName {
	if n == "" {
		return FullName(s)
	}
	return n + "." + FullName(s)
}
