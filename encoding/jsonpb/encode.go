// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb

import (
	"encoding/base64"
	"sort"

	"github.com/golang/protobuf/v2/internal/encoding/json"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/pragma"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// Marshal writes the given proto.Message in JSON format using default options.
func Marshal(m proto.Message) ([]byte, error) {
	return MarshalOptions{}.Marshal(m)
}

// MarshalOptions is a configurable JSON format marshaler.
type MarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// Set Compact to true to have output in a single line with no line breaks.
	Compact bool
}

// Marshal returns the given proto.Message in JSON format using options in MarshalOptions object.
func (o MarshalOptions) Marshal(m proto.Message) ([]byte, error) {
	indent := "  "
	if o.Compact {
		indent = ""
	}

	enc, err := newEncoder(indent)
	if err != nil {
		return nil, err
	}

	var nerr errors.NonFatal
	err = enc.marshalMessage(m.ProtoReflect())
	if !nerr.Merge(err) {
		return nil, err
	}
	return enc.Bytes(), nerr.E
}

// encoder encodes protoreflect values into JSON.
type encoder struct {
	*json.Encoder
}

func newEncoder(indent string) (encoder, error) {
	enc, err := json.NewEncoder(indent)
	if err != nil {
		return encoder{}, errors.New("error in constructing an encoder: %v", err)
	}
	return encoder{enc}, nil
}

// marshalMessage marshals the given protoreflect.Message.
func (e encoder) marshalMessage(m pref.Message) error {
	e.StartObject()
	defer e.EndObject()

	var nerr errors.NonFatal
	fieldDescs := m.Type().Fields()
	knownFields := m.KnownFields()

	for i := 0; i < fieldDescs.Len(); i++ {
		fd := fieldDescs.Get(i)
		num := fd.Number()

		if !knownFields.Has(num) {
			if fd.Cardinality() == pref.Required {
				// Treat unset required fields as a non-fatal error.
				nerr.AppendRequiredNotSet(string(fd.FullName()))
			}
			continue
		}

		name := fd.JSONName()
		if err := e.WriteName(name); !nerr.Merge(err) {
			return err
		}

		val := knownFields.Get(num)
		if err := e.marshalValue(val, fd); !nerr.Merge(err) {
			return err
		}
	}
	return nerr.E
}

// marshalValue marshals the given protoreflect.Value.
func (e encoder) marshalValue(val pref.Value, fd pref.FieldDescriptor) error {
	var nerr errors.NonFatal
	if fd.Cardinality() == pref.Repeated {
		// Map or repeated fields.
		if fd.IsMap() {
			if err := e.marshalMap(val.Map(), fd); !nerr.Merge(err) {
				return err
			}
		} else {
			if err := e.marshalList(val.List(), fd); !nerr.Merge(err) {
				return err
			}
		}
	} else {
		// Required or optional fields.
		if err := e.marshalSingular(val, fd); !nerr.Merge(err) {
			return err
		}
	}
	return nerr.E
}

// marshalSingular marshals the given non-repeated field value. This includes
// all scalar types, enums, messages, and groups.
func (e encoder) marshalSingular(val pref.Value, fd pref.FieldDescriptor) error {
	var nerr errors.NonFatal
	switch kind := fd.Kind(); kind {
	case pref.BoolKind:
		e.WriteBool(val.Bool())

	case pref.StringKind:
		if err := e.WriteString(val.String()); !nerr.Merge(err) {
			return err
		}

	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		e.WriteInt(val.Int())

	case pref.Uint32Kind, pref.Fixed32Kind:
		e.WriteUint(val.Uint())

	case pref.Int64Kind, pref.Sint64Kind, pref.Uint64Kind,
		pref.Sfixed64Kind, pref.Fixed64Kind:
		// 64-bit integers are written out as JSON string.
		e.WriteString(val.String())

	case pref.FloatKind:
		// Encoder.WriteFloat handles the special numbers NaN and infinites.
		e.WriteFloat(val.Float(), 32)

	case pref.DoubleKind:
		// Encoder.WriteFloat handles the special numbers NaN and infinites.
		e.WriteFloat(val.Float(), 64)

	case pref.BytesKind:
		err := e.WriteString(base64.StdEncoding.EncodeToString(val.Bytes()))
		if !nerr.Merge(err) {
			return err
		}

	case pref.EnumKind:
		num := val.Enum()
		if desc := fd.EnumType().Values().ByNumber(num); desc != nil {
			err := e.WriteString(string(desc.Name()))
			if !nerr.Merge(err) {
				return err
			}
		} else {
			// Use numeric value if there is no enum value descriptor.
			e.WriteInt(int64(num))
		}

	case pref.MessageKind, pref.GroupKind:
		if err := e.marshalMessage(val.Message()); !nerr.Merge(err) {
			return err
		}

	default:
		return errors.New("%v has unknown kind: %v", fd.FullName(), kind)
	}
	return nerr.E
}

// marshalList marshals the given protoreflect.List.
func (e encoder) marshalList(list pref.List, fd pref.FieldDescriptor) error {
	e.StartArray()
	defer e.EndArray()

	var nerr errors.NonFatal
	for i := 0; i < list.Len(); i++ {
		item := list.Get(i)
		if err := e.marshalSingular(item, fd); !nerr.Merge(err) {
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
func (e encoder) marshalMap(mmap pref.Map, fd pref.FieldDescriptor) error {
	e.StartObject()
	defer e.EndObject()

	msgFields := fd.MessageType().Fields()
	keyType := msgFields.ByNumber(1)
	valType := msgFields.ByNumber(2)

	// Get a sorted list based on keyType first.
	entries := make([]mapEntry, 0, mmap.Len())
	mmap.Range(func(key pref.MapKey, val pref.Value) bool {
		entries = append(entries, mapEntry{key: key, value: val})
		return true
	})
	sortMap(keyType.Kind(), entries)

	// Write out sorted list.
	var nerr errors.NonFatal
	for _, entry := range entries {
		if err := e.WriteName(entry.key.String()); !nerr.Merge(err) {
			return err
		}

		if err := e.marshalSingular(entry.value, valType); !nerr.Merge(err) {
			return err
		}
	}
	return nerr.E
}

// sortMap orders list based on value of key field for deterministic output.
func sortMap(keyKind pref.Kind, values []mapEntry) {
	less := func(i, j int) bool {
		return values[i].key.String() < values[j].key.String()
	}
	switch keyKind {
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind,
		pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		less = func(i, j int) bool {
			return values[i].key.Int() < values[j].key.Int()
		}
	case pref.Uint32Kind, pref.Fixed32Kind,
		pref.Uint64Kind, pref.Fixed64Kind:
		less = func(i, j int) bool {
			return values[i].key.Uint() < values[j].key.Uint()
		}
	}
	sort.Slice(values, less)
}
