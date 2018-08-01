// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2013 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package generator

import (
	"testing"
)

func TestGetStructTag(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		// skip json/protobuf
		{"`json:\"mid,omitempty\"`", ""},
		{"`protobuf:\"varint,1,opt,name=mid,proto3\"`", ""},
		{"`json:\"mid,omitempty\",foo:\"foo\"`", "foo:\"foo\""},

		// common tags
		{"`validator:\"oneof=red green\"`", "validator:\"oneof=red green\""},
		{"` \t\nfoo:\"foo,a=1\" \t\t bar:\"bar,b=2\" \t\n`", "foo:\"foo,a=1\" bar:\"bar,b=2\""},
		{"`foo_foo:\"foo,a=1\"bar-bar:\"bar,b=2\"`", "foo_foo:\"foo,a=1\" bar-bar:\"bar,b=2\""},

		// only process first
		{"`foo:\"foo\" bar:\"bar\"` xxx `baz:\"baz\"`", "foo:\"foo\" bar:\"bar\""},
	}
	for _, tc := range tests {
		s := getStructTag(tc.in)
		if s != tc.out {
			t.Errorf("tag comment (%q) = %q; should have been %q", tc.in, s, tc.out)
		}
	}
}
