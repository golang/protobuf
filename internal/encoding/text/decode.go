// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"bytes"
	"io"
	"regexp"
	"unicode/utf8"

	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

type syntaxError struct{ error }

func newSyntaxError(f string, x ...interface{}) error {
	return syntaxError{errors.New(f, x...)}
}

// Unmarshal parses b as the proto text format.
// It returns a Value, which is always of the Message type.
func Unmarshal(b []byte) (Value, error) {
	p := decoder{in: b}
	p.consume(0) // trim leading spaces or comments
	v, err := p.unmarshalMessage(false)
	if !p.nerr.Merge(err) {
		if e, ok := err.(syntaxError); ok {
			b = b[:len(b)-len(p.in)] // consumed input
			line := bytes.Count(b, []byte("\n")) + 1
			if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
				b = b[i+1:]
			}
			column := utf8.RuneCount(b) + 1 // ignore multi-rune characters
			err = errors.New("syntax error (line %d:%d): %v", line, column, e.error)
		}
		return Value{}, err
	}
	if len(p.in) > 0 {
		return Value{}, errors.New("%d bytes of unconsumed input", len(p.in))
	}
	return v, p.nerr.E
}

type decoder struct {
	nerr errors.NonFatal
	in   []byte
}

func (p *decoder) unmarshalList() (Value, error) {
	b := p.in
	var elems []Value
	if err := p.consumeChar('[', "at start of list"); err != nil {
		return Value{}, err
	}
	if len(p.in) > 0 && p.in[0] != ']' {
		for len(p.in) > 0 {
			v, err := p.unmarshalValue()
			if !p.nerr.Merge(err) {
				return Value{}, err
			}
			elems = append(elems, v)
			if !p.tryConsumeChar(',') {
				break
			}
		}
	}
	if err := p.consumeChar(']', "at end of list"); err != nil {
		return Value{}, err
	}
	b = b[:len(b)-len(p.in)]
	return rawValueOf(elems, b[:len(b):len(b)]), nil
}

func (p *decoder) unmarshalMessage(checkDelims bool) (Value, error) {
	b := p.in
	var items [][2]Value
	delims := [2]byte{'{', '}'}
	if len(p.in) > 0 && p.in[0] == '<' {
		delims = [2]byte{'<', '>'}
	}
	if checkDelims {
		if err := p.consumeChar(delims[0], "at start of message"); err != nil {
			return Value{}, err
		}
	}
	for len(p.in) > 0 {
		if p.in[0] == '}' || p.in[0] == '>' {
			break
		}
		k, err := p.unmarshalKey()
		if !p.nerr.Merge(err) {
			return Value{}, err
		}
		if !p.tryConsumeChar(':') && len(p.in) > 0 && p.in[0] != '{' && p.in[0] != '<' {
			return Value{}, newSyntaxError("expected ':' after message key")
		}
		v, err := p.unmarshalValue()
		if !p.nerr.Merge(err) {
			return Value{}, err
		}
		if p.tryConsumeChar(';') || p.tryConsumeChar(',') {
			// always optional
		}
		items = append(items, [2]Value{k, v})
	}
	if checkDelims {
		if err := p.consumeChar(delims[1], "at end of message"); err != nil {
			return Value{}, err
		}
	}
	b = b[:len(b)-len(p.in)]
	return rawValueOf(items, b[:len(b):len(b)]), nil
}

// This expression is more liberal than ConsumeAnyTypeUrl in C++.
// However, the C++ parser does not handle many legal URL strings.
// The Go implementation is more liberal to be backwards compatible with
// the historical Go implementation which was overly liberal (and buggy).
var urlRegexp = regexp.MustCompile(`^[-_a-zA-Z0-9]+([./][-_a-zA-Z0-9]+)*`)

