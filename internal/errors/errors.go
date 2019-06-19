// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errors implements functions to manipulate errors.
package errors

import (
	"fmt"
)

// New formats a string according to the format specifier and arguments and
// returns an error that has a "proto" prefix.
func New(f string, x ...interface{}) error {
	for i := 0; i < len(x); i++ {
		if e, ok := x[i].(*prefixError); ok {
			x[i] = e.s // avoid "proto: " prefix when chaining
		}
	}
	return &prefixError{s: fmt.Sprintf(f, x...)}
}

type prefixError struct{ s string }

func (e *prefixError) Error() string { return "proto: " + e.s }

func InvalidUTF8(name string) error {
	return New("field %v contains invalid UTF-8", name)
}

func RequiredNotSet(name string) error {
	return New("required field %v not set", name)
}
