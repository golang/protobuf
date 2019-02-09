// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb

import (
	"encoding/base64"
	"math"
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

// Marshal writes the given proto.Message in JSON format using options in MarshalOptions object.
func (o MarshalOptions) Marshal(m proto.Message) ([]byte, error) {
	var nerr errors.NonFatal
	v, err := o.marshalMessage(m.ProtoReflect())
	if !nerr.Merge(err) {
		return nil, err
	}

	indent := "  "
	if o.Compact {
		indent = ""
	}

	b, err := json.Marshal(v, indent)
	if !nerr.Merge(err) {
		return nil, err
	}
	return b, nerr.E
}

// marshalMessage converts a protoreflect.Message to a json.Value.
func (o MarshalOptions) marshalMessage(m pref.Message) (json.Value, error) {
	var nerr errors.NonFatal
	var msgFields [][2]json.Value

	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()
	size := fieldDescs.Len()
	for i := 0; i < size; i++ {
		fd := fieldDescs.Get(i)
		num := fd.Number()

		if !knownFields.Has(num) {
			if fd.Cardinality() == pref.Required {
				// Treat unset required fields as a non-fatal error.
				nerr.AppendRequiredNotSet(string(fd.FullName()))
			}
			continue
		}

		name := json.ValueOf(fd.JSONName())
		pval := knownFields.Get(num)
		var err error
		msgFields, err = o.appendField(msgFields, name, pval, fd)
		if !nerr.Merge(err) {
			return json.Value{}, err
		}
	}

	return json.ValueOf(msgFields), nerr.E
}

// appendField marshals a protoreflect.Value and appends it to the given
// [][2]json.Value.
func (o MarshalOptions) appendField(msgFields [][2]json.Value, name json.Value, pval pref.Value, fd pref.FieldDescriptor) ([][2]json.Value, error) {
	var nerr errors.NonFatal
	var jval json.Value
	var err error

	if fd.Cardinality() == pref.Repeated {
		// Map or repeated fields.
		if fd.IsMap() {
			jval, err = o.marshalMap(pval.Map(), fd)
			if !nerr.Merge(err) {
				return msgFields, err
			}
		} else {
			jval, err = o.marshalList(pval.List(), fd)
			if !nerr.Merge(err) {
				return msgFields, err
			}
		}
	} else {
		// Required or optional fields.
		jval, err = o.marshalSingular(pval, fd)
		if !nerr.Merge(err) {
			return msgFields, err
		}
	}

	msgFields = append(msgFields, [2]json.Value{name, jval})
	return msgFields, nerr.E
}

// marshalSingular converts a non-repeated field value to json.Value.
// This includes all scalar types, enums, messages, and groups.
func (o MarshalOptions) marshalSingular(val pref.Value, fd pref.FieldDescriptor) (json.Value, error) {
	kind := fd.Kind()
	switch kind {
	case pref.BoolKind, pref.StringKind,
		pref.Int32Kind, pref.Sint32Kind, pref.Uint32Kind,
		pref.Sfixed32Kind, pref.Fixed32Kind:
		return json.ValueOf(val.Interface()), nil

	case pref.Int64Kind, pref.Sint64Kind, pref.Uint64Kind,
		pref.Sfixed64Kind, pref.Fixed64Kind:
		return json.ValueOf(val.String()), nil

	case pref.FloatKind, pref.DoubleKind:
		n := val.Float()
		switch {
		case math.IsNaN(n):
			return json.ValueOf("NaN"), nil
		case math.IsInf(n, +1):
			return json.ValueOf("Infinity"), nil
		case math.IsInf(n, -1):
			return json.ValueOf("-Infinity"), nil
		default:
			return json.ValueOf(n), nil
		}

	case pref.BytesKind:
		return json.ValueOf(base64.StdEncoding.EncodeToString(val.Bytes())), nil

	case pref.EnumKind:
		num := val.Enum()
		if desc := fd.EnumType().Values().ByNumber(num); desc != nil {
			return json.ValueOf(string(desc.Name())), nil
		}
		// Use numeric value if there is no enum value descriptor.
		return json.ValueOf(int32(num)), nil

	case pref.MessageKind, pref.GroupKind:
		return o.marshalMessage(val.Message())
	}

	return json.Value{}, errors.New("%v has unknown kind: %v", fd.FullName(), kind)
}

// marshalList converts a protoreflect.List to json.Value.
func (o MarshalOptions) marshalList(list pref.List, fd pref.FieldDescriptor) (json.Value, error) {
	var nerr errors.NonFatal
	size := list.Len()
	values := make([]json.Value, 0, size)

	for i := 0; i < size; i++ {
		item := list.Get(i)
		val, err := o.marshalSingular(item, fd)
		if !nerr.Merge(err) {
			return json.Value{}, err
		}
		values = append(values, val)
	}

	return json.ValueOf(values), nerr.E
}

type mapEntry struct {
	key   pref.MapKey
	value pref.Value
}

// marshalMap converts a protoreflect.Map to json.Value.
func (o MarshalOptions) marshalMap(mmap pref.Map, fd pref.FieldDescriptor) (json.Value, error) {
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

	// Convert to list of [2]json.Value.
	var nerr errors.NonFatal
	values := make([][2]json.Value, 0, len(entries))
	for _, entry := range entries {
		jkey := json.ValueOf(entry.key.String())
		jval, err := o.marshalSingular(entry.value, valType)
		if !nerr.Merge(err) {
			return json.Value{}, err
		}
		values = append(values, [2]json.Value{jkey, jval})
	}
	return json.ValueOf(values), nerr.E
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
