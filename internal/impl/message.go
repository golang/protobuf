// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"strconv"
	"strings"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

type MessageInfo struct {
	// TODO: Split fields into dense and sparse maps similar to the current
	// table-driven implementation in v1?
	fields map[pref.FieldNumber]*fieldInfo
}

// generateFieldFuncs generates per-field functions for all common operations
// to be performed on each field. It takes in a reflect.Type representing the
// Go struct, and a protoreflect.MessageDescriptor to match with the fields
// in the struct.
//
// This code assumes that the struct is well-formed and panics if there are
// any discrepancies.
func (mi *MessageInfo) generateFieldFuncs(t reflect.Type, md pref.MessageDescriptor) {
	// Generate a mapping of field numbers and names to Go struct field or type.
	fields := map[pref.FieldNumber]reflect.StructField{}
	oneofs := map[pref.Name]reflect.StructField{}
	oneofFields := map[pref.FieldNumber]reflect.Type{}
	special := map[string]reflect.StructField{}
fieldLoop:
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
			if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
				n, _ := strconv.ParseUint(s, 10, 64)
				fields[pref.FieldNumber(n)] = f
				continue fieldLoop
			}
		}
		if s := f.Tag.Get("protobuf_oneof"); len(s) > 0 {
			oneofs[pref.Name(s)] = f
			continue fieldLoop
		}
		switch f.Name {
		case "XXX_weak", "XXX_unrecognized", "XXX_sizecache", "XXX_extensions", "XXX_InternalExtensions":
			special[f.Name] = f
			continue fieldLoop
		}
	}
	if fn, ok := t.MethodByName("XXX_OneofFuncs"); ok {
		vs := fn.Func.Call([]reflect.Value{reflect.New(fn.Type.In(0)).Elem()})[3]
	oneofLoop:
		for _, v := range vs.Interface().([]interface{}) {
			tf := reflect.TypeOf(v).Elem()
			f := tf.Field(0)
			for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
				if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
					n, _ := strconv.ParseUint(s, 10, 64)
					oneofFields[pref.FieldNumber(n)] = tf
					continue oneofLoop
				}
			}
		}
	}

	mi.fields = map[pref.FieldNumber]*fieldInfo{}
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		fs := fields[fd.Number()]
		var fi fieldInfo
		switch {
		case fd.IsWeak():
			fi = fieldInfoForWeak(fd, special["XXX_weak"])
		case fd.OneofType() != nil:
			fi = fieldInfoForOneof(fd, oneofs[fd.OneofType().Name()], oneofFields[fd.Number()])
		case fd.IsMap():
			fi = fieldInfoForMap(fd, fs)
		case fd.Cardinality() == pref.Repeated:
			fi = fieldInfoForVector(fd, fs)
		case fd.Kind() != pref.MessageKind && fd.Kind() != pref.GroupKind:
			fi = fieldInfoForScalar(fd, fs)
		default:
			fi = fieldInfoForMessage(fd, fs)
		}
		mi.fields[fd.Number()] = &fi
	}
}
