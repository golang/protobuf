// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// TODO: Should the constructors take in a value rather than a pointer?
// TODO: Support initializing StandaloneMessage from a google.protobuf.Type?

// StandaloneMessage is a constructor for a protoreflect.MessageDescriptor
// that does not have a parent and has no child declarations.
type StandaloneMessage struct {
	Syntax                protoreflect.Syntax
	FullName              protoreflect.FullName
	Fields                []Field
	Oneofs                []Oneof
	ReservedNames         []protoreflect.Name
	ReservedRanges        [][2]protoreflect.FieldNumber
	ExtensionRanges       [][2]protoreflect.FieldNumber
	ExtensionRangeOptions []protoreflect.OptionsMessage
	Options               protoreflect.OptionsMessage
	IsMapEntry            bool

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

// NewMessages creates a set of new protoreflect.MessageDescriptors.
//
// This constructor permits the creation of cyclic message types that depend
// on each other. For example, message A may have a field of type message B,
// where message B may have a field of type message A. In such a case,
// a placeholder message is used for these cyclic references.
//
// The caller must relinquish full ownership of the input ts and must not
// access or mutate any fields.
func NewMessages(ts []*StandaloneMessage) ([]protoreflect.MessageDescriptor, error) {
	// TODO: Should this be []*T or []T?
	// TODO: NewMessages is a superset of NewMessage. Do we need NewMessage?
	ms := map[protoreflect.FullName]protoreflect.MessageDescriptor{}
	for _, t := range ts {
		if _, ok := ms[t.FullName]; ok {
			return nil, errors.New("duplicate message %v", t.FullName)
		}
		ms[t.FullName] = standaloneMessage{t}
	}

	var mts []protoreflect.MessageDescriptor
	for _, t := range ts {
		for i, f := range t.Fields {
			// Resolve placeholder messages with a concrete standalone message.
			// If this fails, validateMessage will complain about it later.
			if f.MessageType != nil && f.MessageType.IsPlaceholder() && !f.IsWeak {
				if m, ok := ms[f.MessageType.FullName()]; ok {
					t.Fields[i].MessageType = m
				}
			}
		}
		mt := standaloneMessage{t}
		if err := validateMessage(mt); err != nil {
			return nil, err
		}
		mts = append(mts, mt)
	}
	return mts, nil
}

// StandaloneEnum is a constructor for a protoreflect.EnumDescriptor
// that does not have a parent.
type StandaloneEnum struct {
	Syntax         protoreflect.Syntax
	FullName       protoreflect.FullName
	Values         []EnumValue
	ReservedNames  []protoreflect.Name
	ReservedRanges [][2]protoreflect.EnumNumber
	Options        protoreflect.OptionsMessage

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
	FullName     protoreflect.FullName
	Number       protoreflect.FieldNumber
	Cardinality  protoreflect.Cardinality
	Kind         protoreflect.Kind
	Default      protoreflect.Value
	MessageType  protoreflect.MessageDescriptor
	EnumType     protoreflect.EnumDescriptor
	ExtendedType protoreflect.MessageDescriptor
	Options      protoreflect.OptionsMessage
	IsPacked     OptionalBool

	dv defaultValue
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
