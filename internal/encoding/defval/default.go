// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package defval marshals and unmarshals textual forms of default values.
//
// This package handles both the form historically used in Go struct field tags
// and also the form used by google.protobuf.FieldDescriptorProto.default_value
// since they differ in superficial ways.
package defval

import (
	"fmt"
	"math"
	"strconv"

	ptext "github.com/golang/protobuf/v2/internal/encoding/text"
	errors "github.com/golang/protobuf/v2/internal/errors"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// Format is the serialization format used to represent the default value.
type Format int

const (
	_ Format = iota

	// Descriptor uses the serialization format that protoc uses with the
	// google.protobuf.FieldDescriptorProto.default_value field.
	Descriptor

	// GoTag uses the historical serialization format in Go struct field tags.
	GoTag
)

// Unmarshal deserializes the default string s according to the given kind k.
// When using the Descriptor format on an enum kind, a Value of type string
// representing the enum identifier is returned. It is the caller's
// responsibility to verify that the identifier is valid.
func Unmarshal(s string, k pref.Kind, f Format) (pref.Value, error) {
	switch k {
	case pref.BoolKind:
		if f == GoTag {
			switch s {
			case "1":
				return pref.ValueOf(true), nil
			case "0":
				return pref.ValueOf(false), nil
			}
		} else {
			switch s {
			case "true":
				return pref.ValueOf(true), nil
			case "false":
				return pref.ValueOf(false), nil
			}
		}
	case pref.EnumKind:
		if f == GoTag {
			// Go tags used the numeric form of the enum value.
			if n, err := strconv.ParseInt(s, 10, 32); err == nil {
				return pref.ValueOf(pref.EnumNumber(n)), nil
			}
		} else {
			// Descriptor default_value used the enum identifier.
			if pref.Name(s).IsValid() {
				return pref.ValueOf(s), nil
			}
		}
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		if v, err := strconv.ParseInt(s, 10, 32); err == nil {
			return pref.ValueOf(int32(v)), nil
		}
	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return pref.ValueOf(int64(v)), nil
		}
	case pref.Uint32Kind, pref.Fixed32Kind:
		if v, err := strconv.ParseUint(s, 10, 32); err == nil {
			return pref.ValueOf(uint32(v)), nil
		}
	case pref.Uint64Kind, pref.Fixed64Kind:
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			return pref.ValueOf(uint64(v)), nil
		}
	case pref.FloatKind, pref.DoubleKind:
		var v float64
		var err error
		switch s {
		case "-inf":
			v = math.Inf(-1)
		case "inf":
			v = math.Inf(+1)
		case "nan":
			v = math.NaN()
		default:
			v, err = strconv.ParseFloat(s, 64)
		}
		if err == nil {
			if k == pref.FloatKind {
				return pref.ValueOf(float32(v)), nil
			} else {
				return pref.ValueOf(float64(v)), nil
			}
		}
	case pref.StringKind:
		// String values are already unescaped and can be used as is.
		return pref.ValueOf(s), nil
	case pref.BytesKind:
		if b, ok := unmarshalBytes(s); ok {
			return pref.ValueOf(b), nil
		}
	}
	return pref.Value{}, errors.New("invalid default value for %v: %q", k, s)
}

// Marshal serializes v as the default string according to the given kind k.
// Enums are serialized in numeric form regardless of format chosen.
func Marshal(v pref.Value, k pref.Kind, f Format) (string, error) {
	switch k {
	case pref.BoolKind:
		if f == GoTag {
			if v.Bool() {
				return "1", nil
			} else {
				return "0", nil
			}
		} else {
			if v.Bool() {
				return "true", nil
			} else {
				return "false", nil
			}
		}
	case pref.EnumKind:
		return strconv.FormatInt(int64(v.Enum()), 10), nil
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind, pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		return strconv.FormatInt(v.Int(), 10), nil
	case pref.Uint32Kind, pref.Fixed32Kind, pref.Uint64Kind, pref.Fixed64Kind:
		return strconv.FormatUint(v.Uint(), 10), nil
	case pref.FloatKind, pref.DoubleKind:
		f := v.Float()
		switch {
		case math.IsInf(f, -1):
			return "-inf", nil
		case math.IsInf(f, +1):
			return "inf", nil
		case math.IsNaN(f):
			return "nan", nil
		default:
			if k == pref.FloatKind {
				return strconv.FormatFloat(f, 'g', -1, 32), nil
			} else {
				return strconv.FormatFloat(f, 'g', -1, 64), nil
			}
		}
	case pref.StringKind:
		// String values are serialized as is without any escaping.
		return v.String(), nil
	case pref.BytesKind:
		if s, ok := marshalBytes(v.Bytes()); ok {
			return s, nil
		}
	}
	return "", errors.New("invalid default value for %v: %v", k, v)
}

// unmarshalBytes deserializes bytes by applying C unescaping.
func unmarshalBytes(s string) ([]byte, bool) {
	// Bytes values use the same escaping as the text format,
	// however they lack the surrounding double quotes.
	// TODO: Export unmarshalString in the text package to avoid this hack.
	v, err := ptext.Unmarshal([]byte(`["` + s + `"]:0`))
	if err == nil && len(v.Message()) == 1 {
		s := v.Message()[0][0].String()
		return []byte(s), true
	}
	return nil, false
}

// marshalBytes serializes bytes by using C escaping.
// To match the exact output of protoc, this is identical to the
// CEscape function in strutil.cc of the protoc source code.
func marshalBytes(b []byte) (string, bool) {
	var s []byte
	for _, c := range b {
		switch c {
		case '\n':
			s = append(s, `\n`...)
		case '\r':
			s = append(s, `\r`...)
		case '\t':
			s = append(s, `\t`...)
		case '"':
			s = append(s, `\"`...)
		case '\'':
			s = append(s, `\'`...)
		case '\\':
			s = append(s, `\\`...)
		default:
			if printableASCII := c >= 0x20 && c <= 0x7e; printableASCII {
				s = append(s, c)
			} else {
				s = append(s, fmt.Sprintf(`\%03o`, c)...)
			}
		}
	}
	return string(s), true
}
