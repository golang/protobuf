// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"bytes"
	"testing"

	protoV1a "github.com/golang/protobuf/internal/proto"
	"github.com/golang/protobuf/proto"
	pb "github.com/golang/protobuf/proto/proto3_proto"
	tpb "github.com/golang/protobuf/proto/test_proto"
)

func TestProto3ZeroValues(t *testing.T) {
	tests := []struct {
		desc string
		m    proto.Message
	}{
		{"zero message", &pb.Message{}},
		{"empty bytes field", &pb.Message{Data: []byte{}}},
	}
	for _, test := range tests {
		b, err := proto.Marshal(test.m)
		if err != nil {
			t.Errorf("%s: proto.Marshal: %v", test.desc, err)
			continue
		}
		if len(b) > 0 {
			t.Errorf("%s: Encoding is non-empty: %q", test.desc, b)
		}
	}
}

func TestRoundTripProto3(t *testing.T) {
	m := &pb.Message{
		Name:         "David",          // (2 | 1<<3): 0x0a 0x05 "David"
		Hilarity:     pb.Message_PUNS,  // (0 | 2<<3): 0x10 0x01
		HeightInCm:   178,              // (0 | 3<<3): 0x18 0xb2 0x01
		Data:         []byte("roboto"), // (2 | 4<<3): 0x20 0x06 "roboto"
		ResultCount:  47,               // (0 | 7<<3): 0x38 0x2f
		TrueScotsman: true,             // (0 | 8<<3): 0x40 0x01
		Score:        8.1,              // (5 | 9<<3): 0x4d <8.1>

		Key: []uint64{1, 0xdeadbeef},
		Nested: &pb.Nested{
			Bunny: "Monty",
		},
	}
	t.Logf(" m: %v", m)

	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	t.Logf(" b: %q", b)

	m2 := new(pb.Message)
	if err := proto.Unmarshal(b, m2); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}
	t.Logf("m2: %v", m2)

	if !proto.Equal(m, m2) {
		t.Errorf("proto.Equal returned false:\n m: %v\nm2: %v", m, m2)
	}
}

func TestGettersForBasicTypesExist(t *testing.T) {
	var m pb.Message
	if got := m.GetNested().GetBunny(); got != "" {
		t.Errorf("m.GetNested().GetBunny() = %q, want empty string", got)
	}
	if got := m.GetNested().GetCute(); got {
		t.Errorf("m.GetNested().GetCute() = %t, want false", got)
	}
}

func TestProto3SetDefaults(t *testing.T) {
	in := &pb.Message{
		Terrain: map[string]*pb.Nested{
			"meadow": new(pb.Nested),
		},
		Proto2Field: new(tpb.SubDefaults),
		Proto2Value: map[string]*tpb.SubDefaults{
			"badlands": new(tpb.SubDefaults),
		},
	}

	got := proto.Clone(in).(*pb.Message)
	proto.SetDefaults(got)

	// There are no defaults in proto3.  Everything should be the zero value, but
	// we need to remember to set defaults for nested proto2 messages.
	want := &pb.Message{
		Terrain: map[string]*pb.Nested{
			"meadow": new(pb.Nested),
		},
		Proto2Field: &tpb.SubDefaults{N: proto.Int64(7)},
		Proto2Value: map[string]*tpb.SubDefaults{
			"badlands": &tpb.SubDefaults{N: proto.Int64(7)},
		},
	}

	if !proto.Equal(got, want) {
		t.Errorf("with in = %v\nproto.SetDefaults(in) =>\ngot %v\nwant %v", in, got, want)
	}
}

func TestProto3SetDefaults2(t *testing.T) {
	in := &pb.Message{
		Terrain: map[string]*pb.Nested{
			"meadow": new(pb.Nested),
		},
		Proto2Field: new(tpb.SubDefaults),
		Proto2Value: map[string]*tpb.SubDefaults{
			"badlands": new(tpb.SubDefaults),
		},
	}

	got := proto.Clone(in).(*pb.Message)
	protoV1a.SetDefaults(got)

	// There are no defaults in proto3.  Everything should be the zero value, but
	// we need to remember to set defaults for nested proto2 messages.
	want := &pb.Message{
		Terrain: map[string]*pb.Nested{
			"meadow": new(pb.Nested),
		},
		Proto2Field: &tpb.SubDefaults{N: proto.Int64(7)},
		Proto2Value: map[string]*tpb.SubDefaults{
			"badlands": &tpb.SubDefaults{N: proto.Int64(7)},
		},
	}

	if !proto.Equal(got, want) {
		t.Errorf("with in = %v\nproto.SetDefaults(in) =>\ngot %v\nwant %v", in, got, want)
	}
}

func TestUnknownFieldPreservation(t *testing.T) {
	b1 := "\x0a\x05David"      // Known tag 1
	b2 := "\xc2\x0c\x06Google" // Unknown tag 200
	b := []byte(b1 + b2)

	m := new(pb.Message)
	if err := proto.Unmarshal(b, m); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}

	if !bytes.Equal(m.XXX_unrecognized, []byte(b2)) {
		t.Fatalf("mismatching unknown fields:\ngot  %q\nwant %q", m.XXX_unrecognized, b2)
	}
}
