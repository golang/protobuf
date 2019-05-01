// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto

import (
	"github.com/golang/protobuf/v2/internal/encoding/wire"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/pragma"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"
	"github.com/golang/protobuf/v2/runtime/protoiface"
)

// UnmarshalOptions configures the unmarshaler.
//
// Example usage:
//   err := UnmarshalOptions{DiscardUnknown: true}.Unmarshal(b, m)
type UnmarshalOptions struct {
	// AllowPartial accepts input for messages that will result in missing
	// required fields. If AllowPartial is false (the default), Unmarshal will
	// return an error if there are any missing required fields.
	AllowPartial bool

	// If DiscardUnknown is set, unknown fields are ignored.
	DiscardUnknown bool

	// Resolver is used for looking up types when unmarshaling extension fields.
	// If nil, this defaults to using protoregistry.GlobalTypes.
	Resolver *protoregistry.Types

	pragma.NoUnkeyedLiterals
}

var _ = protoiface.UnmarshalOptions(UnmarshalOptions{})

// Unmarshal parses the wire-format message in b and places the result in m.
func Unmarshal(b []byte, m Message) error {
	return UnmarshalOptions{}.Unmarshal(b, m)
}

// Unmarshal parses the wire-format message in b and places the result in m.
func (o UnmarshalOptions) Unmarshal(b []byte, m Message) error {
	if o.Resolver == nil {
		o.Resolver = protoregistry.GlobalTypes
	}

	// TODO: Reset m?
	err := o.unmarshalMessageFast(b, m)
	if err == errInternalNoFast {
		err = o.unmarshalMessage(b, m.ProtoReflect())
	}
	var nerr errors.NonFatal
	if !nerr.Merge(err) {
		return err
	}
	if !o.AllowPartial {
		nerr.Merge(IsInitialized(m))
	}
	return nerr.E
}

func (o UnmarshalOptions) unmarshalMessageFast(b []byte, m Message) error {
	methods := protoMethods(m)
	if methods == nil || methods.Unmarshal == nil {
		return errInternalNoFast
	}
	return methods.Unmarshal(b, m, protoiface.UnmarshalOptions(o))
}

func (o UnmarshalOptions) unmarshalMessage(b []byte, m protoreflect.Message) error {
	messageDesc := m.Descriptor()
	fieldDescs := messageDesc.Fields()
	knownFields := m.KnownFields()
	unknownFields := m.UnknownFields()
	var nerr errors.NonFatal
	for len(b) > 0 {
		// Parse the tag (field number and wire type).
		num, wtyp, tagLen := wire.ConsumeTag(b)
		if tagLen < 0 {
			return wire.ParseError(tagLen)
		}

		// Parse the field value.
		fieldDesc := fieldDescs.ByNumber(num)
		if fieldDesc == nil {
			extType := knownFields.ExtensionTypes().ByNumber(num)
			if extType == nil && messageDesc.ExtensionRanges().Has(num) {
				var err error
				extType, err = o.Resolver.FindExtensionByNumber(messageDesc.FullName(), num)
				if err != nil && err != protoregistry.NotFound {
					return err
				}
				if extType != nil {
					knownFields.ExtensionTypes().Register(extType)
				}
			}
			if extType != nil {
				fieldDesc = extType.Descriptor()
			}
		}
		var err error
		var valLen int
		switch {
		case fieldDesc == nil:
			err = errUnknown
		case fieldDesc.Cardinality() != protoreflect.Repeated:
			valLen, err = o.unmarshalScalarField(b[tagLen:], wtyp, num, knownFields, fieldDesc)
		case !fieldDesc.IsMap():
			valLen, err = o.unmarshalList(b[tagLen:], wtyp, num, knownFields.Get(num).List(), fieldDesc)
		default:
			valLen, err = o.unmarshalMap(b[tagLen:], wtyp, num, knownFields.Get(num).Map(), fieldDesc)
		}
		if err == errUnknown {
			valLen = wire.ConsumeFieldValue(num, wtyp, b[tagLen:])
			if valLen < 0 {
				return wire.ParseError(valLen)
			}
			unknownFields.Set(num, append(unknownFields.Get(num), b[:tagLen+valLen]...))
		} else if !nerr.Merge(err) {
			return err
		}
		b = b[tagLen+valLen:]
	}
	return nerr.E
}

