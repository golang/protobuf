// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototext

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"google.golang.org/protobuf/internal/encoding/text"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/fieldnum"
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/internal/set"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Unmarshal reads the given []byte into the given proto.Message.
func Unmarshal(b []byte, m proto.Message) error {
	return UnmarshalOptions{}.Unmarshal(b, m)
}

// UnmarshalOptions is a configurable textproto format unmarshaler.
type UnmarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// AllowPartial accepts input for messages that will result in missing
	// required fields. If AllowPartial is false (the default), Unmarshal will
	// return error if there are any missing required fields.
	AllowPartial bool

	// Resolver is used for looking up types when unmarshaling
	// google.protobuf.Any messages or extension fields.
	// If nil, this defaults to using protoregistry.GlobalTypes.
	Resolver interface {
		protoregistry.MessageTypeResolver
		protoregistry.ExtensionTypeResolver
	}
}

// Unmarshal reads the given []byte and populates the given proto.Message using options in
// UnmarshalOptions object.
func (o UnmarshalOptions) Unmarshal(b []byte, m proto.Message) error {
	// Clear all fields before populating it.
	// TODO: Determine if this needs to be consistent with protojson and binary unmarshal where
	// behavior is to merge values into existing message. If decision is to not clear the fields
	// ahead, code will need to be updated properly when merging nested messages.
	proto.Reset(m)

	// Parse into text.Value of message type.
	val, err := text.Unmarshal(b)
	if err != nil {
		return err
	}

	if o.Resolver == nil {
		o.Resolver = protoregistry.GlobalTypes
	}
	err = o.unmarshalMessage(val.Message(), m.ProtoReflect())
	if err != nil {
		return err
	}

	if o.AllowPartial {
		return nil
	}
	return proto.IsInitialized(m)
}

// unmarshalMessage unmarshals a [][2]text.Value message into the given protoreflect.Message.
func (o UnmarshalOptions) unmarshalMessage(tmsg [][2]text.Value, m pref.Message) error {
	messageDesc := m.Descriptor()

	// Handle expanded Any message.
	if messageDesc.FullName() == "google.protobuf.Any" && isExpandedAny(tmsg) {
		return o.unmarshalAny(tmsg[0], m)
	}

	fieldDescs := messageDesc.Fields()
	reservedNames := messageDesc.ReservedNames()
	var seenNums set.Ints
	var seenOneofs set.Ints

	for _, tfield := range tmsg {
		tkey := tfield[0]
		tval := tfield[1]

		var fd pref.FieldDescriptor
		var name pref.Name
		switch tkey.Type() {
		case text.Name:
			name, _ = tkey.Name()
			fd = fieldDescs.ByName(name)
			// The proto name of a group field is in all lowercase. However, the
			// textproto field name is the type name. Check to make sure that
			// group name is correct.
			if fd == nil {
				gd := fieldDescs.ByName(pref.Name(strings.ToLower(string(name))))
				if gd != nil && gd.Kind() == pref.GroupKind && gd.Message().Name() == name {
					fd = gd
				}
			} else {
				if fd.Kind() == pref.GroupKind && fd.Message().Name() != name {
					// Reset fd to nil because name does not match.
					fd = nil
				}
			}
		case text.String:
			// Handle extensions only. This code path is not for Any.
			if messageDesc.FullName() == "google.protobuf.Any" {
				break
			}
			// Extensions have to be registered first in the message's
			// ExtensionTypes before setting a value to it.
			extName := pref.FullName(tkey.String())
			// Check first if it is already registered. This is the case for
			// repeated fields.
			xt, err := o.findExtension(extName)
			if err != nil && err != protoregistry.NotFound {
				return errors.New("unable to resolve [%v]: %v", extName, err)
			}
			fd = xt
		}

		if fd == nil {
			// Ignore reserved names.
			if reservedNames.Has(name) {
				continue
			}
			// TODO: Can provide option to ignore unknown message fields.
			return errors.New("%v contains unknown field: %v", messageDesc.FullName(), tkey)
		}

		switch {
		case fd.IsList():
			// If input is not a list, turn it into a list.
			var items []text.Value
			if tval.Type() != text.List {
				items = []text.Value{tval}
			} else {
				items = tval.List()
			}

			list := m.Mutable(fd).List()
			if err := o.unmarshalList(items, fd, list); err != nil {
				return err
			}
		case fd.IsMap():
			// If input is not a list, turn it into a list.
			var items []text.Value
			if tval.Type() != text.List {
				items = []text.Value{tval}
			} else {
				items = tval.List()
			}

			mmap := m.Mutable(fd).Map()
			if err := o.unmarshalMap(items, fd, mmap); err != nil {
				return err
			}
		default:
			// If field is a oneof, check if it has already been set.
			if od := fd.ContainingOneof(); od != nil {
				idx := uint64(od.Index())
				if seenOneofs.Has(idx) {
					return errors.New("oneof %v is already set", od.FullName())
				}
				seenOneofs.Set(idx)
			}

			// Required or optional fields.
			num := uint64(fd.Number())
			if seenNums.Has(num) {
				return errors.New("non-repeated field %v is repeated", fd.FullName())
			}
			if err := o.unmarshalSingular(tval, fd, m); err != nil {
				return err
			}
			seenNums.Set(num)
		}
	}

	return nil
}

