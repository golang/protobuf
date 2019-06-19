// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protojson

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/internal/encoding/json"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/fieldnum"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// isCustomType returns true if type name has special JSON conversion rules.
// The list of custom types here has to match the ones in marshalCustomType and
// unmarshalCustomType.
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
		"google.protobuf.Empty",
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
func (o MarshalOptions) marshalCustomType(m pref.Message) error {
	name := m.Descriptor().FullName()
	switch name {
	case "google.protobuf.Any":
		return o.marshalAny(m)

	case "google.protobuf.BoolValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.FloatValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.StringValue",
		"google.protobuf.BytesValue":
		return o.marshalWrapperType(m)

	case "google.protobuf.Empty":
		return o.marshalEmpty(m)

	case "google.protobuf.Struct":
		return o.marshalStruct(m)

	case "google.protobuf.ListValue":
		return o.marshalListValue(m)

	case "google.protobuf.Value":
		return o.marshalKnownValue(m)

	case "google.protobuf.Duration":
		return o.marshalDuration(m)

	case "google.protobuf.Timestamp":
		return o.marshalTimestamp(m)

	case "google.protobuf.FieldMask":
		return o.marshalFieldMask(m)
	}

	panic(fmt.Sprintf("%q does not have a custom marshaler", name))
}

// unmarshalCustomType unmarshals given well-known type message that have
// special JSON conversion rules. It needs to be a message type where
// isCustomType returns true, else it will panic.
func (o UnmarshalOptions) unmarshalCustomType(m pref.Message) error {
	name := m.Descriptor().FullName()
	switch name {
	case "google.protobuf.Any":
		return o.unmarshalAny(m)

	case "google.protobuf.BoolValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.FloatValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value",
		"google.protobuf.StringValue",
		"google.protobuf.BytesValue":
		return o.unmarshalWrapperType(m)

	case "google.protobuf.Empty":
		return o.unmarshalEmpty(m)

	case "google.protobuf.Struct":
		return o.unmarshalStruct(m)

	case "google.protobuf.ListValue":
		return o.unmarshalListValue(m)

	case "google.protobuf.Value":
		return o.unmarshalKnownValue(m)

	case "google.protobuf.Duration":
		return o.unmarshalDuration(m)

	case "google.protobuf.Timestamp":
		return o.unmarshalTimestamp(m)

	case "google.protobuf.FieldMask":
		return o.unmarshalFieldMask(m)
	}

	panic(fmt.Sprintf("%q does not have a custom unmarshaler", name))
}

// The JSON representation of an Any message uses the regular representation of
// the deserialized, embedded message, with an additional field `@type` which
// contains the type URL. If the embedded message type is well-known and has a
// custom JSON representation, that representation will be embedded adding a
// field `value` which holds the custom JSON in addition to the `@type` field.

func (o MarshalOptions) marshalAny(m pref.Message) error {
	fds := m.Descriptor().Fields()
	fdType := fds.ByNumber(fieldnum.Any_TypeUrl)
	fdValue := fds.ByNumber(fieldnum.Any_Value)

	// Start writing the JSON object.
	o.encoder.StartObject()
	defer o.encoder.EndObject()

	if !m.Has(fdType) {
		if !m.Has(fdValue) {
			// If message is empty, marshal out empty JSON object.
			return nil
		} else {
			// Return error if type_url field is not set, but value is set.
			return errors.New("%s: type_url is not set", m.Descriptor().FullName())
		}
	}

	typeVal := m.Get(fdType)
	valueVal := m.Get(fdValue)

	// Marshal out @type field.
	typeURL := typeVal.String()
	o.encoder.WriteName("@type")
	var nerr errors.NonFatal
	if err := o.encoder.WriteString(typeURL); !nerr.Merge(err) {
		return err
	}

	// Resolve the type in order to unmarshal value field.
	emt, err := o.Resolver.FindMessageByURL(typeURL)
	if !nerr.Merge(err) {
		return errors.New("%s: unable to resolve %q: %v", m.Descriptor().FullName(), typeURL, err)
	}

	em := emt.New()
	// TODO: If binary unmarshaling returns required not set error, need to
	// return another required not set error that contains both the path to this
	// field and the path inside the embedded message.
	err = proto.UnmarshalOptions{
		AllowPartial: o.AllowPartial,
		Resolver:     o.Resolver,
	}.Unmarshal(valueVal.Bytes(), em.Interface())
	if !nerr.Merge(err) {
		return errors.New("%s: unable to unmarshal %q: %v", m.Descriptor().FullName(), typeURL, err)
	}

	// If type of value has custom JSON encoding, marshal out a field "value"
	// with corresponding custom JSON encoding of the embedded message as a
	// field.
	if isCustomType(emt.Descriptor().FullName()) {
		o.encoder.WriteName("value")
		return o.marshalCustomType(em)
	}

	// Else, marshal out the embedded message's fields in this Any object.
	if err := o.marshalFields(em); !nerr.Merge(err) {
		return err
	}

	return nerr.E
}

