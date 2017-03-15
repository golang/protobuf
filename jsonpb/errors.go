// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2015 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
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

package jsonpb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
)

type fieldError struct {
	fieldStack []string
	nestedErr  error
}

func (f *fieldError) Error() string {
	return "unparsable field " + strings.Join(f.fieldStack, ".") + ": " + f.nestedErr.Error()
}

func (f *fieldError) FieldStack() []string {
	return f.fieldStack
}

func (f *fieldError) UnderlyingError() error {
	return f.nestedErr
}

// newFieldError wraps a given error providing a message call stack.
func newFieldError(fieldName string, err error) error {
	if fErr, ok := err.(*fieldError); ok {
		fErr.fieldStack = append([]string{fieldName}, fErr.fieldStack...)
		return err
	}
	return &fieldError{
		fieldStack: []string{fieldName},
		nestedErr:  err,
	}
}

// correctJsonType gets rid of the dredded json.RawMessage errors and casts them to the right type.
func correctJsonType(err error, realType reflect.Type) error {
	if uErr, ok := err.(*json.UnmarshalTypeError); ok {
		uErr.Type = realType
		return uErr
	}
	return err
}

func getFieldMismatchError(remainingFields map[string]json.RawMessage, structProps *proto.StructProperties) error {
	remaining := []string{}
	for k, _ := range remainingFields {
		remaining = append(remaining, k)
	}
	known := []string{}
	for _, prop := range structProps.Prop {
		jsonNames := acceptedJSONFieldNames(prop)
		if strings.HasPrefix(jsonNames.camel, "XXX_") {
			continue
		}
		known = append(known, jsonNames.camel)
	}
	return fmt.Errorf("fields %v do not exist in set of known fields %v", remaining, known)
}
