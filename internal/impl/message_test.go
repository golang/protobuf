// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
)

func mustMakeMessageDesc(t ptype.StandaloneMessage) pref.MessageDescriptor {
	md, err := ptype.NewMessage(&t)
	if err != nil {
		panic(err)
	}
	return md
}

var V = pref.ValueOf

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

	NamedStrings []MyString
	NamedBytes   []MyBytes
)

// List of test operations to perform on messages, vectors, or maps.
type (
	messageOp  interface{} // equalMessage | hasFields | getFields | setFields | clearFields | vectorFields | mapFields
	messageOps []messageOp

	vectorOp  interface{} // equalVector | lenVector | getVector | setVector | appendVector | truncVector
	vectorOps []vectorOp

	mapOp  interface{} // TODO
	mapOps []mapOp     // TODO
)

// Test operations performed on a message.
type (
	equalMessage  pref.Message
	hasFields     map[pref.FieldNumber]bool
	getFields     map[pref.FieldNumber]pref.Value
	setFields     map[pref.FieldNumber]pref.Value
	clearFields   map[pref.FieldNumber]bool
	vectorFields  map[pref.FieldNumber]vectorOps
	mapFields     map[pref.FieldNumber]mapOps
	messageFields map[pref.FieldNumber]messageOps
	// TODO: Mutable, Range, ExtensionTypes
)

// Test operations performed on a vector.
type (
	equalVector  pref.Vector
	lenVector    int
	getVector    map[int]pref.Value
	setVector    map[int]pref.Value
	appendVector []pref.Value
	truncVector  int
	// TODO: Mutable, MutableAppend
)

func TestScalarProto2(t *testing.T) {
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

	mi := MessageType{Desc: mustMakeMessageDesc(ptype.StandaloneMessage{
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
	})}

	testMessage(t, nil, mi.MessageOf(&ScalarProto2{}), messageOps{
		hasFields{
			1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false,
			12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false, 20: false, 21: false, 22: false,
		},
		getFields{
			1: V(bool(true)), 2: V(int32(2)), 3: V(int64(3)), 4: V(uint32(4)), 5: V(uint64(5)), 6: V(float32(6)), 7: V(float64(7)), 8: V(string("8")), 9: V(string("9")), 10: V([]byte("10")), 11: V([]byte("11")),
			12: V(bool(true)), 13: V(int32(13)), 14: V(int64(14)), 15: V(uint32(15)), 16: V(uint64(16)), 17: V(float32(17)), 18: V(float64(18)), 19: V(string("19")), 20: V(string("20")), 21: V([]byte("21")), 22: V([]byte("22")),
		},
		setFields{
			1: V(bool(false)), 2: V(int32(0)), 3: V(int64(0)), 4: V(uint32(0)), 5: V(uint64(0)), 6: V(float32(0)), 7: V(float64(0)), 8: V(string("")), 9: V(string("")), 10: V([]byte(nil)), 11: V([]byte(nil)),
			12: V(bool(false)), 13: V(int32(0)), 14: V(int64(0)), 15: V(uint32(0)), 16: V(uint64(0)), 17: V(float32(0)), 18: V(float64(0)), 19: V(string("")), 20: V(string("")), 21: V([]byte(nil)), 22: V([]byte(nil)),
		},
		hasFields{
			1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true,
			12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true,
		},
		equalMessage(mi.MessageOf(&ScalarProto2{
			new(bool), new(int32), new(int64), new(uint32), new(uint64), new(float32), new(float64), new(string), []byte{}, []byte{}, new(string),
			new(MyBool), new(MyInt32), new(MyInt64), new(MyUint32), new(MyUint64), new(MyFloat32), new(MyFloat64), new(MyString), MyBytes{}, MyBytes{}, new(MyString),
		})),
		clearFields{
			1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true,
			12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true,
		},
		equalMessage(mi.MessageOf(&ScalarProto2{})),
	})
}

