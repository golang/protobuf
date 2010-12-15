// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 Google Inc.  All rights reserved.
// http://code.google.com/p/goprotobuf/
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package proto

// Functions for parsing the Text protocol buffer format.
// TODO:
//     - message sets, groups.

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

// ParseError satisfies the os.Error interface.
type ParseError struct {
	Message string
	Line    int // 1-based line number
	Offset  int // 0-based byte offset from start of input
}

func (p *ParseError) String() string {
	if p.Line == 1 {
		// show offset only for first line
		return fmt.Sprintf("line 1.%d: %v", p.Offset, p.Message)
	}
	return fmt.Sprintf("line %d: %v", p.Line, p.Message)
}

type token struct {
	value    string
	err      *ParseError
	line     int    // line number
	offset   int    // byte number from start of input, not start of line
	unquoted string // the unquoted version of value, if it was a quoted string
}

func (t *token) String() string {
	if t.err == nil {
		return fmt.Sprintf("%q (line=%d, offset=%d)", t.value, t.line, t.offset)
	}
	return fmt.Sprintf("parse error: %v", t.err)
}

type textParser struct {
	s            string // remaining input
	done         bool   // whether the parsing is finished (success or error)
	backed       bool   // whether back() was called
	offset, line int
	cur          token
}

func newTextParser(s string) *textParser {
	p := new(textParser)
	p.s = s
	p.line = 1
	p.cur.line = 1
	return p
}

func (p *textParser) error(format string, a ...interface{}) *ParseError {
	pe := &ParseError{fmt.Sprintf(format, a...), p.cur.line, p.cur.offset}
	p.cur.err = pe
	p.done = true
	return pe
}

// Numbers and identifiers are matched by [-+._A-Za-z0-9]
func isIdentOrNumberChar(c byte) bool {
	switch {
	case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z':
		return true
	case '0' <= c && c <= '9':
		return true
	}
	switch c {
	case '-', '+', '.', '_':
		return true
	}
	return false
}

func isWhitespace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

func (p *textParser) skipWhitespace() {
	i := 0
	for i < len(p.s) && (isWhitespace(p.s[i]) || p.s[i] == '#') {
		if p.s[i] == '#' {
			// comment; skip to end of line or input
			for i < len(p.s) && p.s[i] != '\n' {
				i++
			}
			if i == len(p.s) {
				break
			}
		}
		if p.s[i] == '\n' {
			p.line++
		}
		i++
	}
	p.offset += i
	p.s = p.s[i:len(p.s)]
	if len(p.s) == 0 {
		p.done = true
	}
}

func (p *textParser) advance() {
	// Skip whitespace
	p.skipWhitespace()
	if p.done {
		return
	}

	// Start of non-whitespace
	p.cur.err = nil
	p.cur.offset, p.cur.line = p.offset, p.line
	p.cur.unquoted = ""
	switch p.s[0] {
	case '<', '>', '{', '}', ':':
		// Single symbol
		p.cur.value, p.s = p.s[0:1], p.s[1:len(p.s)]
	case '"':
		// Quoted string
		i := 1
		for i < len(p.s) && p.s[i] != '"' && p.s[i] != '\n' {
			if p.s[i] == '\\' && i+1 < len(p.s) {
				// skip escaped char
				i++
			}
			i++
		}
		if i >= len(p.s) || p.s[i] != '"' {
			p.error("unmatched quote")
			return
		}
		// TODO: Should be UnquoteC.
		unq, err := strconv.Unquote(p.s[0 : i+1])
		if err != nil {
			p.error("invalid quoted string %v", p.s[0:i+1])
			return
		}
		p.cur.value, p.s = p.s[0:i+1], p.s[i+1:len(p.s)]
		p.cur.unquoted = unq
	default:
		i := 0
		for i < len(p.s) && isIdentOrNumberChar(p.s[i]) {
			i++
		}
		if i == 0 {
			p.error("unexpected byte %#x", p.s[0])
			return
		}
		p.cur.value, p.s = p.s[0:i], p.s[i:len(p.s)]
	}
	p.offset += len(p.cur.value)
}

