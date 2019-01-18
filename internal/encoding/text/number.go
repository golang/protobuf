// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"bytes"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/protobuf/v2/internal/errors"
)

// marshalNumber encodes v as either a Bool, Int, Uint, Float32, or Float64.
func (p *encoder) marshalNumber(v Value) error {
	var err error
	p.out, err = appendNumber(p.out, v)
	return err
}
func appendNumber(out []byte, v Value) ([]byte, error) {
	if len(v.raw) > 0 {
		switch v.Type() {
		case Bool, Int, Uint, Float32, Float64:
			return append(out, v.raw...), nil
		}
	}
	switch v.Type() {
	case Bool:
		if b, _ := v.Bool(); b {
			return append(out, "true"...), nil
		} else {
			return append(out, "false"...), nil
		}
	case Int:
		return strconv.AppendInt(out, int64(v.num), 10), nil
	case Uint:
		return strconv.AppendUint(out, uint64(v.num), 10), nil
	case Float32:
		return appendFloat(out, v, 32)
	case Float64:
		return appendFloat(out, v, 64)
	default:
		return nil, errors.New("invalid type %v, expected bool or number", v.Type())
	}
}

func appendFloat(out []byte, v Value, bitSize int) ([]byte, error) {
	switch n := math.Float64frombits(v.num); {
	case math.IsNaN(n):
		return append(out, "nan"...), nil
	case math.IsInf(n, +1):
		return append(out, "inf"...), nil
	case math.IsInf(n, -1):
		return append(out, "-inf"...), nil
	default:
		return strconv.AppendFloat(out, n, 'g', -1, bitSize), nil
	}
}

// These regular expressions were derived by reverse engineering the C++ code
// in tokenizer.cc and text_format.cc.
var (
	literals = map[string]interface{}{
		// These exact literals are the ones supported in C++.
		// In C++, a 1-bit unsigned integers is also allowed to represent
		// a boolean. This is handled in Value.Bool.
		"t":     true,
		"true":  true,
		"True":  true,
		"f":     false,
		"false": false,
		"False": false,

		// C++ permits "-nan" and the case-insensitive variants of these.
		// However, Go continues to be case-sensitive.
		"nan":  math.NaN(),
		"inf":  math.Inf(+1),
		"-inf": math.Inf(-1),
	}
	literalRegexp = regexp.MustCompile("^-?[a-zA-Z]+")
	intRegexp     = regexp.MustCompile("^-?([1-9][0-9]*|0[xX][0-9a-fA-F]+|0[0-7]*)")
	floatRegexp   = regexp.MustCompile("^-?((0|[1-9][0-9]*)?([.][0-9]*)?([eE][+-]?[0-9]+)?[fF]?)")
)

// unmarshalNumber decodes a Bool, Int, Uint, or Float64 from the input.
func (p *decoder) unmarshalNumber() (Value, error) {
	v, n, err := consumeNumber(p.in)
	p.consume(n)
	return v, err
}
func consumeNumber(in []byte) (Value, int, error) {
	if len(in) == 0 {
		return Value{}, 0, io.ErrUnexpectedEOF
	}
	if n := matchWithDelim(literalRegexp, in); n > 0 {
		if v, ok := literals[string(in[:n])]; ok {
			return rawValueOf(v, in[:n:n]), n, nil
		}
	}
	if n := matchWithDelim(floatRegexp, in); n > 0 {
		if bytes.ContainsAny(in[:n], ".eEfF") {
			s := strings.TrimRight(string(in[:n]), "fF")
			// Always decode float as 64-bit.
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return Value{}, 0, err
			}
			return rawValueOf(f, in[:n:n]), n, nil
		}
	}
	if n := matchWithDelim(intRegexp, in); n > 0 {
		if in[0] == '-' {
			v, err := strconv.ParseInt(string(in[:n]), 0, 64)
			if err != nil {
				return Value{}, 0, err
			}
			return rawValueOf(v, in[:n:n]), n, nil
		} else {
			v, err := strconv.ParseUint(string(in[:n]), 0, 64)
			if err != nil {
				return Value{}, 0, err
			}
			return rawValueOf(v, in[:n:n]), n, nil
		}
	}
	return Value{}, 0, newSyntaxError("invalid %q as number or bool", errRegexp.Find(in))
}
