// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl_test

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	ptype "github.com/golang/protobuf/v2/internal/prototype"
	scalar "github.com/golang/protobuf/v2/internal/scalar"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	cmp "github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"

	// The legacy package must be imported prior to use of any legacy messages.
	// TODO: Remove this when protoV1 registers these hooks for you.
	_ "github.com/golang/protobuf/v2/internal/legacy"

	proto2_20180125 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v1.0.0-20180125-92554152"
	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

// List of test operations to perform on messages, lists, or maps.
type (
	messageOp  interface{ isMessageOp() }
	messageOps []messageOp

	listOp  interface{ isListOp() }
	listOps []listOp

	mapOp  interface{ isMapOp() }
	mapOps []mapOp
)

// Test operations performed on a message.
type (
	// check that the message contents match
	equalMessage struct{ pref.Message }
	// check presence for specific fields in the message
	hasFields map[pref.FieldNumber]bool
	// check that specific message fields match
	getFields map[pref.FieldNumber]pref.Value
	// set specific message fields
	setFields map[pref.FieldNumber]pref.Value
	// clear specific fields in the message
	clearFields []pref.FieldNumber
	// apply messageOps on each specified message field
	messageFields map[pref.FieldNumber]messageOps
	// apply listOps on each specified list field
	listFields map[pref.FieldNumber]listOps
	// apply mapOps on each specified map fields
	mapFields map[pref.FieldNumber]mapOps
	// range through all fields and check that they match
	rangeFields map[pref.FieldNumber]pref.Value
)

func (equalMessage) isMessageOp()  {}
func (hasFields) isMessageOp()     {}
func (getFields) isMessageOp()     {}
func (setFields) isMessageOp()     {}
func (clearFields) isMessageOp()   {}
func (messageFields) isMessageOp() {}
func (listFields) isMessageOp()    {}
func (mapFields) isMessageOp()     {}
func (rangeFields) isMessageOp()   {}

// Test operations performed on a list.
type (
	// check that the list contents match
	equalList struct{ pref.List }
	// check that list length matches
	lenList int
	// check that specific list entries match
	getList map[int]pref.Value
	// set specific list entries
	setList map[int]pref.Value
	// append entries to the list
	appendList []pref.Value
	// apply messageOps on a newly appended message
	appendMessageList messageOps
	// truncate the list to the specified length
	truncList int
)

func (equalList) isListOp()         {}
func (lenList) isListOp()           {}
func (getList) isListOp()           {}
func (setList) isListOp()           {}
func (appendList) isListOp()        {}
func (appendMessageList) isListOp() {}
func (truncList) isListOp()         {}

// Test operations performed on a map.
type (
	// check that the map contents match
	equalMap struct{ pref.Map }
	// check that map length matches
	lenMap int
	// check presence for specific entries in the map
	hasMap map[interface{}]bool
	// check that specific map entries match
	getMap map[interface{}]pref.Value
	// set specific map entries
	setMap map[interface{}]pref.Value
	// clear specific entries in the map
	clearMap []interface{}
	// apply messageOps on each specified message entry
	messageMap map[interface{}]messageOps
	// range through all entries and check that they match
	rangeMap map[interface{}]pref.Value
)

func (equalMap) isMapOp()   {}
func (lenMap) isMapOp()     {}
func (hasMap) isMapOp()     {}
func (getMap) isMapOp()     {}
func (setMap) isMapOp()     {}
func (clearMap) isMapOp()   {}
func (messageMap) isMapOp() {}
func (rangeMap) isMapOp()   {}

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

func mustMakeEnumDesc(t ptype.StandaloneEnum) pref.EnumDescriptor {
	ed, err := ptype.NewEnum(&t)
	if err != nil {
		panic(err)
	}
	return ed
}

func mustMakeMessageDesc(t ptype.StandaloneMessage) pref.MessageDescriptor {
	md, err := ptype.NewMessage(&t)
	if err != nil {
		panic(err)
	}
	return md
}

var V = pref.ValueOf
var VE = func(n pref.EnumNumber) pref.Value { return V(n) }

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

	ListStrings []MyString
	ListBytes   []MyBytes

	MapStrings map[MyString]MyString
	MapBytes   map[MyString]MyBytes
)

var scalarProto2Type = pimpl.MessageType{GoType: reflect.TypeOf(new(ScalarProto2)), PBType: ptype.GoMessage(
	mustMakeMessageDesc(ptype.StandaloneMessage{
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
	}),
	func(pref.MessageType) pref.Message {
		return new(ScalarProto2)
	},
)}

func (m *ScalarProto2) Type() pref.MessageType { return scalarProto2Type.PBType }
func (m *ScalarProto2) KnownFields() pref.KnownFields {
	return scalarProto2Type.MessageOf(m).KnownFields()
}
func (m *ScalarProto2) UnknownFields() pref.UnknownFields {
	return scalarProto2Type.MessageOf(m).UnknownFields()
}
func (m *ScalarProto2) Interface() pref.ProtoMessage { return m }
func (m *ScalarProto2) ProtoReflect() pref.Message   { return m }

