// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2011 Google Inc.  All rights reserved.
// http://code.google.com/p/goprotobuf/
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

package proto_test

import (
	"testing"

	. "goprotobuf.googlecode.com/hg/proto"
	pb "./testdata/_obj/test_proto"
)

var EqualTests = []struct {
	desc string
	a, b interface{}
	exp  bool
}{
	{"different types", &pb.GoEnum{}, &pb.GoTestField{}, false},
	{"one pointer, one value", &pb.GoEnum{}, pb.GoEnum{}, false},
	{"non-protocol buffers", 7, 7, false},
	{"equal empty", &pb.GoEnum{}, &pb.GoEnum{}, true},

	{"one set field, one unset field", &pb.GoTestField{Label: String("foo")}, &pb.GoTestField{}, false},
	{"one set field zero, one unset field", &pb.GoTest{Param: Int32(0)}, &pb.GoTest{}, false},
	{"different set fields", &pb.GoTestField{Label: String("foo")}, &pb.GoTestField{Label: String("bar")}, false},
	{"equal set", &pb.GoTestField{Label: String("foo")}, &pb.GoTestField{Label: String("foo")}, true},

	{"repeated, one set", &pb.GoTest{F_Int32Repeated: []int32{2, 3}}, &pb.GoTest{}, false},
	{"repeated, different length", &pb.GoTest{F_Int32Repeated: []int32{2, 3}}, &pb.GoTest{F_Int32Repeated: []int32{2}}, false},
	{"repeated, different value", &pb.GoTest{F_Int32Repeated: []int32{2}}, &pb.GoTest{F_Int32Repeated: []int32{3}}, false},
	{"repeated, equal", &pb.GoTest{F_Int32Repeated: []int32{2, 4}}, &pb.GoTest{F_Int32Repeated: []int32{2, 4}}, true},

	{
		"nested, different",
		&pb.GoTest{RequiredField: &pb.GoTestField{Label: String("foo")}},
		&pb.GoTest{RequiredField: &pb.GoTestField{Label: String("bar")}},
		false,
	},
	{
		"nested, equal",
		&pb.GoTest{RequiredField: &pb.GoTestField{Label: String("wow")}},
		&pb.GoTest{RequiredField: &pb.GoTestField{Label: String("wow")}},
		true,
	},
}

func TestEqual(t *testing.T) {
	for _, tc := range EqualTests {
		if res := Equal(tc.a, tc.b); res != tc.exp {
			t.Errorf("%v: Equal(%v, %v) = %v, want %v", tc.desc, tc.a, tc.b, res, tc.exp)
		}
	}
}
