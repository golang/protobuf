// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpb

import (
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/golang/protobuf/v2/internal/encoding/text"
	"github.com/golang/protobuf/v2/internal/encoding/wire"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/fieldnum"
	"github.com/golang/protobuf/v2/internal/mapsort"
	"github.com/golang/protobuf/v2/internal/pragma"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"
)

// Marshal writes the given proto.Message in textproto format using default options.
func Marshal(m proto.Message) ([]byte, error) {
	return MarshalOptions{}.Marshal(m)
}

// MarshalOptions is a configurable text format marshaler.
type MarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// AllowPartial allows messages that have missing required fields to marshal
	// without returning an error. If AllowPartial is false (the default),
	// Marshal will return error if there are any missing required fields.
	AllowPartial bool

	// If Indent is a non-empty string, it causes entries for a Message to be
	// preceded by the indent and trailed by a newline. Indent can only be
	// composed of space or tab characters.
	Indent string

	// Resolver is the registry used for type lookups when marshaling out
	// google.protobuf.Any messages in expanded form. If Resolver is not set,
	// marshaling will default to using protoregistry.GlobalTypes.  If a type is
	// not found, an Any message will be marshaled as a regular message.
	Resolver *protoregistry.Types
}

// Marshal writes the given proto.Message in textproto format using options in MarshalOptions object.
func (o MarshalOptions) Marshal(m proto.Message) ([]byte, error) {
	if o.Resolver == nil {
		o.Resolver = protoregistry.GlobalTypes
	}

	var nerr errors.NonFatal
	v, err := o.marshalMessage(m.ProtoReflect())
	if !nerr.Merge(err) {
		return nil, err
	}

	delims := [2]byte{'{', '}'}
	const outputASCII = false
	b, err := text.Marshal(v, o.Indent, delims, outputASCII)
	if !nerr.Merge(err) {
		return nil, err
	}
	if !o.AllowPartial {
		nerr.Merge(proto.IsInitialized(m))
	}
	return b, nerr.E
}

// marshalMessage converts a protoreflect.Message to a text.Value.
func (o MarshalOptions) marshalMessage(m pref.Message) (text.Value, error) {
	var nerr errors.NonFatal
	var msgFields [][2]text.Value
	messageDesc := m.Descriptor()

	// Handle Any expansion.
	if messageDesc.FullName() == "google.protobuf.Any" {
		msg, err := o.marshalAny(m)
		if err == nil || nerr.Merge(err) {
			// Return as is for nil or non-fatal error.
			return msg, nerr.E
		}
		// For other errors, continue on to marshal Any as a regular message.
	}

	// Handle known fields.
	fieldDescs := messageDesc.Fields()
	knownFields := m.KnownFields()
	size := fieldDescs.Len()
	for i := 0; i < size; i++ {
		fd := fieldDescs.Get(i)
		num := fd.Number()

		if !knownFields.Has(num) {
			continue
		}

		name := text.ValueOf(fd.Name())
		// Use type name for group field name.
		if fd.Kind() == pref.GroupKind {
			name = text.ValueOf(fd.Message().Name())
		}
		pval := knownFields.Get(num)
		var err error
		msgFields, err = o.appendField(msgFields, name, pval, fd)
		if !nerr.Merge(err) {
			return text.Value{}, err
		}
	}

	// Handle extensions.
	var err error
	msgFields, err = o.appendExtensions(msgFields, knownFields)
	if !nerr.Merge(err) {
		return text.Value{}, err
	}

	// Handle unknown fields.
	// TODO: Provide option to exclude or include unknown fields.
	m.UnknownFields().Range(func(_ pref.FieldNumber, raw pref.RawFields) bool {
		msgFields = appendUnknown(msgFields, raw)
		return true
	})

	return text.ValueOf(msgFields), nerr.E
}