func TestScalarProto2(t *testing.T) {
	testMessage(t, nil, &ScalarProto2{}, messageOps{
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
		equalMessage{&ScalarProto2{
			new(bool), new(int32), new(int64), new(uint32), new(uint64), new(float32), new(float64), new(string), []byte{}, []byte{}, new(string),
			new(MyBool), new(MyInt32), new(MyInt64), new(MyUint32), new(MyUint64), new(MyFloat32), new(MyFloat64), new(MyString), MyBytes{}, MyBytes{}, new(MyString),
		}},
		clearFields{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22},
		equalMessage{&ScalarProto2{}},
	})

	// Test read-only operations on nil message.
	testMessage(t, nil, (*ScalarProto2)(nil), messageOps{
		hasFields{
			1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false,
			12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false, 20: false, 21: false, 22: false,
		},
		getFields{
			1: V(bool(true)), 2: V(int32(2)), 3: V(int64(3)), 4: V(uint32(4)), 5: V(uint64(5)), 6: V(float32(6)), 7: V(float64(7)), 8: V(string("8")), 9: V(string("9")), 10: V([]byte("10")), 11: V([]byte("11")),
			12: V(bool(true)), 13: V(int32(13)), 14: V(int64(14)), 15: V(uint32(15)), 16: V(uint64(16)), 17: V(float32(17)), 18: V(float64(18)), 19: V(string("19")), 20: V(string("20")), 21: V([]byte("21")), 22: V([]byte("22")),
		},
	})
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

var scalarProto3Type = pimpl.MessageType{GoType: reflect.TypeOf(new(ScalarProto3)), PBType: ptype.GoMessage(
	mustMakeMessageDesc(ptype.StandaloneMessage{
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
	}),
	func(pref.MessageType) pref.Message {
		return new(ScalarProto3)
	},
)}

func (m *ScalarProto3) Type() pref.MessageType { return scalarProto3Type.PBType }
func (m *ScalarProto3) KnownFields() pref.KnownFields {
	return scalarProto3Type.MessageOf(m).KnownFields()
}
func (m *ScalarProto3) UnknownFields() pref.UnknownFields {
	return scalarProto3Type.MessageOf(m).UnknownFields()
}
func (m *ScalarProto3) Interface() pref.ProtoMessage { return m }
func (m *ScalarProto3) ProtoReflect() pref.Message   { return m }

func TestScalarProto3(t *testing.T) {
	testMessage(t, nil, &ScalarProto3{}, messageOps{
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
		equalMessage{&ScalarProto3{}},
		setFields{
			1: V(bool(true)), 2: V(int32(2)), 3: V(int64(3)), 4: V(uint32(4)), 5: V(uint64(5)), 6: V(float32(6)), 7: V(float64(7)), 8: V(string("8")), 9: V(string("9")), 10: V([]byte("10")), 11: V([]byte("11")),
			12: V(bool(true)), 13: V(int32(13)), 14: V(int64(14)), 15: V(uint32(15)), 16: V(uint64(16)), 17: V(float32(17)), 18: V(float64(18)), 19: V(string("19")), 20: V(string("20")), 21: V([]byte("21")), 22: V([]byte("22")),
		},
		hasFields{
			1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true,
			12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true,
		},
		equalMessage{&ScalarProto3{
			true, 2, 3, 4, 5, 6, 7, "8", []byte("9"), []byte("10"), "11",
			true, 13, 14, 15, 16, 17, 18, "19", []byte("20"), []byte("21"), "22",
		}},
		setFields{
			2: V(int32(-2)), 3: V(int64(-3)), 6: V(float32(math.Inf(-1))), 7: V(float64(math.NaN())),
		},
		hasFields{
			2: true, 3: true, 6: true, 7: true,
		},
		clearFields{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22},
		equalMessage{&ScalarProto3{}},
	})

	// Test read-only operations on nil message.
	testMessage(t, nil, (*ScalarProto3)(nil), messageOps{
		hasFields{
			1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false,
			12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false, 20: false, 21: false, 22: false,
		},
		getFields{
			1: V(bool(false)), 2: V(int32(0)), 3: V(int64(0)), 4: V(uint32(0)), 5: V(uint64(0)), 6: V(float32(0)), 7: V(float64(0)), 8: V(string("")), 9: V(string("")), 10: V([]byte(nil)), 11: V([]byte(nil)),
			12: V(bool(false)), 13: V(int32(0)), 14: V(int64(0)), 15: V(uint32(0)), 16: V(uint64(0)), 17: V(float32(0)), 18: V(float64(0)), 19: V(string("")), 20: V(string("")), 21: V([]byte(nil)), 22: V([]byte(nil)),
		},
	})
}

type ListScalars struct {
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

	MyStrings3 ListStrings `protobuf:"16"`
	MyStrings4 ListBytes   `protobuf:"17"`
	MyBytes3   ListBytes   `protobuf:"18"`
	MyBytes4   ListStrings `protobuf:"19"`
}

var listScalarsType = pimpl.MessageType{GoType: reflect.TypeOf(new(ListScalars)), PBType: ptype.GoMessage(
	mustMakeMessageDesc(ptype.StandaloneMessage{
		Syntax:   pref.Proto2,
		FullName: "ListScalars",
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
	}),
	func(pref.MessageType) pref.Message {
		return new(ListScalars)
	},
)}

func (m *ListScalars) Type() pref.MessageType { return listScalarsType.PBType }
func (m *ListScalars) KnownFields() pref.KnownFields {
	return listScalarsType.MessageOf(m).KnownFields()
}
func (m *ListScalars) UnknownFields() pref.UnknownFields {
	return listScalarsType.MessageOf(m).UnknownFields()
}
func (m *ListScalars) Interface() pref.ProtoMessage { return m }
func (m *ListScalars) ProtoReflect() pref.Message   { return m }

func TestListScalars(t *testing.T) {
	empty := &ListScalars{}
	emptyFS := empty.KnownFields()

	want := &ListScalars{
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

		MyStrings3: ListStrings{"16", "", "sixteen"},
		MyStrings4: ListBytes{[]byte("17"), nil, []byte("seventeen")},
		MyBytes3:   ListBytes{[]byte("18"), nil, []byte("eighteen")},
		MyBytes4:   ListStrings{"19", "", "nineteen"},
	}
	wantFS := want.KnownFields()

	testMessage(t, nil, &ListScalars{}, messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false},
		getFields{1: emptyFS.Get(1), 3: emptyFS.Get(3), 5: emptyFS.Get(5), 7: emptyFS.Get(7), 9: emptyFS.Get(9), 11: emptyFS.Get(11), 13: emptyFS.Get(13), 15: emptyFS.Get(15), 17: emptyFS.Get(17), 19: emptyFS.Get(19)},
		setFields{1: wantFS.Get(1), 3: wantFS.Get(3), 5: wantFS.Get(5), 7: wantFS.Get(7), 9: wantFS.Get(9), 11: wantFS.Get(11), 13: wantFS.Get(13), 15: wantFS.Get(15), 17: wantFS.Get(17), 19: wantFS.Get(19)},
		listFields{
			2: {
				lenList(0),
				appendList{V(int32(2)), V(int32(math.MinInt32)), V(int32(math.MaxInt32))},
				getList{0: V(int32(2)), 1: V(int32(math.MinInt32)), 2: V(int32(math.MaxInt32))},
				equalList{wantFS.Get(2).List()},
			},
			4: {
				appendList{V(uint32(0)), V(uint32(0)), V(uint32(0))},
				setList{0: V(uint32(4)), 1: V(uint32(math.MaxUint32 / 2)), 2: V(uint32(math.MaxUint32))},
				lenList(3),
			},
			6: {
				appendList{V(float32(6)), V(float32(math.SmallestNonzeroFloat32)), V(float32(math.NaN())), V(float32(math.MaxFloat32))},
				equalList{wantFS.Get(6).List()},
			},
			8: {
				appendList{V(""), V(""), V(""), V(""), V(""), V("")},
				lenList(6),
				setList{0: V("8"), 2: V("eight")},
				truncList(3),
				equalList{wantFS.Get(8).List()},
			},
			10: {
				appendList{V([]byte(nil)), V([]byte(nil))},
				setList{0: V([]byte("10"))},
				appendList{V([]byte("wrong"))},
				setList{2: V([]byte("ten"))},
				equalList{wantFS.Get(10).List()},
			},
			12: {
				appendList{V("12"), V("wrong"), V("twelve")},
				setList{1: V("")},
				equalList{wantFS.Get(12).List()},
			},
			14: {
				appendList{V([]byte("14")), V([]byte(nil)), V([]byte("fourteen"))},
				equalList{wantFS.Get(14).List()},
			},
			16: {
				appendList{V("16"), V(""), V("sixteen"), V("extra")},
				truncList(3),
				equalList{wantFS.Get(16).List()},
			},
			18: {
				appendList{V([]byte("18")), V([]byte(nil)), V([]byte("eighteen"))},
				equalList{wantFS.Get(18).List()},
			},
		},
		hasFields{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true, 12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true},
		equalMessage{want},
		clearFields{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
		equalMessage{empty},
	})

	// Test read-only operations on nil message.
	testMessage(t, nil, (*ListScalars)(nil), messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false},
		listFields{2: {lenList(0)}, 4: {lenList(0)}, 6: {lenList(0)}, 8: {lenList(0)}, 10: {lenList(0)}, 12: {lenList(0)}, 14: {lenList(0)}, 16: {lenList(0)}, 18: {lenList(0)}},
	})
}