func TestScalarProto3(t *testing.T) {
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

	mi := MessageType{Desc: mustMakeMessageDesc(ptype.StandaloneMessage{
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
	})}

	testMessage(t, nil, mi.MessageOf(&ScalarProto3{}), messageOps{
		hasFields{
			1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false,
			12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false, 20: false, 21: false, 22: false,
		},
		getFields{
			1: V(bool(false)), 2: V(int32(0)), 3: V(int64(0)), 4: V(uint32(0)), 5: V(uint64(0)), 6: V(float32(0)), 7: V(float64(0)), 8: V(string("")), 9: V(string("")), 10: V([]byte(nil)), 11: V([]byte(nil)),
			12: V(bool(false)), 13: V(int32(0)), 14: V(int64(0)), 15: V(uint32(0)), 16: V(uint64(0)), 17: V(float32(0)), 18: V(float64(0)), 19: V(string("")), 20: V(string("")), 21: V([]byte(nil)), 22: V([]byte(nil)),
		},
		setFields{
			1: V(bool(false)), 2: V(int32(0)), 3: V(int64(0)), 4: V(uint32(0)), 5: V(uint64(0)), 6: V(float32(0)), 7: V(float64(0)), 8: V(string("")), 9: V(string("")), 10: V([]byte(nil)), 11: V([]byte(nil)),
			12: V(bool(false)), 13: V(int32(0)), 14: V(int64(0)), 15: V(uint32(0)), 16: V(uint64(0)), 17: V(float32(0)), 18: V(float64(0)), 19: V(string("")), 20: V(string("")), 21: V([]byte(nil)), 22: V([]byte(nil)),
		},
		hasFields{
			1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false,
			12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false, 20: false, 21: false, 22: false,
		},
		equalMessage(mi.MessageOf(&ScalarProto3{})),
		setFields{
			1: V(bool(true)), 2: V(int32(2)), 3: V(int64(3)), 4: V(uint32(4)), 5: V(uint64(5)), 6: V(float32(6)), 7: V(float64(7)), 8: V(string("8")), 9: V(string("9")), 10: V([]byte("10")), 11: V([]byte("11")),
			12: V(bool(true)), 13: V(int32(13)), 14: V(int64(14)), 15: V(uint32(15)), 16: V(uint64(16)), 17: V(float32(17)), 18: V(float64(18)), 19: V(string("19")), 20: V(string("20")), 21: V([]byte("21")), 22: V([]byte("22")),
		},
		hasFields{
			1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true,
			12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true,
		},
		equalMessage(mi.MessageOf(&ScalarProto3{
			true, 2, 3, 4, 5, 6, 7, "8", []byte("9"), []byte("10"), "11",
			true, 13, 14, 15, 16, 17, 18, "19", []byte("20"), []byte("21"), "22",
		})),
		clearFields{
			1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true,
			12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true,
		},
		equalMessage(mi.MessageOf(&ScalarProto3{})),
	})
}

