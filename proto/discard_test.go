// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2017 The Go Authors.  All rights reserved.
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

package proto_test

import (
	"testing"

	"github.com/golang/protobuf/proto"

	proto3pb "github.com/golang/protobuf/proto/proto3_proto"
	pb "github.com/golang/protobuf/proto/test_proto"
)

func TestDiscardUnknown(t *testing.T) {
	tests := []struct {
		desc     string
		in, want proto.Message
	}{{
		desc: "Nil",
		in:   nil, want: nil, // Should not panic
	}, {
		desc: "NilPtr",
		in:   (*proto3pb.Message)(nil), want: (*proto3pb.Message)(nil), // Should not panic
	}, {
		desc: "Nested",
		in: &proto3pb.Message{
			Name:             "Aaron",
			Nested:           &proto3pb.Nested{Cute: true, XXX_unrecognized: []byte("blah")},
			XXX_unrecognized: []byte("blah"),
		},
		want: &proto3pb.Message{
			Name:   "Aaron",
			Nested: &proto3pb.Nested{Cute: true},
		},
	}, {
		desc: "OneOf",
		in: &pb.Communique{
			Union: &pb.Communique_Msg{&pb.Strings{
				StringField:      proto.String("123"),
				XXX_unrecognized: []byte("blah"),
			}},
			XXX_unrecognized: []byte("blah"),
		},
		want: &pb.Communique{
			Union: &pb.Communique_Msg{&pb.Strings{StringField: proto.String("123")}},
		},
	}, {
		desc: "Map",
		in: &pb.MessageWithMap{MsgMapping: map[int64]*pb.FloatingPoint{
			0x4002: &pb.FloatingPoint{
				Exact:            proto.Bool(true),
				XXX_unrecognized: []byte("blah"),
			},
		}},
		want: &pb.MessageWithMap{MsgMapping: map[int64]*pb.FloatingPoint{
			0x4002: &pb.FloatingPoint{Exact: proto.Bool(true)},
		}},
	}, {
		desc: "Extension",
		in: func() proto.Message {
			m := &pb.MyMessage{
				Count: proto.Int32(42),
				Somegroup: &pb.MyMessage_SomeGroup{
					GroupField:       proto.Int32(6),
					XXX_unrecognized: []byte("blah"),
				},
				XXX_unrecognized: []byte("blah"),
			}
			proto.SetExtension(m, pb.E_Ext_More, &pb.Ext{
				Data:             proto.String("extension"),
				XXX_unrecognized: []byte("blah"),
			})
			return m
		}(),
		want: func() proto.Message {
			m := &pb.MyMessage{
				Count:     proto.Int32(42),
				Somegroup: &pb.MyMessage_SomeGroup{GroupField: proto.Int32(6)},
			}
			proto.SetExtension(m, pb.E_Ext_More, &pb.Ext{Data: proto.String("extension")})
			return m
		}(),
	}}

	for _, tt := range tests {
		proto.DiscardUnknown(tt.in)
		if !proto.Equal(tt.in, tt.want) {
			t.Errorf("test %s, expected unknown fields to be discarde\ngot  %v\nwant %v", tt.desc, tt.in, tt.want)
		}
	}
}