type MapScalars struct {
	KeyBools   map[bool]string   `protobuf:"1"`
	KeyInt32s  map[int32]string  `protobuf:"2"`
	KeyInt64s  map[int64]string  `protobuf:"3"`
	KeyUint32s map[uint32]string `protobuf:"4"`
	KeyUint64s map[uint64]string `protobuf:"5"`
	KeyStrings map[string]string `protobuf:"6"`

	ValBools    map[string]bool    `protobuf:"7"`
	ValInt32s   map[string]int32   `protobuf:"8"`
	ValInt64s   map[string]int64   `protobuf:"9"`
	ValUint32s  map[string]uint32  `protobuf:"10"`
	ValUint64s  map[string]uint64  `protobuf:"11"`
	ValFloat32s map[string]float32 `protobuf:"12"`
	ValFloat64s map[string]float64 `protobuf:"13"`
	ValStrings  map[string]string  `protobuf:"14"`
	ValStringsA map[string][]byte  `protobuf:"15"`
	ValBytes    map[string][]byte  `protobuf:"16"`
	ValBytesA   map[string]string  `protobuf:"17"`

	MyStrings1 map[MyString]MyString `protobuf:"18"`
	MyStrings2 map[MyString]MyBytes  `protobuf:"19"`
	MyBytes1   map[MyString]MyBytes  `protobuf:"20"`
	MyBytes2   map[MyString]MyString `protobuf:"21"`

	MyStrings3 MapStrings `protobuf:"22"`
	MyStrings4 MapBytes   `protobuf:"23"`
	MyBytes3   MapBytes   `protobuf:"24"`
	MyBytes4   MapStrings `protobuf:"25"`
}

func mustMakeMapEntry(n pref.FieldNumber, keyKind, valKind pref.Kind) ptype.Field {
	return ptype.Field{
		Name:        pref.Name(fmt.Sprintf("f%d", n)),
		Number:      n,
		Cardinality: pref.Repeated,
		Kind:        pref.MessageKind,
		MessageType: mustMakeMessageDesc(ptype.StandaloneMessage{
			Syntax:   pref.Proto2,
			FullName: pref.FullName(fmt.Sprintf("MapScalars.F%dEntry", n)),
			Fields: []ptype.Field{
				{Name: "key", Number: 1, Cardinality: pref.Optional, Kind: keyKind},
				{Name: "value", Number: 2, Cardinality: pref.Optional, Kind: valKind},
			},
			Options:    &descriptorpb.MessageOptions{MapEntry: scalar.Bool(true)},
			IsMapEntry: true,
		}),
	}
}

var mapScalarsType = pimpl.MessageType{GoType: reflect.TypeOf(new(MapScalars)), PBType: ptype.GoMessage(
	mustMakeMessageDesc(ptype.StandaloneMessage{
		Syntax:   pref.Proto2,
		FullName: "MapScalars",
		Fields: []ptype.Field{
			mustMakeMapEntry(1, pref.BoolKind, pref.StringKind),
			mustMakeMapEntry(2, pref.Int32Kind, pref.StringKind),
			mustMakeMapEntry(3, pref.Int64Kind, pref.StringKind),
			mustMakeMapEntry(4, pref.Uint32Kind, pref.StringKind),
			mustMakeMapEntry(5, pref.Uint64Kind, pref.StringKind),
			mustMakeMapEntry(6, pref.StringKind, pref.StringKind),

			mustMakeMapEntry(7, pref.StringKind, pref.BoolKind),
			mustMakeMapEntry(8, pref.StringKind, pref.Int32Kind),
			mustMakeMapEntry(9, pref.StringKind, pref.Int64Kind),
			mustMakeMapEntry(10, pref.StringKind, pref.Uint32Kind),
			mustMakeMapEntry(11, pref.StringKind, pref.Uint64Kind),
			mustMakeMapEntry(12, pref.StringKind, pref.FloatKind),
			mustMakeMapEntry(13, pref.StringKind, pref.DoubleKind),
			mustMakeMapEntry(14, pref.StringKind, pref.StringKind),
			mustMakeMapEntry(15, pref.StringKind, pref.StringKind),
			mustMakeMapEntry(16, pref.StringKind, pref.BytesKind),
			mustMakeMapEntry(17, pref.StringKind, pref.BytesKind),

			mustMakeMapEntry(18, pref.StringKind, pref.StringKind),
			mustMakeMapEntry(19, pref.StringKind, pref.StringKind),
			mustMakeMapEntry(20, pref.StringKind, pref.BytesKind),
			mustMakeMapEntry(21, pref.StringKind, pref.BytesKind),

			mustMakeMapEntry(22, pref.StringKind, pref.StringKind),
			mustMakeMapEntry(23, pref.StringKind, pref.StringKind),
			mustMakeMapEntry(24, pref.StringKind, pref.BytesKind),
			mustMakeMapEntry(25, pref.StringKind, pref.BytesKind),
		},
	}),
	func(pref.MessageType) pref.Message {
		return new(MapScalars)
	},
)}

func (m *MapScalars) Type() pref.MessageType { return mapScalarsType.PBType }
func (m *MapScalars) KnownFields() pref.KnownFields {
	return mapScalarsType.MessageOf(m).KnownFields()
}
func (m *MapScalars) UnknownFields() pref.UnknownFields {
	return mapScalarsType.MessageOf(m).UnknownFields()
}
func (m *MapScalars) Interface() pref.ProtoMessage { return m }
func (m *MapScalars) ProtoReflect() pref.Message   { return m }

