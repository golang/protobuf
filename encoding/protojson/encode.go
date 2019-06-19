// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protojson

import (
	"encoding/base64"
	"fmt"
	"sort"

	"google.golang.org/protobuf/internal/encoding/json"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Marshal writes the given proto.Message in JSON format using default options.
func Marshal(m proto.Message) ([]byte, error) {
	return MarshalOptions{}.Marshal(m)
}

// MarshalOptions is a configurable JSON format marshaler.
type MarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// AllowPartial allows messages that have missing required fields to marshal
	// without returning an error. If AllowPartial is false (the default),
	// Marshal will return error if there are any missing required fields.
	AllowPartial bool

	// If Indent is a non-empty string, it causes entries for an Array or Object
	// to be preceded by the indent and trailed by a newline. Indent can only be
	// composed of space or tab characters.
	Indent string

	// Resolver is used for looking up types when expanding google.protobuf.Any
	// messages. If nil, this defaults to using protoregistry.GlobalTypes.
	Resolver interface {
		protoregistry.ExtensionTypeResolver
		protoregistry.MessageTypeResolver
	}

	encoder *json.Encoder
}

// Marshal marshals the given proto.Message in the JSON format using options in
// MarshalOptions.
func (o MarshalOptions) Marshal(m proto.Message) ([]byte, error) {
	var err error
	o.encoder, err = json.NewEncoder(o.Indent)
	if err != nil {
		return nil, err
	}
	if o.Resolver == nil {
		o.Resolver = protoregistry.GlobalTypes
	}

	var nerr errors.NonFatal
	err = o.marshalMessage(m.ProtoReflect())
	if !nerr.Merge(err) {
		return nil, err
	}
	if !o.AllowPartial {
		nerr.Merge(proto.IsInitialized(m))
	}
	return o.encoder.Bytes(), nerr.E
}

// marshalMessage marshals the given protoreflect.Message.
func (o MarshalOptions) marshalMessage(m pref.Message) error {
	var nerr errors.NonFatal

	if isCustomType(m.Descriptor().FullName()) {
		return o.marshalCustomType(m)
	}

	o.encoder.StartObject()
	defer o.encoder.EndObject()
	if err := o.marshalFields(m); !nerr.Merge(err) {
		return err
	}

	return nerr.E
}

// marshalFields marshals the fields in the given protoreflect.Message.
func (o MarshalOptions) marshalFields(m pref.Message) error {
	var nerr errors.NonFatal

	// Marshal out known fields.
	fieldDescs := m.Descriptor().Fields()
	for i := 0; i < fieldDescs.Len(); i++ {
		fd := fieldDescs.Get(i)
		if !m.Has(fd) {
			continue
		}

		name := fd.JSONName()
		val := m.Get(fd)
		if err := o.encoder.WriteName(name); !nerr.Merge(err) {
			return err
		}
		if err := o.marshalValue(val, fd); !nerr.Merge(err) {
			return err
		}
	}

	// Marshal out extensions.
	if err := o.marshalExtensions(m); !nerr.Merge(err) {
		return err
	}
	return nerr.E
}

// marshalValue marshals the given protoreflect.Value.
func (o MarshalOptions) marshalValue(val pref.Value, fd pref.FieldDescriptor) error {
	switch {
	case fd.IsList():
		return o.marshalList(val.List(), fd)
	case fd.IsMap():
		return o.marshalMap(val.Map(), fd)
	default:
		return o.marshalSingular(val, fd)
	}
}