func (o UnmarshalOptions) unmarshalAny(m pref.Message) error {
	// Use Peek to check for json.StartObject to avoid advancing a read.
	if o.decoder.Peek() != json.StartObject {
		jval, _ := o.decoder.Read()
		return unexpectedJSONError{jval}
	}

	// Use another json.Decoder to parse the unread bytes from o.decoder for
	// @type field. This avoids advancing a read from o.decoder because the
	// current JSON object may contain the fields of the embedded type.
	dec := o.decoder.Clone()
	typeURL, err := findTypeURL(dec)
	if err == errEmptyObject {
		// An empty JSON object translates to an empty Any message.
		o.decoder.Read() // Read json.StartObject.
		o.decoder.Read() // Read json.EndObject.
		return nil
	}
	if o.DiscardUnknown && err == errMissingType {
		// Treat all fields as unknowns, similar to an empty object.
		return skipJSONValue(o.decoder)
	}
	var nerr errors.NonFatal
	if !nerr.Merge(err) {
		return errors.New("google.protobuf.Any: %v", err)
	}

	emt, err := o.Resolver.FindMessageByURL(typeURL)
	if err != nil {
		return errors.New("google.protobuf.Any: unable to resolve type %q: %v", typeURL, err)
	}

	// Create new message for the embedded message type and unmarshal into it.
	em := emt.New()
	if isCustomType(emt.Descriptor().FullName()) {
		// If embedded message is a custom type, unmarshal the JSON "value" field
		// into it.
		if err := o.unmarshalAnyValue(em); !nerr.Merge(err) {
			return errors.New("google.protobuf.Any: %v", err)
		}
	} else {
		// Else unmarshal the current JSON object into it.
		if err := o.unmarshalMessage(em, true); !nerr.Merge(err) {
			return errors.New("google.protobuf.Any: %v", err)
		}
	}
	// Serialize the embedded message and assign the resulting bytes to the
	// proto value field.
	// TODO: If binary marshaling returns required not set error, need to return
	// another required not set error that contains both the path to this field
	// and the path inside the embedded message.
	b, err := proto.MarshalOptions{
		AllowPartial:  o.AllowPartial,
		Deterministic: true,
	}.Marshal(em.Interface())
	if !nerr.Merge(err) {
		return errors.New("google.protobuf.Any: %v", err)
	}

	fds := m.Descriptor().Fields()
	fdType := fds.ByNumber(fieldnum.Any_TypeUrl)
	fdValue := fds.ByNumber(fieldnum.Any_Value)

	m.Set(fdType, pref.ValueOf(typeURL))
	m.Set(fdValue, pref.ValueOf(b))
	return nerr.E
}

var errEmptyObject = errors.New(`empty object`)
var errMissingType = errors.New(`missing "@type" field`)

