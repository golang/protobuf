// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb

import (
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/golang/protobuf/v2/internal/encoding/json"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/pragma"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"

	descpb "github.com/golang/protobuf/v2/types/descriptor"
)

// Marshal writes the given proto.Message in JSON format using default options.
func Marshal(m proto.Message) ([]byte, error) {
	return MarshalOptions{}.Marshal(m)
}

// MarshalOptions is a configurable JSON format marshaler.
type MarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// If Indent is a non-empty string, it causes entries for an Array or Object
	// to be preceded by the indent and trailed by a newline. Indent can only be
	// composed of space or tab characters.
	Indent string

	// Resolver is the registry used for type lookups when marshaling
	// google.protobuf.Any messages. If Resolver is not set, marshaling will
	// default to using protoregistry.GlobalTypes.
	Resolver *protoregistry.Types
}

// Marshal marshals the given proto.Message in the JSON format using options in
// MarshalOptions.
func (o MarshalOptions) Marshal(m proto.Message) ([]byte, error) {
	enc, err := newEncoder(o.Indent, o.Resolver)
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
	resolver *protoregistry.Types
}

func newEncoder(indent string, resolver *protoregistry.Types) (encoder, error) {
	enc, err := json.NewEncoder(indent)
	if err != nil {
		return encoder{}, err
	}
	if resolver == nil {
		resolver = protoregistry.GlobalTypes
	}
	return encoder{
		Encoder:  enc,
		resolver: resolver,
	}, nil
}

// marshalMessage marshals the given protoreflect.Message.
func (e encoder) marshalMessage(m pref.Message) error {
	var nerr errors.NonFatal

	if isCustomType(m.Type().FullName()) {
		return e.marshalCustomType(m)
	}

	e.StartObject()
	defer e.EndObject()
	if err := e.marshalFields(m); !nerr.Merge(err) {
		return err
	}

	return nerr.E
}

// marshalFields marshals the fields in the given protoreflect.Message.
func (e encoder) marshalFields(m pref.Message) error {
	var nerr errors.NonFatal
	fieldDescs := m.Type().Fields()
	knownFields := m.KnownFields()

	// Marshal out known fields.
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

		// An empty google.protobuf.Value should NOT be marshaled out.
		// Hence need to check ahead for this.
		val := knownFields.Get(num)
		if isEmptyKnownValue(val, fd.MessageType()) {
			continue
		}

		name := fd.JSONName()
		if err := e.WriteName(name); !nerr.Merge(err) {
			return err
		}
		if err := e.marshalValue(val, fd); !nerr.Merge(err) {
			return err
		}
	}

	// Marshal out extensions.
	if err := e.marshalExtensions(knownFields); !nerr.Merge(err) {
		return err
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
		enumType := fd.EnumType()
		num := val.Enum()

		if enumType.FullName() == "google.protobuf.NullValue" {
			e.WriteNull()
		} else if desc := enumType.Values().ByNumber(num); desc != nil {
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
		panic(fmt.Sprintf("%v has unknown kind: %v", fd.FullName(), kind))
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
func (e encoder) marshalExtensions(knownFields pref.KnownFields) error {
	type xtEntry struct {
		key    string
		value  pref.Value
		xtType pref.ExtensionType
	}

	xtTypes := knownFields.ExtensionTypes()

	// Get a sorted list based on field key first.
	entries := make([]xtEntry, 0, xtTypes.Len())
	xtTypes.Range(func(xt pref.ExtensionType) bool {
		name := xt.FullName()
		// If extended type is a MessageSet, set field name to be the message type name.
		if isMessageSetExtension(xt) {
			name = xt.MessageType().FullName()
		}

		num := xt.Number()
		if knownFields.Has(num) {
			// Use [name] format for JSON field name.
			pval := knownFields.Get(num)
			entries = append(entries, xtEntry{
				key:    string(name),
				value:  pval,
				xtType: xt,
			})
		}
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
		if err := e.WriteName("[" + entry.key + "]"); !nerr.Merge(err) {
			return err
		}
		if err := e.marshalValue(entry.value, entry.xtType); !nerr.Merge(err) {
			return err
		}
	}
	return nerr.E
}

// isMessageSetExtension reports whether extension extends a message set.
func isMessageSetExtension(xt pref.ExtensionType) bool {
	if xt.Name() != "message_set_extension" {
		return false
	}
	mt := xt.MessageType()
	if mt == nil {
		return false
	}
	if xt.FullName().Parent() != mt.FullName() {
		return false
	}
	xmt := xt.ExtendedType()
	if xmt.Fields().Len() != 0 {
		return false
	}
	opt := xmt.Options().(*descpb.MessageOptions)
	if opt == nil {
		return false
	}
	return opt.GetMessageSetWireFormat()
}
