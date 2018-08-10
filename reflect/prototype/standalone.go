// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"google.golang.org/proto/reflect/protoreflect"
)

// TODO: Should the constructors take in a value rather than a pointer?
// TODO: Support initializing StandaloneMessage from a google.protobuf.Type?

// StandaloneMessage is a constructor for a protoreflect.MessageDescriptor
// that does not have a parent and has no child declarations.
type StandaloneMessage struct {
	Syntax          protoreflect.Syntax
	FullName        protoreflect.FullName
	IsMapEntry      bool
	Fields          []Field
	Oneofs          []Oneof
	ExtensionRanges [][2]protoreflect.FieldNumber

	fields fieldsMeta
	oneofs oneofsMeta
	nums   numbersMeta
}

// NewMessage creates a new protoreflect.MessageDescriptor.
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields.
func NewMessage(t *StandaloneMessage) (protoreflect.MessageDescriptor, error) {
	mt := standaloneMessage{t}
	if err := validateMessage(mt); err != nil {
		return nil, err
	}
	return mt, nil
}

// StandaloneEnum is a constructor for a protoreflect.EnumDescriptor
// that does not have a parent.
type StandaloneEnum struct {
	Syntax   protoreflect.Syntax
	FullName protoreflect.FullName
	Values   []EnumValue

	vals enumValuesMeta
}

// NewEnum creates a new protoreflect.EnumDescriptor.
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields.
func NewEnum(t *StandaloneEnum) (protoreflect.EnumDescriptor, error) {
	et := standaloneEnum{t}
	if err := validateEnum(et); err != nil {
		return nil, err
	}
	return et, nil
}

// StandaloneExtension is a constructor for a protoreflect.ExtensionDescriptor
// that does not have a parent.
type StandaloneExtension struct {
	Syntax       protoreflect.Syntax
	FullName     protoreflect.FullName
	Number       protoreflect.FieldNumber
	Cardinality  protoreflect.Cardinality
	Kind         protoreflect.Kind
	IsPacked     bool
	Default      protoreflect.Value
	MessageType  protoreflect.MessageDescriptor
	EnumType     protoreflect.EnumDescriptor
	ExtendedType protoreflect.MessageDescriptor
}

// NewExtension creates a new protoreflect.ExtensionDescriptor.
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields.
func NewExtension(t *StandaloneExtension) (protoreflect.ExtensionDescriptor, error) {
	xt := standaloneExtension{t}
	if err := validateExtension(xt); err != nil {
		return nil, err
	}
	return xt, nil
}
