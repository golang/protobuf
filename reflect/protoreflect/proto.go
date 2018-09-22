// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protoreflect provides interfaces to dynamically manipulate messages.
//
// Every Go type that represents a protocol buffer type must implement the
// proto.Message or proto.Enum interface, which has a ProtoReflect method that
// returns a protoreflect.Message or protoreflect.Enum.
// These interfaces provide programs with the ability to manipulate a
// message value or to explore the protobuf type descriptor.
//
// The defined interfaces can be categorized as either a type descriptor
// or a value interface.
//
// Type Descriptors
//
// The type descriptors (e.g., MessageDescriptor or EnumDescriptor)
// are immutable objects that represent protobuf type information.
// They are wrappers around the messages declared in descriptor.proto.
//
// Value Interfaces
//
// The value is a reflective interface (e.g., Message) for a message instance.
// The Message interface provides the ability to manipulate the fields of a
// message using getters and setters.
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

// ProtoEnum is the top-level interface that all proto enums implement.
// This is declared in the protoreflect package to avoid a cyclic dependency;
// use the proto.Enum type instead, which aliases this type.
type ProtoEnum interface{ ProtoReflect() Enum }

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
