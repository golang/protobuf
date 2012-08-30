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

	"code.google.com/p/goprotobuf/proto"

	pb "./testdata"
)

var cloneTestMessage = &pb.MyMessage{
	Count: proto.Int32(42),
	Name:  proto.String("Dave"),
	Pet:   []string{"bunny", "kitty", "horsey"},
	Inner: &pb.InnerMessage{
		Host:      proto.String("niles"),
		Port:      proto.Int32(9099),
		Connected: proto.Bool(true),
	},
	Others: []*pb.OtherMessage{
		&pb.OtherMessage{
			Value: []byte("some bytes"),
		},
	},
	RepBytes: [][]byte{[]byte("sham"), []byte("wow")},
}

func init() {
	ext := &pb.Ext{
		Data: proto.String("extension"),
	}
	if err := proto.SetExtension(cloneTestMessage, pb.E_Ext_More, ext); err != nil {
		panic("SetExtension: " + err.Error())
	}
}

func TestClone(t *testing.T) {
	m := proto.Clone(cloneTestMessage).(*pb.MyMessage)
	if !proto.Equal(m, cloneTestMessage) {
		t.Errorf("Clone(%v) = %v", cloneTestMessage, m)
	}

	// Verify it was a deep copy.
	*m.Inner.Port++
	if proto.Equal(m, cloneTestMessage) {
		t.Error("Mutating clone changed the original")
	}
}

func TestCloneNil(t *testing.T) {
	var m *pb.MyMessage
	if c := proto.Clone(m); !proto.Equal(m, c) {
		t.Errorf("Clone(%v) = %v", m, c)
	}
}
