// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"errors"
	"testing"
)

type testErrorList []error

func (e testErrorList) Error() string {
	return "testErrorList"
}

type invalidUTF8Error struct{}

func (e invalidUTF8Error) Error() string     { return "" }
func (e invalidUTF8Error) InvalidUTF8() bool { return true }

type requiredNotSetError struct{}

func (e requiredNotSetError) Error() string        { return "" }
func (e requiredNotSetError) RequiredNotSet() bool { return true }

func TestNonFatalErrors(t *testing.T) {
	tests := []struct {
		input error
		want  errorMask
	}{{
		input: errors.New("not one of them"),
	}, {
		input: testErrorList{},
	}, {
		input: testErrorList{
			invalidUTF8Error{},
		},
		want: errInvalidUTF8,
	}, {
		input: testErrorList{
			requiredNotSetError{},
		},
		want: errRequiredNotSet,
	}, {
		input: testErrorList{
			invalidUTF8Error{},
			requiredNotSetError{},
		},
		want: errInvalidUTF8 | errRequiredNotSet,
	}}

	for _, tc := range tests {
		tc := tc
		t.Run("", func(t *testing.T) {
			got := nonFatalErrors(tc.input)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
