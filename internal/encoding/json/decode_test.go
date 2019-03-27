// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/golang/protobuf/v2/internal/encoding/json"
)

type R struct {
	// T is expected Type returned from calling Decoder.Read.
	T json.Type
	// E is expected error substring from calling Decoder.Read if set.
	E string
	// V is expected value from calling
	// Value.{Bool()|Float()|Int()|Uint()|String()} depending on type.
	V interface{}
	// VE is expected error substring from calling
	// Value.{Bool()|Float()|Int()|Uint()|String()} depending on type if set.
	VE string
}

func TestDecoder(t *testing.T) {
	const space = " \n\r\t"

	tests := []struct {
		input string
		// want is a list of expected values returned from calling
		// Decoder.Read. An item makes the test code invoke
		// Decoder.Read and compare against R.T and R.E.  For Bool,
		// Number and String tokens, it invokes the corresponding getter method
		// and compares the returned value against R.V or R.VE if it returned an
		// error.
		want []R
	}{
		{
			input: ``,
			want:  []R{{T: json.EOF}},
		},
		{
			input: space,
			want:  []R{{T: json.EOF}},
		},
		{
			// Calling Read after EOF will keep returning EOF for
			// succeeding Read calls.
			input: space,
			want: []R{
				{T: json.EOF},
				{T: json.EOF},
				{T: json.EOF},
			},
		},

		// JSON literals.
		{
			input: space + `null` + space,
			want: []R{
				{T: json.Null},
				{T: json.EOF},
			},
		},
		{
			input: space + `true` + space,
			want: []R{
				{T: json.Bool, V: true},
				{T: json.EOF},
			},
		},
		{
			input: space + `false` + space,
			want: []R{
				{T: json.Bool, V: false},
				{T: json.EOF},
			},
		},
		{
			// Error returned will produce the same error again.
			input: space + `foo` + space,
			want: []R{
				{E: `invalid value foo`},
				{E: `invalid value foo`},
			},
		},

		// JSON strings.
		{
			input: space + `""` + space,
			want: []R{
				{T: json.String, V: ""},
				{T: json.EOF},
			},
		},
		{
			input: space + `"hello"` + space,
			want: []R{
				{T: json.String, V: "hello"},
				{T: json.EOF},
			},
		},
		{
			input: `"hello`,
			want:  []R{{E: `unexpected EOF`}},
		},
		{
			input: "\"\x00\"",
			want:  []R{{E: `invalid character '\x00' in string`}},
		},
		{
			input: "\"\u0031\u0032\"",
			want: []R{
				{T: json.String, V: "12"},
				{T: json.EOF},
			},
		},
		{
			// Invalid UTF-8 error is returned in ReadString instead of Read.
			input: "\"\xff\"",
			want: []R{
				{T: json.String, E: `invalid UTF-8 detected`, V: string("\xff")},
				{T: json.EOF},
			},
		},
		{
			input: `"` + string(utf8.RuneError) + `"`,
			want: []R{
				{T: json.String, V: string(utf8.RuneError)},
				{T: json.EOF},
			},
		},
		{
			input: `"\uFFFD"`,
			want: []R{
				{T: json.String, V: string(utf8.RuneError)},
				{T: json.EOF},
			},
		},
		{
			input: `"\x"`,
			want:  []R{{E: `invalid escape code "\\x" in string`}},
		},
		{
			input: `"\uXXXX"`,
			want:  []R{{E: `invalid escape code "\\uXXXX" in string`}},
		},
		{
			input: `"\uDEAD"`, // unmatched surrogate pair
			want:  []R{{E: `unexpected EOF`}},
		},
		{
			input: `"\uDEAD\uBEEF"`, // invalid surrogate half
			want:  []R{{E: `invalid escape code "\\uBEEF" in string`}},
		},
		{
			input: `"\uD800\udead"`, // valid surrogate pair
			want: []R{
				{T: json.String, V: `𐊭`},
				{T: json.EOF},
			},
		},
		{
			input: `"\u0000\"\\\/\b\f\n\r\t"`,
			want: []R{
				{T: json.String, V: "\u0000\"\\/\b\f\n\r\t"},
				{T: json.EOF},
			},
		},

		// Invalid JSON numbers.
		{
			input: `-`,
			want:  []R{{E: `invalid number -`}},
		},
		{
			input: `+0`,
			want:  []R{{E: `invalid value +0`}},
		},
		{
			input: `-+`,
			want:  []R{{E: `invalid number -+`}},
		},
		{
			input: `0.`,
			want:  []R{{E: `invalid number 0.`}},
		},
		{
			input: `.1`,
			want:  []R{{E: `invalid value .1`}},
		},
		{
			input: `1.0.1`,
			want:  []R{{E: `invalid number 1.0.1`}},
		},
		{
			input: `1..1`,
			want:  []R{{E: `invalid number 1..1`}},
		},
		{
			input: `-1-2`,
			want:  []R{{E: `invalid number -1-2`}},
		},
		{
			input: `01`,
			want:  []R{{E: `invalid number 01`}},
		},
		{
			input: `1e`,
			want:  []R{{E: `invalid number 1e`}},
		},
		{
			input: `1e1.2`,
			want:  []R{{E: `invalid number 1e1.2`}},
		},
		{
			input: `1Ee`,
			want:  []R{{E: `invalid number 1Ee`}},
		},
		{
			input: `1.e1`,
			want:  []R{{E: `invalid number 1.e1`}},
		},
		{
			input: `1.e+`,
			want:  []R{{E: `invalid number 1.e+`}},
		},
		{
			input: `1e+-2`,
			want:  []R{{E: `invalid number 1e+-2`}},
		},
		{
			input: `1e--2`,
			want:  []R{{E: `invalid number 1e--2`}},
		},
		{
			input: `1.0true`,
			want:  []R{{E: `invalid number 1.0true`}},
		},

		// JSON numbers as floating point.
		{
			input: space + `0.0` + space,
			want: []R{
				{T: json.Number, V: float32(0)},
				{T: json.EOF},
			},
		},
		{
			input: space + `0` + space,
			want: []R{
				{T: json.Number, V: float32(0)},
				{T: json.EOF},
			},
		},
		{
			input: space + `-0` + space,
			want: []R{
				{T: json.Number, V: float32(0)},
				{T: json.EOF},
			},
		},
		{
			input: `-1.02`,
			want: []R{
				{T: json.Number, V: float32(-1.02)},
				{T: json.EOF},
			},
		},
		{
			input: `1.020000`,
			want: []R{
				{T: json.Number, V: float32(1.02)},
				{T: json.EOF},
			},
		},
		{
			input: `-1.0e0`,
			want: []R{
				{T: json.Number, V: float32(-1)},
				{T: json.EOF},
			},
		},
		{
			input: `1.0e-000`,
			want: []R{
				{T: json.Number, V: float32(1)},
				{T: json.EOF},
			},
		},
		{
			input: `1e+00`,
			want: []R{
				{T: json.Number, V: float32(1)},
				{T: json.EOF},
			},
		},
		{
			input: `1.02e3`,
			want: []R{
				{T: json.Number, V: float32(1.02e3)},
				{T: json.EOF},
			},
		},
		{
			input: `-1.02E03`,
			want: []R{
				{T: json.Number, V: float32(-1.02e3)},
				{T: json.EOF},
			},
		},
		{
			input: `1.0200e+3`,
			want: []R{
				{T: json.Number, V: float32(1.02e3)},
				{T: json.EOF},
			},
		},
		{
			input: `-1.0200E+03`,
			want: []R{
				{T: json.Number, V: float32(-1.02e3)},
				{T: json.EOF},
			},
		},
		{
			input: `1.0200e-3`,
			want: []R{
				{T: json.Number, V: float32(1.02e-3)},
				{T: json.EOF},
			},
		},
		{
			input: `-1.0200E-03`,
			want: []R{
				{T: json.Number, V: float32(-1.02e-3)},
				{T: json.EOF},
			},
		},
		{
			// Exceeds max float32 limit, but should be ok for float64.
			input: `3.4e39`,
			want: []R{
				{T: json.Number, V: float64(3.4e39)},
				{T: json.EOF},
			},
		},
		{
			// Exceeds max float32 limit.
			input: `3.4e39`,
			want: []R{
				{T: json.Number, V: float32(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},
		{
			// Less than negative max float32 limit.
			input: `-3.4e39`,
			want: []R{
				{T: json.Number, V: float32(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},
		{
			// Exceeds max float64 limit.
			input: `1.79e+309`,
			want: []R{
				{T: json.Number, V: float64(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},
		{
			// Less than negative max float64 limit.
			input: `-1.79e+309`,
			want: []R{
				{T: json.Number, V: float64(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},

		// JSON numbers as signed integers.
		{
			input: space + `0` + space,
			want: []R{
				{T: json.Number, V: int32(0)},
				{T: json.EOF},
			},
		},
		{
			input: space + `-0` + space,
			want: []R{
				{T: json.Number, V: int32(0)},
				{T: json.EOF},
			},
		},
		{
			// Fractional part equals 0 is ok.
			input: `1.00000`,
			want: []R{
				{T: json.Number, V: int32(1)},
				{T: json.EOF},
			},
		},
		{
			// Fractional part not equals 0 returns error.
			input: `1.0000000001`,
			want: []R{
				{T: json.Number, V: int32(0), VE: `cannot convert 1.0000000001 to integer`},
				{T: json.EOF},
			},
		},
		{
			input: `0e0`,
			want: []R{
				{T: json.Number, V: int32(0)},
				{T: json.EOF},
			},
		},
		{
			input: `0.0E0`,
			want: []R{
				{T: json.Number, V: int32(0)},
				{T: json.EOF},
			},
		},
		{
			input: `0.0E10`,
			want: []R{
				{T: json.Number, V: int32(0)},
				{T: json.EOF},
			},
		},
		{
			input: `-1`,
			want: []R{
				{T: json.Number, V: int32(-1)},
				{T: json.EOF},
			},
		},
		{
			input: `1.0e+0`,
			want: []R{
				{T: json.Number, V: int32(1)},
				{T: json.EOF},
			},
		},
		{
			input: `-1E-0`,
			want: []R{
				{T: json.Number, V: int32(-1)},
				{T: json.EOF},
			},
		},
		{
			input: `1E1`,
			want: []R{
				{T: json.Number, V: int32(10)},
				{T: json.EOF},
			},
		},
		{
			input: `-100.00e-02`,
			want: []R{
				{T: json.Number, V: int32(-1)},
				{T: json.EOF},
			},
		},
		{
			input: `0.1200E+02`,
			want: []R{
				{T: json.Number, V: int64(12)},
				{T: json.EOF},
			},
		},
		{
			input: `0.012e2`,
			want: []R{
				{T: json.Number, V: int32(0), VE: `cannot convert 0.012e2 to integer`},
				{T: json.EOF},
			},
		},
		{
			input: `12e-2`,
			want: []R{
				{T: json.Number, V: int32(0), VE: `cannot convert 12e-2 to integer`},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MaxInt32.
			input: `2147483648`,
			want: []R{
				{T: json.Number, V: int32(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MinInt32.
			input: `-2147483649`,
			want: []R{
				{T: json.Number, V: int32(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MaxInt32, but ok for int64.
			input: `2147483648`,
			want: []R{
				{T: json.Number, V: int64(2147483648)},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MinInt32, but ok for int64.
			input: `-2147483649`,
			want: []R{
				{T: json.Number, V: int64(-2147483649)},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MaxInt64.
			input: `9223372036854775808`,
			want: []R{
				{T: json.Number, V: int64(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MinInt64.
			input: `-9223372036854775809`,
			want: []R{
				{T: json.Number, V: int64(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},

		// JSON numbers as unsigned integers.
		{
			input: space + `0` + space,
			want: []R{
				{T: json.Number, V: uint32(0)},
				{T: json.EOF},
			},
		},
		{
			input: space + `-0` + space,
			want: []R{
				{T: json.Number, V: uint32(0)},
				{T: json.EOF},
			},
		},
		{
			input: `-1`,
			want: []R{
				{T: json.Number, V: uint32(0), VE: `invalid syntax`},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MaxUint32.
			input: `4294967296`,
			want: []R{
				{T: json.Number, V: uint32(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},
		{
			// Exceeds math.MaxUint64.
			input: `18446744073709551616`,
			want: []R{
				{T: json.Number, V: uint64(0), VE: `value out of range`},
				{T: json.EOF},
			},
		},

		// JSON sequence of values.
		{
			input: `true null`,
			want: []R{
				{T: json.Bool, V: true},
				{E: `unexpected value null`},
			},
		},
		{
			input: "null false",
			want: []R{
				{T: json.Null},
				{E: `unexpected value false`},
			},
		},
		{
			input: `true,false`,
			want: []R{
				{T: json.Bool, V: true},
				{E: `unexpected character ,`},
			},
		},
		{
			input: `47"hello"`,
			want: []R{
				{T: json.Number, V: int32(47)},
				{E: `unexpected value "hello"`},
			},
		},
		{
			input: `47 "hello"`,
			want: []R{
				{T: json.Number, V: int32(47)},
				{E: `unexpected value "hello"`},
			},
		},
		{
			input: `true 42`,
			want: []R{
				{T: json.Bool, V: true},
				{E: `unexpected value 42`},
			},
		},

		// JSON arrays.
		{
			input: space + `[]` + space,
			want: []R{
				{T: json.StartArray},
				{T: json.EndArray},
				{T: json.EOF},
			},
		},
		{
			input: space + `[` + space + `]` + space,
			want: []R{
				{T: json.StartArray},
				{T: json.EndArray},
				{T: json.EOF},
			},
		},
		{
			input: space + `[` + space,
			want: []R{
				{T: json.StartArray},
				{E: `unexpected EOF`},
			},
		},
		{
			input: space + `]` + space,
			want:  []R{{E: `unexpected character ]`}},
		},
		{
			input: `[null,true,false,  1e1, "hello"   ]`,
			want: []R{
				{T: json.StartArray},
				{T: json.Null},
				{T: json.Bool, V: true},
				{T: json.Bool, V: false},
				{T: json.Number, V: int32(10)},
				{T: json.String, V: "hello"},
				{T: json.EndArray},
				{T: json.EOF},
			},
		},
		{
			input: `[` + space + `true` + space + `,` + space + `"hello"` + space + `]`,
			want: []R{
				{T: json.StartArray},
				{T: json.Bool, V: true},
				{T: json.String, V: "hello"},
				{T: json.EndArray},
				{T: json.EOF},
			},
		},
		{
			input: `[` + space + `true` + space + `,` + space + `]`,
			want: []R{
				{T: json.StartArray},
				{T: json.Bool, V: true},
				{E: `unexpected character ]`},
			},
		},
		{
			input: `[` + space + `false` + space + `]`,
			want: []R{
				{T: json.StartArray},
				{T: json.Bool, V: false},
				{T: json.EndArray},
				{T: json.EOF},
			},
		},
		{
			input: `[` + space + `1` + space + `0` + space + `]`,
			want: []R{
				{T: json.StartArray},
				{T: json.Number, V: int64(1)},
				{E: `unexpected value 0`},
			},
		},
		{
			input: `[null`,
			want: []R{
				{T: json.StartArray},
				{T: json.Null},
				{E: `unexpected EOF`},
			},
		},
		{
			input: `[foo]`,
			want: []R{
				{T: json.StartArray},
				{E: `invalid value foo`},
			},
		},
		{
			input: `[{}, "hello", [true, false], null]`,
			want: []R{
				{T: json.StartArray},
				{T: json.StartObject},
				{T: json.EndObject},
				{T: json.String, V: "hello"},
				{T: json.StartArray},
				{T: json.Bool, V: true},
				{T: json.Bool, V: false},
				{T: json.EndArray},
				{T: json.Null},
				{T: json.EndArray},
				{T: json.EOF},
			},
		},
		{
			input: `[{ ]`,
			want: []R{
				{T: json.StartArray},
				{T: json.StartObject},
				{E: `unexpected character ]`},
			},
		},
		{
			input: `[[ ]`,
			want: []R{
				{T: json.StartArray},
				{T: json.StartArray},
				{T: json.EndArray},
				{E: `unexpected EOF`},
			},
		},
		{
			input: `[,]`,
			want: []R{
				{T: json.StartArray},
				{E: `unexpected character ,`},
			},
		},
		{
			input: `[true "hello"]`,
			want: []R{
				{T: json.StartArray},
				{T: json.Bool, V: true},
				{E: `unexpected value "hello"`},
			},
		},
		{
			input: `[] null`,
			want: []R{
				{T: json.StartArray},
				{T: json.EndArray},
				{E: `unexpected value null`},
			},
		},
		{
			input: `true []`,
			want: []R{
				{T: json.Bool, V: true},
				{E: `unexpected character [`},
			},
		},

		// JSON objects.
		{
			input: space + `{}` + space,
			want: []R{
				{T: json.StartObject},
				{T: json.EndObject},
				{T: json.EOF},
			},
		},
		{
			input: space + `{` + space + `}` + space,
			want: []R{
				{T: json.StartObject},
				{T: json.EndObject},
				{T: json.EOF},
			},
		},
		{
			input: space + `{` + space,
			want: []R{
				{T: json.StartObject},
				{E: `unexpected EOF`},
			},
		},
		{
			input: space + `}` + space,
			want:  []R{{E: `unexpected character }`}},
		},
		{
			input: `{` + space + `null` + space + `}`,
			want: []R{
				{T: json.StartObject},
				{E: `unexpected value null`},
			},
		},
		{
			input: `{[]}`,
			want: []R{
				{T: json.StartObject},
				{E: `unexpected character [`},
			},
		},
		{
			input: `{,}`,
			want: []R{
				{T: json.StartObject},
				{E: `unexpected character ,`},
			},
		},
		{
			input: `{"345678"}`,
			want: []R{
				{T: json.StartObject},
				{E: `unexpected character }, missing ":" after object name`},
			},
		},
		{
			input: `{` + space + `"hello"` + space + `:` + space + `"world"` + space + `}`,
			want: []R{
				{T: json.StartObject},
				{T: json.Name, V: "hello"},
				{T: json.String, V: "world"},
				{T: json.EndObject},
				{T: json.EOF},
			},
		},
		{
			input: `{"hello" "world"}`,
			want: []R{
				{T: json.StartObject},
				{E: `unexpected character ", missing ":" after object name`},
			},
		},
		{
			input: `{"hello":`,
			want: []R{
				{T: json.StartObject},
				{T: json.Name, V: "hello"},
				{E: `unexpected EOF`},
			},
		},
		{
			input: `{"hello":"world"`,
			want: []R{
				{T: json.StartObject},
				{T: json.Name, V: "hello"},
				{T: json.String, V: "world"},
				{E: `unexpected EOF`},
			},
		},
		{
			input: `{"hello":"world",`,
			want: []R{
				{T: json.StartObject},
				{T: json.Name, V: "hello"},
				{T: json.String, V: "world"},
				{E: `unexpected EOF`},
			},
		},
		{
			input: `{"34":"89",}`,
			want: []R{
				{T: json.StartObject},
				{T: json.Name, V: "34"},
				{T: json.String, V: "89"},
				{E: `syntax error (line 1:12): unexpected character }`},
			},
		},
		{
			input: `{
  "number": 123e2,
  "bool"  : false,
  "object": {"string": "world"},
  "null"  : null,
  "array" : [1.01, "hello", true],
  "string": "hello"
}`,
			want: []R{
				{T: json.StartObject},

				{T: json.Name, V: "number"},
				{T: json.Number, V: int32(12300)},

				{T: json.Name, V: "bool"},
				{T: json.Bool, V: false},

				{T: json.Name, V: "object"},
				{T: json.StartObject},
				{T: json.Name, V: "string"},
				{T: json.String, V: "world"},
				{T: json.EndObject},

				{T: json.Name, V: "null"},
				{T: json.Null},

				{T: json.Name, V: "array"},
				{T: json.StartArray},
				{T: json.Number, V: float32(1.01)},
				{T: json.String, V: "hello"},
				{T: json.Bool, V: true},
				{T: json.EndArray},

				{T: json.Name, V: "string"},
				{T: json.String, V: "hello"},

				{T: json.EndObject},
				{T: json.EOF},
			},
		},
		{
			input: `[
  {"object": {"number": 47}},
  ["list"],
  null
]`,
			want: []R{
				{T: json.StartArray},

				{T: json.StartObject},
				{T: json.Name, V: "object"},
				{T: json.StartObject},
				{T: json.Name, V: "number"},
				{T: json.Number, V: uint32(47)},
				{T: json.EndObject},
				{T: json.EndObject},

				{T: json.StartArray},
				{T: json.String, V: "list"},
				{T: json.EndArray},

				{T: json.Null},

				{T: json.EndArray},
				{T: json.EOF},
			},
		},

		// Tests for line and column info.
		{
			input: `12345678 x`,
			want: []R{
				{T: json.Number, V: int64(12345678)},
				{E: `syntax error (line 1:10): invalid value x`},
			},
		},
		{
			input: "\ntrue\n   x",
			want: []R{
				{T: json.Bool, V: true},
				{E: `syntax error (line 3:4): invalid value x`},
			},
		},
		{
			input: `"💩"x`,
			want: []R{
				{T: json.String, V: "💩"},
				{E: `syntax error (line 1:4): invalid value x`},
			},
		},
		{
			input: "\n\n[\"🔥🔥🔥\"x",
			want: []R{
				{T: json.StartArray},
				{T: json.String, V: "🔥🔥🔥"},
				{E: `syntax error (line 3:7): invalid value x`},
			},
		},
		{
			// Multi-rune emojis.
			input: `["👍🏻👍🏿"x`,
			want: []R{
				{T: json.StartArray},
				{T: json.String, V: "👍🏻👍🏿"},
				{E: `syntax error (line 1:8): invalid value x`},
			},
		},
		{
			input: `{
  "45678":-1
}`,
			want: []R{
				{T: json.StartObject},
				{T: json.Name, V: "45678"},
				{T: json.Number, V: uint64(1), VE: "error (line 2:11)"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run("", func(t *testing.T) {
			dec := json.NewDecoder([]byte(tc.input))
			for i, want := range tc.want {
				typ := dec.Peek()
				if typ != want.T {
					t.Errorf("input: %v\nPeek() got %v want %v", tc.input, typ, want.T)
				}
				value, err := dec.Read()
				if err != nil {
					if want.E == "" {
						t.Errorf("input: %v\nRead() got unexpected error: %v", tc.input, err)

					} else if !strings.Contains(err.Error(), want.E) {
						t.Errorf("input: %v\nRead() got %q, want %q", tc.input, err, want.E)
					}
				} else {
					if want.E != "" {
						t.Errorf("input: %v\nRead() got nil error, want %q", tc.input, want.E)
					}
				}
				token := value.Type()
				if token != want.T {
					t.Errorf("input: %v\nRead() got %v, want %v", tc.input, token, want.T)
					break
				}
				checkValue(t, value, i, want)
			}
		})
	}
}

func checkValue(t *testing.T, value json.Value, wantIdx int, want R) {
	var got interface{}
	var err error
	switch value.Type() {
	case json.Bool:
		got, err = value.Bool()
	case json.String:
		got = value.String()
	case json.Name:
		got, err = value.Name()
	case json.Number:
		switch want.V.(type) {
		case float32:
			got, err = value.Float(32)
			got = float32(got.(float64))
		case float64:
			got, err = value.Float(64)
		case int32:
			got, err = value.Int(32)
			got = int32(got.(int64))
		case int64:
			got, err = value.Int(64)
		case uint32:
			got, err = value.Uint(32)
			got = uint32(got.(uint64))
		case uint64:
			got, err = value.Uint(64)
		}
	default:
		return
	}

	if err != nil {
		if want.VE == "" {
			t.Errorf("want%d: %v got unexpected error: %v", wantIdx, value, err)
		} else if !strings.Contains(err.Error(), want.VE) {
			t.Errorf("want#%d: %v got %q, want %q", wantIdx, value, err, want.VE)
		}
		return
	} else {
		if want.VE != "" {
			t.Errorf("want#%d: %v got nil error, want %q", wantIdx, value, want.VE)
			return
		}
	}

	if got != want.V {
		t.Errorf("want#%d: %v got %v, want %v", wantIdx, value, got, want.V)
	}
}

func TestClone(t *testing.T) {
	input := `{"outer":{"str":"hello", "number": 123}}`
	dec := json.NewDecoder([]byte(input))

	// Clone at the start should produce the same reads as the original.
	clone := dec.Clone()
	compareDecoders(t, dec, clone)

	// Advance to inner object, clone and compare again.
	dec.Read() // Read StartObject.
	dec.Read() // Read Name.
	clone = dec.Clone()
	compareDecoders(t, dec, clone)
}

func compareDecoders(t *testing.T, d1 *json.Decoder, d2 *json.Decoder) {
	for {
		v1, err1 := d1.Read()
		v2, err2 := d2.Read()
		if v1.Type() != v2.Type() {
			t.Errorf("cloned decoder: got Type %v, want %v", v2.Type(), v1.Type())
		}
		if v1.Raw() != v2.Raw() {
			t.Errorf("cloned decoder: got Raw %v, want %v", v2.Raw(), v1.Raw())
		}
		if err1 != err2 {
			t.Errorf("cloned decoder: got error %v, want %v", err2, err1)
		}
		if v1.Type() == json.EOF {
			break
		}
	}
}