// findExtension returns protoreflect.ExtensionType from the Resolver if found.
func (o UnmarshalOptions) findExtension(xtName pref.FullName) (pref.ExtensionType, error) {
	xt, err := o.Resolver.FindExtensionByName(xtName)
	if err == nil {
		return xt, nil
	}

	// Check if this is a MessageSet extension field.
	xt, err = o.Resolver.FindExtensionByName(xtName + ".message_set_extension")
	if err == nil && isMessageSetExtension(xt) {
		return xt, nil
	}
	return nil, protoregistry.NotFound
}

// unmarshalSingular unmarshals given text.Value into the non-repeated field.
func (o UnmarshalOptions) unmarshalSingular(input text.Value, fd pref.FieldDescriptor, m pref.Message) error {
	var val pref.Value
	switch fd.Kind() {
	case pref.MessageKind, pref.GroupKind:
		if input.Type() != text.Message {
			return errors.New("%v contains invalid message/group value: %v", fd.FullName(), input)
		}
		m2 := m.NewMessage(fd)
		if err := o.unmarshalMessage(input.Message(), m2); err != nil {
			return err
		}
		val = pref.ValueOf(m2)
	default:
		var err error
		val, err = unmarshalScalar(input, fd)
		if err != nil {
			return err
		}
	}
	m.Set(fd, val)

	return nil
}

// unmarshalScalar converts the given text.Value to a scalar/enum protoreflect.Value specified in
// the given FieldDescriptor. Caller should not pass in a FieldDescriptor for a message/group kind.
func unmarshalScalar(input text.Value, fd pref.FieldDescriptor) (pref.Value, error) {
	const b32 = false
	const b64 = true

	switch kind := fd.Kind(); kind {
	case pref.BoolKind:
		if b, ok := input.Bool(); ok {
			return pref.ValueOf(bool(b)), nil
		}
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		if n, ok := input.Int(b32); ok {
			return pref.ValueOf(int32(n)), nil
		}
	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		if n, ok := input.Int(b64); ok {
			return pref.ValueOf(int64(n)), nil
		}
	case pref.Uint32Kind, pref.Fixed32Kind:
		if n, ok := input.Uint(b32); ok {
			return pref.ValueOf(uint32(n)), nil
		}
	case pref.Uint64Kind, pref.Fixed64Kind:
		if n, ok := input.Uint(b64); ok {
			return pref.ValueOf(uint64(n)), nil
		}
	case pref.FloatKind:
		if n, ok := input.Float(b32); ok {
			return pref.ValueOf(float32(n)), nil
		}
	case pref.DoubleKind:
		if n, ok := input.Float(b64); ok {
			return pref.ValueOf(float64(n)), nil
		}
	case pref.StringKind:
		if input.Type() == text.String {
			s := input.String()
			if utf8.ValidString(s) {
				return pref.ValueOf(s), nil
			}
			return pref.Value{}, errors.InvalidUTF8(string(fd.FullName()))
		}
	case pref.BytesKind:
		if input.Type() == text.String {
			return pref.ValueOf([]byte(input.String())), nil
		}
	case pref.EnumKind:
		// If input is int32, use directly.
		if n, ok := input.Int(b32); ok {
			return pref.ValueOf(pref.EnumNumber(n)), nil
		}
		if name, ok := input.Name(); ok {
			// Lookup EnumNumber based on name.
			if enumVal := fd.Enum().Values().ByName(name); enumVal != nil {
				return pref.ValueOf(enumVal.Number()), nil
			}
		}
	default:
		panic(fmt.Sprintf("invalid scalar kind %v", kind))
	}

	return pref.Value{}, errors.New("%v contains invalid scalar value: %v", fd.FullName(), input)
}

// unmarshalList unmarshals given []text.Value into given protoreflect.List.
func (o UnmarshalOptions) unmarshalList(inputList []text.Value, fd pref.FieldDescriptor, list pref.List) error {
	switch fd.Kind() {
	case pref.MessageKind, pref.GroupKind:
		for _, input := range inputList {
			if input.Type() != text.Message {
				return errors.New("%v contains invalid message/group value: %v", fd.FullName(), input)
			}
			m := list.NewMessage()
			if err := o.unmarshalMessage(input.Message(), m); err != nil {
				return err
			}
			list.Append(pref.ValueOf(m))
		}
	default:
		for _, input := range inputList {
			val, err := unmarshalScalar(input, fd)
			if err != nil {
				return err
			}
			list.Append(val)
		}
	}

	return nil
}

