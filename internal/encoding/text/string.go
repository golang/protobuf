// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"bytes"
	"io"
	"math"
	"math/bits"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/golang/protobuf/v2/internal/errors"
)

func (p *encoder) marshalString(v Value) error {
	var err error
	p.out, err = appendString(p.out, v, p.outputASCII)
	return err
}
func appendString(out []byte, v Value, outputASCII bool) ([]byte, error) {
	if v.Type() != String {
		return nil, errors.New("invalid type %v, expected string", v.Type())
	}
	if len(v.raw) > 0 {
		return append(out, v.raw...), nil
	}
	in := v.String()

	out = append(out, '"')
	i := indexNeedEscape(in)
	in, out = in[i:], append(out, in[:i]...)
	for len(in) > 0 {
		switch r, n := utf8.DecodeRuneInString(in); {
		case r == utf8.RuneError && n == 1:
			// We do not report invalid UTF-8 because strings in the text format
			// are used to represent both the proto string and bytes type.
			r = rune(in[0])
			fallthrough
		case r < ' ' || r == '"' || r == '\\':
			out = append(out, '\\')
			switch r {
			case '"', '\\':
				out = append(out, byte(r))
			case '\n':
				out = append(out, 'n')
			case '\r':
				out = append(out, 'r')
			case '\t':
				out = append(out, 't')
			default:
				out = append(out, 'x')
				out = append(out, "00"[1+(bits.Len32(uint32(r))-1)/4:]...)
				out = strconv.AppendUint(out, uint64(r), 16)
			}
			in = in[n:]
		case outputASCII && r >= utf8.RuneSelf:
			out = append(out, '\\')
			if r <= math.MaxUint16 {
				out = append(out, 'u')
				out = append(out, "0000"[1+(bits.Len32(uint32(r))-1)/4:]...)
				out = strconv.AppendUint(out, uint64(r), 16)
			} else {
				out = append(out, 'U')
				out = append(out, "00000000"[1+(bits.Len32(uint32(r))-1)/4:]...)
				out = strconv.AppendUint(out, uint64(r), 16)
			}
			in = in[n:]
		default:
			i := indexNeedEscape(in[n:])
			in, out = in[n+i:], append(out, in[:n+i]...)
		}
	}
	out = append(out, '"')
	return out, nil
}