// Back off the parser by one token. Can only be done between calls to next().
// It makes the next advance() a no-op.
func (p *textParser) back() { p.backed = true }

// Advances the parser and returns the new current token.
func (p *textParser) next() *token {
	if p.backed || p.done {
		p.backed = false
		return &p.cur
	}
	p.advance()
	if p.done {
		p.cur.value = ""
	} else if len(p.cur.value) > 0 && p.cur.value[0] == '"' {
		// Look for multiple quoted strings separated by whitespace,
		// and concatenate them.
		cat := p.cur
		for {
			p.skipWhitespace()
			if p.done || p.s[0] != '"' {
				break
			}
			p.advance()
			if p.cur.err != nil {
				return &p.cur
			}
			cat.value += " " + p.cur.value
			cat.unquoted += p.cur.unquoted
		}
		p.done = false // parser may have seen EOF, but we want to return cat
		p.cur = cat
	}
	return &p.cur
}

type nillable interface {
	IsNil() bool
}

// Return an error indicating which required field was not set.
func (p *textParser) missingRequiredFieldError(sv *reflect.StructValue) *ParseError {
	st := sv.Type().(*reflect.StructType)
	sprops := GetProperties(st)
	for i := 0; i < st.NumField(); i++ {
		// All protocol buffer fields are nillable, but let's be careful.
		nfv, ok := sv.Field(i).(nillable)
		if !ok || !nfv.IsNil() {
			continue
		}

		props := sprops.Prop[i]
		if props.Required {
			return p.error("message %v missing required field %q", st, props.OrigName)
		}
	}
	return p.error("message %v missing required field", st) // should not happen
}

// Returns the index in the struct for the named field, as well as the parsed tag properties.
func structFieldByName(st *reflect.StructType, name string) (int, *Properties, bool) {
	sprops := GetProperties(st)
	i, ok := sprops.origNames[name]
	if ok {
		return i, sprops.Prop[i], true
	}
	return -1, nil, false
}

func (p *textParser) readStruct(sv *reflect.StructValue, terminator string) *ParseError {
	st := sv.Type().(*reflect.StructType)
	reqCount := GetProperties(st).reqCount
	// A struct is a sequence of "name: value", terminated by one of
	// '>' or '}', or the end of the input.
	for {
		tok := p.next()
		if tok.err != nil {
			return tok.err
		}
		if tok.value == terminator {
			break
		}

		fi, props, ok := structFieldByName(st, tok.value)
		if !ok {
			return p.error("unknown field name %q in %v", tok.value, st)
		}

		// Check that it's not already set if it's not a repeated field.
		if !props.Repeated {
			if nfv, ok := sv.Field(fi).(nillable); ok && !nfv.IsNil() {
				return p.error("non-repeated field %q was repeated", tok.value)
			}
		}

		tok = p.next()
		if tok.err != nil {
			return tok.err
		}
		if tok.value != ":" {
			// Colon is optional when the field is a group or message.
			needColon := true
			switch props.Wire {
			case "group":
				needColon = false
			case "bytes":
				// A "bytes" field is either a message, a string, or a repeated field;
				// those three become *T, *string and []T respectively, so we can check for
				// this field being a pointer to a non-string.
				typ := st.Field(fi).Type
				if pt, ok := typ.(*reflect.PtrType); ok {
					// *T or *string
					if _, ok := pt.Elem().(*reflect.StringType); ok {
						break
					}
				} else if st, ok := typ.(*reflect.SliceType); ok {
					// []T or []*T
					if _, ok := st.Elem().(*reflect.PtrType); !ok {
						break
					}
				}
				needColon = false
			}
			if needColon {
				return p.error("expected ':', found %q", tok.value)
			}
			p.back()
		}

		// Parse into the field.
		if err := p.readAny(sv.Field(fi), props); err != nil {
			return err
		}

		if props.Required {
			reqCount--
		}
	}

	if reqCount > 0 {
		return p.missingRequiredFieldError(sv)
	}
	return nil
}