// marshalSingular marshals the given non-repeated field value. This includes
// all scalar types, enums, messages, and groups.
func (o MarshalOptions) marshalSingular(val pref.Value, fd pref.FieldDescriptor) error {
	var nerr errors.NonFatal
	switch kind := fd.Kind(); kind {
	case pref.BoolKind:
		o.encoder.WriteBool(val.Bool())

	case pref.StringKind:
		if err := o.encoder.WriteString(val.String()); !nerr.Merge(err) {
			return err
		}

	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		o.encoder.WriteInt(val.Int())

	case pref.Uint32Kind, pref.Fixed32Kind:
		o.encoder.WriteUint(val.Uint())

	case pref.Int64Kind, pref.Sint64Kind, pref.Uint64Kind,
		pref.Sfixed64Kind, pref.Fixed64Kind:
		// 64-bit integers are written out as JSON string.
		o.encoder.WriteString(val.String())

	case pref.FloatKind:
		// Encoder.WriteFloat handles the special numbers NaN and infinites.
		o.encoder.WriteFloat(val.Float(), 32)

	case pref.DoubleKind:
		// Encoder.WriteFloat handles the special numbers NaN and infinites.
		o.encoder.WriteFloat(val.Float(), 64)

	case pref.BytesKind:
		err := o.encoder.WriteString(base64.StdEncoding.EncodeToString(val.Bytes()))
		if !nerr.Merge(err) {
			return err
		}

	case pref.EnumKind:
		if fd.Enum().FullName() == "google.protobuf.NullValue" {
			o.encoder.WriteNull()
		} else if desc := fd.Enum().Values().ByNumber(val.Enum()); desc != nil {
			err := o.encoder.WriteString(string(desc.Name()))
			if !nerr.Merge(err) {
				return err
			}
		} else {
			// Use numeric value if there is no enum value descriptor.
			o.encoder.WriteInt(int64(val.Enum()))
		}

	case pref.MessageKind, pref.GroupKind:
		if err := o.marshalMessage(val.Message()); !nerr.Merge(err) {
			return err
		}

	default:
		panic(fmt.Sprintf("%v has unknown kind: %v", fd.FullName(), kind))
	}
	return nerr.E
}

// marshalList marshals the given protoreflect.List.
func (o MarshalOptions) marshalList(list pref.List, fd pref.FieldDescriptor) error {
	o.encoder.StartArray()
	defer o.encoder.EndArray()

	var nerr errors.NonFatal
	for i := 0; i < list.Len(); i++ {
		item := list.Get(i)
		if err := o.marshalSingular(item, fd); !nerr.Merge(err) {
			return err
		}
	}
	return nerr.E
}

type mapEntry struct {
	key   pref.MapKey
	value pref.Value
}

// marshalMap marshals given protoreflect.Map.
func (o MarshalOptions) marshalMap(mmap pref.Map, fd pref.FieldDescriptor) error {
	o.encoder.StartObject()
	defer o.encoder.EndObject()

	// Get a sorted list based on keyType first.
	entries := make([]mapEntry, 0, mmap.Len())
	mmap.Range(func(key pref.MapKey, val pref.Value) bool {
		entries = append(entries, mapEntry{key: key, value: val})
		return true
	})
	sortMap(fd.MapKey().Kind(), entries)

	// Write out sorted list.
	var nerr errors.NonFatal
	for _, entry := range entries {
		if err := o.encoder.WriteName(entry.key.String()); !nerr.Merge(err) {
			return err
		}
		if err := o.marshalSingular(entry.value, fd.MapValue()); !nerr.Merge(err) {
			return err
		}
	}
	return nerr.E
}

// sortMap orders list based on value of key field for deterministic ordering.
func sortMap(keyKind pref.Kind, values []mapEntry) {
	sort.Slice(values, func(i, j int) bool {
		switch keyKind {
		case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind,
			pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
			return values[i].key.Int() < values[j].key.Int()

		case pref.Uint32Kind, pref.Fixed32Kind,
			pref.Uint64Kind, pref.Fixed64Kind:
			return values[i].key.Uint() < values[j].key.Uint()
		}
		return values[i].key.String() < values[j].key.String()
	})
}

// marshalExtensions marshals extension fields.
func (o MarshalOptions) marshalExtensions(m pref.Message) error {
	type entry struct {
		key   string
		value pref.Value
		desc  pref.FieldDescriptor
	}

	// Get a sorted list based on field key first.
	var entries []entry
	m.Range(func(fd pref.FieldDescriptor, v pref.Value) bool {
		if !fd.IsExtension() {
			return true
		}
		xt := fd.(pref.ExtensionType)

		// If extended type is a MessageSet, set field name to be the message type name.
		name := xt.Descriptor().FullName()
		if isMessageSetExtension(xt) {
			name = xt.Descriptor().Message().FullName()
		}

		// Use [name] format for JSON field name.
		entries = append(entries, entry{
			key:   string(name),
			value: v,
			desc:  fd,
		})
		return true
	})

	// Sort extensions lexicographically.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	// Write out sorted list.
	var nerr errors.NonFatal
	for _, entry := range entries {
		// JSON field name is the proto field name enclosed in [], similar to
		// textproto. This is consistent with Go v1 lib. C++ lib v3.7.0 does not
		// marshal out extension fields.
		if err := o.encoder.WriteName("[" + entry.key + "]"); !nerr.Merge(err) {
			return err
		}
		if err := o.marshalValue(entry.value, entry.desc); !nerr.Merge(err) {
			return err
		}
	}
	return nerr.E
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
