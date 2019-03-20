// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/fieldnum"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// isCustomType returns true if type name has special JSON conversion rules.
// The list of custom types here has to match the ones in marshalCustomTypes.
func isCustomType(name pref.FullName) bool {
	switch name {
	case "google.protobuf.Any",
		"google.protobuf.BoolValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.FloatValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.StringValue",
		"google.protobuf.BytesValue",
		"google.protobuf.Struct",
		"google.protobuf.ListValue",
		"google.protobuf.Value",
		"google.protobuf.Duration",
		"google.protobuf.Timestamp",
		"google.protobuf.FieldMask":
		return true
	}
	return false
}

// marshalCustomType marshals given well-known type message that have special
// JSON conversion rules. It needs to be a message type where isCustomType
// returns true, else it will panic.
func (e encoder) marshalCustomType(m pref.Message) error {
	name := m.Type().FullName()
	switch name {
	case "google.protobuf.Any":
		return e.marshalAny(m)

	case "google.protobuf.BoolValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.FloatValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.StringValue",
		"google.protobuf.BytesValue":
		return e.marshalKnownScalar(m)

	case "google.protobuf.Struct":
		return e.marshalStruct(m)

	case "google.protobuf.ListValue":
		return e.marshalListValue(m)

	case "google.protobuf.Value":
		return e.marshalKnownValue(m)

	case "google.protobuf.Duration":
		return e.marshalDuration(m)

	case "google.protobuf.Timestamp":
		return e.marshalTimestamp(m)

	case "google.protobuf.FieldMask":
		return e.marshalFieldMask(m)
	}

	panic(fmt.Sprintf("encoder.marshalCustomTypes(%q) does not have a custom marshaler", name))
}

func (e encoder) marshalAny(m pref.Message) error {
	var nerr errors.NonFatal
	msgType := m.Type()
	knownFields := m.KnownFields()

	// Start writing the JSON object.
	e.StartObject()
	defer e.EndObject()

	if !knownFields.Has(fieldnum.Any_TypeUrl) {
		if !knownFields.Has(fieldnum.Any_Value) {
			// If message is empty, marshal out empty JSON object.
			return nil
		} else {
			// Return error if type_url field is not set, but value is set.
			return errors.New("field %s.type_url is not set", msgType.FullName())
		}
	}

	typeVal := knownFields.Get(fieldnum.Any_TypeUrl)
	valueVal := knownFields.Get(fieldnum.Any_Value)

	// Marshal out @type field.
	typeURL := typeVal.String()
	e.WriteName("@type")
	if err := e.WriteString(typeURL); !nerr.Merge(err) {
		return err
	}

	// Resolve the type in order to unmarshal value field.
	emt, err := e.resolver.FindMessageByURL(typeURL)
	if !nerr.Merge(err) {
		return errors.New("unable to resolve %v: %v", typeURL, err)
	}

	em := emt.New()
	// TODO: Need to set types registry in binary unmarshaling.
	err = proto.Unmarshal(valueVal.Bytes(), em.Interface())
	if !nerr.Merge(err) {
		return errors.New("unable to unmarshal %v: %v", typeURL, err)
	}

	// If type of value has custom JSON encoding, marshal out a field "value"
	// with corresponding custom JSON encoding of the embedded message as a
	// field.
	if isCustomType(emt.FullName()) {
		// An empty google.protobuf.Value should NOT be marshaled out.
		if isEmptyKnownValue(pref.ValueOf(em), emt) {
			return nil
		}
		e.WriteName("value")
		return e.marshalCustomType(em)
	}

	// Else, marshal out the embedded message's fields in this Any object.
	if err := e.marshalFields(em); !nerr.Merge(err) {
		return err
	}

	return nerr.E
}

func (e encoder) marshalKnownScalar(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	// The "value" field has the same field number for all wrapper types.
	const num = fieldnum.BoolValue_Value
	fd := fieldDescs.ByNumber(num)
	val := knownFields.Get(num)
	return e.marshalSingular(val, fd)
}

func (e encoder) marshalStruct(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	fd := fieldDescs.ByNumber(fieldnum.Struct_Fields)
	val := knownFields.Get(fieldnum.Struct_Fields)
	return e.marshalMap(val.Map(), fd)
}

func (e encoder) marshalListValue(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	fd := fieldDescs.ByNumber(fieldnum.ListValue_Values)
	val := knownFields.Get(fieldnum.ListValue_Values)
	return e.marshalList(val.List(), fd)
}

// isEmptyKnownValue returns true if given val is of type google.protobuf.Value
// and does not have any of its oneof fields set.
func isEmptyKnownValue(val pref.Value, md pref.MessageDescriptor) bool {
	return md != nil &&
		md.FullName() == "google.protobuf.Value" &&
		val.Message().KnownFields().Len() == 0
}

