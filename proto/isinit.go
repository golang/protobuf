// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/v2/internal/errors"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
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
	md := m.Type()
	known := m.KnownFields()
	fields := md.Fields()
	for i, nums := 0, md.RequiredNumbers(); i < nums.Len(); i++ {
		num := nums.Get(i)
		if !known.Has(num) {
			stack = append(stack, fields.ByNumber(num).Name())
			return newRequiredNotSetError(stack)
		}
	}
	var err error
	known.Range(func(num pref.FieldNumber, v pref.Value) bool {
		field := fields.ByNumber(num)
		if field == nil {
			field = known.ExtensionTypes().ByNumber(num)
		}
		if field == nil {
			panic(fmt.Errorf("no descriptor for field %d in %q", num, md.FullName()))
		}
		// Look for fields containing a message: Messages, groups, and maps
		// with a message or group value.
		ft := field.MessageType()
		if ft == nil {
			return true
		}
		if field.IsMap() {
			if ft.Fields().ByNumber(2).MessageType() == nil {
				return true
			}
		}
		// Recurse into the field
		stack := append(stack, field.Name())
		switch {
		case field.IsMap():
			v.Map().Range(func(key pref.MapKey, v pref.Value) bool {
				stack := append(stack, "[", key, "].")
				err = isInitialized(v.Message(), stack)
				return err == nil
			})
		case field.Cardinality() == pref.Repeated:
			for i, list := 0, v.List(); i < list.Len(); i++ {
				stack := append(stack, "[", i, "].")
				err = isInitialized(list.Get(i).Message(), stack)
				if err != nil {
					break
				}
			}
		default:
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