func TestMapScalars(t *testing.T) {
	empty := &MapScalars{}
	emptyFS := empty.KnownFields()

	want := &MapScalars{
		KeyBools:   map[bool]string{true: "true", false: "false"},
		KeyInt32s:  map[int32]string{0: "zero", -1: "one", 2: "two"},
		KeyInt64s:  map[int64]string{0: "zero", -10: "ten", 20: "twenty"},
		KeyUint32s: map[uint32]string{0: "zero", 1: "one", 2: "two"},
		KeyUint64s: map[uint64]string{0: "zero", 10: "ten", 20: "twenty"},
		KeyStrings: map[string]string{"": "", "foo": "bar"},

		ValBools:    map[string]bool{"true": true, "false": false},
		ValInt32s:   map[string]int32{"one": 1, "two": 2, "three": 3},
		ValInt64s:   map[string]int64{"ten": 10, "twenty": -20, "thirty": 30},
		ValUint32s:  map[string]uint32{"0x00": 0x00, "0xff": 0xff, "0xdead": 0xdead},
		ValUint64s:  map[string]uint64{"0x00": 0x00, "0xff": 0xff, "0xdead": 0xdead},
		ValFloat32s: map[string]float32{"nan": float32(math.NaN()), "pi": float32(math.Pi)},
		ValFloat64s: map[string]float64{"nan": float64(math.NaN()), "pi": float64(math.Pi)},
		ValStrings:  map[string]string{"s1": "s1", "s2": "s2"},
		ValStringsA: map[string][]byte{"s1": []byte("s1"), "s2": []byte("s2")},
		ValBytes:    map[string][]byte{"s1": []byte("s1"), "s2": []byte("s2")},
		ValBytesA:   map[string]string{"s1": "s1", "s2": "s2"},

		MyStrings1: map[MyString]MyString{"s1": "s1", "s2": "s2"},
		MyStrings2: map[MyString]MyBytes{"s1": []byte("s1"), "s2": []byte("s2")},
		MyBytes1:   map[MyString]MyBytes{"s1": []byte("s1"), "s2": []byte("s2")},
		MyBytes2:   map[MyString]MyString{"s1": "s1", "s2": "s2"},

		MyStrings3: MapStrings{"s1": "s1", "s2": "s2"},
		MyStrings4: MapBytes{"s1": []byte("s1"), "s2": []byte("s2")},
		MyBytes3:   MapBytes{"s1": []byte("s1"), "s2": []byte("s2")},
		MyBytes4:   MapStrings{"s1": "s1", "s2": "s2"},
	}
	wantFS := want.KnownFields()

	testMessage(t, nil, &MapScalars{}, messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false, 20: false, 21: false, 22: false, 23: false, 24: false, 25: false},
		getFields{1: emptyFS.Get(1), 3: emptyFS.Get(3), 5: emptyFS.Get(5), 7: emptyFS.Get(7), 9: emptyFS.Get(9), 11: emptyFS.Get(11), 13: emptyFS.Get(13), 15: emptyFS.Get(15), 17: emptyFS.Get(17), 19: emptyFS.Get(19), 21: emptyFS.Get(21), 23: emptyFS.Get(23), 25: emptyFS.Get(25)},
		setFields{1: wantFS.Get(1), 3: wantFS.Get(3), 5: wantFS.Get(5), 7: wantFS.Get(7), 9: wantFS.Get(9), 11: wantFS.Get(11), 13: wantFS.Get(13), 15: wantFS.Get(15), 17: wantFS.Get(17), 19: wantFS.Get(19), 21: wantFS.Get(21), 23: wantFS.Get(23), 25: wantFS.Get(25)},
		mapFields{
			2: {
				lenMap(0),
				hasMap{int32(0): false, int32(-1): false, int32(2): false},
				setMap{int32(0): V("zero")},
				lenMap(1),
				hasMap{int32(0): true, int32(-1): false, int32(2): false},
				setMap{int32(-1): V("one")},
				lenMap(2),
				hasMap{int32(0): true, int32(-1): true, int32(2): false},
				setMap{int32(2): V("two")},
				lenMap(3),
				hasMap{int32(0): true, int32(-1): true, int32(2): true},
			},
			4: {
				setMap{uint32(0): V("zero"), uint32(1): V("one"), uint32(2): V("two")},
				equalMap{wantFS.Get(4).Map()},
			},
			6: {
				clearMap{"noexist"},
				setMap{"foo": V("bar")},
				setMap{"": V("empty")},
				getMap{"": V("empty"), "foo": V("bar"), "noexist": V(nil)},
				setMap{"": V(""), "extra": V("extra")},
				clearMap{"extra", "noexist"},
			},
			8: {
				equalMap{emptyFS.Get(8).Map()},
				setMap{"one": V(int32(1)), "two": V(int32(2)), "three": V(int32(3))},
			},
			10: {
				setMap{"0x00": V(uint32(0x00)), "0xff": V(uint32(0xff)), "0xdead": V(uint32(0xdead))},
				lenMap(3),
				equalMap{wantFS.Get(10).Map()},
				getMap{"0x00": V(uint32(0x00)), "0xff": V(uint32(0xff)), "0xdead": V(uint32(0xdead)), "0xdeadbeef": V(nil)},
			},
			12: {
				setMap{"nan": V(float32(math.NaN())), "pi": V(float32(math.Pi)), "e": V(float32(math.E))},
				clearMap{"e", "phi"},
				rangeMap{"nan": V(float32(math.NaN())), "pi": V(float32(math.Pi))},
			},
			14: {
				equalMap{emptyFS.Get(14).Map()},
				setMap{"s1": V("s1"), "s2": V("s2")},
			},
			16: {
				setMap{"s1": V([]byte("s1")), "s2": V([]byte("s2"))},
				equalMap{wantFS.Get(16).Map()},
			},
			18: {
				hasMap{"s1": false, "s2": false, "s3": false},
				setMap{"s1": V("s1"), "s2": V("s2")},
				hasMap{"s1": true, "s2": true, "s3": false},
			},
			20: {
				equalMap{emptyFS.Get(20).Map()},
				setMap{"s1": V([]byte("s1")), "s2": V([]byte("s2"))},
			},
			22: {
				rangeMap{},
				setMap{"s1": V("s1"), "s2": V("s2")},
				rangeMap{"s1": V("s1"), "s2": V("s2")},
				lenMap(2),
			},
			24: {
				setMap{"s1": V([]byte("s1")), "s2": V([]byte("s2"))},
				equalMap{wantFS.Get(24).Map()},
			},
		},
		hasFields{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true, 12: true, 13: true, 14: true, 15: true, 16: true, 17: true, 18: true, 19: true, 20: true, 21: true, 22: true, 23: true, 24: true, 25: true},
		equalMessage{want},
		clearFields{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25},
		equalMessage{empty},
	})

	// Test read-only operations on nil message.
	testMessage(t, nil, (*MapScalars)(nil), messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: false, 14: false, 15: false, 16: false, 17: false, 18: false, 19: false, 20: false, 21: false, 22: false, 23: false, 24: false, 25: false},
		mapFields{2: {lenMap(0)}, 4: {lenMap(0)}, 6: {lenMap(0)}, 8: {lenMap(0)}, 10: {lenMap(0)}, 12: {lenMap(0)}, 14: {lenMap(0)}, 16: {lenMap(0)}, 18: {lenMap(0)}, 20: {lenMap(0)}, 22: {lenMap(0)}, 24: {lenMap(0)}},
	})
}

type OneofScalars struct {
	Union isOneofScalars_Union `protobuf_oneof:"union"`
}

