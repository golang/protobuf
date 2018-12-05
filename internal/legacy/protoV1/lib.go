// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoV1

import (
	"fmt"

	"github.com/golang/protobuf/protoapi"
)

// RequiredNotSetError is an error type returned by either Marshal or Unmarshal.
// Marshal reports this when a required field is not initialized.
// Unmarshal reports this when a required field is missing from the wire data.
type RequiredNotSetError struct{ field string }

func (e *RequiredNotSetError) Error() string {
	if e.field == "" {
		return fmt.Sprintf("proto: required field not set")
	}
	return fmt.Sprintf("proto: required field %q not set", e.field)
}
func (e *RequiredNotSetError) RequiredNotSet() bool {
	return true
}

type invalidUTF8Error struct{ field string }

func (e *invalidUTF8Error) Error() string {
	if e.field == "" {
		return "proto: invalid UTF-8 detected"
	}
	return fmt.Sprintf("proto: field %q contains invalid UTF-8", e.field)
}
func (e *invalidUTF8Error) InvalidUTF8() bool {
	return true
}

// errInvalidUTF8 is a sentinel error to identify fields with invalid UTF-8.
// This error should not be exposed to the external API as such errors should
// be recreated with the field information.
var errInvalidUTF8 = &invalidUTF8Error{}

// isNonFatal reports whether the error is either a RequiredNotSet error
// or a InvalidUTF8 error.
func isNonFatal(err error) bool {
	if re, ok := err.(interface{ RequiredNotSet() bool }); ok && re.RequiredNotSet() {
		return true
	}
	if re, ok := err.(interface{ InvalidUTF8() bool }); ok && re.InvalidUTF8() {
		return true
	}
	return false
}

type nonFatal struct{ E error }

// Merge merges err into nf and reports whether it was successful.
// Otherwise it returns false for any fatal non-nil errors.
func (nf *nonFatal) Merge(err error) (ok bool) {
	if err == nil {
		return true // not an error
	}
	if !isNonFatal(err) {
		return false // fatal error
	}
	if nf.E == nil {
		nf.E = err // store first instance of non-fatal error
	}
	return true
}

type (
	Message                = protoapi.Message
	Extension              = protoapi.ExtensionField
	ExtensionRange         = protoapi.ExtensionRange
	XXX_InternalExtensions = protoapi.XXX_InternalExtensions
)

// A Buffer is a buffer manager for marshaling and unmarshaling
// protocol buffers.  It may be reused between invocations to
// reduce memory usage.  It is not necessary to use a Buffer;
// the global functions Marshal and Unmarshal create a
// temporary Buffer and are fine for most applications.
type Buffer struct {
	buf   []byte // encode/decode byte stream
	index int    // read point
}

// NewBuffer allocates a new Buffer and initializes its internal data to
// the contents of the argument slice.
func NewBuffer(e []byte) *Buffer {
	return &Buffer{buf: e}
}

// InternalMessageInfo is a type used internally by generated .pb.go files.
// This type is not intended to be used by non-generated code.
// This type is not subject to any compatibility guarantee.
type InternalMessageInfo struct {
	unmarshal *unmarshalInfo
}
