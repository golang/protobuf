// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

import (
	"errors"
	"reflect"
	"testing"
)

func TestNonFatal(t *testing.T) {
	type (
		method = interface{} // merge | appendRequiredNotSet | appendInvalidUTF8
		merge  struct {
			inErr  error
			wantOk bool
		}
		appendRequiredNotSet struct{ inField string }
		appendInvalidUTF8    struct{ inField string }
	)

	tests := []struct {
		label   string
		methods []method
		wantErr error
	}{{
		label: "IgnoreNil",
		methods: []method{
			merge{inErr: nil, wantOk: true},
		},
	}, {
		label: "IgnoreFatal",
		methods: []method{
			merge{inErr: errors.New("fatal error")},
		},
	}, {
		label: "MergeNonFatal",
		methods: []method{
			appendRequiredNotSet{"foo"},
			merge{inErr: customRequiredNotSetError{}, wantOk: true},
			appendInvalidUTF8{"bar"},
			merge{inErr: customInvalidUTF8Error{}, wantOk: true},
			merge{inErr: NonFatalErrors{
				requiredNotSetError("fizz"),
				invalidUTF8Error("buzz"),
			}, wantOk: true},
			merge{inErr: errors.New("fatal error")}, // not stored
		},
		wantErr: NonFatalErrors{
			requiredNotSetError("foo"),
			customRequiredNotSetError{},
			invalidUTF8Error("bar"),
			customInvalidUTF8Error{},
			requiredNotSetError("fizz"),
			invalidUTF8Error("buzz"),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			var nerr NonFatal
			for _, m := range tt.methods {
				switch m := m.(type) {
				case merge:
					if gotOk := nerr.Merge(m.inErr); gotOk != m.wantOk {
						t.Errorf("Merge() = %v, want %v", gotOk, m.wantOk)
					}
				case appendRequiredNotSet:
					nerr.AppendRequiredNotSet(m.inField)
				case appendInvalidUTF8:
					nerr.AppendInvalidUTF8(m.inField)
				default:
					t.Fatalf("invalid method: %T", m)
				}
			}
			if !reflect.DeepEqual(nerr.E, tt.wantErr) {
				t.Errorf("NonFatal.E = %v, want %v", nerr.E, tt.wantErr)
			}
		})
	}
}

type customInvalidUTF8Error struct{}

func (customInvalidUTF8Error) Error() string     { return "invalid UTF-8 detected" }
func (customInvalidUTF8Error) InvalidUTF8() bool { return true }

type customRequiredNotSetError struct{}

func (customRequiredNotSetError) Error() string        { return "required field not set" }
func (customRequiredNotSetError) RequiredNotSet() bool { return true }