const (
	minInt32  = -1 << 31
	maxInt32  = 1<<31 - 1
	maxUint32 = 1<<32 - 1
)

func (p *textParser) readAny(v reflect.Value, props *Properties) *ParseError {
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value == "" {
		return p.error("unexpected EOF")
	}

	switch fv := v.(type) {
	case *reflect.SliceValue:
		at := v.Type().(*reflect.SliceType)
		if at.Elem().Kind() == reflect.Uint8 {
			// Special case for []byte
			if tok.value[0] != '"' {
				// Deliberately written out here, as the error after
				// this switch statement would write "invalid []byte: ...",
				// which is not as user-friendly.
				return p.error("invalid string: %v", tok.value)
			}
			bytes := []byte(tok.unquoted)
			fv.Set(reflect.NewValue(bytes).(*reflect.SliceValue))
			return nil
		}
		// Repeated field. May already exist.
		flen := fv.Len()
		if flen == fv.Cap() {
			nav := reflect.MakeSlice(at, flen, 2*flen+1)
			reflect.Copy(nav, fv)
			fv.Set(nav)
		}
		fv.SetLen(flen + 1)

		// Read one.
		p.back()
		return p.readAny(fv.Elem(flen), nil) // TODO: pass properties?
	case *reflect.BoolValue:
		// Either "true", "false", 1 or 0.
		switch tok.value {
		case "true", "1":
			fv.Set(true)
			return nil
		case "false", "0":
			fv.Set(false)
			return nil
		}
	case *reflect.FloatValue:
		if f, err := strconv.AtofN(tok.value, fv.Type().Bits()); err == nil {
			fv.Set(f)
			return nil
		}
	case *reflect.IntValue:
		switch fv.Type().Bits() {
		case 32:
			if x, err := strconv.Atoi64(tok.value); err == nil && minInt32 <= x && x <= maxInt32 {
				fv.Set(x)
				return nil
			}
			if len(props.Enum) == 0 {
				break
			}
			m, ok := enumValueMaps[props.Enum]
			if !ok {
				break
			}
			x, ok := m[tok.value]
			if !ok {
				break
			}
			fv.Set(int64(x))
			return nil
		case 64:
			if x, err := strconv.Atoi64(tok.value); err == nil {
				fv.Set(x)
				return nil
			}
		}
	case *reflect.PtrValue:
		// A basic field (indirected through pointer), or a repeated message/group
		p.back()
		fv.PointTo(reflect.MakeZero(fv.Type().(*reflect.PtrType).Elem()))
		return p.readAny(fv.Elem(), props)
	case *reflect.StringValue:
		if tok.value[0] == '"' {
			fv.Set(tok.unquoted)
			return nil
		}
	case *reflect.StructValue:
		var terminator string
		switch tok.value {
		case "{":
			terminator = "}"
		case "<":
			terminator = ">"
		default:
			return p.error("expected '{' or '<', found %q", tok.value)
		}
		return p.readStruct(fv, terminator)
	case *reflect.UintValue:
		switch fv.Type().Bits() {
		case 32:
			if x, err := strconv.Atoui64(tok.value); err == nil && x <= maxUint32 {
				fv.Set(uint64(x))
				return nil
			}
		case 64:
			if x, err := strconv.Atoui64(tok.value); err == nil {
				fv.Set(x)
				return nil
			}
		}
	}
	return p.error("invalid %v: %v", v.Type(), tok.value)
}

var notPtrStruct os.Error = &ParseError{"destination is not a pointer to a struct", 0, 0}

// UnmarshalText reads a protobuffer in Text format.
func UnmarshalText(s string, pb interface{}) os.Error {
	pv, ok := reflect.NewValue(pb).(*reflect.PtrValue)
	if !ok {
		return notPtrStruct
	}
	sv, ok := pv.Elem().(*reflect.StructValue)
	if !ok {
		return notPtrStruct
	}
	if pe := newTextParser(s).readStruct(sv, ""); pe != nil {
		return pe
	}
	return nil
}
