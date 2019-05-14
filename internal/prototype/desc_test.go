// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"reflect"
	"testing"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// TestDescriptors tests that the implementations do not declare additional
// methods that do not exist on the interface types.
func TestDescriptors(t *testing.T) {
	tests := []interface{}{
		[]pref.FileDescriptor{placeholderFile{}, fileDesc{}},
		[]pref.MessageDescriptor{placeholderMessage{}, standaloneMessage{}, messageDesc{}},
		[]pref.FieldDescriptor{standaloneExtension{}, fieldDesc{}, extensionDesc{}},
		[]pref.OneofDescriptor{oneofDesc{}},
		[]pref.EnumDescriptor{placeholderEnum{}, standaloneEnum{}, enumDesc{}},
		[]pref.EnumValueDescriptor{enumValueDesc{}},
		[]pref.ServiceDescriptor{serviceDesc{}},
		[]pref.MethodDescriptor{methodDesc{}},

		[]pref.FileImports{(*fileImports)(nil)},
		[]pref.MessageDescriptors{(*messages)(nil)},
		[]pref.Names{(*names)(nil)},
		[]pref.FieldNumbers{(*numbers)(nil)},
		[]pref.FieldRanges{(*fieldRanges)(nil)},
		[]pref.FieldDescriptors{(*fields)(nil), (*oneofFields)(nil)},
		[]pref.OneofDescriptors{(*oneofs)(nil)},
		[]pref.ExtensionDescriptors{(*extensions)(nil)},
		[]pref.EnumRanges{(*enumRanges)(nil)},
		[]pref.EnumDescriptors{(*enums)(nil)},
		[]pref.EnumValueDescriptors{(*enumValues)(nil)},
		[]pref.ServiceDescriptors{(*services)(nil)},
		[]pref.MethodDescriptors{(*methods)(nil)},
	}

	for _, tt := range tests {
		v := reflect.ValueOf(tt) // []T where T is an interface
		ifaceType := v.Type().Elem()
		for i := 0; i < v.Len(); i++ {
			implType := v.Index(i).Elem().Type()

			var hasName bool
			for j := 0; j < implType.NumMethod(); j++ {
				if name := implType.Method(j).Name; name == "Format" {
					hasName = true
				} else if _, ok := ifaceType.MethodByName(name); !ok {
					t.Errorf("spurious method: %v.%v", implType, name)
				}
			}
			if !hasName {
				t.Errorf("missing method: %v.Format", implType)
			}
		}
	}
}
