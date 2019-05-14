// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

// Functions for writing the text protocol buffer format.

import (
	"bytes"
	"encoding"
	"io"
	"reflect"

	"google.golang.org/protobuf/encoding/textpb"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// TextMarshaler is a configurable text format marshaler.
type TextMarshaler struct {
	Compact   bool // use compact text format in one line without the trailing newline character
	ExpandAny bool // expand google.protobuf.Any messages of known types
}

// Marshal writes a given protocol buffer in text format.
func (tm *TextMarshaler) Marshal(w io.Writer, pb Message) error {
	val := reflect.ValueOf(pb)
	// V1 supports passing in nil interface or pointer and outputs <nil>, while
	// V2 will panic on nil interface and outputs nothing for nil pointer.
	if pb == nil || val.IsNil() {
		w.Write([]byte("<nil>"))
		return nil
	}

	// V1-specific override in marshaling.
	if etm, ok := pb.(encoding.TextMarshaler); ok {
		text, err := etm.MarshalText()
		if err != nil {
			return err
		}
		if _, err = w.Write(text); err != nil {
			return err
		}
		return nil
	}

	var ind string
	if !tm.Compact {
		ind = "  "
	}
	mo := textpb.MarshalOptions{
		AllowPartial: true,
		Indent:       ind,
	}
	if !tm.ExpandAny {
		mo.Resolver = emptyResolver
	}
	b, err := mo.Marshal(protoimpl.X.MessageOf(pb).Interface())
	mask := nonFatalErrors(err)
	// V1 does not return invalid UTF-8 error.
	if err != nil && mask&errInvalidUTF8 == 0 {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

// Text is the same as Marshal, but returns the string directly.
func (tm *TextMarshaler) Text(pb Message) string {
	var buf bytes.Buffer
	tm.Marshal(&buf, pb)
	return buf.String()
}

var (
	emptyResolver        = preg.NewTypes()
	defaultTextMarshaler = TextMarshaler{}
	compactTextMarshaler = TextMarshaler{Compact: true}
)

// MarshalText writes a given protocol buffer in text format.
func MarshalText(w io.Writer, pb Message) error { return defaultTextMarshaler.Marshal(w, pb) }

// MarshalTextString is the same as MarshalText, but returns the string directly.
func MarshalTextString(pb Message) string { return defaultTextMarshaler.Text(pb) }

// CompactText writes a given protocol buffer in compact text format (one line).
func CompactText(w io.Writer, pb Message) error { return compactTextMarshaler.Marshal(w, pb) }

// CompactTextString is the same as CompactText, but returns the string directly.
func CompactTextString(pb Message) string { return compactTextMarshaler.Text(pb) }

// UnmarshalText reads a protocol buffer in text format. UnmarshalText resets pb
// before starting to unmarshal, so any existing data in pb is always removed.
// If a required field is not set and no other error occurs, UnmarshalText
// returns *RequiredNotSetError.
func UnmarshalText(s string, m Message) error {
	if um, ok := m.(encoding.TextUnmarshaler); ok {
		return um.UnmarshalText([]byte(s))
	}
	err := textpb.Unmarshal(protoimpl.X.MessageOf(m).Interface(), []byte(s))
	// Return RequiredNotSetError for required not set errors and ignore invalid
	// UTF-8 errors.
	mask := nonFatalErrors(err)
	if mask&errRequiredNotSet > 0 {
		return &RequiredNotSetError{}
	}
	if mask&errInvalidUTF8 > 0 {
		return nil
	}
	// Otherwise return error which can either be nil or fatal.
	return err
}