// unmarshalKey parses the key, which may be a Name, String, or Uint.
func (p *decoder) unmarshalKey() (v Value, err error) {
	if p.tryConsumeChar('[') {
		if len(p.in) == 0 {
			return Value{}, io.ErrUnexpectedEOF
		}
		if p.in[0] == '\'' || p.in[0] == '"' {
			// Historically, Go's parser allowed a string for the Any type URL.
			// This is specific to Go and contrary to the C++ implementation,
			// which does not support strings for the Any type URL.
			v, err = p.unmarshalString()
			if !p.nerr.Merge(err) {
				return Value{}, err
			}
		} else if n := matchWithDelim(urlRegexp, p.in); n > 0 {
			v = rawValueOf(string(p.in[:n]), p.in[:n:n])
			p.consume(n)
		} else {
			return Value{}, newSyntaxError("invalid %q as identifier", errRegexp.Find(p.in))
		}
		if err := p.consumeChar(']', "at end of extension name"); err != nil {
			return Value{}, err
		}
		return v, nil
	}
	if matchWithDelim(intRegexp, p.in) > 0 && p.in[0] != '-' {
		return p.unmarshalNumber()
	}
	return p.unmarshalName()
}

func (p *decoder) unmarshalValue() (Value, error) {
	if len(p.in) == 0 {
		return Value{}, io.ErrUnexpectedEOF
	}
	switch p.in[0] {
	case '"', '\'':
		return p.unmarshalStrings()
	case '[':
		return p.unmarshalList()
	case '{', '<':
		return p.unmarshalMessage(true)
	default:
		n := matchWithDelim(nameRegexp, p.in) // zero if no match
		if n > 0 && literals[string(p.in[:n])] == nil {
			return p.unmarshalName()
		}
		return p.unmarshalNumber()
	}
}

// This expression matches all valid proto identifiers.
var nameRegexp = regexp.MustCompile(`^[_a-zA-Z][_a-zA-Z0-9]*`)

// unmarshalName unmarshals an unquoted identifier.
//
// E.g., `field_name` => ValueOf(protoreflect.Name("field_name"))
func (p *decoder) unmarshalName() (Value, error) {
	if n := matchWithDelim(nameRegexp, p.in); n > 0 {
		v := rawValueOf(protoreflect.Name(p.in[:n]), p.in[:n:n])
		p.consume(n)
		return v, nil
	}
	return Value{}, newSyntaxError("invalid %q as identifier", errRegexp.Find(p.in))
}

func (p *decoder) consumeChar(c byte, msg string) error {
	if p.tryConsumeChar(c) {
		return nil
	}
	if len(p.in) == 0 {
		return io.ErrUnexpectedEOF
	}
	return newSyntaxError("invalid character %q, expected %q %s", p.in[0], c, msg)
}

func (p *decoder) tryConsumeChar(c byte) bool {
	if len(p.in) > 0 && p.in[0] == c {
		p.consume(1)
		return true
	}
	return false
}

// consume consumes n bytes of input and any subsequent whitespace or comments.
func (p *decoder) consume(n int) {
	p.in = p.in[n:]
	for len(p.in) > 0 {
		switch p.in[0] {
		case ' ', '\n', '\r', '\t':
			p.in = p.in[1:]
		case '#':
			if i := bytes.IndexByte(p.in, '\n'); i >= 0 {
				p.in = p.in[i+len("\n"):]
			} else {
				p.in = nil
			}
		default:
			return
		}
	}
}

// Any sequence that looks like a non-delimiter (for error reporting).
var errRegexp = regexp.MustCompile("^([-+._a-zA-Z0-9]{1,32}|.)")

// matchWithDelim matches r with the input b and verifies that the match
// terminates with a delimiter of some form (e.g., r"[^-+_.a-zA-Z0-9]").
// As a special case, EOF is considered a delimiter.
func matchWithDelim(r *regexp.Regexp, b []byte) int {
	n := len(r.Find(b))
	if n < len(b) {
		// Check that that the next character is a delimiter.
		c := b[n]
		notDelim := (c == '-' || c == '+' || c == '.' || c == '_' ||
			('a' <= c && c <= 'z') ||
			('A' <= c && c <= 'Z') ||
			('0' <= c && c <= '9'))
		if notDelim {
			return 0
		}
	}
	return n
}