// appendField marshals a protoreflect.Value and appends it to the given [][2]text.Value.
func (o MarshalOptions) appendField(msgFields [][2]text.Value, name text.Value, pval pref.Value, fd pref.FieldDescriptor) ([][2]text.Value, error) {
	var nerr errors.NonFatal

	switch {
	case fd.IsList():
		items, err := o.marshalList(pval.List(), fd)
		if !nerr.Merge(err) {
			return msgFields, err
		}

		for _, item := range items {
			msgFields = append(msgFields, [2]text.Value{name, item})
		}
	case fd.IsMap():
		items, err := o.marshalMap(pval.Map(), fd)
		if !nerr.Merge(err) {
			return msgFields, err
		}

		for _, item := range items {
			msgFields = append(msgFields, [2]text.Value{name, item})
		}
	default:
		tval, err := o.marshalSingular(pval, fd)
		if !nerr.Merge(err) {
			return msgFields, err
		}
		msgFields = append(msgFields, [2]text.Value{name, tval})
	}

	return msgFields, nerr.E
}

// marshalSingular converts a non-repeated field value to text.Value.
// This includes all scalar types, enums, messages, and groups.
func (o MarshalOptions) marshalSingular(val pref.Value, fd pref.FieldDescriptor) (text.Value, error) {
	kind := fd.Kind()
	switch kind {
	case pref.BoolKind,
		pref.Int32Kind, pref.Sint32Kind, pref.Uint32Kind,
		pref.Int64Kind, pref.Sint64Kind, pref.Uint64Kind,
		pref.Sfixed32Kind, pref.Fixed32Kind,
		pref.Sfixed64Kind, pref.Fixed64Kind,
		pref.FloatKind, pref.DoubleKind,
		pref.BytesKind:
		return text.ValueOf(val.Interface()), nil

	case pref.StringKind:
		s := val.String()
		if utf8.ValidString(s) {
			return text.ValueOf(s), nil
		}
		var nerr errors.NonFatal
		nerr.AppendInvalidUTF8(string(fd.FullName()))
		return text.ValueOf(s), nerr.E

	case pref.EnumKind:
		num := val.Enum()
		if desc := fd.Enum().Values().ByNumber(num); desc != nil {
			return text.ValueOf(desc.Name()), nil
		}
		// Use numeric value if there is no enum description.
		return text.ValueOf(int32(num)), nil

	case pref.MessageKind, pref.GroupKind:
		return o.marshalMessage(val.Message())
	}

	panic(fmt.Sprintf("%v has unknown kind: %v", fd.FullName(), kind))
}

// marshalList converts a protoreflect.List to []text.Value.
func (o MarshalOptions) marshalList(list pref.List, fd pref.FieldDescriptor) ([]text.Value, error) {
	var nerr errors.NonFatal
	size := list.Len()
	values := make([]text.Value, 0, size)

	for i := 0; i < size; i++ {
		item := list.Get(i)
		val, err := o.marshalSingular(item, fd)
		if !nerr.Merge(err) {
			// Return already marshaled values.
			return values, err
		}
		values = append(values, val)
	}

	return values, nerr.E
}

var (
	mapKeyName   = text.ValueOf(pref.Name("key"))
	mapValueName = text.ValueOf(pref.Name("value"))
)

// marshalMap converts a protoreflect.Map to []text.Value.
func (o MarshalOptions) marshalMap(mmap pref.Map, fd pref.FieldDescriptor) ([]text.Value, error) {
	var nerr errors.NonFatal
	// values is a list of messages.
	values := make([]text.Value, 0, mmap.Len())

	var err error
	mapsort.Range(mmap, fd.MapKey().Kind(), func(key pref.MapKey, val pref.Value) bool {
		var keyTxtVal text.Value
		keyTxtVal, err = o.marshalSingular(key.Value(), fd.MapKey())
		if !nerr.Merge(err) {
			return false
		}
		var valTxtVal text.Value
		valTxtVal, err = o.marshalSingular(val, fd.MapValue())
		if !nerr.Merge(err) {
			return false
		}
		// Map entry (message) contains 2 fields, first field for key and second field for value.
		msg := text.ValueOf([][2]text.Value{
			{mapKeyName, keyTxtVal},
			{mapValueName, valTxtVal},
		})
		values = append(values, msg)
		err = nil
		return true
	})
	if err != nil {
		return nil, err
	}

	return values, nerr.E
}

