// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"bytes"
	"strings"

	"github.com/golang/protobuf/v2/internal/detrand"
	"github.com/golang/protobuf/v2/internal/errors"
)

// Marshal serializes v as the proto text format, where v must be a Message.
// In the proto text format, the top-level value is always a message where the
// delimiters are elided.
//
// If indent is a non-empty string, it causes every entry in a List or Message
// to be preceded by the indent and trailed by a newline.
//
// If delims is not the zero value, it controls the delimiter characters used
// for messages (e.g., "{}" vs "<>").
//
// If outputASCII is true, strings will be serialized in such a way that
// multi-byte UTF-8 sequences are escaped. This property ensures that the
// overall output is ASCII (as opposed to UTF-8).
func Marshal(v Value, indent string, delims [2]byte, outputASCII bool) ([]byte, error) {
	p := encoder{}
	if len(indent) > 0 {
		if strings.Trim(indent, " \t") != "" {
			return nil, errors.New("indent may only be composed of space and tab characters")
		}
		p.indent = indent
		p.newline = "\n"
	}
	switch delims {
	case [2]byte{0, 0}:
		p.delims = [2]byte{'{', '}'}
	case [2]byte{'{', '}'}, [2]byte{'<', '>'}:
		p.delims = delims
	default:
		return nil, errors.New("delimiters may only be \"{}\" or \"<>\"")
	}
	p.outputASCII = outputASCII

	err := p.marshalMessage(v, false)
	if !p.nerr.Merge(err) {
		return nil, err
	}
	if len(indent) > 0 {
		return append(bytes.TrimRight(p.out, "\n"), '\n'), p.nerr.E
	}
	return p.out, p.nerr.E
}

type encoder struct {
	nerr errors.NonFatal
	out  []byte

	indent      string
	indents     []byte
	newline     string // set to "\n" if len(indent) > 0
	delims      [2]byte
	outputASCII bool
}

func (p *encoder) marshalList(v Value) error {
	if v.Type() != List {
		return errors.New("invalid type %v, expected list", v.Type())
	}
	elems := v.List()
	p.out = append(p.out, '[')
	p.indents = append(p.indents, p.indent...)
	if len(elems) > 0 {
		p.out = append(p.out, p.newline...)
	}
	for i, elem := range elems {
		p.out = append(p.out, p.indents...)
		if err := p.marshalValue(elem); !p.nerr.Merge(err) {
			return err
		}
		if i < len(elems)-1 {
			p.out = append(p.out, ',')
		}
		p.out = append(p.out, p.newline...)
	}
	p.indents = p.indents[:len(p.indents)-len(p.indent)]
	if len(elems) > 0 {
		p.out = append(p.out, p.indents...)
	}
	p.out = append(p.out, ']')
	return nil
}

func (p *encoder) marshalMessage(v Value, emitDelims bool) error {
	if v.Type() != Message {
		return errors.New("invalid type %v, expected message", v.Type())
	}
	items := v.Message()
	if emitDelims {
		p.out = append(p.out, p.delims[0])
		p.indents = append(p.indents, p.indent...)
		if len(items) > 0 {
			p.out = append(p.out, p.newline...)
		}
	}
	for i, item := range items {
		p.out = append(p.out, p.indents...)
		if err := p.marshalKey(item[0]); !p.nerr.Merge(err) {
			return err
		}
		p.out = append(p.out, ':')
		if len(p.indent) > 0 {
			p.out = append(p.out, ' ')
		}
		// For multi-line output, add a random extra space after key: per message to
		// make output unstable.
		if len(p.indent) > 0 && detrand.Bool() {
			p.out = append(p.out, ' ')
		}

		if err := p.marshalValue(item[1]); !p.nerr.Merge(err) {
			return err
		}
		if i < len(items)-1 && len(p.indent) == 0 {
			p.out = append(p.out, ' ')
		}
		// For single-line output, add a random extra space after a field per message to
		// make output unstable.
		if len(p.indent) == 0 && detrand.Bool() && i != len(items)-1 {
			p.out = append(p.out, ' ')
		}
		p.out = append(p.out, p.newline...)
	}
	if emitDelims {
		p.indents = p.indents[:len(p.indents)-len(p.indent)]
		if len(items) > 0 {
			p.out = append(p.out, p.indents...)
		}
		p.out = append(p.out, p.delims[1])
	}
	return nil
}

func (p *encoder) marshalKey(v Value) error {
	switch v.Type() {
	case String:
		var err error
		p.out = append(p.out, '[')
		if len(urlRegexp.FindString(v.str)) == len(v.str) {
			p.out = append(p.out, v.str...)
		} else {
			err = p.marshalString(v)
		}
		p.out = append(p.out, ']')
		return err
	case Uint:
		return p.marshalNumber(v)
	case Name:
		s, _ := v.Name()
		p.out = append(p.out, s...)
		return nil
	default:
		return errors.New("invalid type %v to encode key", v.Type())
	}
}

func (p *encoder) marshalValue(v Value) error {
	switch v.Type() {
	case Bool, Int, Uint, Float32, Float64:
		return p.marshalNumber(v)
	case String:
		return p.marshalString(v)
	case List:
		return p.marshalList(v)
	case Message:
		return p.marshalMessage(v, true)
	case Name:
		s, _ := v.Name()
		p.out = append(p.out, s...)
		return nil
	default:
		return errors.New("invalid type %v to encode value", v.Type())
	}
}
