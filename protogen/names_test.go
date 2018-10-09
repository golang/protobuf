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
	}
	for _, tc := range tests {
		if got := camelCase(tc.in); got != tc.want {
			t.Errorf("CamelCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
