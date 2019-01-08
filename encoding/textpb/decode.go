// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpb

import (
	"fmt"
	"strings"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/internal/encoding/text"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/pragma"
	"github.com/golang/protobuf/v2/internal/set"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"
)

// Unmarshal reads the given []byte into the given proto.Message.
// TODO: may want to describe when Unmarshal returns error.
func Unmarshal(m proto.Message, b []byte) error {
	return UnmarshalOptions{}.Unmarshal(m, b)
}

// UnmarshalOptions is a configurable textproto format parser.
type UnmarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// Resolver is the registry used for type lookups when unmarshaling extensions
	// and processing Any. If Resolver is not set, unmarshaling will default to
	// using protoregistry.GlobalTypes.
	Resolver *protoregistry.Types
}

// Unmarshal reads the given []byte and populates the given proto.Message using options in
// UnmarshalOptions object.
func (o UnmarshalOptions) Unmarshal(m proto.Message, b []byte) error {
	var nerr errors.NonFatal

	mr := m.ProtoReflect()
	// Clear all fields before populating it.
	// TODO: Determine if this needs to be consistent with jsonpb and binary unmarshal where
	// behavior is to merge values into existing message. If decision is to not clear the fields
	// ahead, code will need to be updated properly when merging nested messages.
	resetMessage(mr)

	// Parse into text.Value of message type.
	val, err := text.Unmarshal(b)
	if !nerr.Merge(err) {
		return err
	}

	if o.Resolver == nil {
		o.Resolver = protoregistry.GlobalTypes
	}
	err = o.unmarshalMessage(val.Message(), mr)
	if !nerr.Merge(err) {
		return err
	}

	return nerr.E
}

// resetMessage clears all fields of given protoreflect.Message.
// TODO: This should go into the proto package.
func resetMessage(m pref.Message) {
	knownFields := m.KnownFields()
	knownFields.Range(func(num pref.FieldNumber, _ pref.Value) bool {
		knownFields.Clear(num)
		return true
	})
	unknownFields := m.UnknownFields()
	unknownFields.Range(func(num pref.FieldNumber, _ pref.RawFields) bool {
		unknownFields.Set(num, nil)
		return true
	})
	extTypes := knownFields.ExtensionTypes()
	extTypes.Range(func(xt pref.ExtensionType) bool {
		extTypes.Remove(xt)
		return true
	})
}