// unmarshalMap unmarshals given []text.Value into given protoreflect.Map.
func (o UnmarshalOptions) unmarshalMap(input []text.Value, fd pref.FieldDescriptor, mmap pref.Map) error {
	// Determine ahead whether map entry is a scalar type or a message type in order to call the
	// appropriate unmarshalMapValue func inside the for loop below.
	unmarshalMapValue := unmarshalMapScalarValue
	switch fd.MapValue().Kind() {
	case pref.MessageKind, pref.GroupKind:
		unmarshalMapValue = o.unmarshalMapMessageValue
	}

	for _, entry := range input {
		if entry.Type() != text.Message {
			return errors.New("%v contains invalid map entry: %v", fd.FullName(), entry)
		}
		tkey, tval, err := parseMapEntry(entry.Message(), fd.FullName())
		if err != nil {
			return err
		}
		pkey, err := unmarshalMapKey(tkey, fd.MapKey())
		if err != nil {
			return err
		}
		err = unmarshalMapValue(tval, pkey, fd.MapValue(), mmap)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseMapEntry parses [][2]text.Value for field names key and value, and return corresponding
// field values. If there are duplicate field names, the value for the last field is returned. If
// the field name does not exist, it will return the zero value of text.Value. It will return an
// error if there are unknown field names.
func parseMapEntry(mapEntry [][2]text.Value, name pref.FullName) (key text.Value, value text.Value, err error) {
	for _, field := range mapEntry {
		keyStr, ok := field[0].Name()
		if ok {
			switch keyStr {
			case "key":
				if key.Type() != 0 {
					return key, value, errors.New("%v contains duplicate key field", name)
				}
				key = field[1]
			case "value":
				if value.Type() != 0 {
					return key, value, errors.New("%v contains duplicate value field", name)
				}
				value = field[1]
			default:
				ok = false
			}
		}
		if !ok {
			// TODO: Do not return error if ignore unknown option is added and enabled.
			return key, value, errors.New("%v contains unknown map entry name: %v", name, field[0])
		}
	}
	return key, value, nil
}

// unmarshalMapKey converts given text.Value into a protoreflect.MapKey. A map key type is any
// integral or string type.
func unmarshalMapKey(input text.Value, fd pref.FieldDescriptor) (pref.MapKey, error) {
	// If input is not set, use the zero value.
	if input.Type() == 0 {
		return fd.Default().MapKey(), nil
	}

	val, err := unmarshalScalar(input, fd)
	if err != nil {
		return pref.MapKey{}, errors.New("%v contains invalid key: %v", fd.FullName(), input)
	}
	return val.MapKey(), nil
}

// unmarshalMapMessageValue unmarshals given message-type text.Value into a protoreflect.Map for
// the given MapKey.
func (o UnmarshalOptions) unmarshalMapMessageValue(input text.Value, pkey pref.MapKey, _ pref.FieldDescriptor, mmap pref.Map) error {
	var value [][2]text.Value
	if input.Type() != 0 {
		value = input.Message()
	}
	m := mmap.NewMessage()
	if err := o.unmarshalMessage(value, m); err != nil {
		return err
	}
	mmap.Set(pkey, pref.ValueOf(m))
	return nil
}

// unmarshalMapScalarValue unmarshals given scalar-type text.Value into a protoreflect.Map
// for the given MapKey.
func unmarshalMapScalarValue(input text.Value, pkey pref.MapKey, fd pref.FieldDescriptor, mmap pref.Map) error {
	var val pref.Value
	if input.Type() == 0 {
		val = fd.Default()
	} else {
		var err error
		val, err = unmarshalScalar(input, fd)
		if err != nil {
			return err
		}
	}
	mmap.Set(pkey, val)
	return nil
}

// isExpandedAny returns true if given [][2]text.Value may be an expanded Any that contains only one
// field with key type of text.String type and value type of text.Message.
func isExpandedAny(tmsg [][2]text.Value) bool {
	if len(tmsg) != 1 {
		return false
	}

	field := tmsg[0]
	return field[0].Type() == text.String && field[1].Type() == text.Message
}

// unmarshalAny unmarshals an expanded Any textproto. This method assumes that the given
// tfield has key type of text.String and value type of text.Message.
func (o UnmarshalOptions) unmarshalAny(tfield [2]text.Value, m pref.Message) error {
	typeURL := tfield[0].String()
	value := tfield[1].Message()

	mt, err := o.Resolver.FindMessageByURL(typeURL)
	if err != nil {
		return errors.New("unable to resolve message [%v]: %v", typeURL, err)
	}
	// Create new message for the embedded message type and unmarshal the
	// value into it.
	m2 := mt.New()
	if err := o.unmarshalMessage(value, m2); err != nil {
		return err
	}
	// Serialize the embedded message and assign the resulting bytes to the value field.
	b, err := proto.MarshalOptions{
		AllowPartial:  true, // never check required fields inside an Any
		Deterministic: true,
	}.Marshal(m2.Interface())
	if err != nil {
		return err
	}

	fds := m.Descriptor().Fields()
	fdType := fds.ByNumber(fieldnum.Any_TypeUrl)
	fdValue := fds.ByNumber(fieldnum.Any_Value)

	m.Set(fdType, pref.ValueOf(typeURL))
	m.Set(fdValue, pref.ValueOf(b))

	return nil
}