func (o UnmarshalOptions) unmarshalScalarField(b []byte, wtyp wire.Type, num wire.Number, knownFields protoreflect.KnownFields, field protoreflect.FieldDescriptor) (n int, err error) {
	var nerr errors.NonFatal
	v, n, err := o.unmarshalScalar(b, wtyp, num, field)
	if !nerr.Merge(err) {
		return 0, err
	}
	switch field.Kind() {
	case protoreflect.GroupKind, protoreflect.MessageKind:
		// Messages are merged with any existing message value,
		// unless the message is part of a oneof.
		//
		// TODO: C++ merges into oneofs, while v1 does not.
		// Evaluate which behavior to pick.
		var m protoreflect.Message
		if knownFields.Has(num) && field.Oneof() == nil {
			m = knownFields.Get(num).Message()
		} else {
			m = knownFields.NewMessage(num)
			knownFields.Set(num, protoreflect.ValueOf(m))
		}
		// Pass up errors (fatal and otherwise).
		if err := o.unmarshalMessage(v.Bytes(), m); !nerr.Merge(err) {
			return n, err
		}
	default:
		// Non-message scalars replace the previous value.
		knownFields.Set(num, v)
	}
	return n, nerr.E
}

func (o UnmarshalOptions) unmarshalMap(b []byte, wtyp wire.Type, num wire.Number, mapv protoreflect.Map, field protoreflect.FieldDescriptor) (n int, err error) {
	if wtyp != wire.BytesType {
		return 0, errUnknown
	}
	b, n = wire.ConsumeBytes(b)
	if n < 0 {
		return 0, wire.ParseError(n)
	}
	var (
		keyField = field.Message().Fields().ByNumber(1)
		valField = field.Message().Fields().ByNumber(2)
		key      protoreflect.Value
		val      protoreflect.Value
		haveKey  bool
		haveVal  bool
	)
	switch valField.Kind() {
	case protoreflect.GroupKind, protoreflect.MessageKind:
		val = protoreflect.ValueOf(mapv.NewMessage())
	}
	// Map entries are represented as a two-element message with fields
	// containing the key and value.
	var nerr errors.NonFatal
	for len(b) > 0 {
		num, wtyp, n := wire.ConsumeTag(b)
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		b = b[n:]
		err = errUnknown
		switch num {
		case 1:
			key, n, err = o.unmarshalScalar(b, wtyp, num, keyField)
			if !nerr.Merge(err) {
				break
			}
			err = nil
			haveKey = true
		case 2:
			var v protoreflect.Value
			v, n, err = o.unmarshalScalar(b, wtyp, num, valField)
			if !nerr.Merge(err) {
				break
			}
			err = nil
			switch valField.Kind() {
			case protoreflect.GroupKind, protoreflect.MessageKind:
				if err := o.unmarshalMessage(v.Bytes(), val.Message()); !nerr.Merge(err) {
					return 0, err
				}
			default:
				val = v
			}
			haveVal = true
		}
		if err == errUnknown {
			n = wire.ConsumeFieldValue(num, wtyp, b)
			if n < 0 {
				return 0, wire.ParseError(n)
			}
		} else if err != nil {
			return 0, err
		}
		b = b[n:]
	}
	// Every map entry should have entries for key and value, but this is not strictly required.
	if !haveKey {
		key = keyField.Default()
	}
	if !haveVal {
		switch valField.Kind() {
		case protoreflect.GroupKind, protoreflect.MessageKind:
		default:
			val = valField.Default()
		}
	}
	mapv.Set(key.MapKey(), val)
	return n, nerr.E
}

// errUnknown is used internally to indicate fields which should be added
// to the unknown field set of a message. It is never returned from an exported
// function.
var errUnknown = errors.New("BUG: internal error (unknown)")
