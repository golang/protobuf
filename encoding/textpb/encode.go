// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpb

import (
	"fmt"
	"sort"

	"github.com/golang/protobuf/v2/internal/encoding/text"
	"github.com/golang/protobuf/v2/internal/encoding/wire"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/pragma"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"

	descpb "github.com/golang/protobuf/v2/types/descriptor"
)

// Marshal writes the given proto.Message in textproto format using default options.
// TODO: may want to describe when Marshal returns error.
func Marshal(m proto.Message) ([]byte, error) {
	return MarshalOptions{}.Marshal(m)
}

// MarshalOptions is a configurable text format marshaler.
type MarshalOptions struct {
	pragma.NoUnkeyedLiterals

	// Set Compact to true to have output in a single line with no line breaks.
	Compact bool

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

	indent := "  "
	if o.Compact {
		indent = ""
	}
	delims := [2]byte{'{', '}'}

	const outputASCII = false
	b, err := text.Marshal(v, indent, delims, outputASCII)
	if !nerr.Merge(err) {
		return nil, err
	}
	return b, nerr.E
}

// marshalMessage converts a protoreflect.Message to a text.Value.
func (o MarshalOptions) marshalMessage(m pref.Message) (text.Value, error) {
	var nerr errors.NonFatal
	var msgFields [][2]text.Value
	msgType := m.Type()

	// Handle Any expansion.
	if msgType.FullName() == "google.protobuf.Any" {
		msg, err := o.marshalAny(m)
		if err == nil || nerr.Merge(err) {
			// Return as is for nil or non-fatal error.
			return msg, nerr.E
		}
		// For other errors, continue on to marshal Any as a regular message.
	}

	// Handle known fields.
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

		name := text.ValueOf(fd.Name())
		// Use type name for group field name.
		if fd.Kind() == pref.GroupKind {
			name = text.ValueOf(fd.MessageType().Name())
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

	if fd.Cardinality() == pref.Repeated {
		// Map or repeated fields.
		var items []text.Value
		var err error
		if fd.IsMap() {
			items, err = o.marshalMap(pval.Map(), fd)
			if !nerr.Merge(err) {
				return msgFields, err
			}
		} else {
			items, err = o.marshalList(pval.List(), fd)
			if !nerr.Merge(err) {
				return msgFields, err
			}
		}

		// Add each item as key: value field.
		for _, item := range items {
			msgFields = append(msgFields, [2]text.Value{name, item})
		}
	} else {
		// Required or optional fields.
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
		pref.StringKind, pref.BytesKind:
		return text.ValueOf(val.Interface()), nil

	case pref.EnumKind:
		num := val.Enum()
		if desc := fd.EnumType().Values().ByNumber(num); desc != nil {
			return text.ValueOf(desc.Name()), nil
		}
		// Use numeric value if there is no enum description.
		return text.ValueOf(int32(num)), nil

	case pref.MessageKind, pref.GroupKind:
		return o.marshalMessage(val.Message())
	}

	return text.Value{}, errors.New("%v has unknown kind: %v", fd.FullName(), kind)
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
	msgFields := fd.MessageType().Fields()
	keyType := msgFields.ByNumber(1)
	valType := msgFields.ByNumber(2)

	mmap.Range(func(key pref.MapKey, val pref.Value) bool {
		keyTxtVal, err := o.marshalSingular(key.Value(), keyType)
		if !nerr.Merge(err) {
			return false
		}
		valTxtVal, err := o.marshalSingular(val, valType)
		if !nerr.Merge(err) {
			return false
		}
		// Map entry (message) contains 2 fields, first field for key and second field for value.
		msg := text.ValueOf([][2]text.Value{
			{mapKeyName, keyTxtVal},
			{mapValueName, valTxtVal},
		})
		values = append(values, msg)
		return true
	})

	sortMap(keyType.Kind(), values)
	return values, nerr.E
}

// sortMap orders list based on value of key field for deterministic output.
// TODO: Improve sort comparison of text.Value for map keys.
func sortMap(keyKind pref.Kind, values []text.Value) {
	less := func(i, j int) bool {
		mi := values[i].Message()
		mj := values[j].Message()
		return mi[0][1].String() < mj[0][1].String()
	}
	switch keyKind {
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		less = func(i, j int) bool {
			mi := values[i].Message()
			mj := values[j].Message()
			ni, _ := mi[0][1].Int(false)
			nj, _ := mj[0][1].Int(false)
			return ni < nj
		}
	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		less = func(i, j int) bool {
			mi := values[i].Message()
			mj := values[j].Message()
			ni, _ := mi[0][1].Int(true)
			nj, _ := mj[0][1].Int(true)
			return ni < nj
		}

	case pref.Uint32Kind, pref.Fixed32Kind:
		less = func(i, j int) bool {
			mi := values[i].Message()
			mj := values[j].Message()
			ni, _ := mi[0][1].Uint(false)
			nj, _ := mj[0][1].Uint(false)
			return ni < nj
		}
	case pref.Uint64Kind, pref.Fixed64Kind:
		less = func(i, j int) bool {
			mi := values[i].Message()
			mj := values[j].Message()
			ni, _ := mi[0][1].Uint(true)
			nj, _ := mj[0][1].Uint(true)
			return ni < nj
		}
	}
	sort.Slice(values, less)
}

// appendExtensions marshals extension fields and appends them to the given [][2]text.Value.
func (o MarshalOptions) appendExtensions(msgFields [][2]text.Value, knownFields pref.KnownFields) ([][2]text.Value, error) {
	var nerr errors.NonFatal
	xtTypes := knownFields.ExtensionTypes()
	xtFields := make([][2]text.Value, 0, xtTypes.Len())

	var err error
	xtTypes.Range(func(xt pref.ExtensionType) bool {
		name := xt.FullName()
		// If extended type is a MessageSet, set field name to be the message type name.
		if isMessageSetExtension(xt) {
			name = xt.MessageType().FullName()
		}

		num := xt.Number()
		if knownFields.Has(num) {
			// Use string type to produce [name] format.
			tname := text.ValueOf(string(name))
			pval := knownFields.Get(num)
			xtFields, err = o.appendField(xtFields, tname, pval, xt)
			if err != nil {
				return false
			}
		}
		return true
	})
	if !nerr.Merge(err) {
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

	fds := m.Type().Fields()
	tfd := fds.ByName("type_url")
	if tfd == nil || tfd.Kind() != pref.StringKind {
		return text.Value{}, errors.New("invalid google.protobuf.Any message")
	}
	vfd := fds.ByName("value")
	if vfd == nil || vfd.Kind() != pref.BytesKind {
		return text.Value{}, errors.New("invalid google.protobuf.Any message")
	}

	knownFields := m.KnownFields()
	typeURL := knownFields.Get(tfd.Number()).String()
	value := knownFields.Get(vfd.Number())

	emt, err := o.Resolver.FindMessageByURL(typeURL)
	if !nerr.Merge(err) {
		return text.Value{}, err
	}
	em := emt.New().Interface()
	// TODO: Need to set types registry in binary unmarshaling.
	err = proto.Unmarshal(value.Bytes(), em)
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
