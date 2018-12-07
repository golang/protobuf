// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package defval

import (
	"math"
	"reflect"
	"testing"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

func Test(t *testing.T) {
	V := pref.ValueOf
	tests := []struct {
		val   pref.Value
		kind  pref.Kind
		strPB string
		strGo string
	}{
		{V(bool(true)), pref.BoolKind, "true", "1"},
		{V(int32(-0x1234)), pref.Int32Kind, "-4660", "-4660"},
		{V(float32(math.Pi)), pref.FloatKind, "3.1415927", "3.1415927"},
		{V(float64(math.Pi)), pref.DoubleKind, "3.141592653589793", "3.141592653589793"},
		{V(string("hello, \xde\xad\xbe\xef\n")), pref.StringKind, "hello, \xde\xad\xbe\xef\n", "hello, \xde\xad\xbe\xef\n"},
		{V([]byte("hello, \xde\xad\xbe\xef\n")), pref.BytesKind, "hello, \\336\\255\\276\\357\\n", "hello, \\336\\255\\276\\357\\n"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			gotStrPB, _ := Marshal(tt.val, tt.kind, Descriptor)
			if gotStrPB != tt.strPB {
				t.Errorf("Marshal(%v, %v, Descriptor) = %q, want %q", tt.val, tt.kind, gotStrPB, tt.strPB)
			}

			gotStrGo, _ := Marshal(tt.val, tt.kind, GoTag)
			if gotStrGo != tt.strGo {
				t.Errorf("Marshal(%v, %v, GoTag) = %q, want %q", tt.val, tt.kind, gotStrGo, tt.strGo)
			}

			gotValPB, _ := Unmarshal(tt.strPB, tt.kind, Descriptor)
			if !reflect.DeepEqual(gotValPB.Interface(), tt.val.Interface()) {
				t.Errorf("Unmarshal(%v, %v, Descriptor) = %q, want %q", tt.strPB, tt.kind, gotValPB, tt.val)
			}

			gotValGo, _ := Unmarshal(tt.strGo, tt.kind, GoTag)
			if !reflect.DeepEqual(gotValGo.Interface(), tt.val.Interface()) {
				t.Errorf("Unmarshal(%v, %v, GoTag) = %q, want %q", tt.strGo, tt.kind, gotValGo, tt.val)
			}
		})
	}
}
