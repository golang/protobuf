// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package typefmt

import (
	"reflect"
	"testing"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

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