// unmarshalMessage unmarshals a [][2]text.Value message into the given protoreflect.Message.
func (o UnmarshalOptions) unmarshalMessage(tmsg [][2]text.Value, m pref.Message) error {
	var nerr errors.NonFatal

	msgType := m.Type()
	knownFields := m.KnownFields()

	// Handle expanded Any message.
	if msgType.FullName() == "google.protobuf.Any" && isExpandedAny(tmsg) {
		return o.unmarshalAny(tmsg[0], knownFields)
	}

	fieldDescs := msgType.Fields()
	reservedNames := msgType.ReservedNames()
	xtTypes := knownFields.ExtensionTypes()
	var reqNums set.Ints
	var seenNums set.Ints

	for _, tfield := range tmsg {
		tkey := tfield[0]
		tval := tfield[1]

		var fd pref.FieldDescriptor
		var name pref.Name
		switch tkey.Type() {
		case text.Name:
			name, _ = tkey.Name()
			fd = fieldDescs.ByName(name)
			if fd == nil {
				// Check if this is a group field.
				fd = fieldDescs.ByName(pref.Name(strings.ToLower(string(name))))
			}
		case text.String:
			// Handle extensions only. This code path is not for Any.
			if msgType.FullName() == "google.protobuf.Any" {
				break
			}
			// Extensions have to be registered first in the message's
			// ExtensionTypes before setting a value to it.
			xtName := pref.FullName(tkey.String())
			// Check first if it is already registered. This is the case for
			// repeated fields.
			xt := xtTypes.ByName(xtName)
			if xt == nil {
				var err error
				xt, err = o.findExtension(xtName)
				if err != nil && err != protoregistry.NotFound {
					return errors.New("unable to resolve [%v]: %v", xtName, err)
				}
				if xt != nil {
					xtTypes.Register(xt)
				}
			}
			fd = xt
		}

		if fd == nil {
			// Ignore reserved names.
			if reservedNames.Has(name) {
				continue
			}
			// TODO: Can provide option to ignore unknown message fields.
			return errors.New("%v contains unknown field: %v", msgType.FullName(), tkey)
		}

		if cardinality := fd.Cardinality(); cardinality == pref.Repeated {
			// Map or list fields have cardinality of repeated.
			if err := o.unmarshalRepeated(tval, fd, knownFields); !nerr.Merge(err) {
				return err
			}
		} else {
			// Required or optional fields.
			num := uint64(fd.Number())
			if seenNums.Has(num) {
				return errors.New("non-repeated field %v is repeated", fd.FullName())
			}
			if err := o.unmarshalSingular(tval, fd, knownFields); !nerr.Merge(err) {
				return err
			}
			if cardinality == pref.Required {
				reqNums.Set(num)
			}
			seenNums.Set(num)
		}
	}

	// Check for any missing required fields.
	allReqNums := msgType.RequiredNumbers()
	if reqNums.Len() != allReqNums.Len() {
		for i := 0; i < allReqNums.Len(); i++ {
			if num := allReqNums.Get(i); !reqNums.Has(uint64(num)) {
				nerr.AppendRequiredNotSet(string(fieldDescs.ByNumber(num).FullName()))
			}
		}
	}

	return nerr.E
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
func (o UnmarshalOptions) unmarshalSingular(input text.Value, fd pref.FieldDescriptor, knownFields pref.KnownFields) error {
	num := fd.Number()

	var nerr errors.NonFatal
	var val pref.Value
	switch fd.Kind() {
	case pref.MessageKind, pref.GroupKind:
		if input.Type() != text.Message {
			return errors.New("%v contains invalid message/group value: %v", fd.FullName(), input)
		}
		m := knownFields.NewMessage(num)
		if err := o.unmarshalMessage(input.Message(), m); !nerr.Merge(err) {
			return err
		}
		val = pref.ValueOf(m)
	default:
		var err error
		val, err = unmarshalScalar(input, fd)
		if !nerr.Merge(err) {
			return err
		}
	}
	knownFields.Set(num, val)

	return nerr.E
}

// unmarshalRepeated unmarshals given text.Value into a repeated field. Caller should only
// call this for cardinality=repeated.
func (o UnmarshalOptions) unmarshalRepeated(input text.Value, fd pref.FieldDescriptor, knownFields pref.KnownFields) error {
	var items []text.Value
	// If input is not a list, turn it into a list.
	if input.Type() != text.List {
		items = []text.Value{input}
	} else {
		items = input.List()
	}

	var nerr errors.NonFatal
	num := fd.Number()
	val := knownFields.Get(num)
	if !fd.IsMap() {
		if err := o.unmarshalList(items, fd, val.List()); !nerr.Merge(err) {
			return err
		}
	} else {
		if err := o.unmarshalMap(items, fd, val.Map()); !nerr.Merge(err) {
			return err
		}
	}

	return nerr.E
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
		if n, ok := input.Float32(); ok {
			return pref.ValueOf(float32(n)), nil
		}
	case pref.DoubleKind:
		if n, ok := input.Float64(); ok {
			return pref.ValueOf(float64(n)), nil
		}
	case pref.StringKind:
		if input.Type() == text.String {
			return pref.ValueOf(string(input.String())), nil
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
			if enumVal := fd.EnumType().Values().ByName(name); enumVal != nil {
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
	var nerr errors.NonFatal

	switch fd.Kind() {
	case pref.MessageKind, pref.GroupKind:
		for _, input := range inputList {
			if input.Type() != text.Message {
				return errors.New("%v contains invalid message/group value: %v", fd.FullName(), input)
			}
			m := list.NewMessage()
			if err := o.unmarshalMessage(input.Message(), m); !nerr.Merge(err) {
				return err
			}
			list.Append(pref.ValueOf(m))
		}
	default:
		for _, input := range inputList {
			val, err := unmarshalScalar(input, fd)
			if !nerr.Merge(err) {
				return err
			}
			list.Append(val)
		}
	}

	return nerr.E
}

// unmarshalMap unmarshals given []text.Value into given protoreflect.Map.
func (o UnmarshalOptions) unmarshalMap(input []text.Value, fd pref.FieldDescriptor, mmap pref.Map) error {
	var nerr errors.NonFatal
	fields := fd.MessageType().Fields()
	keyDesc := fields.ByNumber(1)
	valDesc := fields.ByNumber(2)

	// Determine ahead whether map entry is a scalar type or a message type in order to call the
	// appropriate unmarshalMapValue func inside the for loop below.
	unmarshalMapValue := unmarshalMapScalarValue
	switch valDesc.Kind() {
	case pref.MessageKind, pref.GroupKind:
		unmarshalMapValue = o.unmarshalMapMessageValue
	}

	for _, entry := range input {
		if entry.Type() != text.Message {
			return errors.New("%v contains invalid map entry: %v", fd.FullName(), entry)
		}
		tkey, tval, err := parseMapEntry(entry.Message(), fd.FullName())
		if !nerr.Merge(err) {
			return err
		}
		pkey, err := unmarshalMapKey(tkey, keyDesc)
		if !nerr.Merge(err) {
			return err
		}
		err = unmarshalMapValue(tval, pkey, valDesc, mmap)
		if !nerr.Merge(err) {
			return err
		}
	}

	return nerr.E
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
	var nerr errors.NonFatal
	var value [][2]text.Value
	if input.Type() != 0 {
		value = input.Message()
	}
	m := mmap.NewMessage()
	if err := o.unmarshalMessage(value, m); !nerr.Merge(err) {
		return err
	}
	mmap.Set(pkey, pref.ValueOf(m))
	return nerr.E
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
func (o UnmarshalOptions) unmarshalAny(tfield [2]text.Value, knownFields pref.KnownFields) error {
	var nerr errors.NonFatal

	typeURL := tfield[0].String()
	value := tfield[1].Message()

	mt, err := o.Resolver.FindMessageByURL(typeURL)
	if !nerr.Merge(err) {
		return errors.New("unable to resolve message [%v]: %v", typeURL, err)
	}
	// Create new message for the embedded message type and unmarshal the
	// value into it.
	m := mt.New()
	if err := o.unmarshalMessage(value, m); !nerr.Merge(err) {
		return err
	}
	// Serialize the embedded message and assign the resulting bytes to the value field.
	// TODO: Switch to V2 marshal and enable deterministic option when ready.
	var mv1 protoV1.Message
	if mtmp, ok := m.(pvalue.Unwrapper); ok {
		mv1 = mtmp.ProtoUnwrap().(protoV1.Message)
	} else {
		mv1 = m.Interface().(protoV1.Message)
	}
	b, err := protoV1.Marshal(mv1)
	if !nerr.Merge(err) {
		return err
	}

	knownFields.Set(pref.FieldNumber(1), pref.ValueOf(typeURL))
	knownFields.Set(pref.FieldNumber(2), pref.ValueOf(b))

	return nerr.E
}
