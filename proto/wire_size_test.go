// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"log"
	"math"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	pb2 "github.com/golang/protobuf/internal/testprotos/proto2_proto"
	pb3 "github.com/golang/protobuf/internal/testprotos/proto3_proto"
)

var messageWithExtension1 = &pb2.MyMessage{Count: proto.Int32(7)}

// messageWithExtension2 is in equal_test.go.
var messageWithExtension3 = &pb2.MyMessage{Count: proto.Int32(8)}

func init() {
	if err := proto.SetExtension(messageWithExtension1, pb2.E_Ext_More, &pb2.Ext{Data: proto.String("Abbott")}); err != nil {
		log.Panicf("proto.SetExtension: %v", err)
	}
	if err := proto.SetExtension(messageWithExtension3, pb2.E_Ext_More, &pb2.Ext{Data: proto.String("Costello")}); err != nil {
		log.Panicf("proto.SetExtension: %v", err)
	}

	// Force messageWithExtension3 to have the extension encoded.
	proto.Marshal(messageWithExtension3)

}

// non-pointer custom message
type nonptrMessage struct{}

func (m nonptrMessage) ProtoMessage()  {}
func (m nonptrMessage) Reset()         {}
func (m nonptrMessage) String() string { return "" }

func (m nonptrMessage) Marshal() ([]byte, error) {
	return []byte{42}, nil
}

