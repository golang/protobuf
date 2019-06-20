// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"testing"

	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/internal/scalar"
	"google.golang.org/protobuf/proto"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

func TestMerge(t *testing.T) {
	dst := new(testpb.TestAllTypes)
	src := (*testpb.TestAllTypes)(nil)
	proto.Merge(dst, src)

	// Mutating the source should not affect dst.

	tests := []struct {
		desc    string
		dst     proto.Message
		src     proto.Message
		want    proto.Message
		mutator func(proto.Message) // if provided, is run on src after merging

		skipMarshalUnmarshal bool // TODO: Remove this when proto.Unmarshal is fixed for messages in oneofs
	}{{
		desc: "merge from nil message",
		dst:  new(testpb.TestAllTypes),
		src:  (*testpb.TestAllTypes)(nil),
		want: new(testpb.TestAllTypes),
	}, {
		desc: "clone a large message",
		dst:  new(testpb.TestAllTypes),
		src: &testpb.TestAllTypes{
			OptionalInt64:      scalar.Int64(0),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(1).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: scalar.Int32(100),
			},
			RepeatedSfixed32: []int32{1, 2, 3},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: scalar.Int32(200)},
				{A: scalar.Int32(300)},
			},
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 400,
				"buzz": 500,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: scalar.Int32(600)},
				"bar": {A: scalar.Int32(700)},
			},
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: scalar.Int32(800),
				},
			},
		},
		want: &testpb.TestAllTypes{
			OptionalInt64:      scalar.Int64(0),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(1).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: scalar.Int32(100),
			},
			RepeatedSfixed32: []int32{1, 2, 3},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: scalar.Int32(200)},
				{A: scalar.Int32(300)},
			},
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 400,
				"buzz": 500,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: scalar.Int32(600)},
				"bar": {A: scalar.Int32(700)},
			},
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: scalar.Int32(800),
				},
			},
		},
		mutator: func(mi proto.Message) {
			m := mi.(*testpb.TestAllTypes)
			*m.OptionalInt64++
			*m.OptionalNestedEnum++
			*m.OptionalNestedMessage.A++
			m.RepeatedSfixed32[0]++
			*m.RepeatedNestedMessage[0].A++
			delete(m.MapStringNestedEnum, "fizz")
			*m.MapStringNestedMessage["foo"].A++
			*m.OneofField.(*testpb.TestAllTypes_OneofNestedMessage).OneofNestedMessage.A++
		},
	}, {
		desc: "merge bytes",
		dst: &testpb.TestAllTypes{
			OptionalBytes:  []byte{1, 2, 3},
			RepeatedBytes:  [][]byte{{1, 2}, {3, 4}},
			MapStringBytes: map[string][]byte{"alpha": {1, 2, 3}},
		},
		src: &testpb.TestAllTypes{
			OptionalBytes:  []byte{4, 5, 6},
			RepeatedBytes:  [][]byte{{5, 6}, {7, 8}},
			MapStringBytes: map[string][]byte{"alpha": {4, 5, 6}, "bravo": {1, 2, 3}},
		},
		want: &testpb.TestAllTypes{
			OptionalBytes:  []byte{4, 5, 6},
			RepeatedBytes:  [][]byte{{1, 2}, {3, 4}, {5, 6}, {7, 8}},
			MapStringBytes: map[string][]byte{"alpha": {4, 5, 6}, "bravo": {1, 2, 3}},
		},
		mutator: func(mi proto.Message) {
			m := mi.(*testpb.TestAllTypes)
			m.OptionalBytes[0]++
			m.RepeatedBytes[0][0]++
			m.MapStringBytes["alpha"][0]++
		},
	}, {
		desc: "merge singular fields",
		dst: &testpb.TestAllTypes{
			OptionalInt32:      scalar.Int32(1),
			OptionalInt64:      scalar.Int64(1),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(10).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: scalar.Int32(100),
				Corecursive: &testpb.TestAllTypes{
					OptionalInt64: scalar.Int64(1000),
				},
			},
		},
		src: &testpb.TestAllTypes{
			OptionalInt64:      scalar.Int64(2),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(20).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: scalar.Int32(200),
			},
		},
		want: &testpb.TestAllTypes{
			OptionalInt32:      scalar.Int32(1),
			OptionalInt64:      scalar.Int64(2),
			OptionalNestedEnum: testpb.TestAllTypes_NestedEnum(20).Enum(),
			OptionalNestedMessage: &testpb.TestAllTypes_NestedMessage{
				A: scalar.Int32(200),
				Corecursive: &testpb.TestAllTypes{
					OptionalInt64: scalar.Int64(1000),
				},
			},
		},
		mutator: func(mi proto.Message) {
			m := mi.(*testpb.TestAllTypes)
			*m.OptionalInt64++
			*m.OptionalNestedEnum++
			*m.OptionalNestedMessage.A++
		},
	}, {
		desc: "merge list fields",
		dst: &testpb.TestAllTypes{
			RepeatedSfixed32: []int32{1, 2, 3},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: scalar.Int32(100)},
				{A: scalar.Int32(200)},
			},
		},
		src: &testpb.TestAllTypes{
			RepeatedSfixed32: []int32{4, 5, 6},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: scalar.Int32(300)},
				{A: scalar.Int32(400)},
			},
		},
		want: &testpb.TestAllTypes{
			RepeatedSfixed32: []int32{1, 2, 3, 4, 5, 6},
			RepeatedNestedMessage: []*testpb.TestAllTypes_NestedMessage{
				{A: scalar.Int32(100)},
				{A: scalar.Int32(200)},
				{A: scalar.Int32(300)},
				{A: scalar.Int32(400)},
			},
		},
		mutator: func(mi proto.Message) {
			m := mi.(*testpb.TestAllTypes)
			m.RepeatedSfixed32[0]++
			*m.RepeatedNestedMessage[0].A++
		},
	}, {
		desc: "merge map fields",
		dst: &testpb.TestAllTypes{
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 100,
				"buzz": 200,
				"guzz": 300,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: scalar.Int32(400)},
			},
		},
		src: &testpb.TestAllTypes{
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 1000,
				"buzz": 2000,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: scalar.Int32(3000)},
				"bar": {},
			},
		},
		want: &testpb.TestAllTypes{
			MapStringNestedEnum: map[string]testpb.TestAllTypes_NestedEnum{
				"fizz": 1000,
				"buzz": 2000,
				"guzz": 300,
			},
			MapStringNestedMessage: map[string]*testpb.TestAllTypes_NestedMessage{
				"foo": {A: scalar.Int32(3000)},
				"bar": {},
			},
		},
		mutator: func(mi proto.Message) {
			m := mi.(*testpb.TestAllTypes)
			delete(m.MapStringNestedEnum, "fizz")
			m.MapStringNestedMessage["bar"].A = scalar.Int32(1)
		},
	}, {
		desc: "merge oneof message fields",
		dst: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: scalar.Int32(100),
				},
			},
		},
		src: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					Corecursive: &testpb.TestAllTypes{
						OptionalInt64: scalar.Int64(1000),
					},
				},
			},
		},
		want: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofNestedMessage{
				&testpb.TestAllTypes_NestedMessage{
					A: scalar.Int32(100),
					Corecursive: &testpb.TestAllTypes{
						OptionalInt64: scalar.Int64(1000),
					},
				},
			},
		},
		mutator: func(mi proto.Message) {
			m := mi.(*testpb.TestAllTypes)
			*m.OneofField.(*testpb.TestAllTypes_OneofNestedMessage).OneofNestedMessage.Corecursive.OptionalInt64++
		},
		skipMarshalUnmarshal: true,
	}, {
		desc: "merge oneof scalar fields",
		dst: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofUint32{100},
		},
		src: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofFloat{3.14152},
		},
		want: &testpb.TestAllTypes{
			OneofField: &testpb.TestAllTypes_OneofFloat{3.14152},
		},
		mutator: func(mi proto.Message) {
			m := mi.(*testpb.TestAllTypes)
			m.OneofField.(*testpb.TestAllTypes_OneofFloat).OneofFloat++
		},
	}, {
		desc: "merge extension fields",
		dst: func() proto.Message {
			m := new(testpb.TestAllExtensions)
			m.ProtoReflect().Set(
				testpb.E_OptionalInt32Extension.Type,
				testpb.E_OptionalInt32Extension.Type.ValueOf(int32(32)),
			)
			m.ProtoReflect().Set(
				testpb.E_OptionalNestedMessageExtension.Type,
				testpb.E_OptionalNestedMessageExtension.Type.ValueOf(&testpb.TestAllTypes_NestedMessage{
					A: scalar.Int32(50),
				}),
			)
			m.ProtoReflect().Set(
				testpb.E_RepeatedFixed32Extension.Type,
				testpb.E_RepeatedFixed32Extension.Type.ValueOf(&[]uint32{1, 2, 3}),
			)
			return m
		}(),
		src: func() proto.Message {
			m := new(testpb.TestAllExtensions)
			m.ProtoReflect().Set(
				testpb.E_OptionalInt64Extension.Type,
				testpb.E_OptionalInt64Extension.Type.ValueOf(int64(64)),
			)
			m.ProtoReflect().Set(
				testpb.E_OptionalNestedMessageExtension.Type,
				testpb.E_OptionalNestedMessageExtension.Type.ValueOf(&testpb.TestAllTypes_NestedMessage{
					Corecursive: &testpb.TestAllTypes{
						OptionalInt64: scalar.Int64(1000),
					},
				}),
			)
			m.ProtoReflect().Set(
				testpb.E_RepeatedFixed32Extension.Type,
				testpb.E_RepeatedFixed32Extension.Type.ValueOf(&[]uint32{4, 5, 6}),
			)
			return m
		}(),
		want: func() proto.Message {
			m := new(testpb.TestAllExtensions)
			m.ProtoReflect().Set(
				testpb.E_OptionalInt32Extension.Type,
				testpb.E_OptionalInt32Extension.Type.ValueOf(int32(32)),
			)
			m.ProtoReflect().Set(
				testpb.E_OptionalInt64Extension.Type,
				testpb.E_OptionalInt64Extension.Type.ValueOf(int64(64)),
			)
			m.ProtoReflect().Set(
				testpb.E_OptionalNestedMessageExtension.Type,
				testpb.E_OptionalNestedMessageExtension.Type.ValueOf(&testpb.TestAllTypes_NestedMessage{
					A: scalar.Int32(50),
					Corecursive: &testpb.TestAllTypes{
						OptionalInt64: scalar.Int64(1000),
					},
				}),
			)
			m.ProtoReflect().Set(
				testpb.E_RepeatedFixed32Extension.Type,
				testpb.E_RepeatedFixed32Extension.Type.ValueOf(&[]uint32{1, 2, 3, 4, 5, 6}),
			)
			return m
		}(),
	}, {
		desc: "merge unknown fields",
		dst: func() proto.Message {
			m := new(testpb.TestAllTypes)
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
			}.Marshal())
			return m
		}(),
		src: func() proto.Message {
			m := new(testpb.TestAllTypes)
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 500000, Type: pack.VarintType}, pack.Svarint(-50),
			}.Marshal())
			return m
		}(),
		want: func() proto.Message {
			m := new(testpb.TestAllTypes)
			m.ProtoReflect().SetUnknown(pack.Message{
				pack.Tag{Number: 50000, Type: pack.VarintType}, pack.Svarint(-5),
				pack.Tag{Number: 500000, Type: pack.VarintType}, pack.Svarint(-50),
			}.Marshal())
			return m
		}(),
	}}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Merge should be semantically equivalent to unmarshaling the
			// encoded form of src into the current dst.
			b1, err := proto.MarshalOptions{AllowPartial: true}.Marshal(tt.dst)
			if err != nil {
				t.Fatalf("Marshal(dst) error: %v", err)
			}
			b2, err := proto.MarshalOptions{AllowPartial: true}.Marshal(tt.src)
			if err != nil {
				t.Fatalf("Marshal(src) error: %v", err)
			}
			dst := tt.dst.ProtoReflect().New().Interface()
			err = proto.UnmarshalOptions{AllowPartial: true}.Unmarshal(append(b1, b2...), dst)
			if err != nil {
				t.Fatalf("Unmarshal() error: %v", err)
			}
			if !proto.Equal(dst, tt.want) && !tt.skipMarshalUnmarshal {
				t.Fatalf("Unmarshal(Marshal(dst)+Marshal(src)) mismatch: got %v, want %v", dst, tt.want)
			}

			proto.Merge(tt.dst, tt.src)
			if tt.mutator != nil {
				tt.mutator(tt.src) // should not be observable by dst
			}
			if !proto.Equal(tt.dst, tt.want) {
				t.Fatalf("Merge() mismatch: got %v, want %v", tt.dst, tt.want)
			}
		})
	}
}
