// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"io"
	"regexp"
	"unicode/utf8"

	"google.golang.org/proto/internal/errors"
)

type syntaxError struct{ error }

func newSyntaxError(f string, x ...interface{}) error {
	return syntaxError{errors.New(f, x...)}
}

// Unmarshal parses b as the JSON format.
// It returns a Value, which represents the input as an AST.
func Unmarshal(b []byte) (Value, error) {
	p := decoder{in: b}
	p.consume(0) // trim leading spaces
	v, err := p.unmarshalValue()
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

var literalRegexp = regexp.MustCompile("^(null|true|false)")

func (p *decoder) unmarshalValue() (Value, error) {
	if len(p.in) == 0 {
		return Value{}, io.ErrUnexpectedEOF
	}
	switch p.in[0] {
	case 'n', 't', 'f':
		if n := matchWithDelim(literalRegexp, p.in); n > 0 {
			var v Value
			switch p.in[0] {
			case 'n':
				v = rawValueOf(nil, p.in[:n:n])
			case 't':
				v = rawValueOf(true, p.in[:n:n])
			case 'f':
				v = rawValueOf(false, p.in[:n:n])
			}
			p.consume(n)
			return v, nil
		}
		return Value{}, newSyntaxError("invalid %q as literal", errRegexp.Find(p.in))
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return p.unmarshalNumber()
	case '"':
		return p.unmarshalString()
	case '[':
		return p.unmarshalArray()
	case '{':
		return p.unmarshalObject()
	default:
		return Value{}, newSyntaxError("invalid %q as value", errRegexp.Find(p.in))
	}
}

func (p *decoder) unmarshalArray() (Value, error) {
	b := p.in
	var elems []Value
	if err := p.consumeChar('[', "at start of array"); err != nil {
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
	if err := p.consumeChar(']', "at end of array"); err != nil {
		return Value{}, err
	}
	b = b[:len(b)-len(p.in)]
	return rawValueOf(elems, b[:len(b):len(b)]), nil
}

func (p *decoder) unmarshalObject() (Value, error) {
	b := p.in
	var items [][2]Value
	if err := p.consumeChar('{', "at start of object"); err != nil {
		return Value{}, err
	}
	if len(p.in) > 0 && p.in[0] != '}' {
		for len(p.in) > 0 {
			k, err := p.unmarshalString()
			if !p.nerr.Merge(err) {
				return Value{}, err
			}
			if err := p.consumeChar(':', "in object"); err != nil {
				return Value{}, err
			}
			v, err := p.unmarshalValue()
			if !p.nerr.Merge(err) {
				return Value{}, err
			}
			items = append(items, [2]Value{k, v})
			if !p.tryConsumeChar(',') {
				break
			}
		}
	}
	if err := p.consumeChar('}', "at end of object"); err != nil {
		return Value{}, err
	}
	b = b[:len(b)-len(p.in)]
	return rawValueOf(items, b[:len(b):len(b)]), nil
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

// consume consumes n bytes of input and any subsequent whitespace.
func (p *decoder) consume(n int) {
	p.in = p.in[n:]
	for len(p.in) > 0 {
		switch p.in[0] {
		case ' ', '\n', '\r', '\t':
			p.in = p.in[1:]
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