func (e encoder) marshalKnownValue(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Oneofs().Get(0).Fields()
	knownFields := m.KnownFields()

	for i := 0; i < fieldDescs.Len(); i++ {
		fd := fieldDescs.Get(i)
		num := fd.Number()
		if !knownFields.Has(num) {
			continue
		}
		// Only one field should be set.
		val := knownFields.Get(num)
		return e.marshalSingular(val, fd)
	}

	// None of the fields are set.
	return nil
}

const (
	secondsInNanos       = int64(time.Second / time.Nanosecond)
	maxSecondsInDuration = int64(315576000000)
)

func (e encoder) marshalDuration(m pref.Message) error {
	msgType := m.Type()
	knownFields := m.KnownFields()

	secsVal := knownFields.Get(fieldnum.Duration_Seconds)
	nanosVal := knownFields.Get(fieldnum.Duration_Nanos)
	secs := secsVal.Int()
	nanos := nanosVal.Int()
	if secs < -maxSecondsInDuration || secs > maxSecondsInDuration {
		return errors.New("%s.seconds out of range", msgType.FullName())
	}
	if nanos <= -secondsInNanos || nanos >= secondsInNanos {
		return errors.New("%s.nanos out of range", msgType.FullName())
	}
	if (secs > 0 && nanos < 0) || (secs < 0 && nanos > 0) {
		return errors.New("%s: signs of seconds and nanos do not match", msgType.FullName())
	}
	// Generated output always contains 0, 3, 6, or 9 fractional digits,
	// depending on required precision, followed by the suffix "s".
	f := "%d.%09d"
	if nanos < 0 {
		nanos = -nanos
		if secs == 0 {
			f = "-%d.%09d"
		}
	}
	x := fmt.Sprintf(f, secs, nanos)
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, ".000")
	e.WriteString(x + "s")
	return nil
}

const (
	maxTimestampSeconds = 253402300799
	minTimestampSeconds = -62135596800
)

func (e encoder) marshalTimestamp(m pref.Message) error {
	msgType := m.Type()
	knownFields := m.KnownFields()

	secsVal := knownFields.Get(fieldnum.Timestamp_Seconds)
	nanosVal := knownFields.Get(fieldnum.Timestamp_Nanos)
	secs := secsVal.Int()
	nanos := nanosVal.Int()
	if secs < minTimestampSeconds || secs > maxTimestampSeconds {
		return errors.New("%s.seconds out of range", msgType.FullName())
	}
	if nanos < 0 || nanos >= secondsInNanos {
		return errors.New("%s.nanos out of range", msgType.FullName())
	}
	// Uses RFC 3339, where generated output will be Z-normalized and uses 0, 3,
	// 6 or 9 fractional digits.
	t := time.Unix(secs, nanos).UTC()
	x := t.Format("2006-01-02T15:04:05.000000000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, ".000")
	e.WriteString(x + "Z")
	return nil
}

func (e encoder) marshalFieldMask(m pref.Message) error {
	msgType := m.Type()
	knownFields := m.KnownFields()
	name := msgType.FullName()

	val := knownFields.Get(fieldnum.FieldMask_Paths)
	list := val.List()
	paths := make([]string, 0, list.Len())

	for i := 0; i < list.Len(); i++ {
		s := list.Get(i).String()
		// Return error if conversion to camelCase is not reversible.
		cc := camelCase(s)
		if s != snakeCase(cc) {
			return errors.New("%s.paths contains irreversible value %q", name, s)
		}
		paths = append(paths, cc)
	}

	e.WriteString(strings.Join(paths, ","))
	return nil
}

// camelCase converts given string into camelCase where ASCII character after _
// is turned into uppercase and _'s are removed.
func camelCase(s string) string {
	var b []byte
	var afterUnderscore bool
	for i := 0; i < len(s); i++ {
		c := s[i]
		if afterUnderscore {
			if isASCIILower(c) {
				c -= 'a' - 'A'
			}
		}
		if c == '_' {
			afterUnderscore = true
			continue
		}
		afterUnderscore = false
		b = append(b, c)
	}
	return string(b)
}

// snakeCase converts given string into snake_case where ASCII uppercase
// character is turned into _ + lowercase.
func snakeCase(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isASCIIUpper(c) {
			c += 'a' - 'A'
			b = append(b, '_', c)
		} else {
			b = append(b, c)
		}
	}
	return string(b)
}

func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

func isASCIIUpper(c byte) bool {
	return 'A' <= c && c <= 'Z'
}