func TestRepeatedScalars(t *testing.T) {
	type RepeatedScalars struct {
		Bools    []bool    `protobuf:"1"`
		Int32s   []int32   `protobuf:"2"`
		Int64s   []int64   `protobuf:"3"`
		Uint32s  []uint32  `protobuf:"4"`
		Uint64s  []uint64  `protobuf:"5"`
		Float32s []float32 `protobuf:"6"`
		Float64s []float64 `protobuf:"7"`
		Strings  []string  `protobuf:"8"`
		StringsA [][]byte  `protobuf:"9"`
		Bytes    [][]byte  `protobuf:"10"`
		BytesA   []string  `protobuf:"11"`

		MyStrings1 []MyString `protobuf:"12"`
		MyStrings2 []MyBytes  `protobuf:"13"`
		MyBytes1   []MyBytes  `protobuf:"14"`
		MyBytes2   []MyString `protobuf:"15"`

		MyStrings3 NamedStrings `protobuf:"16"`
		MyStrings4 NamedBytes   `protobuf:"17"`
		MyBytes3   NamedBytes   `protobuf:"18"`
		MyBytes4   NamedStrings `protobuf:"19"`
	}

	mi := MessageType{Desc: mustMakeMessageDesc(ptype.StandaloneMessage{
		Syntax:   pref.Proto2,
		FullName: "RepeatedScalars",
		Fields: []ptype.Field{
			{Name: "f1", Number: 1, Cardinality: pref.Repeated, Kind: pref.BoolKind},
			{Name: "f2", Number: 2, Cardinality: pref.Repeated, Kind: pref.Int32Kind},
			{Name: "f3", Number: 3, Cardinality: pref.Repeated, Kind: pref.Int64Kind},
			{Name: "f4", Number: 4, Cardinality: pref.Repeated, Kind: pref.Uint32Kind},
			{Name: "f5", Number: 5, Cardinality: pref.Repeated, Kind: pref.Uint64Kind},
			{Name: "f6", Number: 6, Cardinality: pref.Repeated, Kind: pref.FloatKind},
			{Name: "f7", Number: 7, Cardinality: pref.Repeated, Kind: pref.DoubleKind},
			{Name: "f8", Number: 8, Cardinality: pref.Repeated, Kind: pref.StringKind},
			{Name: "f9", Number: 9, Cardinality: pref.Repeated, Kind: pref.StringKind},
			{Name: "f10", Number: 10, Cardinality: pref.Repeated, Kind: pref.BytesKind},
			{Name: "f11", Number: 11, Cardinality: pref.Repeated, Kind: pref.BytesKind},

			{Name: "f12", Number: 12, Cardinality: pref.Repeated, Kind: pref.StringKind},
			{Name: "f13", Number: 13, Cardinality: pref.Repeated, Kind: pref.StringKind},
			{Name: "f14", Number: 14, Cardinality: pref.Repeated, Kind: pref.BytesKind},
			{Name: "f15", Number: 15, Cardinality: pref.Repeated, Kind: pref.BytesKind},

			{Name: "f16", Number: 16, Cardinality: pref.Repeated, Kind: pref.StringKind},
			{Name: "f17", Number: 17, Cardinality: pref.Repeated, Kind: pref.StringKind},
			{Name: "f18", Number: 18, Cardinality: pref.Repeated, Kind: pref.BytesKind},
			{Name: "f19", Number: 19, Cardinality: pref.Repeated, Kind: pref.BytesKind},
		},
	})}

	empty := mi.MessageOf(&RepeatedScalars{})
	emptyFS := empty.KnownFields()

	want := mi.MessageOf(&RepeatedScalars{
		Bools:    []bool{true, false, true},
		Int32s:   []int32{2, math.MinInt32, math.MaxInt32},
		Int64s:   []int64{3, math.MinInt64, math.MaxInt64},
		Uint32s:  []uint32{4, math.MaxUint32 / 2, math.MaxUint32},
		Uint64s:  []uint64{5, math.MaxUint64 / 2, math.MaxUint64},
		Float32s: []float32{6, math.SmallestNonzeroFloat32, float32(math.NaN()), math.MaxFloat32},
		Float64s: []float64{7, math.SmallestNonzeroFloat64, float64(math.NaN()), math.MaxFloat64},
		Strings:  []string{"8", "", "eight"},
		StringsA: [][]byte{[]byte("9"), nil, []byte("nine")},
		Bytes:    [][]byte{[]byte("10"), nil, []byte("ten")},
		BytesA:   []string{"11", "", "eleven"},

		MyStrings1: []MyString{"12", "", "twelve"},
		MyStrings2: []MyBytes{[]byte("13"), nil, []byte("thirteen")},
		MyBytes1:   []MyBytes{[]byte("14"), nil, []byte("fourteen")},
		MyBytes2:   []MyString{"15", "", "fifteen"},

		MyStrings3: NamedStrings{"16", "", "sixteen"},
		MyStrings4: NamedBytes{[]byte("17"), nil, []byte("seventeen")},
		MyBytes3:   NamedBytes{[]byte("18"), nil, []byte("eighteen")},
		MyBytes4:   NamedStrings{"19", "", "nineteen"},
	})
	wantFS := want.KnownFields()

	testMessage(t, nil, mi.MessageOf(&RepeatedScalars{}), messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false},
		getFields{1: emptyFS.Get(1), 3: emptyFS.Get(3), 5: emptyFS.Get(5), 7: emptyFS.Get(7), 9: emptyFS.Get(9), 11: emptyFS.Get(11), 13: emptyFS.Get(13), 15: emptyFS.Get(15), 17: emptyFS.Get(17), 19: emptyFS.Get(19)},
		setFields{1: wantFS.Get(1), 3: wantFS.Get(3), 5: wantFS.Get(5), 7: wantFS.Get(7), 9: wantFS.Get(9), 11: wantFS.Get(11), 13: wantFS.Get(13), 15: wantFS.Get(15), 17: wantFS.Get(17), 19: wantFS.Get(19)},
		vectorFields{
			2: {
				lenVector(0),
				appendVector{V(int32(2)), V(int32(math.MinInt32)), V(int32(math.MaxInt32))},
				getVector{0: V(int32(2)), 1: V(int32(math.MinInt32)), 2: V(int32(math.MaxInt32))},
				equalVector(wantFS.Get(2).Vector()),
			},
			4: {
				appendVector{V(uint32(0)), V(uint32(0)), V(uint32(0))},
				setVector{0: V(uint32(4)), 1: V(uint32(math.MaxUint32 / 2)), 2: V(uint32(math.MaxUint32))},
				lenVector(3),
			},
			6: {
				appendVector{V(float32(6)), V(float32(math.SmallestNonzeroFloat32)), V(float32(math.NaN())), V(float32(math.MaxFloat32))},
				equalVector(wantFS.Get(6).Vector()),
			},
			8: {
				appendVector{V(""), V(""), V(""), V(""), V(""), V("")},
				lenVector(6),
				setVector{0: V("8"), 2: V("eight")},
				truncVector(3),
				equalVector(wantFS.Get(8).Vector()),
			},
			10: {
				appendVector{V([]byte(nil)), V([]byte(nil))},
				setVector{0: V([]byte("10"))},
				appendVector{V([]byte("wrong"))},
				setVector{2: V([]byte("ten"))},
				equalVector(wantFS.Get(10).Vector()),
			},
			12: {
				appendVector{V("12"), V("wrong"), V("twelve")},
				setVector{1: V("")},
				equalVector(wantFS.Get(12).Vector()),
			},
			14: {
				appendVector{V([]byte("14")), V([]byte(nil)), V([]byte("fourteen"))},
				equalVector(wantFS.Get(14).Vector()),
			},
			16: {
				appendVector{V("16"), V(""), V("sixteen"), V("extra")},
				truncVector(3),
				equalVector(wantFS.Get(16).Vector()),
			},
			18: {
				appendVector{V([]byte("18")), V([]byte(nil)), V([]byte("eighteen"))},
				equalVector(wantFS.Get(18).Vector()),
			},
		},
		hasFields{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true, 12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true},
		equalMessage(want),
		clearFields{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true, 12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true},
		equalMessage(mi.MessageOf(&RepeatedScalars{})),
	})
}

