// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errors implements functions to manipulate errors.
package errors

import (
	"fmt"
	"sort"
	"strings"
)

// TODO: This package currently only provides functionality for constructing
// non-fatal errors. However, it does not currently provide functionality
// to test for a specific kind of non-fatal error, which is necessary
// for the end user.
//
// When that functionality is added, we need to think carefully about whether
// a user only cares that some kind of non-fatal error was present or whether
// all of the errors are of the same kind of non-fatal error.

// NonFatalErrors is a list of non-fatal errors where each error
// must either be a RequiredNotSet error or InvalidUTF8 error.
// The list must not be empty.
type NonFatalErrors []error

func (es NonFatalErrors) Error() string {
	ms := map[string]struct{}{}
	for _, e := range es {
		ms[e.Error()] = struct{}{}
	}
	var ss []string
	for s := range ms {
		ss = append(ss, s)
	}
	sort.Strings(ss)
	return "proto: " + strings.Join(ss, "; ")
}

// NonFatal contains non-fatal errors, which are errors that permit execution
// to continue, but should return with a non-nil error. As such, NonFatal is
// a data structure useful for swallowing non-fatal errors, but being able to
// reproduce them at the end of the function.
// An error is non-fatal if it is collection of non-fatal errors, or is
// an individual error where IsRequiredNotSet or IsInvalidUTF8 reports true.
//
// Typical usage pattern:
//	var nerr errors.NonFatal
//	...
//	if err := MyFunction(); !nerr.Merge(err) {
//		return nil, err // immediately return if err is fatal
//	}
//	...
//	return out, nerr.E
type NonFatal struct{ E error }

// Merge merges err into nf and reports whether it was successful.
// Otherwise it returns false for any fatal non-nil errors.
func (nf *NonFatal) Merge(err error) (ok bool) {
	if err == nil {
		return true // not an error
	}
	if es, ok := err.(NonFatalErrors); ok {
		nf.append(es...)
		return true // merged a list of non-fatal errors
	}
	if e, ok := err.(interface{ RequiredNotSet() bool }); ok && e.RequiredNotSet() {
		nf.append(err)
		return true // non-fatal RequiredNotSet error
	}
	if e, ok := err.(interface{ InvalidUTF8() bool }); ok && e.InvalidUTF8() {
		nf.append(err)
		return true // non-fatal InvalidUTF8 error
	}
	return false // fatal error
}

// AppendRequiredNotSet appends a RequiredNotSet error.
func (nf *NonFatal) AppendRequiredNotSet(field string) {
	nf.append(requiredNotSetError(field))
}

// AppendInvalidUTF8 appends an InvalidUTF8 error.
func (nf *NonFatal) AppendInvalidUTF8(field string) {
	nf.append(invalidUTF8Error(field))
}

func (nf *NonFatal) append(errs ...error) {
	es, _ := nf.E.(NonFatalErrors)
	es = append(es, errs...)
	nf.E = es
}

type requiredNotSetError string

func (e requiredNotSetError) Error() string {
	if e == "" {
		return "required field not set"
	}
	return string("required field " + e + " not set")
}
func (requiredNotSetError) RequiredNotSet() bool { return true }

type invalidUTF8Error string

func (e invalidUTF8Error) Error() string {
	if e == "" {
		return "invalid UTF-8 detected"
	}
	return string("field " + e + " contains invalid UTF-8")
}
func (invalidUTF8Error) InvalidUTF8() bool { return true }

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