// findTypeURL returns the "@type" field value from the given JSON bytes. It is
// expected that the given bytes start with json.StartObject. It returns
// errEmptyObject if the JSON object is empty. It returns error if the object
// does not contain the field or other decoding problems.
func findTypeURL(dec *json.Decoder) (string, error) {
	var typeURL string
	var nerr errors.NonFatal
	numFields := 0
	// Skip start object.
	dec.Read()

Loop:
	for {
		jval, err := dec.Read()
		if !nerr.Merge(err) {
			return "", err
		}

		switch jval.Type() {
		case json.EndObject:
			if typeURL == "" {
				// Did not find @type field.
				if numFields > 0 {
					return "", errMissingType
				}
				return "", errEmptyObject
			}
			break Loop

		case json.Name:
			numFields++
			name, err := jval.Name()
			if !nerr.Merge(err) {
				return "", err
			}
			if name != "@type" {
				// Skip value.
				if err := skipJSONValue(dec); !nerr.Merge(err) {
					return "", err
				}
				continue
			}

			// Return error if this was previously set already.
			if typeURL != "" {
				return "", errors.New(`duplicate "@type" field`)
			}
			// Read field value.
			jval, err := dec.Read()
			if !nerr.Merge(err) {
				return "", err
			}
			if jval.Type() != json.String {
				return "", unexpectedJSONError{jval}
			}
			typeURL = jval.String()
			if typeURL == "" {
				return "", errors.New(`"@type" field contains empty value`)
			}
		}
	}

	return typeURL, nerr.E
}

// skipJSONValue makes the given decoder parse a JSON value (null, boolean,
// string, number, object and array) in order to advance the read to the next
// JSON value. It relies on Decoder.Read returning an error if the types are
// not in valid sequence.
func skipJSONValue(dec *json.Decoder) error {
	var nerr errors.NonFatal
	jval, err := dec.Read()
	if !nerr.Merge(err) {
		return err
	}
	// Only need to continue reading for objects and arrays.
	switch jval.Type() {
	case json.StartObject:
		for {
			jval, err := dec.Read()
			if !nerr.Merge(err) {
				return err
			}
			switch jval.Type() {
			case json.EndObject:
				return nil
			case json.Name:
				// Skip object field value.
				if err := skipJSONValue(dec); !nerr.Merge(err) {
					return err
				}
			}
		}

	case json.StartArray:
		for {
			switch dec.Peek() {
			case json.EndArray:
				dec.Read()
				return nil
			case json.Invalid:
				_, err := dec.Read()
				return err
			default:
				// Skip array item.
				if err := skipJSONValue(dec); !nerr.Merge(err) {
					return err
				}
			}
		}
	}
	return nerr.E
}

// unmarshalAnyValue unmarshals the given custom-type message from the JSON
// object's "value" field.
func (o UnmarshalOptions) unmarshalAnyValue(m pref.Message) error {
	var nerr errors.NonFatal
	// Skip StartObject, and start reading the fields.
	o.decoder.Read()

	var found bool // Used for detecting duplicate "value".
	for {
		jval, err := o.decoder.Read()
		if !nerr.Merge(err) {
			return err
		}
		switch jval.Type() {
		case json.EndObject:
			if !found {
				return errors.New(`missing "value" field`)
			}
			return nerr.E

		case json.Name:
			name, err := jval.Name()
			if !nerr.Merge(err) {
				return err
			}
			switch name {
			default:
				if o.DiscardUnknown {
					if err := skipJSONValue(o.decoder); !nerr.Merge(err) {
						return err
					}
					continue
				}
				return errors.New("unknown field %q", name)

			case "@type":
				// Skip the value as this was previously parsed already.
				o.decoder.Read()

			case "value":
				if found {
					return errors.New(`duplicate "value" field`)
				}
				// Unmarshal the field value into the given message.
				if err := o.unmarshalCustomType(m); !nerr.Merge(err) {
					return err
				}
				found = true
			}
		}
	}
}

// Wrapper types are encoded as JSON primitives like string, number or boolean.

// The "value" field has the same field number for all wrapper types.
const wrapperFieldNumber = fieldnum.BoolValue_Value