func (p *decoder) unmarshalString() (Value, error) {
	v, n, err := consumeString(p.in)
	p.consume(n)
	return v, err
}
func consumeString(in []byte) (Value, int, error) {
	var nerr errors.NonFatal
	in0 := in
	if len(in) == 0 {
		return Value{}, 0, io.ErrUnexpectedEOF
	}
	quote := in[0]
	if in[0] != '"' && in[0] != '\'' {
		return Value{}, 0, newSyntaxError("invalid character %q at start of string", in[0])
	}
	in = in[1:]
	i := indexNeedEscape(string(in))
	in, out := in[i:], in[:i:i] // set cap to prevent mutations
	for len(in) > 0 {
		switch r, n := utf8.DecodeRune(in); {
		case r == utf8.RuneError && n == 1:
			nerr.AppendInvalidUTF8("")
			in, out = in[1:], append(out, in[0]) // preserve invalid byte
		case r == 0 || r == '\n':
			return Value{}, 0, newSyntaxError("invalid character %q in string", r)
		case r == rune(quote):
			in = in[1:]
			n := len(in0) - len(in)
			v := rawValueOf(string(out), in0[:n:n])
			return v, n, nerr.E
		case r == '\\':
			if len(in) < 2 {
				return Value{}, 0, io.ErrUnexpectedEOF
			}
			switch r := in[1]; r {
			case '"', '\'', '\\', '?':
				in, out = in[2:], append(out, r)
			case 'a':
				in, out = in[2:], append(out, '\a')
			case 'b':
				in, out = in[2:], append(out, '\b')
			case 'n':
				in, out = in[2:], append(out, '\n')
			case 'r':
				in, out = in[2:], append(out, '\r')
			case 't':
				in, out = in[2:], append(out, '\t')
			case 'v':
				in, out = in[2:], append(out, '\v')
			case 'f':
				in, out = in[2:], append(out, '\f')
			case '0', '1', '2', '3', '4', '5', '6', '7':
				// One, two, or three octal characters.
				n := len(in[1:]) - len(bytes.TrimLeft(in[1:], "01234567"))
				if n > 3 {
					n = 3
				}
				v, err := strconv.ParseUint(string(in[1:1+n]), 8, 8)
				if err != nil {
					return Value{}, 0, newSyntaxError("invalid octal escape code %q in string", in[:1+n])
				}
				in, out = in[1+n:], append(out, byte(v))
			case 'x':
				// One or two hexadecimal characters.
				n := len(in[2:]) - len(bytes.TrimLeft(in[2:], "0123456789abcdefABCDEF"))
				if n > 2 {
					n = 2
				}
				v, err := strconv.ParseUint(string(in[2:2+n]), 16, 8)
				if err != nil {
					return Value{}, 0, newSyntaxError("invalid hex escape code %q in string", in[:2+n])
				}
				in, out = in[2+n:], append(out, byte(v))
			case 'u', 'U':
				// Four or eight hexadecimal characters
				n := 6
				if r == 'U' {
					n = 10
				}
				if len(in) < n {
					return Value{}, 0, io.ErrUnexpectedEOF
				}
				v, err := strconv.ParseUint(string(in[2:n]), 16, 32)
				if utf8.MaxRune < v || err != nil {
					return Value{}, 0, newSyntaxError("invalid Unicode escape code %q in string", in[:n])
				}
				in = in[n:]

				r := rune(v)
				if utf16.IsSurrogate(r) {
					if len(in) < 6 {
						return Value{}, 0, io.ErrUnexpectedEOF
					}
					v, err := strconv.ParseUint(string(in[2:6]), 16, 16)
					r = utf16.DecodeRune(r, rune(v))
					if in[0] != '\\' || in[1] != 'u' || r == unicode.ReplacementChar || err != nil {
						return Value{}, 0, newSyntaxError("invalid Unicode escape code %q in string", in[:6])
					}
					in = in[6:]
				}
				out = append(out, string(r)...)
			default:
				return Value{}, 0, newSyntaxError("invalid escape code %q in string", in[:2])
			}
		default:
			i := indexNeedEscape(string(in[n:]))
			in, out = in[n+i:], append(out, in[:n+i]...)
		}
	}
	return Value{}, 0, io.ErrUnexpectedEOF
}

// unmarshalStrings unmarshals multiple strings.
// This differs from unmarshalString since the text format allows
// multiple back-to-back string literals where they are semantically treated
// as a single large string with all values concatenated.
//
// E.g., `"foo" "bar" "baz"` => ValueOf("foobarbaz")
func (p *decoder) unmarshalStrings() (Value, error) {
	// Note that the ending quote is sufficient to unambiguously mark the end
	// of a string. Thus, the text grammar does not require intervening
	// whitespace or control characters in-between strings.
	// Thus, the following is valid:
	//	`"foo"'bar'"baz"` => ValueOf("foobarbaz")
	b := p.in
	var ss []string
	for len(p.in) > 0 && (p.in[0] == '"' || p.in[0] == '\'') {
		v, err := p.unmarshalString()
		if !p.nerr.Merge(err) {
			return Value{}, err
		}
		ss = append(ss, v.String())
	}
	b = b[:len(b)-len(p.in)]
	return rawValueOf(strings.Join(ss, ""), b[:len(b):len(b)]), nil
}

// indexNeedEscape returns the index of the next character that needs escaping.
// If no characters need escaping, this returns the input length.
func indexNeedEscape(s string) int {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < ' ' || c == '"' || c == '\'' || c == '\\' || c >= utf8.RuneSelf {
			return i
		}
	}
	return len(s)
}
