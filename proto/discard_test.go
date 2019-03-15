// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"testing"

	"github.com/golang/protobuf/proto"

	proto3pb "github.com/golang/protobuf/proto/proto3_proto"
	pb "github.com/golang/protobuf/proto/test_proto"
)

const rawFields = "\x2d\xc3\xd2\xe1\xf0"

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
			Nested:           &proto3pb.Nested{Cute: true, XXX_unrecognized: []byte(rawFields)},
			XXX_unrecognized: []byte(rawFields),
		},
		want: &proto3pb.Message{
			Name:   "Aaron",
			Nested: &proto3pb.Nested{Cute: true},
		},
	}, {
		desc: "Slice",
		in: &proto3pb.Message{
			Name: "Aaron",
			Children: []*proto3pb.Message{
				{Name: "Sarah", XXX_unrecognized: []byte(rawFields)},
				{Name: "Abraham", XXX_unrecognized: []byte(rawFields)},
			},
			XXX_unrecognized: []byte(rawFields),
		},
		want: &proto3pb.Message{
			Name: "Aaron",
			Children: []*proto3pb.Message{
				{Name: "Sarah"},
				{Name: "Abraham"},
			},
		},
	}, {
		desc: "OneOf",
		in: &pb.Communique{
			Union: &pb.Communique_Msg{&pb.Strings{
				StringField:      proto.String("123"),
				XXX_unrecognized: []byte(rawFields),
			}},
			XXX_unrecognized: []byte(rawFields),
		},
		want: &pb.Communique{
			Union: &pb.Communique_Msg{&pb.Strings{StringField: proto.String("123")}},
		},
	}, {
		desc: "Map",
		in: &pb.MessageWithMap{MsgMapping: map[int64]*pb.FloatingPoint{
			0x4002: &pb.FloatingPoint{
				Exact:            proto.Bool(true),
				XXX_unrecognized: []byte(rawFields),
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
					XXX_unrecognized: []byte(rawFields),
				},
				XXX_unrecognized: []byte(rawFields),
			}
			proto.SetExtension(m, pb.E_Ext_More, &pb.Ext{
				Data:             proto.String("extension"),
				XXX_unrecognized: []byte(rawFields),
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
			t.Errorf("test %s, expected unknown fields to be discarded\ngot  %v\nwant %v", tt.desc, tt.in, tt.want)
		}
	}
}