var oneofScalarsType = pimpl.MessageType{GoType: reflect.TypeOf(new(OneofScalars)), PBType: ptype.GoMessage(
	mustMakeMessageDesc(ptype.StandaloneMessage{
		Syntax:   pref.Proto2,
		FullName: "OneofScalars",
		Fields: []ptype.Field{
			{Name: "f1", Number: 1, Cardinality: pref.Optional, Kind: pref.BoolKind, Default: V(bool(true)), OneofName: "union"},
			{Name: "f2", Number: 2, Cardinality: pref.Optional, Kind: pref.Int32Kind, Default: V(int32(2)), OneofName: "union"},
			{Name: "f3", Number: 3, Cardinality: pref.Optional, Kind: pref.Int64Kind, Default: V(int64(3)), OneofName: "union"},
			{Name: "f4", Number: 4, Cardinality: pref.Optional, Kind: pref.Uint32Kind, Default: V(uint32(4)), OneofName: "union"},
			{Name: "f5", Number: 5, Cardinality: pref.Optional, Kind: pref.Uint64Kind, Default: V(uint64(5)), OneofName: "union"},
			{Name: "f6", Number: 6, Cardinality: pref.Optional, Kind: pref.FloatKind, Default: V(float32(6)), OneofName: "union"},
			{Name: "f7", Number: 7, Cardinality: pref.Optional, Kind: pref.DoubleKind, Default: V(float64(7)), OneofName: "union"},
			{Name: "f8", Number: 8, Cardinality: pref.Optional, Kind: pref.StringKind, Default: V(string("8")), OneofName: "union"},
			{Name: "f9", Number: 9, Cardinality: pref.Optional, Kind: pref.StringKind, Default: V(string("9")), OneofName: "union"},
			{Name: "f10", Number: 10, Cardinality: pref.Optional, Kind: pref.StringKind, Default: V(string("10")), OneofName: "union"},
			{Name: "f11", Number: 11, Cardinality: pref.Optional, Kind: pref.BytesKind, Default: V([]byte("11")), OneofName: "union"},
			{Name: "f12", Number: 12, Cardinality: pref.Optional, Kind: pref.BytesKind, Default: V([]byte("12")), OneofName: "union"},
			{Name: "f13", Number: 13, Cardinality: pref.Optional, Kind: pref.BytesKind, Default: V([]byte("13")), OneofName: "union"},
		},
		Oneofs: []ptype.Oneof{{Name: "union"}},
	}),
	func(pref.MessageType) pref.Message {
		return new(OneofScalars)
	},
)}

func (m *OneofScalars) Type() pref.MessageType { return oneofScalarsType.PBType }
func (m *OneofScalars) KnownFields() pref.KnownFields {
	return oneofScalarsType.MessageOf(m).KnownFields()
}
func (m *OneofScalars) UnknownFields() pref.UnknownFields {
	return oneofScalarsType.MessageOf(m).UnknownFields()
}
func (m *OneofScalars) Interface() pref.ProtoMessage { return m }
func (m *OneofScalars) ProtoReflect() pref.Message   { return m }

func (*OneofScalars) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*OneofScalars_Bool)(nil),
		(*OneofScalars_Int32)(nil),
		(*OneofScalars_Int64)(nil),
		(*OneofScalars_Uint32)(nil),
		(*OneofScalars_Uint64)(nil),
		(*OneofScalars_Float32)(nil),
		(*OneofScalars_Float64)(nil),
		(*OneofScalars_String)(nil),
		(*OneofScalars_StringA)(nil),
		(*OneofScalars_StringB)(nil),
		(*OneofScalars_Bytes)(nil),
		(*OneofScalars_BytesA)(nil),
		(*OneofScalars_BytesB)(nil),
	}
}

type (
	isOneofScalars_Union interface {
		isOneofScalars_Union()
	}
	OneofScalars_Bool struct {
		Bool bool `protobuf:"1"`
	}
	OneofScalars_Int32 struct {
		Int32 MyInt32 `protobuf:"2"`
	}
	OneofScalars_Int64 struct {
		Int64 int64 `protobuf:"3"`
	}
	OneofScalars_Uint32 struct {
		Uint32 MyUint32 `protobuf:"4"`
	}
	OneofScalars_Uint64 struct {
		Uint64 uint64 `protobuf:"5"`
	}
	OneofScalars_Float32 struct {
		Float32 MyFloat32 `protobuf:"6"`
	}
	OneofScalars_Float64 struct {
		Float64 float64 `protobuf:"7"`
	}
	OneofScalars_String struct {
		String string `protobuf:"8"`
	}
	OneofScalars_StringA struct {
		StringA []byte `protobuf:"9"`
	}
	OneofScalars_StringB struct {
		StringB MyString `protobuf:"10"`
	}
	OneofScalars_Bytes struct {
		Bytes []byte `protobuf:"11"`
	}
	OneofScalars_BytesA struct {
		BytesA string `protobuf:"12"`
	}
	OneofScalars_BytesB struct {
		BytesB MyBytes `protobuf:"13"`
	}
)

func (*OneofScalars_Bool) isOneofScalars_Union()    {}
func (*OneofScalars_Int32) isOneofScalars_Union()   {}
func (*OneofScalars_Int64) isOneofScalars_Union()   {}
func (*OneofScalars_Uint32) isOneofScalars_Union()  {}
func (*OneofScalars_Uint64) isOneofScalars_Union()  {}
func (*OneofScalars_Float32) isOneofScalars_Union() {}
func (*OneofScalars_Float64) isOneofScalars_Union() {}
func (*OneofScalars_String) isOneofScalars_Union()  {}
func (*OneofScalars_StringA) isOneofScalars_Union() {}
func (*OneofScalars_StringB) isOneofScalars_Union() {}
func (*OneofScalars_Bytes) isOneofScalars_Union()   {}
func (*OneofScalars_BytesA) isOneofScalars_Union()  {}
func (*OneofScalars_BytesB) isOneofScalars_Union()  {}