// TODO: Need to test singular and repeated messages

var cmpOpts = cmp.Options{
	cmp.Transformer("UnwrapValue", func(v pref.Value) interface{} {
		return v.Interface()
	}),
	cmp.Transformer("UnwrapMessage", func(m pref.Message) interface{} {
		v := m.Interface()
		if v, ok := v.(interface{ Unwrap() interface{} }); ok {
			return v.Unwrap()
		}
		return v
	}),
	cmp.Transformer("UnwrapVector", func(v pref.Vector) interface{} {
		return v.(interface{ Unwrap() interface{} }).Unwrap()
	}),
	cmp.Transformer("UnwrapMap", func(m pref.Map) interface{} {
		return m.(interface{ Unwrap() interface{} }).Unwrap()
	}),
	cmpopts.EquateNaNs(),
}

func testMessage(t *testing.T, p path, m pref.Message, tt messageOps) {
	fs := m.KnownFields()
	for i, op := range tt {
		p.Push(i)
		switch op := op.(type) {
		case equalMessage:
			if diff := cmp.Diff(op, m, cmpOpts); diff != "" {
				t.Errorf("operation %v, message mismatch (-want, +got):\n%s", p, diff)
			}
		case hasFields:
			got := map[pref.FieldNumber]bool{}
			want := map[pref.FieldNumber]bool(op)
			for n := range want {
				got[n] = fs.Has(n)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("operation %v, KnownFields.Has mismatch (-want, +got):\n%s", p, diff)
			}
		case getFields:
			got := map[pref.FieldNumber]pref.Value{}
			want := map[pref.FieldNumber]pref.Value(op)
			for n := range want {
				got[n] = fs.Get(n)
			}
			if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
				t.Errorf("operation %v, KnownFields.Get mismatch (-want, +got):\n%s", p, diff)
			}
		case setFields:
			for n, v := range op {
				fs.Set(n, v)
			}
		case clearFields:
			for n, ok := range op {
				if ok {
					fs.Clear(n)
				}
			}
		case vectorFields:
			for n, tt := range op {
				p.Push(int(n))
				testVectors(t, p, fs.Mutable(n).(pref.Vector), tt)
				p.Pop()
			}
		default:
			t.Fatalf("operation %v, invalid operation: %T", p, op)
		}
		p.Pop()
	}
}

func testVectors(t *testing.T, p path, v pref.Vector, tt vectorOps) {
	for i, op := range tt {
		p.Push(i)
		switch op := op.(type) {
		case equalVector:
			if diff := cmp.Diff(op, v, cmpOpts); diff != "" {
				t.Errorf("operation %v, vector mismatch (-want, +got):\n%s", p, diff)
			}
		case lenVector:
			if got, want := v.Len(), int(op); got != want {
				t.Errorf("operation %v, Vector.Len = %d, want %d", p, got, want)
			}
		case getVector:
			got := map[int]pref.Value{}
			want := map[int]pref.Value(op)
			for n := range want {
				got[n] = v.Get(n)
			}
			if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
				t.Errorf("operation %v, Vector.Get mismatch (-want, +got):\n%s", p, diff)
			}
		case setVector:
			for n, e := range op {
				v.Set(n, e)
			}
		case appendVector:
			for _, e := range op {
				v.Append(e)
			}
		case truncVector:
			v.Truncate(int(op))
		default:
			t.Fatalf("operation %v, invalid operation: %T", p, op)
		}
		p.Pop()
	}
}

type path []int

func (p *path) Push(i int) { *p = append(*p, i) }
func (p *path) Pop()       { *p = (*p)[:len(*p)-1] }
func (p path) String() string {
	var ss []string
	for _, i := range p {
		ss = append(ss, fmt.Sprint(i))
	}
	return strings.Join(ss, ".")
}