func (o MarshalOptions) marshalWrapperType(m pref.Message) error {
	fd := m.Descriptor().Fields().ByNumber(wrapperFieldNumber)
	val := m.Get(fd)
	return o.marshalSingular(val, fd)
}

func (o UnmarshalOptions) unmarshalWrapperType(m pref.Message) error {
	fd := m.Descriptor().Fields().ByNumber(wrapperFieldNumber)
	val, err := o.unmarshalScalar(fd)
	var nerr errors.NonFatal
	if !nerr.Merge(err) {
		return err
	}
	m.Set(fd, val)
	return nerr.E
}

// The JSON representation for Empty is an empty JSON object.

func (o MarshalOptions) marshalEmpty(pref.Message) error {
	o.encoder.StartObject()
	o.encoder.EndObject()
	return nil
}

func (o UnmarshalOptions) unmarshalEmpty(pref.Message) error {
	var nerr errors.NonFatal
	jval, err := o.decoder.Read()
	if err != nil {
		return err
	}
	if jval.Type() != json.StartObject {
		return unexpectedJSONError{jval}
	}

	for {
		jval, err := o.decoder.Read()
		if !nerr.Merge(err) {
			return err
		}
		switch jval.Type() {
		case json.EndObject:
			return nerr.E

		case json.Name:
			if o.DiscardUnknown {
				if err := skipJSONValue(o.decoder); !nerr.Merge(err) {
					return err
				}
				continue
			}
			name, _ := jval.Name()
			return errors.New("unknown field %q", name)

		default:
			return unexpectedJSONError{jval}
		}
	}
}

// The JSON representation for Struct is a JSON object that contains the encoded
// Struct.fields map and follows the serialization rules for a map.

func (o MarshalOptions) marshalStruct(m pref.Message) error {
	fd := m.Descriptor().Fields().ByNumber(fieldnum.Struct_Fields)
	return o.marshalMap(m.Get(fd).Map(), fd)
}

func (o UnmarshalOptions) unmarshalStruct(m pref.Message) error {
	fd := m.Descriptor().Fields().ByNumber(fieldnum.Struct_Fields)
	return o.unmarshalMap(m.Mutable(fd).Map(), fd)
}

// The JSON representation for ListValue is JSON array that contains the encoded
// ListValue.values repeated field and follows the serialization rules for a
// repeated field.

func (o MarshalOptions) marshalListValue(m pref.Message) error {
	fd := m.Descriptor().Fields().ByNumber(fieldnum.ListValue_Values)
	return o.marshalList(m.Get(fd).List(), fd)
}

func (o UnmarshalOptions) unmarshalListValue(m pref.Message) error {
	fd := m.Descriptor().Fields().ByNumber(fieldnum.ListValue_Values)
	return o.unmarshalList(m.Mutable(fd).List(), fd)
}

// The JSON representation for a Value is dependent on the oneof field that is
// set. Each of the field in the oneof has its own custom serialization rule. A
// Value message needs to be a oneof field set, else it is an error.

func (o MarshalOptions) marshalKnownValue(m pref.Message) error {
	od := m.Descriptor().Oneofs().ByName("kind")
	fd := m.WhichOneof(od)
	if fd == nil {
		return errors.New("%s: none of the oneof fields is set", m.Descriptor().FullName())
	}
	return o.marshalSingular(m.Get(fd), fd)
}

