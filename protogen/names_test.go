// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protogen

import "testing"

func TestCamelCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"one", "One"},
		{"one_two", "OneTwo"},
		{"_my_field_name_2", "XMyFieldName_2"},
		{"Something_Capped", "Something_Capped"},
		{"my_Name", "My_Name"},
		{"OneTwo", "OneTwo"},
		{"_", "X"},
		{"_a_", "XA_"},
		{"one.two", "OneTwo"},
		{"one.Two", "One_Two"},
		{"one_two.three_four", "OneTwoThreeFour"},
		{"one_two.Three_four", "OneTwo_ThreeFour"},
		{"_one._two", "XOne_XTwo"},
		{"SCREAMING_SNAKE_CASE", "SCREAMING_SNAKE_CASE"},
		{"double__underscore", "Double_Underscore"},
		{"camelCase", "CamelCase"},
		{"go2proto", "Go2Proto"},
		{"世界", "世界"},
		{"x世界", "X世界"},
		{"foo_bar世界", "FooBar世界"},
	}
	for _, tc := range tests {
		if got := camelCase(tc.in); got != tc.want {
			t.Errorf("CamelCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCleanGoName(t *testing.T) {
	tests := []struct {
		in, want, wantExported string
	}{
		{"", "", "X"},
		{"hello", "hello", "Hello"},
		{"hello-world!!", "hello_world__", "Hello_world__"},
		{"hello-\xde\xad\xbe\xef\x00", "hello_____", "Hello_____"},
		{"hello 世界", "hello_世界", "Hello_世界"},
		{"世界", "世界", "X世界"},
	}
	for _, tc := range tests {
		if got := cleanGoName(tc.in, false); got != tc.want {
			t.Errorf("cleanGoName(%q, false) = %q, want %q", tc.in, got, tc.want)
		}
		if got := cleanGoName(tc.in, true); got != tc.wantExported {
			t.Errorf("cleanGoName(%q, true) = %q, want %q", tc.in, got, tc.wantExported)
		}
	}
}
