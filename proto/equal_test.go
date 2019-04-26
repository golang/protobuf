// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"testing"

	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/internal/scalar"
	testpb "google.golang.org/protobuf/internal/testprotos/test"
	test3pb "google.golang.org/protobuf/internal/testprotos/test3"
	"google.golang.org/protobuf/proto"
)

func TestEqual(t *testing.T) {
	for _, test := range inequalities {
		if !proto.Equal(test.a, test.a) {
			t.Errorf("Equal(a, a) = false, want true\na = %T %v", test.a, marshalText(test.a))
		}
		if proto.Equal(test.a, test.b) {
			t.Errorf("Equal(a, b) = true, want false\na = %T %v\nb = %T %v", test.a, marshalText(test.a), test.b, marshalText(test.b))
		}
	}
}

var inequalities = []struct{ a, b proto.Message }{
	// Scalar values.
	{
		&testpb.TestAllTypes{OptionalInt32: scalar.Int32(1)},
		&testpb.TestAllTypes{OptionalInt32: scalar.Int32(2)},
	},
	{
		&testpb.TestAllTypes{OptionalInt64: scalar.Int64(1)},
		&testpb.TestAllTypes{OptionalInt64: scalar.Int64(2)},
	},
	{
		&testpb.TestAllTypes{OptionalUint32: scalar.Uint32(1)},
		&testpb.TestAllTypes{OptionalUint32: scalar.Uint32(2)},
	},
	{
		&testpb.TestAllTypes{OptionalUint64: scalar.Uint64(1)},
		&testpb.TestAllTypes{OptionalUint64: scalar.Uint64(2)},
	},
	{
		&testpb.TestAllTypes{OptionalSint32: scalar.Int32(1)},
		&testpb.TestAllTypes{OptionalSint32: scalar.Int32(2)},
	},
	{
		&testpb.TestAllTypes{OptionalSint64: scalar.Int64(1)},
		&testpb.TestAllTypes{OptionalSint64: scalar.Int64(2)},
	},
	{
		&testpb.TestAllTypes{OptionalFixed32: scalar.Uint32(1)},
		&testpb.TestAllTypes{OptionalFixed32: scalar.Uint32(2)},
	},
	{
		&testpb.TestAllTypes{OptionalFixed64: scalar.Uint64(1)},
		&testpb.TestAllTypes{OptionalFixed64: scalar.Uint64(2)},
	},
	{
		&testpb.TestAllTypes{OptionalSfixed32: scalar.Int32(1)},
		&testpb.TestAllTypes{OptionalSfixed32: scalar.Int32(2)},
	},
	{
		&testpb.TestAllTypes{OptionalSfixed64: scalar.Int64(1)},
		&testpb.TestAllTypes{OptionalSfixed64: scalar.Int64(2)},
	},
	{
		&testpb.TestAllTypes{OptionalFloat: scalar.Float32(1)},
		&testpb.TestAllTypes{OptionalFloat: scalar.Float32(2)},
	},
	{
		&testpb.TestAllTypes{OptionalDouble: scalar.Float64(1)},
		&testpb.TestAllTypes{OptionalDouble: scalar.Float64(2)},
	},
	{
		&testpb.TestAllTypes{OptionalBool: scalar.Bool(true)},
		&testpb.TestAllTypes{OptionalBool: scalar.Bool(false)},
	},
	{
		&testpb.TestAllTypes{OptionalString: scalar.String("a")},
		&testpb.TestAllTypes{OptionalString: scalar.String("b")},
	},
	{
		&testpb.TestAllTypes{OptionalBytes: []byte("a")},
		&testpb.TestAllTypes{OptionalBytes: []byte("b")},
	},
	{
		&testpb.TestAllTypes{OptionalNestedEnum: testpb.TestAllTypes_FOO.Enum()},
		&testpb.TestAllTypes{OptionalNestedEnum: testpb.TestAllTypes_BAR.Enum()},
	},
	// Proto2 presence.
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalInt32: scalar.Int32(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalInt64: scalar.Int64(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalUint32: scalar.Uint32(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalUint64: scalar.Uint64(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalSint32: scalar.Int32(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalSint64: scalar.Int64(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalFixed32: scalar.Uint32(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalFixed64: scalar.Uint64(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalSfixed32: scalar.Int32(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalSfixed64: scalar.Int64(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalFloat: scalar.Float32(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalDouble: scalar.Float64(0)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalBool: scalar.Bool(false)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalString: scalar.String("")},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalBytes: []byte{}},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalNestedEnum: testpb.TestAllTypes_FOO.Enum()},
	},
	// Groups.
	{
		&testpb.TestAllTypes{Optionalgroup: &testpb.TestAllTypes_OptionalGroup{
			A: scalar.Int32(1),
		}},
		&testpb.TestAllTypes{Optionalgroup: &testpb.TestAllTypes_OptionalGroup{
			A: scalar.Int32(2),
		}},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{Optionalgroup: &testpb.TestAllTypes_OptionalGroup{}},
	},
	// Messages.
	{
		&testpb.TestAllTypes{OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
			A: scalar.Int32(1),
		}},
		&testpb.TestAllTypes{OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
			A: scalar.Int32(2),
		}},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{}},
	},
	{
		&test3pb.TestAllTypes{},
		&test3pb.TestAllTypes{OptionalNestedMessage: &test3pb.TestAllTypes_NestedMessage{}},
	},
	// Lists.
	{
		&testpb.TestAllTypes{RepeatedInt32: []int32{1}},
		&testpb.TestAllTypes{RepeatedInt32: []int32{1, 2}},
	},
	{
		&testpb.TestAllTypes{RepeatedInt32: []int32{1, 2}},
		&testpb.TestAllTypes{RepeatedInt32: []int32{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedInt64: []int64{1, 2}},
		&testpb.TestAllTypes{RepeatedInt64: []int64{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedUint32: []uint32{1, 2}},
		&testpb.TestAllTypes{RepeatedUint32: []uint32{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedUint64: []uint64{1, 2}},
		&testpb.TestAllTypes{RepeatedUint64: []uint64{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedSint32: []int32{1, 2}},
		&testpb.TestAllTypes{RepeatedSint32: []int32{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedSint64: []int64{1, 2}},
		&testpb.TestAllTypes{RepeatedSint64: []int64{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedFixed32: []uint32{1, 2}},
		&testpb.TestAllTypes{RepeatedFixed32: []uint32{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedFixed64: []uint64{1, 2}},
		&testpb.TestAllTypes{RepeatedFixed64: []uint64{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedSfixed32: []int32{1, 2}},
		&testpb.TestAllTypes{RepeatedSfixed32: []int32{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedSfixed64: []int64{1, 2}},
		&testpb.TestAllTypes{RepeatedSfixed64: []int64{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedFloat: []float32{1, 2}},
		&testpb.TestAllTypes{RepeatedFloat: []float32{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedDouble: []float64{1, 2}},
		&testpb.TestAllTypes{RepeatedDouble: []float64{1, 3}},
	},
	{
		&testpb.TestAllTypes{RepeatedBool: []bool{true, false}},
		&testpb.TestAllTypes{RepeatedBool: []bool{true, true}},
	},
	{
		&testpb.TestAllTypes{RepeatedString: []string{"a", "b"}},
		&testpb.TestAllTypes{RepeatedString: []string{"a", "c"}},
	},
	{
		&testpb.TestAllTypes{RepeatedBytes: [][]byte{[]byte("a"), []byte("b")}},
		&testpb.TestAllTypes{RepeatedBytes: [][]byte{[]byte("a"), []byte("c")}},
	},
	{
		&testpb.TestAllTypes{RepeatedNestedEnum: []testpb.TestAllTypes_NestedEnum{testpb.TestAllTypes_FOO}},
		&testpb.TestAllTypes{RepeatedNestedEnum: []testpb.TestAllTypes_NestedEnum{testpb.TestAllTypes_BAR}},
	},
	{
		&testpb.TestAllTypes{Repeatedgroup: []*testpb.TestAllTypes_RepeatedGroup{
			{A: scalar.Int32(1)},
			{A: scalar.Int32(2)},
		}},
		&testpb.TestAllTypes{Repeatedgroup: []*testpb.TestAllTypes_RepeatedGroup{
			{A: scalar.Int32(1)},
			{A: scalar.Int32(3)},
		}},
	},
	{
		&testpb.TestAllTypes{RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
			{A: scalar.Int32(1)},
			{A: scalar.Int32(2)},
		}},
		&testpb.TestAllTypes{RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
			{A: scalar.Int32(1)},
			{A: scalar.Int32(3)},
		}},
	},
	// Maps: various configurations.
	{
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{1: 2}},
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{3: 4}},
	},
	{
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{1: 2}},
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{1: 2, 3: 4}},
	},
	{
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{1: 2}},
	},
	// Maps: various types.
	{
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapInt32Int32: map[int32]int32{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapInt64Int64: map[int64]int64{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapInt64Int64: map[int64]int64{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapUint32Uint32: map[uint32]uint32{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapUint32Uint32: map[uint32]uint32{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapUint64Uint64: map[uint64]uint64{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapUint64Uint64: map[uint64]uint64{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapSint32Sint32: map[int32]int32{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapSint32Sint32: map[int32]int32{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapSint64Sint64: map[int64]int64{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapSint64Sint64: map[int64]int64{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapFixed32Fixed32: map[uint32]uint32{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapFixed32Fixed32: map[uint32]uint32{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapFixed64Fixed64: map[uint64]uint64{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapFixed64Fixed64: map[uint64]uint64{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapSfixed32Sfixed32: map[int32]int32{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapSfixed32Sfixed32: map[int32]int32{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapSfixed64Sfixed64: map[int64]int64{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapSfixed64Sfixed64: map[int64]int64{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapInt32Float: map[int32]float32{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapInt32Float: map[int32]float32{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapInt32Double: map[int32]float64{1: 2, 3: 4}},
		&testpb.TestAllTypes{MapInt32Double: map[int32]float64{1: 2, 3: 5}},
	},
	{
		&testpb.TestAllTypes{MapBoolBool: map[bool]bool{true: false, false: true}},
		&testpb.TestAllTypes{MapBoolBool: map[bool]bool{true: false, false: false}},
	},
	{
		&testpb.TestAllTypes{MapStringString: map[string]string{"a": "b", "c": "d"}},
		&testpb.TestAllTypes{MapStringString: map[string]string{"a": "b", "c": "e"}},
	},
	{
		&testpb.TestAllTypes{MapStringBytes: map[string][]byte{"a": []byte("b"), "c": []byte("d")}},
		&testpb.TestAllTypes{MapStringBytes: map[string][]byte{"a": []byte("b"), "c": []byte("e")}},
	},
	{
		&testpb.TestAllTypes{MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
			"a": {A: scalar.Int32(1)},
			"b": {A: scalar.Int32(2)},
		}},
		&testpb.TestAllTypes{MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
			"a": {A: scalar.Int32(1)},
			"b": {A: scalar.Int32(3)},
		}},
	},
	{
		&testpb.TestAllTypes{MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
			"a": testpb.TestAllTypes_FOO,
			"b": testpb.TestAllTypes_BAR,
		}},
		&testpb.TestAllTypes{MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
			"a": testpb.TestAllTypes_FOO,
			"b": testpb.TestAllTypes_BAZ,
		}},
	},
	// Unknown fields.
	{
		build(&testpb.TestAllTypes{}, unknown(pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(1),
		}.Marshal())),
		build(&testpb.TestAllTypes{}, unknown(pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(2),
		}.Marshal())),
	},
	{
		build(&testpb.TestAllTypes{}, unknown(pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(1),
		}.Marshal())),
		&testpb.TestAllTypes{},
	},
	{
		&testpb.TestAllTypes{},
		build(&testpb.TestAllTypes{}, unknown(pack.Message{
			pack.Tag{100000, pack.VarintType}, pack.Varint(1),
		}.Marshal())),
	},
	// Extensions.
	{
		build(&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalInt32Extension, scalar.Int32(1)),
		),
		build(&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalInt32Extension, scalar.Int32(2)),
		),
	},
	{
		&testpb.TestAllExtensions{},
		build(&testpb.TestAllExtensions{},
			extend(testpb.E_OptionalInt32Extension, scalar.Int32(2)),
		),
	},
	// Proto2 default values are not considered by Equal, so the following are still unequal.
	{
		&testpb.TestAllTypes{DefaultInt32: scalar.Int32(81)},
		&testpb.TestAllTypes{},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultInt32: scalar.Int32(81)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultInt64: scalar.Int64(82)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultUint32: scalar.Uint32(83)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultUint64: scalar.Uint64(84)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultSint32: scalar.Int32(-85)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultSint64: scalar.Int64(86)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultFixed32: scalar.Uint32(87)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultFixed64: scalar.Uint64(88)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultSfixed32: scalar.Int32(89)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultSfixed64: scalar.Int64(-90)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultFloat: scalar.Float32(91.5)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultDouble: scalar.Float64(92e3)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultBool: scalar.Bool(true)},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultString: scalar.String("hello")},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultBytes: []byte("world")},
	},
	{
		&testpb.TestAllTypes{},
		&testpb.TestAllTypes{DefaultNestedEnum: testpb.TestAllTypes_BAR.Enum()},
	},
}
