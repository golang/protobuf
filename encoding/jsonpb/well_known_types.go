// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/v2/internal/encoding/json"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/fieldnum"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
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
	name := m.Type().FullName()
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
	name := m.Type().FullName()
	switch name {
	case "google.protobuf.Any":
		panic("unmarshaling of google.protobuf.Any is not implemented yet")

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
// contains the type URL.  If the embedded message type is well-known and has a
// custom JSON representation, that representation will be embedded adding a
// field `value` which holds the custom JSON in addition to the `@type` field.

func (o MarshalOptions) marshalAny(m pref.Message) error {
	var nerr errors.NonFatal
	msgType := m.Type()
	knownFields := m.KnownFields()

	// Start writing the JSON object.
	o.encoder.StartObject()
	defer o.encoder.EndObject()

	if !knownFields.Has(fieldnum.Any_TypeUrl) {
		if !knownFields.Has(fieldnum.Any_Value) {
			// If message is empty, marshal out empty JSON object.
			return nil
		} else {
			// Return error if type_url field is not set, but value is set.
			return errors.New("%s: type_url is not set", msgType.FullName())
		}
	}

	typeVal := knownFields.Get(fieldnum.Any_TypeUrl)
	valueVal := knownFields.Get(fieldnum.Any_Value)

	// Marshal out @type field.
	typeURL := typeVal.String()
	o.encoder.WriteName("@type")
	if err := o.encoder.WriteString(typeURL); !nerr.Merge(err) {
		return err
	}

	// Resolve the type in order to unmarshal value field.
	emt, err := o.Resolver.FindMessageByURL(typeURL)
	if !nerr.Merge(err) {
		return errors.New("%s: unable to resolve %q: %v", msgType.FullName(), typeURL, err)
	}

	em := emt.New()
	// TODO: Need to set types registry in binary unmarshaling.
	err = proto.Unmarshal(valueVal.Bytes(), em.Interface())
	if !nerr.Merge(err) {
		return errors.New("%s: unable to unmarshal %q: %v", msgType.FullName(), typeURL, err)
	}

	// If type of value has custom JSON encoding, marshal out a field "value"
	// with corresponding custom JSON encoding of the embedded message as a
	// field.
	if isCustomType(emt.FullName()) {
		o.encoder.WriteName("value")
		return o.marshalCustomType(em)
	}

	// Else, marshal out the embedded message's fields in this Any object.
	if err := o.marshalFields(em); !nerr.Merge(err) {
		return err
	}

	return nerr.E
}

// Wrapper types are encoded as JSON primitives like string, number or boolean.

func (o MarshalOptions) marshalWrapperType(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	// The "value" field has the same field number for all wrapper types.
	const num = fieldnum.BoolValue_Value
	fd := fieldDescs.ByNumber(num)
	val := knownFields.Get(num)
	return o.marshalSingular(val, fd)
}

func (o UnmarshalOptions) unmarshalWrapperType(m pref.Message) error {
	var nerr errors.NonFatal
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	// The "value" field has the same field number for all wrapper types.
	const num = fieldnum.BoolValue_Value
	fd := fieldDescs.ByNumber(num)
	val, err := o.unmarshalScalar(fd)
	if !nerr.Merge(err) {
		return err
	}
	knownFields.Set(num, val)
	return nerr.E
}

// The JSON representation for Empty is an empty JSON object.

func (o MarshalOptions) marshalEmpty(pref.Message) error {
	o.encoder.StartObject()
	o.encoder.EndObject()
	return nil
}

func (o UnmarshalOptions) unmarshalEmpty(pref.Message) error {
	jval, err := o.decoder.Read()
	if err != nil {
		return err
	}
	if jval.Type() != json.StartObject {
		return unexpectedJSONError{jval}
	}
	jval, err = o.decoder.Read()
	if err != nil {
		return err
	}
	if jval.Type() != json.EndObject {
		return unexpectedJSONError{jval}
	}
	return nil
}

// The JSON representation for Struct is a JSON object that contains the encoded
// Struct.fields map and follows the serialization rules for a map.

func (o MarshalOptions) marshalStruct(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	fd := fieldDescs.ByNumber(fieldnum.Struct_Fields)
	val := knownFields.Get(fieldnum.Struct_Fields)
	return o.marshalMap(val.Map(), fd)
}

func (o UnmarshalOptions) unmarshalStruct(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	fd := fieldDescs.ByNumber(fieldnum.Struct_Fields)
	val := knownFields.Get(fieldnum.Struct_Fields)
	return o.unmarshalMap(val.Map(), fd)
}

// The JSON representation for ListValue is JSON array that contains the encoded
// ListValue.values repeated field and follows the serialization rules for a
// repeated field.

func (o MarshalOptions) marshalListValue(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	fd := fieldDescs.ByNumber(fieldnum.ListValue_Values)
	val := knownFields.Get(fieldnum.ListValue_Values)
	return o.marshalList(val.List(), fd)
}

func (o UnmarshalOptions) unmarshalListValue(m pref.Message) error {
	msgType := m.Type()
	fieldDescs := msgType.Fields()
	knownFields := m.KnownFields()

	fd := fieldDescs.ByNumber(fieldnum.ListValue_Values)
	val := knownFields.Get(fieldnum.ListValue_Values)
	return o.unmarshalList(val.List(), fd)
}

// The JSON representation for a Value is dependent on the oneof field that is
// set. Each of the field in the oneof has its own custom serialization rule. A
// Value message needs to be a oneof field set, else it is an error.

func (o MarshalOptions) marshalKnownValue(m pref.Message) error {
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
		return o.marshalSingular(val, fd)
	}

	// Return error if none of the fields are set.
	return errors.New("%s: none of the variants is set", msgType.FullName())
}