var SizeTests = []struct {
	desc string
	pb   proto.Message
}{
	{"empty", &pb2.OtherMessage{}},
	// Basic types.
	{"bool", &pb2.Defaults{F_Bool: proto.Bool(true)}},
	{"int32", &pb2.Defaults{F_Int32: proto.Int32(12)}},
	{"negative int32", &pb2.Defaults{F_Int32: proto.Int32(-1)}},
	{"small int64", &pb2.Defaults{F_Int64: proto.Int64(1)}},
	{"big int64", &pb2.Defaults{F_Int64: proto.Int64(1 << 20)}},
	{"negative int64", &pb2.Defaults{F_Int64: proto.Int64(-1)}},
	{"fixed32", &pb2.Defaults{F_Fixed32: proto.Uint32(71)}},
	{"fixed64", &pb2.Defaults{F_Fixed64: proto.Uint64(72)}},
	{"uint32", &pb2.Defaults{F_Uint32: proto.Uint32(123)}},
	{"uint64", &pb2.Defaults{F_Uint64: proto.Uint64(124)}},
	{"float", &pb2.Defaults{F_Float: proto.Float32(12.6)}},
	{"double", &pb2.Defaults{F_Double: proto.Float64(13.9)}},
	{"string", &pb2.Defaults{F_String: proto.String("niles")}},
	{"bytes", &pb2.Defaults{F_Bytes: []byte("wowsa")}},
	{"bytes, empty", &pb2.Defaults{F_Bytes: []byte{}}},
	{"sint32", &pb2.Defaults{F_Sint32: proto.Int32(65)}},
	{"sint64", &pb2.Defaults{F_Sint64: proto.Int64(67)}},
	{"enum", &pb2.Defaults{F_Enum: pb2.Defaults_BLUE.Enum()}},
	// Repeated.
	{"empty repeated bool", &pb2.MoreRepeated{Bools: []bool{}}},
	{"repeated bool", &pb2.MoreRepeated{Bools: []bool{false, true, true, false}}},
	{"packed repeated bool", &pb2.MoreRepeated{BoolsPacked: []bool{false, true, true, false, true, true, true}}},
	{"repeated int32", &pb2.MoreRepeated{Ints: []int32{1, 12203, 1729, -1}}},
	{"repeated int32 packed", &pb2.MoreRepeated{IntsPacked: []int32{1, 12203, 1729}}},
	{"repeated int64 packed", &pb2.MoreRepeated{Int64SPacked: []int64{
		// Need enough large numbers to verify that the header is counting the number of bytes
		// for the field, not the number of elements.
		1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62,
		1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62, 1 << 62,
	}}},
	{"repeated string", &pb2.MoreRepeated{Strings: []string{"r", "ken", "gri"}}},
	{"repeated fixed", &pb2.MoreRepeated{Fixeds: []uint32{1, 2, 3, 4}}},
	// Nested.
	{"nested", &pb2.OldMessage{Nested: &pb2.OldMessage_Nested{Name: proto.String("whatever")}}},
	{"group", &pb2.GroupOld{G: &pb2.GroupOld_G{X: proto.Int32(12345)}}},
	// Other things.
	{"unrecognized", &pb2.MoreRepeated{XXX_unrecognized: []byte{13<<3 | 0, 4}}},
	{"extension (unencoded)", messageWithExtension1},
	{"extension (encoded)", messageWithExtension3},
	// proto3 message
	{"proto3 empty", &pb3.Message{}},
	{"proto3 bool", &pb3.Message{TrueScotsman: true}},
	{"proto3 int64", &pb3.Message{ResultCount: 1}},
	{"proto3 uint32", &pb3.Message{HeightInCm: 123}},
	{"proto3 float", &pb3.Message{Score: 12.6}},
	{"proto3 string", &pb3.Message{Name: "Snezana"}},
	{"proto3 bytes", &pb3.Message{Data: []byte("wowsa")}},
	{"proto3 bytes, empty", &pb3.Message{Data: []byte{}}},
	{"proto3 enum", &pb3.Message{Hilarity: pb3.Message_PUNS}},
	{"proto3 map field with empty bytes", &pb3.MessageWithMap{ByteMapping: map[bool][]byte{false: []byte{}}}},

	{"map field", &pb2.MessageWithMap{NameMapping: map[int32]string{1: "Rob", 7: "Andrew"}}},
	{"map field with message", &pb2.MessageWithMap{MsgMapping: map[int64]*pb2.FloatingPoint{0x7001: &pb2.FloatingPoint{F: proto.Float64(2.0)}}}},
	{"map field with bytes", &pb2.MessageWithMap{ByteMapping: map[bool][]byte{true: []byte("this time for sure")}}},
	{"map field with empty bytes", &pb2.MessageWithMap{ByteMapping: map[bool][]byte{true: []byte{}}}},

	{"map field with big entry", &pb2.MessageWithMap{NameMapping: map[int32]string{8: strings.Repeat("x", 125)}}},
	{"map field with big key and val", &pb2.MessageWithMap{StrToStr: map[string]string{strings.Repeat("x", 70): strings.Repeat("y", 70)}}},
	{"map field with big numeric key", &pb2.MessageWithMap{NameMapping: map[int32]string{0xf00d: "om nom nom"}}},

	{"oneof not set", &pb2.Oneof{}},
	{"oneof bool", &pb2.Oneof{Union: &pb2.Oneof_F_Bool{true}}},
	{"oneof zero int32", &pb2.Oneof{Union: &pb2.Oneof_F_Int32{0}}},
	{"oneof big int32", &pb2.Oneof{Union: &pb2.Oneof_F_Int32{1 << 20}}},
	{"oneof int64", &pb2.Oneof{Union: &pb2.Oneof_F_Int64{42}}},
	{"oneof fixed32", &pb2.Oneof{Union: &pb2.Oneof_F_Fixed32{43}}},
	{"oneof fixed64", &pb2.Oneof{Union: &pb2.Oneof_F_Fixed64{44}}},
	{"oneof uint32", &pb2.Oneof{Union: &pb2.Oneof_F_Uint32{45}}},
	{"oneof uint64", &pb2.Oneof{Union: &pb2.Oneof_F_Uint64{46}}},
	{"oneof float", &pb2.Oneof{Union: &pb2.Oneof_F_Float{47.1}}},
	{"oneof double", &pb2.Oneof{Union: &pb2.Oneof_F_Double{48.9}}},
	{"oneof string", &pb2.Oneof{Union: &pb2.Oneof_F_String{"Rhythmic Fman"}}},
	{"oneof bytes", &pb2.Oneof{Union: &pb2.Oneof_F_Bytes{[]byte("let go")}}},
	{"oneof sint32", &pb2.Oneof{Union: &pb2.Oneof_F_Sint32{50}}},
	{"oneof sint64", &pb2.Oneof{Union: &pb2.Oneof_F_Sint64{51}}},
	{"oneof enum", &pb2.Oneof{Union: &pb2.Oneof_F_Enum{pb2.MyMessage_BLUE}}},
	{"message for oneof", &pb2.GoTestField{Label: proto.String("k"), Type: proto.String("v")}},
	{"oneof message", &pb2.Oneof{Union: &pb2.Oneof_F_Message{&pb2.GoTestField{Label: proto.String("k"), Type: proto.String("v")}}}},
	{"oneof group", &pb2.Oneof{Union: &pb2.Oneof_FGroup{&pb2.Oneof_F_Group{X: proto.Int32(52)}}}},
	{"oneof largest tag", &pb2.Oneof{Union: &pb2.Oneof_F_Largest_Tag{1}}},
	{"multiple oneofs", &pb2.Oneof{Union: &pb2.Oneof_F_Int32{1}, Tormato: &pb2.Oneof_Value{2}}},

	{"non-pointer message", nonptrMessage{}},
}

func TestSize(t *testing.T) {
	for _, tc := range SizeTests {
		t.Run(tc.desc, func(t *testing.T) {
			size := proto.Size(tc.pb)
			b, err := proto.Marshal(tc.pb)
			if err != nil {
				t.Errorf("%v: Marshal failed: %v", tc.desc, err)
				return
			}
			if size != len(b) {
				t.Errorf("%v: Size(%v) = %d, want %d", tc.desc, tc.pb, size, len(b))
				t.Logf("%v: bytes: %#v", tc.desc, b)
			}
		})
	}
}

func TestVarintSize(t *testing.T) {
	// Check the edge cases carefully.
	testCases := []struct {
		n    uint64
		size int
	}{
		{0, 1},
		{1, 1},
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
		{math.MaxInt64, 9},
		{math.MaxInt64 + 1, 10},
	}
	for _, tc := range testCases {
		size := proto.SizeVarint(tc.n)
		if size != tc.size {
			t.Errorf("sizeVarint(%d) = %d, want %d", tc.n, size, tc.size)
		}
	}
}
