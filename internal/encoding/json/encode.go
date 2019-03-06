// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"strconv"
	"strings"

	"github.com/golang/protobuf/v2/internal/errors"
)

// Encoder provides methods to write out JSON constructs and values. The user is
// responsible for producing valid sequences of JSON constructs and values.
type Encoder struct {
	indent   string
	lastType Type
	indents  []byte
	out      []byte
}

// NewEncoder returns an Encoder.
//
// If indent is a non-empty string, it causes every entry for an Array or Object
// to be preceded by the indent and trailed by a newline.
func NewEncoder(indent string) (*Encoder, error) {
	e := &Encoder{}
	if len(indent) > 0 {
		if strings.Trim(indent, " \t") != "" {
			return nil, errors.New("indent may only be composed of space or tab characters")
		}
		e.indent = indent
	}
	return e, nil
}

// Bytes returns the content of the written bytes.
func (e *Encoder) Bytes() []byte {
	return e.out
}

// WriteNull writes out the null value.
func (e *Encoder) WriteNull() {
	e.prepareNext(Null)
	e.out = append(e.out, "null"...)
}

// WriteBool writes out the given boolean value.
func (e *Encoder) WriteBool(b bool) {
	e.prepareNext(Bool)
	if b {
		e.out = append(e.out, "true"...)
	} else {
		e.out = append(e.out, "false"...)
	}
}

// WriteString writes out the given string in JSON string value.
func (e *Encoder) WriteString(s string) error {
	e.prepareNext(String)
	var err error
	if e.out, err = appendString(e.out, s); err != nil {
		return err
	}
	return nil
}

// WriteFloat writes out the given float and bitSize in JSON number value.
func (e *Encoder) WriteFloat(n float64, bitSize int) {
	e.prepareNext(Number)
	e.out = appendFloat(e.out, n, bitSize)
}

// WriteInt writes out the given signed integer in JSON number value.
func (e *Encoder) WriteInt(n int64) {
	e.prepareNext(Number)
	e.out = append(e.out, strconv.FormatInt(n, 10)...)
}

// WriteUint writes out the given unsigned integer in JSON number value.
func (e *Encoder) WriteUint(n uint64) {
	e.prepareNext(Number)
	e.out = append(e.out, strconv.FormatUint(n, 10)...)
}

// StartObject writes out the '{' symbol.
func (e *Encoder) StartObject() {
	e.prepareNext(StartObject)
	e.out = append(e.out, '{')
}

// EndObject writes out the '}' symbol.
func (e *Encoder) EndObject() {
	e.prepareNext(EndObject)
	e.out = append(e.out, '}')
}

// WriteName writes out the given string in JSON string value and the name
// separator ':'.
func (e *Encoder) WriteName(s string) error {
	e.prepareNext(Name)
	// Errors returned by appendString() are non-fatal.
	var err error
	e.out, err = appendString(e.out, s)
	e.out = append(e.out, ':')
	return err
}

// StartArray writes out the '[' symbol.
func (e *Encoder) StartArray() {
	e.prepareNext(StartArray)
	e.out = append(e.out, '[')
}

// EndArray writes out the ']' symbol.
func (e *Encoder) EndArray() {
	e.prepareNext(EndArray)
	e.out = append(e.out, ']')
}

// prepareNext adds possible comma and indentation for the next value based
// on last type and indent option. It also updates lastType to next.
func (e *Encoder) prepareNext(next Type) {
	defer func() {
		// Set lastType to next.
		e.lastType = next
	}()

	if len(e.indent) == 0 {
		// Need to add comma on the following condition.
		if e.lastType&(Null|Bool|Number|String|EndObject|EndArray) != 0 &&
			next&(Name|Null|Bool|Number|String|StartObject|StartArray) != 0 {
			e.out = append(e.out, ',')
		}
		return
	}

	switch {
	case e.lastType&(StartObject|StartArray) != 0:
		// If next type is NOT closing, add indent and newline.
		if next&(EndObject|EndArray) == 0 {
			e.indents = append(e.indents, e.indent...)
			e.out = append(e.out, '\n')
			e.out = append(e.out, e.indents...)
		}

	case e.lastType&(Null|Bool|Number|String|EndObject|EndArray) != 0:
		switch {
		// If next type is either a value or name, add comma and newline.
		case next&(Name|Null|Bool|Number|String|StartObject|StartArray) != 0:
			e.out = append(e.out, ',', '\n')

		// If next type is a closing object or array, adjust indentation.
		case next&(EndObject|EndArray) != 0:
			e.indents = e.indents[:len(e.indents)-len(e.indent)]
			e.out = append(e.out, '\n')
		}
		e.out = append(e.out, e.indents...)

	case e.lastType&Name != 0:
		e.out = append(e.out, ' ')
	}
}