func isKnownValue(fd pref.FieldDescriptor) bool {
	md := fd.MessageType()
	return md != nil && md.FullName() == "google.protobuf.Value"
}

func (o UnmarshalOptions) unmarshalKnownValue(m pref.Message) error {
	var nerr errors.NonFatal
	knownFields := m.KnownFields()

	switch o.decoder.Peek() {
	case json.Null:
		o.decoder.Read()
		knownFields.Set(fieldnum.Value_NullValue, pref.ValueOf(pref.EnumNumber(0)))

	case json.Bool:
		jval, err := o.decoder.Read()
		if err != nil {
			return err
		}
		val, err := unmarshalBool(jval)
		if err != nil {
			return err
		}
		knownFields.Set(fieldnum.Value_BoolValue, val)

	case json.Number:
		jval, err := o.decoder.Read()
		if err != nil {
			return err
		}
		val, err := unmarshalFloat(jval, 64)
		if err != nil {
			return err
		}
		knownFields.Set(fieldnum.Value_NumberValue, val)

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
		knownFields.Set(fieldnum.Value_StringValue, val)

	case json.StartObject:
		m := knownFields.NewMessage(fieldnum.Value_StructValue)
		if err := o.unmarshalStruct(m); !nerr.Merge(err) {
			return err
		}
		knownFields.Set(fieldnum.Value_StructValue, pref.ValueOf(m))

	case json.StartArray:
		m := knownFields.NewMessage(fieldnum.Value_ListValue)
		if err := o.unmarshalListValue(m); !nerr.Merge(err) {
			return err
		}
		knownFields.Set(fieldnum.Value_ListValue, pref.ValueOf(m))

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
	msgType := m.Type()
	knownFields := m.KnownFields()

	secsVal := knownFields.Get(fieldnum.Duration_Seconds)
	nanosVal := knownFields.Get(fieldnum.Duration_Nanos)
	secs := secsVal.Int()
	nanos := nanosVal.Int()
	if secs < -maxSecondsInDuration || secs > maxSecondsInDuration {
		return errors.New("%s: seconds out of range", msgType.FullName())
	}
	if nanos <= -secondsInNanos || nanos >= secondsInNanos {
		return errors.New("%s: nanos out of range", msgType.FullName())
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

	msgType := m.Type()
	input := jval.String()
	secs, nanos, ok := parseDuration(input)
	if !ok {
		return errors.New("%s: invalid duration value %q", msgType.FullName(), input)
	}
	// Validate seconds. No need to validate nanos because parseDuration would
	// have covered that already.
	if secs < -maxSecondsInDuration || secs > maxSecondsInDuration {
		return errors.New("%s: out of range %q", msgType.FullName(), input)
	}

	knownFields := m.KnownFields()
	knownFields.Set(fieldnum.Duration_Seconds, pref.ValueOf(secs))
	knownFields.Set(fieldnum.Duration_Nanos, pref.ValueOf(nanos))
	return nerr.E
}

// Regular expression for Duration type in JSON format. This allows for values
// like 1s, 0.1s, 1.s, .1s. It limits fractional part to 9 digits only for
// nanoseconds precision, regardless of whether there are trailing zero digits.
var durationRE = regexp.MustCompile(`^-?([0-9]|[1-9][0-9]+)?(\.[0-9]{0,9})?s$`)

func parseDuration(input string) (int64, int32, bool) {
	b := []byte(input)
	// TODO: Parse input directly instead of using a regular expression.
	matched := durationRE.FindSubmatch(b)
	if len(matched) != 3 {
		return 0, 0, false
	}

	var neg bool
	if b[0] == '-' {
		neg = true
	}
	var secb []byte
	if len(matched[1]) == 0 {
		secb = []byte{'0'}
	} else {
		secb = matched[1]
	}
	var nanob []byte
	if len(matched[2]) <= 1 {
		nanob = []byte{'0'}
	} else {
		nanob = matched[2][1:]
		// Right-pad with 0s for nanosecond-precision.
		for i := len(nanob); i < 9; i++ {
			nanob = append(nanob, '0')
		}
		// Remove unnecessary 0s in the left.
		nanob = bytes.TrimLeft(nanob, "0")
	}

	secs, err := strconv.ParseInt(string(secb), 10, 64)
	if err != nil {
		return 0, 0, false
	}

	nanos, err := strconv.ParseInt(string(nanob), 10, 32)
	if err != nil {
		return 0, 0, false
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
	msgType := m.Type()
	knownFields := m.KnownFields()

	secsVal := knownFields.Get(fieldnum.Timestamp_Seconds)
	nanosVal := knownFields.Get(fieldnum.Timestamp_Nanos)
	secs := secsVal.Int()
	nanos := nanosVal.Int()
	if secs < minTimestampSeconds || secs > maxTimestampSeconds {
		return errors.New("%s: seconds out of range %q", msgType.FullName(), secs)
	}
	if nanos < 0 || nanos >= secondsInNanos {
		return errors.New("%s: nanos out of range %q", msgType.FullName(), nanos)
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

	msgType := m.Type()
	input := jval.String()
	t, err := time.Parse(time.RFC3339Nano, input)
	if err != nil {
		return errors.New("%s: invalid timestamp value %q", msgType.FullName(), input)
	}
	// Validate seconds. No need to validate nanos because time.Parse would have
	// covered that already.
	secs := t.Unix()
	if secs < minTimestampSeconds || secs > maxTimestampSeconds {
		return errors.New("%s: out of range %q", msgType.FullName(), input)
	}

	knownFields := m.KnownFields()
	knownFields.Set(fieldnum.Timestamp_Seconds, pref.ValueOf(secs))
	knownFields.Set(fieldnum.Timestamp_Nanos, pref.ValueOf(int32(t.Nanosecond())))
	return nerr.E
}

// The JSON representation for a FieldMask is a JSON string where paths are
// separated by a comma. Fields name in each path are converted to/from
// lower-camel naming conventions. Encoding should fail if the path name would
// end up differently after a round-trip.

func (o MarshalOptions) marshalFieldMask(m pref.Message) error {
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

	knownFields := m.KnownFields()
	val := knownFields.Get(fieldnum.FieldMask_Paths)
	list := val.List()

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
