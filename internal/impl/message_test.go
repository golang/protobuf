// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"

	pref "google.golang.org/proto/reflect/protoreflect"
	ptype "google.golang.org/proto/reflect/prototype"
)

type (
	MyBool    bool
	MyInt32   int32
	MyInt64   int64
	MyUint32  uint32
	MyUint64  uint64
	MyFloat32 float32
	MyFloat64 float64
	MyString  string
	MyBytes   []byte
)

type ScalarProto2 struct {
	Bool    *bool    `protobuf:"1"`
	Int32   *int32   `protobuf:"2"`
	Int64   *int64   `protobuf:"3"`
	Uint32  *uint32  `protobuf:"4"`
	Uint64  *uint64  `protobuf:"5"`
	Float32 *float32 `protobuf:"6"`
	Float64 *float64 `protobuf:"7"`
	String  *string  `protobuf:"8"`
	StringA []byte   `protobuf:"9"`
	Bytes   []byte   `protobuf:"10"`
	BytesA  *string  `protobuf:"11"`

	MyBool    *MyBool    `protobuf:"12"`
	MyInt32   *MyInt32   `protobuf:"13"`
	MyInt64   *MyInt64   `protobuf:"14"`
	MyUint32  *MyUint32  `protobuf:"15"`
	MyUint64  *MyUint64  `protobuf:"16"`
	MyFloat32 *MyFloat32 `protobuf:"17"`
	MyFloat64 *MyFloat64 `protobuf:"18"`
	MyString  *MyString  `protobuf:"19"`
	MyStringA MyBytes    `protobuf:"20"`
	MyBytes   MyBytes    `protobuf:"21"`
	MyBytesA  *MyString  `protobuf:"22"`
}

type ScalarProto3 struct {
	Bool    bool    `protobuf:"1"`
	Int32   int32   `protobuf:"2"`
	Int64   int64   `protobuf:"3"`
	Uint32  uint32  `protobuf:"4"`
	Uint64  uint64  `protobuf:"5"`
	Float32 float32 `protobuf:"6"`
	Float64 float64 `protobuf:"7"`
	String  string  `protobuf:"8"`
	StringA []byte  `protobuf:"9"`
	Bytes   []byte  `protobuf:"10"`
	BytesA  string  `protobuf:"11"`

	MyBool    MyBool    `protobuf:"12"`
	MyInt32   MyInt32   `protobuf:"13"`
	MyInt64   MyInt64   `protobuf:"14"`
	MyUint32  MyUint32  `protobuf:"15"`
	MyUint64  MyUint64  `protobuf:"16"`
	MyFloat32 MyFloat32 `protobuf:"17"`
	MyFloat64 MyFloat64 `protobuf:"18"`
	MyString  MyString  `protobuf:"19"`
	MyStringA MyBytes   `protobuf:"20"`
	MyBytes   MyBytes   `protobuf:"21"`
	MyBytesA  MyString  `protobuf:"22"`
}

