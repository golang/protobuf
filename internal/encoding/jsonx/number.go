// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"io"
	"math"
	"regexp"
	"strconv"

	"github.com/golang/protobuf/v2/internal/errors"
)

// marshalNumber encodes v as a Number.
func (p *encoder) marshalNumber(v Value) error {
	var err error
	p.out, err = appendNumber(p.out, v)
	return err
}
func appendNumber(out []byte, v Value) ([]byte, error) {
	if v.Type() != Number {
		return nil, errors.New("invalid type %v, expected number", v.Type())
	}
	if len(v.raw) > 0 {
		return append(out, v.raw...), nil
	}
	n := v.Number()
	if math.IsInf(n, 0) || math.IsNaN(n) {
		return nil, errors.New("invalid number value: %v", n)
	}

	// JSON number formatting logic based on encoding/json.
	// See floatEncoder.encode for reference.
	bits := 64
	if float64(float32(n)) == n {
		bits = 32
	}
	fmt := byte('f')
	if abs := math.Abs(n); abs != 0 {
		if bits == 64 && (abs < 1e-6 || abs >= 1e21) || bits == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
			fmt = 'e'
		}
	}
	out = strconv.AppendFloat(out, n, fmt, -1, bits)
	if fmt == 'e' {
		n := len(out)
		if n >= 4 && out[n-4] == 'e' && out[n-3] == '-' && out[n-2] == '0' {
			out[n-2] = out[n-1]
			out = out[:n-1]
		}
	}
	return out, nil
}

// Exact expression to match a JSON floating-point number.
// JSON's grammar for floats is more restrictive than Go's grammar.
var floatRegexp = regexp.MustCompile("^-?(0|[1-9][0-9]*)([.][0-9]+)?([eE][+-]?[0-9]+)?")

// unmarshalNumber decodes a Number from the input.
func (p *decoder) unmarshalNumber() (Value, error) {
	v, n, err := consumeNumber(p.in)
	p.consume(n)
	return v, err
}
func consumeNumber(in []byte) (Value, int, error) {
	if len(in) == 0 {
		return Value{}, 0, io.ErrUnexpectedEOF
	}
	if n := matchWithDelim(floatRegexp, in); n > 0 {
		v, err := strconv.ParseFloat(string(in[:n]), 64)
		if err != nil {
			return Value{}, 0, err
		}
		return rawValueOf(v, in[:n:n]), n, nil
	}
	return Value{}, 0, newSyntaxError("invalid %q as number", errRegexp.Find(in))
}