func (o UnmarshalOptions) unmarshalKnownValue(m pref.Message) error {
	var nerr errors.NonFatal
	switch o.decoder.Peek() {
	case json.Null:
		o.decoder.Read()
		fd := m.Descriptor().Fields().ByNumber(fieldnum.Value_NullValue)
		m.Set(fd, pref.ValueOf(pref.EnumNumber(0)))

	case json.Bool:
		jval, err := o.decoder.Read()
		if err != nil {
			return err
		}
		val, err := unmarshalBool(jval)
		if err != nil {
			return err
		}
		fd := m.Descriptor().Fields().ByNumber(fieldnum.Value_BoolValue)
		m.Set(fd, val)

	case json.Number:
		jval, err := o.decoder.Read()
		if err != nil {
			return err
		}
		val, err := unmarshalFloat(jval, 64)
		if err != nil {
			return err
		}
		fd := m.Descriptor().Fields().ByNumber(fieldnum.Value_NumberValue)
		m.Set(fd, val)

	case json.String:
		// A JSON string may have been encoded from the number_value field,
		// e.g. "NaN", "Infinity", etc. Parsing a proto double type also allows
		// for it to be in JSON string form. Given this custom encoding spec,
		// however, there is no way to identify that and hence a JSON string is
		// always assigned to the string_value field, which means that certain
		// encoding cannot be parsed back to the same field.
		jval, err := o.decoder.Read()
		if !nerr.Merge(err) {
			return err
		}
		val, err := unmarshalString(jval)
		if !nerr.Merge(err) {
			return err
		}
		fd := m.Descriptor().Fields().ByNumber(fieldnum.Value_StringValue)
		m.Set(fd, val)

	case json.StartObject:
		fd := m.Descriptor().Fields().ByNumber(fieldnum.Value_StructValue)
		m2 := m.NewMessage(fd)
		if err := o.unmarshalStruct(m2); !nerr.Merge(err) {
			return err
		}
		m.Set(fd, pref.ValueOf(m2))

	case json.StartArray:
		fd := m.Descriptor().Fields().ByNumber(fieldnum.Value_ListValue)
		m2 := m.NewMessage(fd)
		if err := o.unmarshalListValue(m2); !nerr.Merge(err) {
			return err
		}
		m.Set(fd, pref.ValueOf(m2))

	default:
		jval, err := o.decoder.Read()
		if err != nil {
			return err
		}
		return unexpectedJSONError{jval}
	}
	return nerr.E
}

// The JSON representation for a Duration is a JSON string that ends in the
// suffix "s" (indicating seconds) and is preceded by the number of seconds,
// with nanoseconds expressed as fractional seconds.
//
// Durations less than one second are represented with a 0 seconds field and a
// positive or negative nanos field. For durations of one second or more, a
// non-zero value for the nanos field must be of the same sign as the seconds
// field.
//
// Duration.seconds must be from -315,576,000,000 to +315,576,000,000 inclusive.
// Duration.nanos must be from -999,999,999 to +999,999,999 inclusive.

const (
	secondsInNanos       = 999999999
	maxSecondsInDuration = 315576000000
)