func TestFieldFuncs(t *testing.T) {
	V := pref.ValueOf
	type (
		// has checks that each field matches the list.
		hasOp []bool
		// get checks that each field returns values matching the list.
		getOp []pref.Value
		// set calls set on each field with the given value in the list.
		setOp []pref.Value
		// clear calls clear on each field.
		clearOp []bool
		// equal checks that the current message equals the provided value.
		equalOp struct{ want interface{} }

		testOp interface{} // has | get | set | clear | equal
	)

	tests := []struct {
		structType  reflect.Type
		messageDesc ptype.StandaloneMessage
		testOps     []testOp
	}{{
		structType: reflect.TypeOf(ScalarProto2{}),
		messageDesc: ptype.StandaloneMessage{
			Syntax:   pref.Proto2,
			FullName: "ScalarProto2",
			Fields: []ptype.Field{
				{Name: "f1", Number: 1, Cardinality: pref.Optional, Kind: pref.BoolKind, Default: V(bool(true))},
				{Name: "f2", Number: 2, Cardinality: pref.Optional, Kind: pref.Int32Kind, Default: V(int32(2))},
				{Name: "f3", Number: 3, Cardinality: pref.Optional, Kind: pref.Int64Kind, Default: V(int64(3))},
				{Name: "f4", Number: 4, Cardinality: pref.Optional, Kind: pref.Uint32Kind, Default: V(uint32(4))},
				{Name: "f5", Number: 5, Cardinality: pref.Optional, Kind: pref.Uint64Kind, Default: V(uint64(5))},
				{Name: "f6", Number: 6, Cardinality: pref.Optional, Kind: pref.FloatKind, Default: V(float32(6))},
				{Name: "f7", Number: 7, Cardinality: pref.Optional, Kind: pref.DoubleKind, Default: V(float64(7))},
				{Name: "f8", Number: 8, Cardinality: pref.Optional, Kind: pref.StringKind, Default: V(string("8"))},
				{Name: "f9", Number: 9, Cardinality: pref.Optional, Kind: pref.StringKind, Default: V(string("9"))},
				{Name: "f10", Number: 10, Cardinality: pref.Optional, Kind: pref.BytesKind, Default: V([]byte("10"))},
				{Name: "f11", Number: 11, Cardinality: pref.Optional, Kind: pref.BytesKind, Default: V([]byte("11"))},

				{Name: "f12", Number: 12, Cardinality: pref.Optional, Kind: pref.BoolKind, Default: V(bool(true))},
				{Name: "f13", Number: 13, Cardinality: pref.Optional, Kind: pref.Int32Kind, Default: V(int32(13))},
				{Name: "f14", Number: 14, Cardinality: pref.Optional, Kind: pref.Int64Kind, Default: V(int64(14))},
				{Name: "f15", Number: 15, Cardinality: pref.Optional, Kind: pref.Uint32Kind, Default: V(uint32(15))},
				{Name: "f16", Number: 16, Cardinality: pref.Optional, Kind: pref.Uint64Kind, Default: V(uint64(16))},
				{Name: "f17", Number: 17, Cardinality: pref.Optional, Kind: pref.FloatKind, Default: V(float32(17))},
				{Name: "f18", Number: 18, Cardinality: pref.Optional, Kind: pref.DoubleKind, Default: V(float64(18))},
				{Name: "f19", Number: 19, Cardinality: pref.Optional, Kind: pref.StringKind, Default: V(string("19"))},
				{Name: "f20", Number: 20, Cardinality: pref.Optional, Kind: pref.StringKind, Default: V(string("20"))},
				{Name: "f21", Number: 21, Cardinality: pref.Optional, Kind: pref.BytesKind, Default: V([]byte("21"))},
				{Name: "f22", Number: 22, Cardinality: pref.Optional, Kind: pref.BytesKind, Default: V([]byte("22"))},
			},
		},
		testOps: []testOp{
			hasOp([]bool{
				false, false, false, false, false, false, false, false, false, false, false,
				false, false, false, false, false, false, false, false, false, false, false,
			}),
			getOp([]pref.Value{
				V(bool(true)), V(int32(2)), V(int64(3)), V(uint32(4)), V(uint64(5)), V(float32(6)), V(float64(7)), V(string("8")), V(string("9")), V([]byte("10")), V([]byte("11")),
				V(bool(true)), V(int32(13)), V(int64(14)), V(uint32(15)), V(uint64(16)), V(float32(17)), V(float64(18)), V(string("19")), V(string("20")), V([]byte("21")), V([]byte("22")),
			}),
			setOp([]pref.Value{
				V(bool(false)), V(int32(0)), V(int64(0)), V(uint32(0)), V(uint64(0)), V(float32(0)), V(float64(0)), V(string("")), V(string("")), V([]byte(nil)), V([]byte(nil)),
				V(bool(false)), V(int32(0)), V(int64(0)), V(uint32(0)), V(uint64(0)), V(float32(0)), V(float64(0)), V(string("")), V(string("")), V([]byte(nil)), V([]byte(nil)),
			}),
			hasOp([]bool{
				true, true, true, true, true, true, true, true, true, true, true,
				true, true, true, true, true, true, true, true, true, true, true,
			}),
			equalOp{&ScalarProto2{
				new(bool), new(int32), new(int64), new(uint32), new(uint64), new(float32), new(float64), new(string), []byte{}, []byte{}, new(string),
				new(MyBool), new(MyInt32), new(MyInt64), new(MyUint32), new(MyUint64), new(MyFloat32), new(MyFloat64), new(MyString), MyBytes{}, MyBytes{}, new(MyString),
			}},
			clearOp([]bool{
				true, true, true, true, true, true, true, true, true, true, true,
				true, true, true, true, true, true, true, true, true, true, true,
			}),
			equalOp{&ScalarProto2{}},
		},
	}, {
		structType: reflect.TypeOf(ScalarProto3{}),
		messageDesc: ptype.StandaloneMessage{
			Syntax:   pref.Proto3,
			FullName: "ScalarProto3",
			Fields: []ptype.Field{
				{Name: "f1", Number: 1, Cardinality: pref.Optional, Kind: pref.BoolKind},
				{Name: "f2", Number: 2, Cardinality: pref.Optional, Kind: pref.Int32Kind},
				{Name: "f3", Number: 3, Cardinality: pref.Optional, Kind: pref.Int64Kind},
				{Name: "f4", Number: 4, Cardinality: pref.Optional, Kind: pref.Uint32Kind},
				{Name: "f5", Number: 5, Cardinality: pref.Optional, Kind: pref.Uint64Kind},
				{Name: "f6", Number: 6, Cardinality: pref.Optional, Kind: pref.FloatKind},
				{Name: "f7", Number: 7, Cardinality: pref.Optional, Kind: pref.DoubleKind},
				{Name: "f8", Number: 8, Cardinality: pref.Optional, Kind: pref.StringKind},
				{Name: "f9", Number: 9, Cardinality: pref.Optional, Kind: pref.StringKind},
				{Name: "f10", Number: 10, Cardinality: pref.Optional, Kind: pref.BytesKind},
				{Name: "f11", Number: 11, Cardinality: pref.Optional, Kind: pref.BytesKind},

				{Name: "f12", Number: 12, Cardinality: pref.Optional, Kind: pref.BoolKind},
				{Name: "f13", Number: 13, Cardinality: pref.Optional, Kind: pref.Int32Kind},
				{Name: "f14", Number: 14, Cardinality: pref.Optional, Kind: pref.Int64Kind},
				{Name: "f15", Number: 15, Cardinality: pref.Optional, Kind: pref.Uint32Kind},
				{Name: "f16", Number: 16, Cardinality: pref.Optional, Kind: pref.Uint64Kind},
				{Name: "f17", Number: 17, Cardinality: pref.Optional, Kind: pref.FloatKind},
				{Name: "f18", Number: 18, Cardinality: pref.Optional, Kind: pref.DoubleKind},
				{Name: "f19", Number: 19, Cardinality: pref.Optional, Kind: pref.StringKind},
				{Name: "f20", Number: 20, Cardinality: pref.Optional, Kind: pref.StringKind},
				{Name: "f21", Number: 21, Cardinality: pref.Optional, Kind: pref.BytesKind},
				{Name: "f22", Number: 22, Cardinality: pref.Optional, Kind: pref.BytesKind},
			},
		},
		testOps: []testOp{
			hasOp([]bool{
				false, false, false, false, false, false, false, false, false, false, false,
				false, false, false, false, false, false, false, false, false, false, false,
			}),
			getOp([]pref.Value{
				V(bool(false)), V(int32(0)), V(int64(0)), V(uint32(0)), V(uint64(0)), V(float32(0)), V(float64(0)), V(string("")), V(string("")), V([]byte(nil)), V([]byte(nil)),
				V(bool(false)), V(int32(0)), V(int64(0)), V(uint32(0)), V(uint64(0)), V(float32(0)), V(float64(0)), V(string("")), V(string("")), V([]byte(nil)), V([]byte(nil)),
			}),
			setOp([]pref.Value{
				V(bool(false)), V(int32(0)), V(int64(0)), V(uint32(0)), V(uint64(0)), V(float32(0)), V(float64(0)), V(string("")), V(string("")), V([]byte(nil)), V([]byte(nil)),
				V(bool(false)), V(int32(0)), V(int64(0)), V(uint32(0)), V(uint64(0)), V(float32(0)), V(float64(0)), V(string("")), V(string("")), V([]byte(nil)), V([]byte(nil)),
			}),
			hasOp([]bool{
				false, false, false, false, false, false, false, false, false, false, false,
				false, false, false, false, false, false, false, false, false, false, false,
			}),
			equalOp{&ScalarProto3{}},
			setOp([]pref.Value{
				V(bool(true)), V(int32(2)), V(int64(3)), V(uint32(4)), V(uint64(5)), V(float32(6)), V(float64(7)), V(string("8")), V(string("9")), V([]byte("10")), V([]byte("11")),
				V(bool(true)), V(int32(13)), V(int64(14)), V(uint32(15)), V(uint64(16)), V(float32(17)), V(float64(18)), V(string("19")), V(string("20")), V([]byte("21")), V([]byte("22")),
			}),
			hasOp([]bool{
				true, true, true, true, true, true, true, true, true, true, true,
				true, true, true, true, true, true, true, true, true, true, true,
			}),
			equalOp{&ScalarProto3{
				true, 2, 3, 4, 5, 6, 7, "8", []byte("9"), []byte("10"), "11",
				true, 13, 14, 15, 16, 17, 18, "19", []byte("20"), []byte("21"), "22",
			}},
			clearOp([]bool{
				true, true, true, true, true, true, true, true, true, true, true,
				true, true, true, true, true, true, true, true, true, true, true,
			}),
			equalOp{&ScalarProto3{}},
		},
	}}

	for _, tt := range tests {
		t.Run(tt.structType.Name(), func(t *testing.T) {
			// Construct the message descriptor.
			md, err := ptype.NewMessage(&tt.messageDesc)
			if err != nil {
				t.Fatalf("NewMessage error: %v", err)
			}

			// Generate the field functions from the message descriptor.
			var mi MessageInfo
			mi.generateFieldFuncs(tt.structType, md) // must not panic

			// Test the field functions.
			m := reflect.New(tt.structType)
			p := pointerOfValue(m)
			for i, op := range tt.testOps {
				switch op := op.(type) {
				case hasOp:
					got := map[pref.FieldNumber]bool{}
					want := map[pref.FieldNumber]bool{}
					for j, ok := range op {
						n := pref.FieldNumber(j + 1)
						got[n] = mi.fields[n].has(p)
						want[n] = ok
					}
					if diff := cmp.Diff(want, got); diff != "" {
						t.Errorf("operation %d, has mismatch (-want, +got):\n%s", i, diff)
					}
				case getOp:
					got := map[pref.FieldNumber]pref.Value{}
					want := map[pref.FieldNumber]pref.Value{}
					for j, v := range op {
						n := pref.FieldNumber(j + 1)
						got[n] = mi.fields[n].get(p)
						want[n] = v
					}
					xformValue := cmp.Transformer("", func(v pref.Value) interface{} {
						return v.Interface()
					})
					if diff := cmp.Diff(want, got, xformValue); diff != "" {
						t.Errorf("operation %d, get mismatch (-want, +got):\n%s", i, diff)
					}
				case setOp:
					for j, v := range op {
						n := pref.FieldNumber(j + 1)
						mi.fields[n].set(p, v)
					}
				case clearOp:
					for j, ok := range op {
						n := pref.FieldNumber(j + 1)
						if ok {
							mi.fields[n].clear(p)
						}
					}
				case equalOp:
					got := m.Interface()
					if diff := cmp.Diff(op.want, got); diff != "" {
						t.Errorf("operation %d, equal mismatch (-want, +got):\n%s", i, diff)
					}
				}
			}
		})
	}
}