func TestOneofs(t *testing.T) {
	empty := &OneofScalars{}
	want1 := &OneofScalars{Union: &OneofScalars_Bool{true}}
	want2 := &OneofScalars{Union: &OneofScalars_Int32{20}}
	want3 := &OneofScalars{Union: &OneofScalars_Int64{30}}
	want4 := &OneofScalars{Union: &OneofScalars_Uint32{40}}
	want5 := &OneofScalars{Union: &OneofScalars_Uint64{50}}
	want6 := &OneofScalars{Union: &OneofScalars_Float32{60}}
	want7 := &OneofScalars{Union: &OneofScalars_Float64{70}}
	want8 := &OneofScalars{Union: &OneofScalars_String{string("80")}}
	want9 := &OneofScalars{Union: &OneofScalars_StringA{[]byte("90")}}
	want10 := &OneofScalars{Union: &OneofScalars_StringB{MyString("100")}}
	want11 := &OneofScalars{Union: &OneofScalars_Bytes{[]byte("110")}}
	want12 := &OneofScalars{Union: &OneofScalars_BytesA{string("120")}}
	want13 := &OneofScalars{Union: &OneofScalars_BytesB{MyBytes("130")}}

	testMessage(t, nil, &OneofScalars{}, messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: false},
		getFields{1: V(bool(true)), 2: V(int32(2)), 3: V(int64(3)), 4: V(uint32(4)), 5: V(uint64(5)), 6: V(float32(6)), 7: V(float64(7)), 8: V(string("8")), 9: V(string("9")), 10: V(string("10")), 11: V([]byte("11")), 12: V([]byte("12")), 13: V([]byte("13"))},

		setFields{1: V(bool(true))}, hasFields{1: true}, equalMessage{want1},
		setFields{2: V(int32(20))}, hasFields{2: true}, equalMessage{want2},
		setFields{3: V(int64(30))}, hasFields{3: true}, equalMessage{want3},
		setFields{4: V(uint32(40))}, hasFields{4: true}, equalMessage{want4},
		setFields{5: V(uint64(50))}, hasFields{5: true}, equalMessage{want5},
		setFields{6: V(float32(60))}, hasFields{6: true}, equalMessage{want6},
		setFields{7: V(float64(70))}, hasFields{7: true}, equalMessage{want7},
		setFields{8: V(string("80"))}, hasFields{8: true}, equalMessage{want8},
		setFields{9: V(string("90"))}, hasFields{9: true}, equalMessage{want9},
		setFields{10: V(string("100"))}, hasFields{10: true}, equalMessage{want10},
		setFields{11: V([]byte("110"))}, hasFields{11: true}, equalMessage{want11},
		setFields{12: V([]byte("120"))}, hasFields{12: true}, equalMessage{want12},
		setFields{13: V([]byte("130"))}, hasFields{13: true}, equalMessage{want13},

		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: true},
		getFields{1: V(bool(true)), 2: V(int32(2)), 3: V(int64(3)), 4: V(uint32(4)), 5: V(uint64(5)), 6: V(float32(6)), 7: V(float64(7)), 8: V(string("8")), 9: V(string("9")), 10: V(string("10")), 11: V([]byte("11")), 12: V([]byte("12")), 13: V([]byte("130"))},
		clearFields{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		equalMessage{want13},
		clearFields{13},
		equalMessage{empty},
	})

	// Test read-only operations on nil message.
	testMessage(t, nil, (*OneofScalars)(nil), messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false, 13: false},
		getFields{1: V(bool(true)), 2: V(int32(2)), 3: V(int64(3)), 4: V(uint32(4)), 5: V(uint64(5)), 6: V(float32(6)), 7: V(float64(7)), 8: V(string("8")), 9: V(string("9")), 10: V(string("10")), 11: V([]byte("11")), 12: V([]byte("12")), 13: V([]byte("13"))},
	})
}

type EnumProto2 int32

var enumProto2Type = ptype.GoEnum(
	mustMakeEnumDesc(ptype.StandaloneEnum{
		Syntax:   pref.Proto2,
		FullName: "EnumProto2",
		Values:   []ptype.EnumValue{{Name: "DEAD", Number: 0xdead}, {Name: "BEEF", Number: 0xbeef}},
	}),
	func(_ pref.EnumType, n pref.EnumNumber) pref.Enum {
		return EnumProto2(n)
	},
)

func (e EnumProto2) Enum() *EnumProto2       { return &e }
func (e EnumProto2) Type() pref.EnumType     { return enumProto2Type }
func (e EnumProto2) Number() pref.EnumNumber { return pref.EnumNumber(e) }

type EnumProto3 int32

var enumProto3Type = ptype.GoEnum(
	mustMakeEnumDesc(ptype.StandaloneEnum{
		Syntax:   pref.Proto3,
		FullName: "EnumProto3",
		Values:   []ptype.EnumValue{{Name: "ALPHA", Number: 0}, {Name: "BRAVO", Number: 1}},
	}),
	func(_ pref.EnumType, n pref.EnumNumber) pref.Enum {
		return EnumProto3(n)
	},
)

func (e EnumProto3) Enum() *EnumProto3       { return &e }
func (e EnumProto3) Type() pref.EnumType     { return enumProto3Type }
func (e EnumProto3) Number() pref.EnumNumber { return pref.EnumNumber(e) }

type EnumMessages struct {
	EnumP2        *EnumProto2              `protobuf:"1"`
	EnumP3        *EnumProto3              `protobuf:"2"`
	MessageLegacy *proto2_20180125.Message `protobuf:"3"`
	MessageCycle  *EnumMessages            `protobuf:"4"`
	EnumList      []EnumProto2             `protobuf:"5"`
	MessageList   []*ScalarProto2          `protobuf:"6"`
	EnumMap       map[string]EnumProto3    `protobuf:"7"`
	MessageMap    map[string]*ScalarProto3 `protobuf:"8"`
	Union         isEnumMessages_Union     `protobuf_oneof:"union"`
}

var enumMessagesType = pimpl.MessageType{GoType: reflect.TypeOf(new(EnumMessages)), PBType: ptype.GoMessage(
	mustMakeMessageDesc(ptype.StandaloneMessage{
		Syntax:   pref.Proto2,
		FullName: "EnumMessages",
		Fields: []ptype.Field{
			{Name: "f1", Number: 1, Cardinality: pref.Optional, Kind: pref.EnumKind, Default: V("BEEF"), EnumType: enumProto2Type},
			{Name: "f2", Number: 2, Cardinality: pref.Optional, Kind: pref.EnumKind, Default: V("BRAVO"), EnumType: enumProto3Type},
			{Name: "f3", Number: 3, Cardinality: pref.Optional, Kind: pref.MessageKind, MessageType: pimpl.Export{}.MessageOf(new(proto2_20180125.Message)).Type()},
			{Name: "f4", Number: 4, Cardinality: pref.Optional, Kind: pref.MessageKind, MessageType: ptype.PlaceholderMessage("EnumMessages")},
			{Name: "f5", Number: 5, Cardinality: pref.Repeated, Kind: pref.EnumKind, EnumType: enumProto2Type},
			{Name: "f6", Number: 6, Cardinality: pref.Repeated, Kind: pref.MessageKind, MessageType: scalarProto2Type.PBType},
			{Name: "f7", Number: 7, Cardinality: pref.Repeated, Kind: pref.MessageKind, MessageType: enumMapDesc},
			{Name: "f8", Number: 8, Cardinality: pref.Repeated, Kind: pref.MessageKind, MessageType: messageMapDesc},
			{Name: "f9", Number: 9, Cardinality: pref.Optional, Kind: pref.EnumKind, Default: V("BEEF"), OneofName: "union", EnumType: enumProto2Type},
			{Name: "f10", Number: 10, Cardinality: pref.Optional, Kind: pref.EnumKind, Default: V("BRAVO"), OneofName: "union", EnumType: enumProto3Type},
			{Name: "f11", Number: 11, Cardinality: pref.Optional, Kind: pref.MessageKind, OneofName: "union", MessageType: scalarProto2Type.PBType},
			{Name: "f12", Number: 12, Cardinality: pref.Optional, Kind: pref.MessageKind, OneofName: "union", MessageType: scalarProto3Type.PBType},
		},
		Oneofs: []ptype.Oneof{{Name: "union"}},
	}),
	func(pref.MessageType) pref.Message {
		return new(EnumMessages)
	},
)}