// appendExtensions marshals extension fields and appends them to the given [][2]text.Value.
func (o MarshalOptions) appendExtensions(msgFields [][2]text.Value, knownFields pref.KnownFields) ([][2]text.Value, error) {
	xtTypes := knownFields.ExtensionTypes()
	xtFields := make([][2]text.Value, 0, xtTypes.Len())

	var nerr errors.NonFatal
	var err error
	xtTypes.Range(func(xt pref.ExtensionType) bool {
		name := xt.Descriptor().FullName()
		// If extended type is a MessageSet, set field name to be the message type name.
		if isMessageSetExtension(xt) {
			name = xt.Descriptor().Message().FullName()
		}

		num := xt.Descriptor().Number()
		if knownFields.Has(num) {
			// Use string type to produce [name] format.
			tname := text.ValueOf(string(name))
			pval := knownFields.Get(num)
			xtFields, err = o.appendField(xtFields, tname, pval, xt.Descriptor())
			if !nerr.Merge(err) {
				return false
			}
			err = nil
		}
		return true
	})
	if err != nil {
		return msgFields, err
	}

	// Sort extensions lexicographically and append to output.
	sort.SliceStable(xtFields, func(i, j int) bool {
		return xtFields[i][0].String() < xtFields[j][0].String()
	})
	return append(msgFields, xtFields...), nerr.E
}

// isMessageSetExtension reports whether extension extends a message set.
func isMessageSetExtension(xt pref.ExtensionType) bool {
	xd := xt.Descriptor()
	if xd.Name() != "message_set_extension" {
		return false
	}
	md := xd.Message()
	if md == nil {
		return false
	}
	if xd.FullName().Parent() != md.FullName() {
		return false
	}
	xmd, ok := xd.ContainingMessage().(interface{ IsMessageSet() bool })
	return ok && xmd.IsMessageSet()
}

// appendUnknown parses the given []byte and appends field(s) into the given fields slice.
// This function assumes proper encoding in the given []byte.
func appendUnknown(fields [][2]text.Value, b []byte) [][2]text.Value {
	for len(b) > 0 {
		var value interface{}
		num, wtype, n := wire.ConsumeTag(b)
		b = b[n:]

		switch wtype {
		case wire.VarintType:
			value, n = wire.ConsumeVarint(b)
		case wire.Fixed32Type:
			value, n = wire.ConsumeFixed32(b)
		case wire.Fixed64Type:
			value, n = wire.ConsumeFixed64(b)
		case wire.BytesType:
			value, n = wire.ConsumeBytes(b)
		case wire.StartGroupType:
			var v []byte
			v, n = wire.ConsumeGroup(num, b)
			var msg [][2]text.Value
			value = appendUnknown(msg, v)
		default:
			panic(fmt.Sprintf("error parsing unknown field wire type: %v", wtype))
		}

		fields = append(fields, [2]text.Value{text.ValueOf(uint32(num)), text.ValueOf(value)})
		b = b[n:]
	}
	return fields
}

// marshalAny converts a google.protobuf.Any protoreflect.Message to a text.Value.
func (o MarshalOptions) marshalAny(m pref.Message) (text.Value, error) {
	var nerr errors.NonFatal
	knownFields := m.KnownFields()
	typeURL := knownFields.Get(fieldnum.Any_TypeUrl).String()
	value := knownFields.Get(fieldnum.Any_Value)

	emt, err := o.Resolver.FindMessageByURL(typeURL)
	if !nerr.Merge(err) {
		return text.Value{}, err
	}
	em := emt.New().Interface()
	// TODO: Need to set types registry in binary unmarshaling.
	// TODO: If binary unmarshaling returns required not set error, need to
	// return another required not set error that contains both the path to this
	// field and the path inside the embedded message.
	err = proto.UnmarshalOptions{
		AllowPartial: o.AllowPartial,
	}.Unmarshal(value.Bytes(), em)
	if !nerr.Merge(err) {
		return text.Value{}, err
	}

	msg, err := o.marshalMessage(em.ProtoReflect())
	if !nerr.Merge(err) {
		return text.Value{}, err
	}
	// Expanded Any field value contains only a single field with the type_url field value as the
	// field name in [] and a text marshaled field value of the embedded message.
	msgFields := [][2]text.Value{
		{
			text.ValueOf(typeURL),
			msg,
		},
	}
	return text.ValueOf(msgFields), nerr.E
}
