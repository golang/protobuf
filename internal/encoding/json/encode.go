// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"strings"

	"google.golang.org/proto/internal/errors"
)

// Marshal serializes v as the JSON format.
//
// If indent is a non-empty string, it causes every entry for an Array or Object
// to be preceded by the indent and trailed by a newline.
func Marshal(v Value, indent string) ([]byte, error) {
	p := encoder{}
	if len(indent) > 0 {
		if strings.Trim(indent, " \t") != "" {
			return nil, errors.New("indent may only be composed of space and tab characters")
		}
		p.indent = indent
		p.newline = "\n"
	}
	err := p.marshalValue(v)
	if !p.nerr.Merge(err) {
		return nil, err
	}
	return p.out, p.nerr.E
}

type encoder struct {
	nerr errors.NonFatal
	out  []byte

	indent  string
	indents []byte
	newline string // set to "\n" if len(indent) > 0
}

func (p *encoder) marshalValue(v Value) error {
	switch v.Type() {
	case Null:
		p.out = append(p.out, "null"...)
		return nil
	case Bool:
		if v.Bool() {
			p.out = append(p.out, "true"...)
		} else {
			p.out = append(p.out, "false"...)
		}
		return nil
	case Number:
		return p.marshalNumber(v)
	case String:
		return p.marshalString(v)
	case Array:
		return p.marshalArray(v)
	case Object:
		return p.marshalObject(v)
	default:
		return errors.New("invalid type %v to encode value", v.Type())
	}
}

func (p *encoder) marshalArray(v Value) error {
	if v.Type() != Array {
		return errors.New("invalid type %v, expected array", v.Type())
	}
	elems := v.Array()
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

func (p *encoder) marshalObject(v Value) error {
	if v.Type() != Object {
		return errors.New("invalid type %v, expected object", v.Type())
	}
	items := v.Object()
	p.out = append(p.out, '{')
	p.indents = append(p.indents, p.indent...)
	if len(items) > 0 {
		p.out = append(p.out, p.newline...)
	}
	for i, item := range items {
		p.out = append(p.out, p.indents...)
		if err := p.marshalString(item[0]); !p.nerr.Merge(err) {
			return err
		}
		p.out = append(p.out, ':')
		if len(p.indent) > 0 {
			p.out = append(p.out, ' ')
		}
		if err := p.marshalValue(item[1]); !p.nerr.Merge(err) {
			return err
		}
		if i < len(items)-1 {
			p.out = append(p.out, ',')
		}
		p.out = append(p.out, p.newline...)
	}
	p.indents = p.indents[:len(p.indents)-len(p.indent)]
	if len(items) > 0 {
		p.out = append(p.out, p.indents...)
	}
	p.out = append(p.out, '}')
	return nil
}
