// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generator

import (
	"testing"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

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
	}
	for _, tc := range tests {
		if got := CamelCase(tc.in); got != tc.want {
			t.Errorf("CamelCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGoPackageOption(t *testing.T) {
	tests := []struct {
		in      string
		impPath GoImportPath
		pkg     GoPackageName
		ok      bool
	}{
		{"", "", "", false},
		{"foo", "", "foo", true},
		{"github.com/golang/bar", "github.com/golang/bar", "bar", true},
		{"github.com/golang/bar;baz", "github.com/golang/bar", "baz", true},
		{"github.com/golang/string", "github.com/golang/string", "string", true},
	}
	for _, tc := range tests {
		d := &FileDescriptor{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Options: &descriptor.FileOptions{
					GoPackage: &tc.in,
				},
			},
		}
		impPath, pkg, ok := d.goPackageOption()
		if impPath != tc.impPath || pkg != tc.pkg || ok != tc.ok {
			t.Errorf("go_package = %q => (%q, %q, %t), want (%q, %q, %t)", tc.in,
				impPath, pkg, ok, tc.impPath, tc.pkg, tc.ok)
		}
	}
}

func TestPackageNames(t *testing.T) {
	g := New()
	g.packageNames = make(map[GoImportPath]GoPackageName)
	g.usedPackageNames = make(map[GoPackageName]bool)
	for _, test := range []struct {
		importPath GoImportPath
		want       GoPackageName
	}{
		{"github.com/golang/foo", "foo"},
		{"github.com/golang/second/package/named/foo", "foo1"},
		{"github.com/golang/third/package/named/foo", "foo2"},
		{"github.com/golang/conflicts/with/predeclared/ident/string", "string1"},
	} {
		if got := g.GoPackageName(test.importPath); got != test.want {
			t.Errorf("GoPackageName(%v) = %v, want %v", test.importPath, got, test.want)
		}
	}
}

func TestUnescape(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		// successful cases, including all kinds of escapes
		{"", ""},
		{"foo bar baz frob nitz", "foo bar baz frob nitz"},
		{`\000\001\002\003\004\005\006\007`, string([]byte{0, 1, 2, 3, 4, 5, 6, 7})},
		{`\a\b\f\n\r\t\v\\\?\'\"`, string([]byte{'\a', '\b', '\f', '\n', '\r', '\t', '\v', '\\', '?', '\'', '"'})},
		{`\x10\x20\x30\x40\x50\x60\x70\x80`, string([]byte{16, 32, 48, 64, 80, 96, 112, 128})},
		// variable length octal escapes
		{`\0\018\222\377\3\04\005\6\07`, string([]byte{0, 1, '8', 0222, 255, 3, 4, 5, 6, 7})},
		// malformed escape sequences left as is
		{"foo \\g bar", "foo \\g bar"},
		{"foo \\xg0 bar", "foo \\xg0 bar"},
		{"\\", "\\"},
		{"\\x", "\\x"},
		{"\\xf", "\\xf"},
		{"\\777", "\\777"}, // overflows byte
	}
	for _, tc := range tests {
		s := unescape(tc.in)
		if s != tc.out {
			t.Errorf("doUnescape(%q) = %q; should have been %q", tc.in, s, tc.out)
		}
	}
}
