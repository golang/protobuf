// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

// TODO: This file exists to provide the illusion to other source files that
// they live within the real proto package by providing functions and types
// that they would otherwise be able to call directly.

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/runtime/protoiface"
	_ "google.golang.org/protobuf/runtime/protolegacy"
)

type (
	Message       = protoiface.MessageV1
	ExtensionDesc = protoiface.ExtensionDescV1
)

// RequiredNotSetError is an error type returned by either Marshal or Unmarshal.
// Marshal reports this when a required field is not initialized.
// Unmarshal reports this when a required field is missing from the input data.
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

type errorMask uint8

const (
	_ errorMask = (1 << iota) / 2
	errInvalidUTF8
	errRequiredNotSet
)

type errorsList = []error

var errorsListType = reflect.TypeOf(errorsList{})

// nonFatalErrors returns an errorMask identifying V2 non-fatal errors.
func nonFatalErrors(err error) errorMask {
	verr := reflect.ValueOf(err)
	if !verr.IsValid() {
		return 0
	}

	if !verr.Type().AssignableTo(errorsListType) {
		return 0
	}

	errs := verr.Convert(errorsListType).Interface().(errorsList)
	var ret errorMask
	for _, e := range errs {
		switch e.(type) {
		case interface{ RequiredNotSet() bool }:
			ret |= errRequiredNotSet
		case interface{ InvalidUTF8() bool }:
			ret |= errInvalidUTF8
		}
	}
	return ret
}
