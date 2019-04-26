// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto

import (
	"bytes"
	"fmt"

	"google.golang.org/protobuf/internal/errors"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// IsInitialized returns an error if any required fields in m are not set.
func IsInitialized(m Message) error {
	if methods := protoMethods(m); methods != nil && methods.IsInitialized != nil {
		// TODO: Do we need a way to disable the fast path here?
		//
		// TODO: Should detailed information about missing
		// fields always be provided by the slow-but-informative
		// reflective implementation?
		return methods.IsInitialized(m)
	}
	return isInitialized(m.ProtoReflect(), nil)
}

// IsInitialized returns an error if any required fields in m are not set.
func isInitialized(m pref.Message, stack []interface{}) error {
	md := m.Descriptor()
	fds := md.Fields()
	for i, nums := 0, md.RequiredNumbers(); i < nums.Len(); i++ {
		fd := fds.ByNumber(nums.Get(i))
		if !m.Has(fd) {
			stack = append(stack, fd.Name())
			return newRequiredNotSetError(stack)
		}
	}
	var err error
	m.Range(func(fd pref.FieldDescriptor, v pref.Value) bool {
		// Recurse into fields containing message values.
		stack := append(stack, fd.Name())
		switch {
		case fd.IsList():
			if fd.Message() == nil {
				return true
			}
			for i, list := 0, v.List(); i < list.Len() && err == nil; i++ {
				stack := append(stack, "[", i, "].")
				err = isInitialized(list.Get(i).Message(), stack)
			}
		case fd.IsMap():
			if fd.MapValue().Message() == nil {
				return true
			}
			v.Map().Range(func(key pref.MapKey, v pref.Value) bool {
				stack := append(stack, "[", key, "].")
				err = isInitialized(v.Message(), stack)
				return err == nil
			})
		default:
			if fd.Message() == nil {
				return true
			}
			stack := append(stack, ".")
			err = isInitialized(v.Message(), stack)
		}
		return err == nil
	})
	return err
}

func newRequiredNotSetError(stack []interface{}) error {
	var buf bytes.Buffer
	for _, s := range stack {
		fmt.Fprint(&buf, s)
	}
	var nerr errors.NonFatal
	nerr.AppendRequiredNotSet(buf.String())
	return nerr.E
}