var enumMapDesc = mustMakeMessageDesc(ptype.StandaloneMessage{
	Syntax:   pref.Proto2,
	FullName: "EnumMessages.F7Entry",
	Fields: []ptype.Field{
		{Name: "key", Number: 1, Cardinality: pref.Optional, Kind: pref.StringKind},
		{Name: "value", Number: 2, Cardinality: pref.Optional, Kind: pref.EnumKind, EnumType: enumProto3Type},
	},
	Options:    &descriptorpb.MessageOptions{MapEntry: scalar.Bool(true)},
	IsMapEntry: true,
})

var messageMapDesc = mustMakeMessageDesc(ptype.StandaloneMessage{
	Syntax:   pref.Proto2,
	FullName: "EnumMessages.F8Entry",
	Fields: []ptype.Field{
		{Name: "key", Number: 1, Cardinality: pref.Optional, Kind: pref.StringKind},
		{Name: "value", Number: 2, Cardinality: pref.Optional, Kind: pref.MessageKind, MessageType: scalarProto3Type.PBType},
	},
	Options:    &descriptorpb.MessageOptions{MapEntry: scalar.Bool(true)},
	IsMapEntry: true,
})

func (m *EnumMessages) Type() pref.MessageType { return enumMessagesType.PBType }
func (m *EnumMessages) KnownFields() pref.KnownFields {
	return enumMessagesType.MessageOf(m).KnownFields()
}
func (m *EnumMessages) UnknownFields() pref.UnknownFields {
	return enumMessagesType.MessageOf(m).UnknownFields()
}
func (m *EnumMessages) Interface() pref.ProtoMessage { return m }
func (m *EnumMessages) ProtoReflect() pref.Message   { return m }

func (*EnumMessages) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*EnumMessages_OneofE2)(nil),
		(*EnumMessages_OneofE3)(nil),
		(*EnumMessages_OneofM2)(nil),
		(*EnumMessages_OneofM3)(nil),
	}
}

type (
	isEnumMessages_Union interface {
		isEnumMessages_Union()
	}
	EnumMessages_OneofE2 struct {
		OneofE2 EnumProto2 `protobuf:"9"`
	}
	EnumMessages_OneofE3 struct {
		OneofE3 EnumProto3 `protobuf:"10"`
	}
	EnumMessages_OneofM2 struct {
		OneofM2 *ScalarProto2 `protobuf:"11"`
	}
	EnumMessages_OneofM3 struct {
		OneofM3 *ScalarProto3 `protobuf:"12"`
	}
)

func (*EnumMessages_OneofE2) isEnumMessages_Union() {}
func (*EnumMessages_OneofE3) isEnumMessages_Union() {}
func (*EnumMessages_OneofM2) isEnumMessages_Union() {}
func (*EnumMessages_OneofM3) isEnumMessages_Union() {}

func TestEnumMessages(t *testing.T) {
	wantL := pimpl.Export{}.MessageOf(&proto2_20180125.Message{OptionalFloat: scalar.Float32(math.E)})
	wantM := &EnumMessages{EnumP2: EnumProto2(1234).Enum()}
	wantM2a := &ScalarProto2{Float32: scalar.Float32(math.Pi)}
	wantM2b := &ScalarProto2{Float32: scalar.Float32(math.Phi)}
	wantM3a := &ScalarProto3{Float32: math.Pi}
	wantM3b := &ScalarProto3{Float32: math.Ln2}

	wantList5 := (&EnumMessages{EnumList: []EnumProto2{333, 222}}).KnownFields().Get(5)
	wantList6 := (&EnumMessages{MessageList: []*ScalarProto2{wantM2a, wantM2b}}).KnownFields().Get(6)

	wantMap7 := (&EnumMessages{EnumMap: map[string]EnumProto3{"one": 1, "two": 2}}).KnownFields().Get(7)
	wantMap8 := (&EnumMessages{MessageMap: map[string]*ScalarProto3{"pi": wantM3a, "ln2": wantM3b}}).KnownFields().Get(8)

	testMessage(t, nil, &EnumMessages{}, messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false},
		getFields{1: VE(0xbeef), 2: VE(1), 3: V(nil), 4: V(nil), 9: VE(0xbeef), 10: VE(1)},

		// Test singular enums.
		setFields{1: VE(0xdead), 2: VE(0)},
		getFields{1: VE(0xdead), 2: VE(0)},
		hasFields{1: true, 2: true},

		// Test singular messages.
		messageFields{3: messageOps{setFields{109: V(float32(math.E))}}},
		messageFields{4: messageOps{setFields{1: VE(1234)}}},
		getFields{3: V(wantL), 4: V(wantM)},
		clearFields{3, 4},
		hasFields{3: false, 4: false},
		setFields{3: V(wantL), 4: V(wantM)},
		hasFields{3: true, 4: true},

		// Test list of enums and messages.
		listFields{
			5: listOps{
				appendList{VE(111), VE(222)},
				setList{0: VE(333)},
				getList{0: VE(333), 1: VE(222)},
				lenList(2),
			},
			6: listOps{
				appendMessageList{setFields{4: V(uint32(1e6))}},
				appendMessageList{setFields{6: V(float32(math.Phi))}},
				setList{0: V(wantM2a)},
				getList{0: V(wantM2a), 1: V(wantM2b)},
			},
		},
		getFields{5: wantList5, 6: wantList6},
		hasFields{5: true, 6: true},
		listFields{5: listOps{truncList(0)}},
		hasFields{5: false, 6: true},

		// Test maps of enums and messages.
		mapFields{
			7: mapOps{
				setMap{"one": VE(1), "two": VE(2)},
				hasMap{"one": true, "two": true, "three": false},
				lenMap(2),
			},
			8: mapOps{
				messageMap{"pi": messageOps{setFields{6: V(float32(math.Pi))}}},
				setMap{"ln2": V(wantM3b)},
				getMap{"pi": V(wantM3a), "ln2": V(wantM3b), "none": V(nil)},
				lenMap(2),
			},
		},
		getFields{7: wantMap7, 8: wantMap8},
		hasFields{7: true, 8: true},
		mapFields{8: mapOps{clearMap{"pi", "ln2", "none"}}},
		hasFields{7: true, 8: false},

		// Test oneofs of enums and messages.
		setFields{9: VE(0xdead)},
		hasFields{1: true, 2: true, 9: true, 10: false, 11: false, 12: false},
		setFields{10: VE(0)},
		hasFields{1: true, 2: true, 9: false, 10: true, 11: false, 12: false},
		messageFields{11: messageOps{setFields{6: V(float32(math.Pi))}}},
		getFields{11: V(wantM2a)},
		hasFields{1: true, 2: true, 9: false, 10: false, 11: true, 12: false},
		messageFields{12: messageOps{setFields{6: V(float32(math.Pi))}}},
		getFields{12: V(wantM3a)},
		hasFields{1: true, 2: true, 9: false, 10: false, 11: false, 12: true},

		// Check entire message.
		rangeFields{1: VE(0xdead), 2: VE(0), 3: V(wantL), 4: V(wantM), 6: wantList6, 7: wantMap7, 12: V(wantM3a)},
		equalMessage{&EnumMessages{
			EnumP2:        EnumProto2(0xdead).Enum(),
			EnumP3:        EnumProto3(0).Enum(),
			MessageLegacy: &proto2_20180125.Message{OptionalFloat: scalar.Float32(math.E)},
			MessageCycle:  wantM,
			MessageList:   []*ScalarProto2{wantM2a, wantM2b},
			EnumMap:       map[string]EnumProto3{"one": 1, "two": 2},
			Union:         &EnumMessages_OneofM3{wantM3a},
		}},
		clearFields{1, 2, 3, 4, 6, 7, 12},
		equalMessage{&EnumMessages{}},
	})

	// Test read-only operations on nil message.
	testMessage(t, nil, (*EnumMessages)(nil), messageOps{
		hasFields{1: false, 2: false, 3: false, 4: false, 5: false, 6: false, 7: false, 8: false, 9: false, 10: false, 11: false, 12: false},
		getFields{1: VE(0xbeef), 2: VE(1), 3: V(nil), 4: V(nil), 9: VE(0xbeef), 10: VE(1), 11: V(nil), 12: V(nil)},
		listFields{5: {lenList(0)}, 6: {lenList(0)}},
		mapFields{7: {lenMap(0)}, 8: {lenMap(0)}},
	})
}

