// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"reflect"
	"testing"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
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
		[]pref.FieldNumbers{(*numbers)(nil)},
		[]pref.FieldRanges{(*ranges)(nil)},
		[]pref.FieldDescriptors{(*fields)(nil), (*oneofFields)(nil)},
		[]pref.OneofDescriptors{(*oneofs)(nil)},
		[]pref.ExtensionDescriptors{(*extensions)(nil)},
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

// TestDescriptorAccessors tests that descriptorAccessors is up-to-date.
func TestDescriptorAccessors(t *testing.T) {
	ignore := map[string]bool{
		"DefaultEnumValue": true,
		"DescriptorByName": true,
		"ProtoType":        true,
	}
	rt := reflect.TypeOf((*pref.Descriptor)(nil)).Elem()
	for i := 0; i < rt.NumMethod(); i++ {
		ignore[rt.Method(i).Name] = true
	}

	for rt, m := range descriptorAccessors {
		got := map[string]bool{}
		for _, s := range m {
			got[s] = true
		}
		want := map[string]bool{}
		for i := 0; i < rt.NumMethod(); i++ {
			want[rt.Method(i).Name] = true
		}

		// Check if descriptorAccessors contains a non-existent accessor.
		// If this test fails, remove the accessor from descriptorAccessors.
		for s := range got {
			if !want[s] && !ignore[s] {
				t.Errorf("%v.%v does not exist", rt, s)
			}
		}

		// Check if there are new protoreflect interface methods that are not
		// handled by the formatter. If this fails, either add the method to
		// ignore or add them to descriptorAccessors.
		for s := range want {
			if !got[s] && !ignore[s] {
				t.Errorf("%v.%v is not called by formatter", rt, s)
			}
		}
	}
}
