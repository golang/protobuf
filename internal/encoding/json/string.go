// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"io"
	"math/bits"
	"strconv"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/golang/protobuf/v2/internal/errors"
)

func appendString(out []byte, in string) ([]byte, error) {
	var nerr errors.NonFatal
	out = append(out, '"')
	i := indexNeedEscape(in)
	in, out = in[i:], append(out, in[:i]...)
	for len(in) > 0 {
		switch r, n := utf8.DecodeRuneInString(in); {
		case r == utf8.RuneError && n == 1:
			nerr.AppendInvalidUTF8("")
			in, out = in[1:], append(out, in[0]) // preserve invalid byte
		case r < ' ' || r == '"' || r == '\\':
			out = append(out, '\\')
			switch r {
			case '"', '\\':
				out = append(out, byte(r))
			case '\b':
				out = append(out, 'b')
			case '\f':
				out = append(out, 'f')
			case '\n':
				out = append(out, 'n')
			case '\r':
				out = append(out, 'r')
			case '\t':
				out = append(out, 't')
			default:
				out = append(out, 'u')
				out = append(out, "0000"[1+(bits.Len32(uint32(r))-1)/4:]...)
				out = strconv.AppendUint(out, uint64(r), 16)
			}
			in = in[n:]
		default:
			i := indexNeedEscape(in[n:])
			in, out = in[n+i:], append(out, in[:n+i]...)
		}
	}
	out = append(out, '"')
	return out, nerr.E
}

func (d *Decoder) parseString(in []byte) (string, int, error) {
	var nerr errors.NonFatal
	in0 := in
	if len(in) == 0 {
		return "", 0, io.ErrUnexpectedEOF
	}
	if in[0] != '"' {
		return "", 0, d.newSyntaxError("invalid character %q at start of string", in[0])
	}
	in = in[1:]
	i := indexNeedEscape(string(in))
	in, out := in[i:], in[:i:i] // set cap to prevent mutations
	for len(in) > 0 {
		switch r, n := utf8.DecodeRune(in); {
		case r == utf8.RuneError && n == 1:
			nerr.AppendInvalidUTF8("")
			in, out = in[1:], append(out, in[0]) // preserve invalid byte
		case r < ' ':
			return "", 0, d.newSyntaxError("invalid character %q in string", r)
		case r == '"':
			in = in[1:]
			n := len(in0) - len(in)
			return string(out), n, nerr.E
		case r == '\\':
			if len(in) < 2 {
				return "", 0, io.ErrUnexpectedEOF
			}
			switch r := in[1]; r {
			case '"', '\\', '/':
				in, out = in[2:], append(out, r)
			case 'b':
				in, out = in[2:], append(out, '\b')
			case 'f':
				in, out = in[2:], append(out, '\f')
			case 'n':
				in, out = in[2:], append(out, '\n')
			case 'r':
				in, out = in[2:], append(out, '\r')
			case 't':
				in, out = in[2:], append(out, '\t')
			case 'u':
				if len(in) < 6 {
					return "", 0, io.ErrUnexpectedEOF
				}
				v, err := strconv.ParseUint(string(in[2:6]), 16, 16)
				if err != nil {
					return "", 0, d.newSyntaxError("invalid escape code %q in string", in[:6])
				}
				in = in[6:]

				r := rune(v)
				if utf16.IsSurrogate(r) {
					if len(in) < 6 {
						return "", 0, io.ErrUnexpectedEOF
					}
					v, err := strconv.ParseUint(string(in[2:6]), 16, 16)
					r = utf16.DecodeRune(r, rune(v))
					if in[0] != '\\' || in[1] != 'u' ||
						r == unicode.ReplacementChar || err != nil {
						return "", 0, d.newSyntaxError("invalid escape code %q in string", in[:6])
					}
					in = in[6:]
				}
				out = append(out, string(r)...)
			default:
				return "", 0, d.newSyntaxError("invalid escape code %q in string", in[:2])
			}
		default:
			i := indexNeedEscape(string(in[n:]))
			in, out = in[n+i:], append(out, in[:n+i]...)
		}
	}
	return "", 0, io.ErrUnexpectedEOF
}

// indexNeedEscape returns the index of the next character that needs escaping.
// If no characters need escaping, this returns the input length.
func indexNeedEscape(s string) int {
	for i, r := range s {
		if r < ' ' || r == '\\' || r == '"' || r == utf8.RuneError {
			return i
		}
	}
	return len(s)
}