var cmpOpts = cmp.Options{
	cmp.Comparer(func(x, y *proto2_20180125.Message) bool {
		return protoV1.Equal(x, y)
	}),
	cmp.Transformer("UnwrapValue", func(pv pref.Value) interface{} {
		return pv.Interface()
	}),
	cmp.Transformer("UnwrapGeneric", func(x pvalue.Unwrapper) interface{} {
		return x.ProtoUnwrap()
	}),
	cmpopts.EquateNaNs(),
	cmpopts.EquateEmpty(),
}

func testMessage(t *testing.T, p path, m pref.Message, tt messageOps) {
	fs := m.KnownFields()
	for i, op := range tt {
		p.Push(i)
		switch op := op.(type) {
		case equalMessage:
			if diff := cmp.Diff(op.Message, m, cmpOpts); diff != "" {
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
			for _, n := range op {
				fs.Clear(n)
			}
		case messageFields:
			for n, tt := range op {
				p.Push(int(n))
				if !fs.Has(n) {
					fs.Set(n, V(fs.NewMessage(n)))
				}
				testMessage(t, p, fs.Get(n).Message(), tt)
				p.Pop()
			}
		case listFields:
			for n, tt := range op {
				p.Push(int(n))
				testLists(t, p, fs.Get(n).List(), tt)
				p.Pop()
			}
		case mapFields:
			for n, tt := range op {
				p.Push(int(n))
				testMaps(t, p, fs.Get(n).Map(), tt)
				p.Pop()
			}
		case rangeFields:
			got := map[pref.FieldNumber]pref.Value{}
			want := map[pref.FieldNumber]pref.Value(op)
			fs.Range(func(n pref.FieldNumber, v pref.Value) bool {
				got[n] = v
				return true
			})
			if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
				t.Errorf("operation %v, KnownFields.Range mismatch (-want, +got):\n%s", p, diff)
			}
		default:
			t.Fatalf("operation %v, invalid operation: %T", p, op)
		}
		p.Pop()
	}
}

func testLists(t *testing.T, p path, v pref.List, tt listOps) {
	for i, op := range tt {
		p.Push(i)
		switch op := op.(type) {
		case equalList:
			if diff := cmp.Diff(op.List, v, cmpOpts); diff != "" {
				t.Errorf("operation %v, list mismatch (-want, +got):\n%s", p, diff)
			}
		case lenList:
			if got, want := v.Len(), int(op); got != want {
				t.Errorf("operation %v, List.Len = %d, want %d", p, got, want)
			}
		case getList:
			got := map[int]pref.Value{}
			want := map[int]pref.Value(op)
			for n := range want {
				got[n] = v.Get(n)
			}
			if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
				t.Errorf("operation %v, List.Get mismatch (-want, +got):\n%s", p, diff)
			}
		case setList:
			for n, e := range op {
				v.Set(n, e)
			}
		case appendList:
			for _, e := range op {
				v.Append(e)
			}
		case appendMessageList:
			m := v.NewMessage()
			v.Append(V(m))
			testMessage(t, p, m, messageOps(op))
		case truncList:
			v.Truncate(int(op))
		default:
			t.Fatalf("operation %v, invalid operation: %T", p, op)
		}
		p.Pop()
	}
}

func testMaps(t *testing.T, p path, m pref.Map, tt mapOps) {
	for i, op := range tt {
		p.Push(i)
		switch op := op.(type) {
		case equalMap:
			if diff := cmp.Diff(op.Map, m, cmpOpts); diff != "" {
				t.Errorf("operation %v, map mismatch (-want, +got):\n%s", p, diff)
			}
		case lenMap:
			if got, want := m.Len(), int(op); got != want {
				t.Errorf("operation %v, Map.Len = %d, want %d", p, got, want)
			}
		case hasMap:
			got := map[interface{}]bool{}
			want := map[interface{}]bool(op)
			for k := range want {
				got[k] = m.Has(V(k).MapKey())
			}
			if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
				t.Errorf("operation %v, Map.Has mismatch (-want, +got):\n%s", p, diff)
			}
		case getMap:
			got := map[interface{}]pref.Value{}
			want := map[interface{}]pref.Value(op)
			for k := range want {
				got[k] = m.Get(V(k).MapKey())
			}
			if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
				t.Errorf("operation %v, Map.Get mismatch (-want, +got):\n%s", p, diff)
			}
		case setMap:
			for k, v := range op {
				m.Set(V(k).MapKey(), v)
			}
		case clearMap:
			for _, k := range op {
				m.Clear(V(k).MapKey())
			}
		case messageMap:
			for k, tt := range op {
				mk := V(k).MapKey()
				if !m.Has(mk) {
					m.Set(mk, V(m.NewMessage()))
				}
				testMessage(t, p, m.Get(mk).Message(), tt)
			}
		case rangeMap:
			got := map[interface{}]pref.Value{}
			want := map[interface{}]pref.Value(op)
			m.Range(func(k pref.MapKey, v pref.Value) bool {
				got[k.Interface()] = v
				return true
			})
			if diff := cmp.Diff(want, got, cmpOpts); diff != "" {
				t.Errorf("operation %v, Map.Range mismatch (-want, +got):\n%s", p, diff)
			}
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