func (o MarshalOptions) marshalDuration(m pref.Message) error {
	fds := m.Descriptor().Fields()
	fdSeconds := fds.ByNumber(fieldnum.Duration_Seconds)
	fdNanos := fds.ByNumber(fieldnum.Duration_Nanos)

	secsVal := m.Get(fdSeconds)
	nanosVal := m.Get(fdNanos)
	secs := secsVal.Int()
	nanos := nanosVal.Int()
	if secs < -maxSecondsInDuration || secs > maxSecondsInDuration {
		return errors.New("%s: seconds out of range %v", m.Descriptor().FullName(), secs)
	}
	if nanos < -secondsInNanos || nanos > secondsInNanos {
		return errors.New("%s: nanos out of range %v", m.Descriptor().FullName(), nanos)
	}
	if (secs > 0 && nanos < 0) || (secs < 0 && nanos > 0) {
		return errors.New("%s: signs of seconds and nanos do not match", m.Descriptor().FullName())
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
	o.encoder.WriteString(x + "s")
	return nil
}

func (o UnmarshalOptions) unmarshalDuration(m pref.Message) error {
	var nerr errors.NonFatal
	jval, err := o.decoder.Read()
	if !nerr.Merge(err) {
		return err
	}
	if jval.Type() != json.String {
		return unexpectedJSONError{jval}
	}

	input := jval.String()
	secs, nanos, ok := parseDuration(input)
	if !ok {
		return errors.New("%s: invalid duration value %q", m.Descriptor().FullName(), input)
	}
	// Validate seconds. No need to validate nanos because parseDuration would
	// have covered that already.
	if secs < -maxSecondsInDuration || secs > maxSecondsInDuration {
		return errors.New("%s: out of range %q", m.Descriptor().FullName(), input)
	}

	fds := m.Descriptor().Fields()
	fdSeconds := fds.ByNumber(fieldnum.Duration_Seconds)
	fdNanos := fds.ByNumber(fieldnum.Duration_Nanos)

	m.Set(fdSeconds, pref.ValueOf(secs))
	m.Set(fdNanos, pref.ValueOf(nanos))
	return nerr.E
}

// parseDuration parses the given input string for seconds and nanoseconds value
// for the Duration JSON format. The format is a decimal number with a suffix
// 's'. It can have optional plus/minus sign. There needs to be at least an
// integer or fractional part. Fractional part is limited to 9 digits only for
// nanoseconds precision, regardless of whether there are trailing zero digits.
// Example values are 1s, 0.1s, 1.s, .1s, +1s, -1s, -.1s.
func parseDuration(input string) (int64, int32, bool) {
	b := []byte(input)
	size := len(b)
	if size < 2 {
		return 0, 0, false
	}
	if b[size-1] != 's' {
		return 0, 0, false
	}
	b = b[:size-1]

	// Read optional plus/minus symbol.
	var neg bool
	switch b[0] {
	case '-':
		neg = true
		b = b[1:]
	case '+':
		b = b[1:]
	}
	if len(b) == 0 {
		return 0, 0, false
	}

	// Read the integer part.
	var intp []byte
	switch {
	case b[0] == '0':
		b = b[1:]

	case '1' <= b[0] && b[0] <= '9':
		intp = b[0:]
		b = b[1:]
		n := 1
		for len(b) > 0 && '0' <= b[0] && b[0] <= '9' {
			n++
			b = b[1:]
		}
		intp = intp[:n]

	case b[0] == '.':
		// Continue below.

	default:
		return 0, 0, false
	}

	hasFrac := false
	var frac [9]byte
	if len(b) > 0 {
		if b[0] != '.' {
			return 0, 0, false
		}
		// Read the fractional part.
		b = b[1:]
		n := 0
		for len(b) > 0 && n < 9 && '0' <= b[0] && b[0] <= '9' {
			frac[n] = b[0]
			n++
			b = b[1:]
		}
		// It is not valid if there are more bytes left.
		if len(b) > 0 {
			return 0, 0, false
		}
		// Pad fractional part with 0s.
		for i := n; i < 9; i++ {
			frac[i] = '0'
		}
		hasFrac = true
	}

	var secs int64
	if len(intp) > 0 {
		var err error
		secs, err = strconv.ParseInt(string(intp), 10, 64)
		if err != nil {
			return 0, 0, false
		}
	}

	var nanos int64
	if hasFrac {
		nanob := bytes.TrimLeft(frac[:], "0")
		if len(nanob) > 0 {
			var err error
			nanos, err = strconv.ParseInt(string(nanob), 10, 32)
			if err != nil {
				return 0, 0, false
			}
		}
	}

	if neg {
		if secs > 0 {
			secs = -secs
		}
		if nanos > 0 {
			nanos = -nanos
		}
	}
	return secs, int32(nanos), true
}

// The JSON representation for a Timestamp is a JSON string in the RFC 3339
// format, i.e. "{year}-{month}-{day}T{hour}:{min}:{sec}[.{frac_sec}]Z" where
// {year} is always expressed using four digits while {month}, {day}, {hour},
// {min}, and {sec} are zero-padded to two digits each. The fractional seconds,
// which can go up to 9 digits, up to 1 nanosecond resolution, is optional. The
// "Z" suffix indicates the timezone ("UTC"); the timezone is required. Encoding
// should always use UTC (as indicated by "Z") and a decoder should be able to
// accept both UTC and other timezones (as indicated by an offset).
//
// Timestamp.seconds must be from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z
// inclusive.
// Timestamp.nanos must be from 0 to 999,999,999 inclusive.

const (
	maxTimestampSeconds = 253402300799
	minTimestampSeconds = -62135596800
)

func (o MarshalOptions) marshalTimestamp(m pref.Message) error {
	fds := m.Descriptor().Fields()
	fdSeconds := fds.ByNumber(fieldnum.Timestamp_Seconds)
	fdNanos := fds.ByNumber(fieldnum.Timestamp_Nanos)

	secsVal := m.Get(fdSeconds)
	nanosVal := m.Get(fdNanos)
	secs := secsVal.Int()
	nanos := nanosVal.Int()
	if secs < minTimestampSeconds || secs > maxTimestampSeconds {
		return errors.New("%s: seconds out of range %v", m.Descriptor().FullName(), secs)
	}
	if nanos < 0 || nanos > secondsInNanos {
		return errors.New("%s: nanos out of range %v", m.Descriptor().FullName(), nanos)
	}
	// Uses RFC 3339, where generated output will be Z-normalized and uses 0, 3,
	// 6 or 9 fractional digits.
	t := time.Unix(secs, nanos).UTC()
	x := t.Format("2006-01-02T15:04:05.000000000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, "000")
	x = strings.TrimSuffix(x, ".000")
	o.encoder.WriteString(x + "Z")
	return nil
}

func (o UnmarshalOptions) unmarshalTimestamp(m pref.Message) error {
	var nerr errors.NonFatal
	jval, err := o.decoder.Read()
	if !nerr.Merge(err) {
		return err
	}
	if jval.Type() != json.String {
		return unexpectedJSONError{jval}
	}

	input := jval.String()
	t, err := time.Parse(time.RFC3339Nano, input)
	if err != nil {
		return errors.New("%s: invalid timestamp value %q", m.Descriptor().FullName(), input)
	}
	// Validate seconds. No need to validate nanos because time.Parse would have
	// covered that already.
	secs := t.Unix()
	if secs < minTimestampSeconds || secs > maxTimestampSeconds {
		return errors.New("%s: out of range %q", m.Descriptor().FullName(), input)
	}

	fds := m.Descriptor().Fields()
	fdSeconds := fds.ByNumber(fieldnum.Timestamp_Seconds)
	fdNanos := fds.ByNumber(fieldnum.Timestamp_Nanos)

	m.Set(fdSeconds, pref.ValueOf(secs))
	m.Set(fdNanos, pref.ValueOf(int32(t.Nanosecond())))
	return nerr.E
}

// The JSON representation for a FieldMask is a JSON string where paths are
// separated by a comma. Fields name in each path are converted to/from
// lower-camel naming conventions. Encoding should fail if the path name would
// end up differently after a round-trip.

func (o MarshalOptions) marshalFieldMask(m pref.Message) error {
	fd := m.Descriptor().Fields().ByNumber(fieldnum.FieldMask_Paths)
	list := m.Get(fd).List()
	paths := make([]string, 0, list.Len())

	for i := 0; i < list.Len(); i++ {
		s := list.Get(i).String()
		// Return error if conversion to camelCase is not reversible.
		cc := camelCase(s)
		if s != snakeCase(cc) {
			return errors.New("%s.paths contains irreversible value %q", m.Descriptor().FullName(), s)
		}
		paths = append(paths, cc)
	}

	o.encoder.WriteString(strings.Join(paths, ","))
	return nil
}

func (o UnmarshalOptions) unmarshalFieldMask(m pref.Message) error {
	var nerr errors.NonFatal
	jval, err := o.decoder.Read()
	if !nerr.Merge(err) {
		return err
	}
	if jval.Type() != json.String {
		return unexpectedJSONError{jval}
	}
	str := strings.TrimSpace(jval.String())
	if str == "" {
		return nil
	}
	paths := strings.Split(str, ",")

	fd := m.Descriptor().Fields().ByNumber(fieldnum.FieldMask_Paths)
	list := m.Mutable(fd).List()

	for _, s := range paths {
		s = strings.TrimSpace(s)
		// Convert to snake_case. Unlike encoding, no validation is done because
		// it is not possible to know the original path names.
		list.Append(pref.ValueOf(snakeCase(s)))
	}
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
