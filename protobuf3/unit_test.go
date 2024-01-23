// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2016 Mist Systems. All rights reserved.
//
// This code is derived from earlier code which was itself:
//
// Copyright 2014 The Go Authors.  All rights reserved.
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

package protobuf3_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/mistsys/protobuf3/protobuf3"
	"github.com/mistsys/protobuf3/protobuf3/internal/unit_tests/duration"
	"github.com/mistsys/protobuf3/protobuf3/internal/unit_tests/proto"
	pb3 "github.com/mistsys/protobuf3/protobuf3/internal/unit_tests/proto3_proto"
	"github.com/mistsys/protobuf3/protobuf3/internal/unit_tests/timestamp"
)

func TestProto3ZeroValues(t *testing.T) {
	protobuf3.XXXHack = true // needed b/c of pb3.Message.Proto2Field.XXX_unrecognized
	defer func() { protobuf3.XXXHack = false }()

	tests := []struct {
		desc string
		m    proto.Message
	}{
		{"zero message", &pb3.Message{}},
		{"empty bytes field", &pb3.Message{Data: []byte{}}},
	}
	for _, test := range tests {
		b, err := protobuf3.Marshal(test.m)
		if err != nil {
			t.Errorf("ERROR %s: protobuf3.Marshal: %v", test.desc, err)
			continue
		}
		if len(b) > 0 {
			t.Errorf("ERROR %s: Encoding is non-empty: %q", test.desc, b)
		}
	}
}

func TestRoundTripProto3(t *testing.T) {
	protobuf3.XXXHack = true // needed b/c of pb3.Message.Proto2Field.XXX_unrecognized
	defer func() { protobuf3.XXXHack = false }()

	m := &pb3.Message{
		Name:         "David",          // (2 | 1<<3): 0x0a 0x05 "David"
		Hilarity:     pb3.Message_PUNS, // (0 | 2<<3): 0x10 0x01
		HeightInCm:   178,              // (0 | 3<<3): 0x18 0xb2 0x01
		Data:         []byte("roboto"), // (2 | 4<<3): 0x20 0x06 "roboto"
		ResultCount:  47,               // (0 | 7<<3): 0x38 0x2f
		TrueScotsman: true,             // (0 | 8<<3): 0x40 0x01
		Score:        8.1,              // (5 | 9<<3): 0x4d <8.1>

		Key: []uint64{1, 0xdeadbeef},
		Nested: &pb3.Nested{
			Bunny: "Monty",
		},
	}
	t.Logf(" m: %v", m)

	b, err := protobuf3.Marshal(m)
	if err != nil {
		t.Fatalf("protobuf3.Marshal: %v", err)
	}
	t.Logf(" b: % x", b)

	// also log the correct answer
	c, err := proto.Marshal(m)
	if err != nil {
		t.Fatalf("protobuf3.Marshal: %v", err)
	}
	t.Logf(" c: % x", c)

	m2 := new(pb3.Message)
	if err := proto.Unmarshal(b, m2); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}
	t.Logf("m2: %v", m2)

	if !proto.Equal(m, m2) {
		t.Errorf("ERROR proto.Equal returned false:\n m: %v\nm2: %v", m, m2)
	}
}

//----------------------------------------------------------------------------------------------

// test message with fixed-sized encoded fields
type FixedMsg struct {
	i32 int32   `protobuf:"fixed32,1"`
	u32 uint32  `protobuf:"fixed32,2"`
	i64 int64   `protobuf:"fixed64,3"`
	u64 uint64  `protobuf:"fixed64,4"`
	f32 float32 `protobuf:"fixed32,8"`
	f64 float64 `protobuf:"fixed64,9"`

	pi32 *int32   `protobuf:"fixed32,11"`
	pu32 *uint32  `protobuf:"fixed32,12"`
	pi64 *int64   `protobuf:"fixed64,13"`
	pu64 *uint64  `protobuf:"fixed64,14"`
	pf32 *float32 `protobuf:"fixed32,18"`
	pf64 *float64 `protobuf:"fixed64,19"`

	si32 []int32   `protobuf:"fixed32,21,packed"`
	su32 []uint32  `protobuf:"fixed32,22,packed"`
	si64 []int64   `protobuf:"fixed64,23,packed"`
	su64 []uint64  `protobuf:"fixed64,24,packed"`
	sf32 []float32 `protobuf:"fixed32,28,packed"`
	sf64 []float64 `protobuf:"fixed64,29,packed"`
}

func (*FixedMsg) ProtoMessage()    {}
func (m *FixedMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *FixedMsg) Reset()         { *m = FixedMsg{} }

// fixed size array fields (split out because regular proto.Marshal can't deal with them)
type FixedArrayMsg struct {
	ai32 [1]int32   `protobuf:"fixed32,21,packed"`
	au32 [2]uint32  `protobuf:"fixed32,22,packed"`
	ai64 [3]int64   `protobuf:"fixed64,23,packed"`
	au64 [4]uint64  `protobuf:"fixed64,24,packed"`
	af32 [5]float32 `protobuf:"fixed32,28,packed"`
	af64 [6]float64 `protobuf:"fixed64,29,packed"`
}

// test message with varint encoded fields
type VarMsg struct {
	i32 int32  `protobuf:"varint,1"`
	u32 uint32 `protobuf:"varint,2"`
	i64 int64  `protobuf:"varint,3"`
	u64 uint64 `protobuf:"varint,4"`
	b   bool   `protobuf:"varint,5"`

	pi32 *int32  `protobuf:"varint,11"`
	pu32 *uint32 `protobuf:"varint,12"`
	pi64 *int64  `protobuf:"varint,13"`
	pu64 *uint64 `protobuf:"varint,14"`
	pb   *bool   `protobuf:"varint,15"`

	si32 []int32  `protobuf:"varint,21,packed"`
	su32 []uint32 `protobuf:"varint,22,packed"`
	si64 []int64  `protobuf:"varint,23,packed"`
	su64 []uint64 `protobuf:"varint,24,packed"`
	sb   []bool   `protobuf:"varint,25,packed"`
}

func (*VarMsg) ProtoMessage()    {}
func (m *VarMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *VarMsg) Reset()         { *m = VarMsg{} }

type VarArrayMsg struct {
	ai32 [1]int32  `protobuf:"varint,21,packed"`
	au32 [2]uint32 `protobuf:"varint,22,packed"`
	ai64 [3]int64  `protobuf:"varint,23,packed"`
	au64 [4]uint64 `protobuf:"varint,24,packed"`
	ab   [5]bool   `protobuf:"varint,25,packed"`
}

// test message with zigzag encodings
type ZigZagMsg struct {
	i32 int32 `protobuf:"zigzag32,1"`
	i64 int64 `protobuf:"zigzag64,2"`

	pi32 *int32 `protobuf:"zigzag32,11"`
	pi64 *int64 `protobuf:"zigzag64,12"`

	si32 []int32 `protobuf:"zigzag32,21,packed"`
	si64 []int64 `protobuf:"zigzag64,22,packed"`
}

func (*ZigZagMsg) ProtoMessage()    {}
func (m *ZigZagMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *ZigZagMsg) Reset()         { *m = ZigZagMsg{} }

type ZigZagArrayMsg struct {
	ai32 [1]int32 `protobuf:"zigzag32,21,packed"`
	ai64 [2]int64 `protobuf:"zigzag64,22,packed"`
}

// test message with bytes encoded fields
type BytesMsg struct {
	s  string   `protobuf:"bytes,1"`
	ps *string  `protobuf:"bytes,2"`
	ss []string `protobuf:"bytes,3,packed"`

	sb []byte `protobuf:"bytes,11,packed"`
}

func (*BytesMsg) ProtoMessage()    {}
func (m *BytesMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *BytesMsg) Reset()         { *m = BytesMsg{} }

type BytesArrayMsg struct {
	ss      [2]string `protobuf:"bytes,3"`
	sb      [3]byte   `protobuf:"bytes,11"`
	skipped int32     `protobuf:"-"`
}

func TestFixedMsg(t *testing.T) {
	i32 := int32(-10)
	u32 := uint32(11)
	i64 := int64(-12)
	u64 := uint64(13)
	f32 := float32(-14.14)
	f64 := float64(15.15)

	m := FixedMsg{
		i32: -1,
		u32: 2,
		i64: -3,
		u64: 4,
		f32: -5.5,
		f64: 6.6,

		pi32: &i32,
		pu32: &u32,
		pi64: &i64,
		pu64: &u64,
		pf32: &f32,
		pf64: &f64,

		si32: []int32{-1},
		su32: []uint32{1, 2},
		si64: []int64{-1, 3, -3},
		su64: []uint64{1, 2, 3, 4},
		sf32: []float32{-1.1, 2.2, -3.3, 4.4},
		sf64: []float64{-1.1, 2.2, -3.3, 4.4},
	}

	check(&m, &m, t)

	var mb, mc FixedMsg
	uncheck(&m, &mb, &mc, t)
	eq("mb", m, mb, t)
	eq("mc", m, mc, t)
}

func TestVarMsg(t *testing.T) {
	i32 := int32(-10)
	u32 := uint32(11)
	i64 := int64(-12)
	u64 := uint64(13)

	m := VarMsg{
		i32: -1,
		u32: 2,
		i64: -3,
		u64: 4,

		pi32: &i32,
		pu32: &u32,
		pi64: &i64,
		pu64: &u64,

		si32: []int32{-1},
		su32: []uint32{1, 2},
		si64: []int64{-1, 3, -3},
		su64: []uint64{1, 2, 3, 4},
	}

	check(&m, &m, t)

	var mb VarMsg
	uncheck(&m, &mb, nil, t)
	eq("mb", m, mb, t)
}

func TestZigZagMsg(t *testing.T) {
	i32 := int32(-10)
	i64 := int64(-12)

	m := ZigZagMsg{
		i32: -1,
		i64: -3,

		pi32: &i32,
		pi64: &i64,

		si32: []int32{-1, 2, -3},
		si64: []int64{-4, 5, -6},
	}

	check(&m, &m, t)

	var mb ZigZagMsg
	uncheck(&m, &mb, nil, t)
	eq("mb", m, mb, t)
}

func TestBytesMsg(t *testing.T) {
	s := "str"

	m := BytesMsg{
		s:  "test1",
		ps: &s,
		ss: []string{"test3", "test4"},
		sb: []byte{3, 2, 1, 0},
	}

	check(&m, &m, t)

	var mb BytesMsg
	uncheck(&m, &mb, nil, t)
	eq("mb", m, mb, t)
}

func TestFixedArrayMsg(t *testing.T) {
	a := FixedArrayMsg{
		ai32: [1]int32{1},
		au32: [2]uint32{2, 3},
		ai64: [3]int64{4, 5, 6},
		au64: [4]uint64{8, 9, 10, 11},
		af32: [5]float32{16, 17, 18, 19, 20},
		af64: [6]float64{32, 33, 34, 35, 36, 37},
	}

	m := FixedMsg{
		si32: []int32{1},
		su32: []uint32{2, 3},
		si64: []int64{4, 5, 6},
		su64: []uint64{8, 9, 10, 11},
		sf32: []float32{16, 17, 18, 19, 20},
		sf64: []float64{32, 33, 34, 35, 36, 37},
	}

	check(&m, &m, t)
	check(&a, &m, t)

	var mb FixedArrayMsg
	var mc FixedMsg
	uncheck(&a, &mb, &mc, t)
	eq("mb", a, mb, t)
	eq("mc", m, mc, t)
}

func TestVarArrayMsg(t *testing.T) {
	a := VarArrayMsg{
		ai32: [1]int32{1},
		au32: [2]uint32{2, 3},
		ai64: [3]int64{4, 5, 6},
		au64: [4]uint64{8, 9, 10, 11},
		ab:   [5]bool{true, false, true, false, true},
	}

	m := VarMsg{
		si32: []int32{1},
		su32: []uint32{2, 3},
		si64: []int64{4, 5, 6},
		su64: []uint64{8, 9, 10, 11},
		sb:   []bool{true, false, true, false, true},
	}

	check(&m, &m, t)
	check(&a, &m, t)

	var mb VarArrayMsg
	var mc VarMsg
	uncheck(&m, &mb, &mc, t)
	eq("mb", a, mb, t)
	eq("mc", m, mc, t)
}

func TestZigZagArrayMsg(t *testing.T) {
	a := ZigZagArrayMsg{
		ai32: [1]int32{-123456789},
		ai64: [2]int64{9876543210123, 4567890987654321},
	}

	m := ZigZagMsg{
		si32: []int32{-123456789},
		si64: []int64{9876543210123, 4567890987654321},
	}

	check(&a, &m, t)

	var mb ZigZagArrayMsg
	var mc ZigZagMsg
	uncheck(&a, &mb, &mc, t)
	eq("mb", a, mb, t)
	eq("mc", m, mc, t)
}

func TestByteArrayMsg(t *testing.T) {
	a := BytesArrayMsg{
		ss:      [2]string{"hello", "world"},
		sb:      [3]byte{0, 1, 2},
		skipped: 99,
	}

	m := BytesMsg{
		ss: []string{"hello", "world"},
		sb: []byte{0, 1, 2},
	}

	check(&m, &m, t)
	check(&a, &m, t)

	var mb BytesArrayMsg
	var mc BytesMsg
	a.skipped = 0 // it should not have been decoded
	uncheck(&a, &mb, &mc, t)
	eq("mb", a, mb, t)
	eq("mc", m, mc, t)
}

func TestZeroMsgs(t *testing.T) {
	f := FixedMsg{}
	check(&f, &f, t)

	v := VarMsg{}
	check(&v, &v, t)

	z := ZigZagMsg{}
	check(&z, &z, t)

	b := BytesMsg{}
	check(&b, &b, t)
}

// check that protobuf3.Marshal(mb) == proto.Marshal(mc)
func check(mb protobuf3.Message, mc proto.Message, t *testing.T) {
	t.Logf("check(%T,%T)", mb, mc)

	b, err := protobuf3.Marshal(mb)
	if err != nil {
		t.Error("ERROR ", err)
		return
	}

	c, err := proto.Marshal(mc)
	if err != nil {
		t.Error("ERROR ", err)
		return
	}

	t.Logf("b = % x", b)
	t.Logf("c = % x", c)

	if !bytes.Equal(b, c) {
		t.Errorf("ERROR Marshal(%T) different between proto and protobuf3", mb)
	}
}

// check that protobuf3.Unmarshal(mb) works like proto.Unmarshal(mc)
func uncheck(mi protobuf3.Message, mb protobuf3.Message, mc proto.Message, t *testing.T) {
	t.Logf("uncheck(%T,%T,%T)", mi, mb, mc)
	t.Logf("mi = %v", mi)

	pb, err := protobuf3.Marshal(mi)
	if err != nil {
		t.Error("ERROR ", err)
		return
	}

	t.Logf("pb = % x", pb)

	err = protobuf3.Unmarshal(pb, mb)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("mb = %v", mb)

	if mc != nil {
		err = proto.Unmarshal(pb, mc)
		if err != nil {
			t.Error("ERROR ", err)
			return
		}
		t.Logf("mc = %v", mc)
	}
}

func eq(name string, x interface{}, y interface{}, t *testing.T) {
	if !reflect.DeepEqual(x, y) {
		t.Errorf("ERROR %s: (%v) %v != (%v) %v", name, reflect.TypeOf(x), x, reflect.TypeOf(y), y)
	}
}

type NestedPtrStructMsg struct {
	first  *InnerMsg   `protobuf:"bytes,1"`
	second *InnerMsg   `protobuf:"bytes,2"`
	many   []*InnerMsg `protobuf:"bytes,3"`
	more   []*InnerMsg `protobuf:"bytes,4"`
	some   []*InnerMsg `protobuf:"bytes,5"`
	empty  []InnerMsg  `protobuf:"bytes,6"`
	ptrs   []*InnerMsg `protobuf:"bytes,7"`
}

func (*NestedPtrStructMsg) ProtoMessage()    {}
func (m *NestedPtrStructMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *NestedPtrStructMsg) Reset()         { *m = NestedPtrStructMsg{} }

type InnerMsg struct {
	i int32 `protobuf:"varint,2"`
}

func TestNestedPtrStructMsg(t *testing.T) {
	m := NestedPtrStructMsg{
		first:  &InnerMsg{0x11},
		second: &InnerMsg{0x22},
		many:   []*InnerMsg{&InnerMsg{0x33}, &InnerMsg{0x44}},
		more:   []*InnerMsg{},
	}

	check(&m, &m, t)

	var mb, mc NestedPtrStructMsg
	uncheck(&m, &mb, &mc, t)

	t.Logf("m = %#v", m)
	t.Logf("mb = %#v", mb)
	t.Logf("mc = %#v", mc)

	eq("mb.first", m.first, mb.first, t)
	eq("mc.first", m.first, mc.first, t)
	eq("mb.second", m.second, mb.second, t)
	eq("mc.second", m.second, mc.second, t)
	eq("mb.many", m.many, mb.many, t)
	eq("mc.many", m.many, mc.many, t)
	eq("mb.empty", m.empty, mb.empty, t)
	eq("mc.empty", m.empty, mc.empty, t)
	eq("mb.ptrs", m.ptrs, mb.ptrs, t)
	eq("mc.ptrs", m.ptrs, mc.ptrs, t)
}

type NestedStructMsg struct {
	first  InnerMsg     `protobuf:"bytes,1"`
	second InnerMsg     `protobuf:"bytes,2"`
	many   []InnerMsg   `protobuf:"bytes,3"`
	more   [3]InnerMsg  `protobuf:"bytes,4"`
	some   [1]*InnerMsg `protobuf:"bytes,5"`
	empty  []InnerMsg   `protobuf:"bytes,6"`
	ptrs   []*InnerMsg  `protobuf:"bytes,7"`
}

func TestNestedStructMsg(t *testing.T) {
	a := NestedStructMsg{
		first:  InnerMsg{0x11},
		second: InnerMsg{0x22},
		many:   []InnerMsg{InnerMsg{0x33}},
		more:   [3]InnerMsg{InnerMsg{0x44}, InnerMsg{0x55}, InnerMsg{0x66}},
		some:   [1]*InnerMsg{&InnerMsg{0x77}},
	}

	m := NestedPtrStructMsg{
		first:  &InnerMsg{0x11},
		second: &InnerMsg{0x22},
		many:   []*InnerMsg{&InnerMsg{0x33}},
		more:   []*InnerMsg{&InnerMsg{0x44}, &InnerMsg{0x55}, &InnerMsg{0x66}},
		some:   []*InnerMsg{&InnerMsg{0x77}},
	}

	check(&m, &m, t)
	check(&a, &m, t)

	var mb NestedStructMsg
	var mc NestedPtrStructMsg
	uncheck(&m, &mb, &mc, t)

	t.Logf("a = %#v", a)
	t.Logf("mb = %#v", mb)
	eq("mb", a, mb, t)

	t.Logf("m = %#v", m)
	t.Logf("mc = %#v", mc)
	eq("mc", m, mc, t)
}

type EmptyStructMsg struct {
	zero   InnerMsg   `protobuf:"bytes,1"`
	zslice []InnerMsg `protobuf:"bytes,2"`
	ptr    *InnerMsg  `protobuf:"bytes,3"` // this should encode the empty InnerMsg because we need to distiguish between a nil pointer and a pointer to a zero-value
	zptr   *InnerMsg  `protobuf:"bytes,4"`
}

func (*EmptyStructMsg) ProtoMessage()    {}
func (m *EmptyStructMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *EmptyStructMsg) Reset()         { *m = EmptyStructMsg{} }

func TestEmptyStructMsg(t *testing.T) {
	m := EmptyStructMsg{}

	pb, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error("ERROR ", err)
		return
	}
	if len(pb) != 0 {
		t.Error("empty struct should have encoded to nothing")
		return
	}

	check(&m, &m, t)

	var mb, mc EmptyStructMsg
	uncheck(&m, &mb, &mc, t)
	eq("mb", m, mb, t)
	eq("mc", m, mc, t)

	// now set one of the pointers to point to a zero-value InnerMsg and retest. this time the zero-value
	// must be encoded, so that we end up with a non-nil pointer when we unmarshal
	m.ptr = &InnerMsg{}

	pb, err = protobuf3.Marshal(&m)
	if err != nil {
		t.Error("ERROR ", err)
		return
	}
	if len(pb) == 0 {
		t.Error("empty struct with non-nil pointer should NOT have encoded to nothing")
		return
	}

	check(&m, &m, t)

	mb, mc = EmptyStructMsg{}, EmptyStructMsg{}
	uncheck(&m, &mb, &mc, t)
	eq("mb", m, mb, t)
	eq("mc", m, mc, t)
}

func (*InnerMsg) ProtoMessage()     {}
func (im *InnerMsg) String() string { return fmt.Sprintf("&InnerMsg{ %d }", im.i) }
func (m *InnerMsg) Reset()          { *m = InnerMsg{} }

type RecursiveTypeMsg struct {
	// type-recursive pointer
	self *RecursiveTypeMsg `protobuf:"bytes,1"`
	b    bool              `protobuf:"varint,22334455"`
}

func (*RecursiveTypeMsg) ProtoMessage()    {}
func (m *RecursiveTypeMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *RecursiveTypeMsg) Reset()         { *m = RecursiveTypeMsg{} }

func TestRecursiveTypeMsg(t *testing.T) {
	m := RecursiveTypeMsg{
		self: &RecursiveTypeMsg{
			b: true,
		},
	}

	check(&m, &m, t)

	var mb RecursiveTypeMsg
	uncheck(&m, &mb, nil, t)
	eq("mb", m, mb, t)
}

type MapMsg struct {
	m map[string]int32   `protobuf:"bytes,3" protobuf_key:"bytes,1" protobuf_val:"varint,2"`
	n map[int32][]byte   `protobuf:"bytes,4" protobuf_key:"varint,1" protobuf_val:"bytes,2"`
	e map[int32]struct{} `protobuf:"bytes,5" protobuf_key:"zigzag32,1" protobuf_val:"bytes,2"` // check that sets encode and decode properly
}

func (*MapMsg) ProtoMessage()    {}
func (m *MapMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MapMsg) Reset()         { *m = MapMsg{} }

func TestMapMsg(t *testing.T) {
	for i, m := range []MapMsg{
		MapMsg{
			m: map[string]int32{"123": 123, "abc": 124},
		},
		MapMsg{
			n: map[int32][]byte{125: []byte("abc"), 126: []byte("def")},
		},
		MapMsg{
			e: map[int32]struct{}{-127: struct{}{}, -128: struct{}{}},
		},
	} {

		// note we can't just use check() because the encoding depends on the map's iteration order,
		// and that is random. So we allow for either result when verifying

		b, err := protobuf3.Marshal(&m)
		if err != nil {
			t.Error("ERROR ", err)
			return
		}

		c, err := proto.Marshal(&m)
		if err != nil {
			t.Error("ERROR ", err)
			return
		}
		if i == 2 && len(m.e) != 0 { // double check, in case someone adds more test cases
			// we have improved our marshaling of emptys struct. we elide them completely. this means that our output is different from that of proto.Marshal
			// and we need to account for this
			c = []byte{0x2a, 0x03, 0x08, 0xfd, 0x01, 0x2a, 0x03, 0x08, 0xff, 0x01}
		}

		t.Logf("m = %#v", m)
		t.Logf("b = % x", b)
		t.Logf("c = % x", c)

		if !bytes.Equal(b, c) {
			// OK, they didn't match, but if we swap the two fields then do they match?
			// the values of the two fields were chosen so they both encoded to the same length, so swappihg the order of the encoding is easy
			ll := len(b) / 2
			b = append(b[ll:], b[:ll]...)
			if !bytes.Equal(b, c) {
				t.Errorf("ERROR Marshal(%T) different between proto and protobuf3", m)
			}
		}

		var mb, mc MapMsg
		uncheck(&m, &mb, &mc, t)
		eq("mb", m, mb, t)
		eq("mc", m, mc, t)
	}

}

// test encoding and decoding int and uint as zigzag and varint
// (proto package doesn't support this, but we do, since an int encoded in
// zigzag is always safe irrespective of the sizeof(int), as is a uint
// encoded as a varint. int encoded as a varint, if the int is negative,
// will not work if the encoder as 32-bit and the decoder 64-bit)
type IntMsg struct {
	i   int    `protobuf:"varint,1"`
	u   uint   `protobuf:"varint,2"`
	i8  int8   `protobuf:"varint,3"`
	u8  uint8  `protobuf:"varint,4"`
	i16 int16  `protobuf:"varint,5"`
	u16 uint16 `protobuf:"varint,6"`
	z32 int    `protobuf:"zigzag32,10"`
	z64 int    `protobuf:"zigzag64,11"`

	si   []int    `protobuf:"varint,21"`
	su   []uint   `protobuf:"varint,22"`
	si8  []int8   `protobuf:"varint,23"`
	su8  []uint8  `protobuf:"varint,24"`
	si16 []int16  `protobuf:"varint,25"`
	su16 []uint16 `protobuf:"varint,26"`
	sz32 []int    `protobuf:"zigzag32,30"`
	sz64 []int    `protobuf:"zigzag64,31"`

	u32 U32 `protobuf:"varint,40"`
	i32 I32 `protobuf:"varint,41"`
}

type U32 uint32
type I32 = int32

// same fields, but using types the old package can use
type OldIntMsg struct {
	i   int32  `protobuf:"varint,1"`
	u   uint32 `protobuf:"varint,2"`
	i8  int32  `protobuf:"varint,3"`
	u8  uint32 `protobuf:"varint,4"`
	i16 int32  `protobuf:"varint,5"`
	u16 uint32 `protobuf:"varint,6"`
	z32 int32  `protobuf:"zigzag32,10"`
	z64 int64  `protobuf:"zigzag64,11"`

	si   []int32  `protobuf:"varint,21,packed"`
	su   []uint32 `protobuf:"varint,22,packed"`
	si8  []int32  `protobuf:"varint,23,packed"`
	su8  []uint32 `protobuf:"varint,24,packed"`
	si16 []int32  `protobuf:"varint,25,packed"`
	su16 []uint32 `protobuf:"varint,26,packed"`
	sz32 []int32  `protobuf:"zigzag32,30,packed"`
	sz64 []int64  `protobuf:"zigzag64,31,packed"`

	u32 U32 `protobuf:"varint,40"`
	i32 I32 `protobuf:"varint,41"`
}

func (*OldIntMsg) ProtoMessage()    {}
func (m *OldIntMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *OldIntMsg) Reset()         { *m = OldIntMsg{} }

func TestIntMsg(t *testing.T) {
	m := IntMsg{
		i:   -1,
		u:   2,
		i8:  -3,
		u8:  4,
		i16: -4,
		u16: 5,
		z32: 555,
		z64: -5761760885135729648,

		si:   []int{-1, 1},
		su:   []uint{2, 22},
		si8:  []int8{-3, 3},
		su8:  []uint8{4, 44},
		si16: []int16{-4, 4},
		su16: []uint16{5, 55},
		sz32: []int{555, -555},
		sz64: []int{-5761760885135729648, 5761760885135729648},

		u32: 32,
		i32: 33,
	}

	o := OldIntMsg{
		i:   -1,
		u:   2,
		i8:  -3,
		u8:  4,
		i16: -4,
		u16: 5,
		z32: 555,
		z64: -5761760885135729648,

		si:   []int32{-1, 1},
		su:   []uint32{2, 22},
		si8:  []int32{-3, 3},
		su8:  []uint32{4, 44},
		si16: []int32{-4, 4},
		su16: []uint32{5, 55},
		sz32: []int32{555, -555},
		sz64: []int64{-5761760885135729648, 5761760885135729648},

		u32: 32,
		i32: 33,
	}

	check(&o, &o, t)
	check(&m, &o, t)

	var mb IntMsg
	var mc OldIntMsg
	uncheck(&m, &mb, &mc, t)
	eq("mb", m, mb, t)
	eq("mc", o, mc, t)
}

type TimeMsg struct {
	tm      time.Time        `protobuf:"bytes,1"`
	dur     time.Duration    `protobuf:"bytes,26"`
	dur2    *time.Duration   `protobuf:"bytes,46"`
	dur3    []time.Duration  `protobuf:"bytes,64"`
	dur4    [1]time.Duration `protobuf:"bytes,93"`
	zero_d  time.Duration    `protobuf:"bytes,128"` // leave at the zero-value; it should encode to nothing
	zero_d2 *time.Duration   `protobuf:"bytes,129"` // same
	zero_d3 []time.Duration  `protobuf:"bytes,130"` // same
}

type OldTimeMsg struct {
	tm   *timestamp.Timestamp `protobuf:"bytes,1"`
	dur  *duration.Duration   `protobuf:"bytes,26"`
	dur2 *duration.Duration   `protobuf:"bytes,46"`
	dur3 []*duration.Duration `protobuf:"bytes,64"`
	dur4 []*duration.Duration `protobuf:"bytes,93"`
}

func (*OldTimeMsg) ProtoMessage()    {}
func (m *OldTimeMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *OldTimeMsg) Reset()         { *m = OldTimeMsg{} }

func TestTimeMsg(t *testing.T) {
	d2 := -(time.Second + time.Millisecond)
	m := TimeMsg{
		tm:   time.Unix(112233, 445566).UTC(),
		dur:  time.Second*10 + time.Microsecond,
		dur2: &d2,
		dur3: []time.Duration{15 * time.Second, 365 * 24 * time.Hour},
		dur4: [1]time.Duration{time.Nanosecond},
	}

	o := OldTimeMsg{
		tm: &timestamp.Timestamp{
			Seconds: 112233,
			Nanos:   445566,
		},
		dur: &duration.Duration{
			Seconds: 10,
			Nanos:   1000,
		},
		dur2: &duration.Duration{
			Seconds: -1,
			Nanos:   -1000000,
		},
		dur3: []*duration.Duration{
			&duration.Duration{Seconds: 15},
			&duration.Duration{Seconds: 365 * 24 * 60 * 60},
		},
		dur4: []*duration.Duration{&duration.Duration{Nanos: 1}},
	}

	check(&o, &o, t)
	check(&m, &o, t)

	var mb TimeMsg
	var mc OldTimeMsg
	uncheck(&m, &mb, &mc, t)
	eq("mb", m, mb, t)
	eq("mc", o, mc, t)

	eq("tm", mb.tm, m.tm, t)
	eq("dur", mb.dur, m.dur, t)
	if mb.dur2 != nil {
		eq("dur2", *mb.dur2, *m.dur2, t)
	} else {
		t.Error("failed to unmarshal *time.Duration")
	}
	eq("dur3", mb.dur3, m.dur3, t)
	eq("dur4", mb.dur4, m.dur4, t)
}

type CustomMsg struct {
	Slice  CustomSlice  `protobuf:"bytes,1"`
	Int    CustomInt    `protobuf:"varint,2"`
	Fixedp *CustomFixed `protobuf:"fixed32,3"`
}

type CustomSlice [][]uint32

func (s *CustomSlice) MarshalProtobuf3() ([]byte, error) {
	var buf, tmp protobuf3.Buffer
	for i, ss := range *s {
		tmp.Reset()
		for _, x := range ss {
			tmp.EncodeVarint(uint64(x))
		}
		buf.EncodeBytes(uint32(i)+1, tmp.Bytes())
	}
	return buf.Bytes(), nil
}

func (s *CustomSlice) UnmarshalProtobuf3(data []byte) error {
	buf := protobuf3.NewBuffer(data)
	for !buf.EOF() {
		err := buf.SkipVarint()
		if err != nil {
			return err
		}
		raw, err := buf.DecodeRawBytes()
		if err != nil {
			return err
		}
		tmp := protobuf3.NewBuffer(raw)
		var row []uint32
		for !tmp.EOF() {
			v, err := tmp.DecodeVarint()
			if err != nil {
				return err
			}
			row = append(row, uint32(v))
		}
		*s = append(*s, row)
	}
	return nil
}

type CustomInt uint32

func (i *CustomInt) MarshalProtobuf3() ([]byte, error) {
	var buf protobuf3.Buffer
	buf.EncodeVarint(uint64(*i))
	return buf.Bytes(), nil
}

func (i *CustomInt) UnmarshalProtobuf3(data []byte) error {
	buf := protobuf3.NewBuffer(data)
	x, err := buf.DecodeVarint()
	if err != nil {
		return err
	}
	*i = CustomInt(x)
	return nil
}

type CustomFixed uint32

func (i *CustomFixed) MarshalProtobuf3() ([]byte, error) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(*i))
	return buf[:], nil
}

func (i *CustomFixed) UnmarshalProtobuf3(data []byte) error {
	if len(data) != 4 {
		return fmt.Errorf("fixed32 data length %d is not 4", len(data))
	}
	*i = CustomFixed(binary.LittleEndian.Uint32(data))
	return nil
}

type EquivToCustomMsg struct {
	Custom *EquivCustomSlices `protobuf:"bytes,1"`
	Int    uint32             `protobuf:"varint,2"`
	Fixedp *uint32            `protobuf:"fixed32,3"`
}

type EquivCustomSlices struct {
	Slice1 []uint32 `protobuf:"varint,1,packed"`
	Slice2 []uint32 `protobuf:"varint,2,packed"`
}

func (*EquivToCustomMsg) ProtoMessage()    {}
func (m *EquivToCustomMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *EquivToCustomMsg) Reset()         { *m = EquivToCustomMsg{} }

func (*EquivCustomSlices) ProtoMessage()    {}
func (m *EquivCustomSlices) String() string { return fmt.Sprintf("%+v", *m) }
func (m *EquivCustomSlices) Reset()         { *m = EquivCustomSlices{} }

func TestCustomMsg(t *testing.T) {
	var custom_int = CustomFixed(7)
	m := CustomMsg{
		Slice:  CustomSlice{[]uint32{1, 2}, []uint32{3, 4, 5}},
		Int:    5,
		Fixedp: &custom_int,
	}

	var custom_uint32 = uint32(custom_int)
	o := EquivToCustomMsg{
		Custom: &EquivCustomSlices{
			Slice1: []uint32{1, 2},
			Slice2: []uint32{3, 4, 5},
		},
		Int:    5,
		Fixedp: &custom_uint32,
	}

	check(&o, &o, t)
	check(&m, &o, t)

	var mb CustomMsg
	var mc EquivToCustomMsg
	uncheck(&m, &mb, &mc, t)
	eq("mb", m, mb, t)
	eq("mc", o, mc, t)
}

type SliceMarshalerMsg struct {
	Slice []TestMarshaler `protobuf:"bytes,1"`
}

type TestMarshaler [4]byte

func (t *TestMarshaler) MarshalProtobuf3() ([]byte, error) {
	return t[:], nil
}

func (t *TestMarshaler) UnmarshalProtobuf3(data []byte) error {
	copy(t[:], data)
	return nil
}

type EquivSliceMarshalerMsg struct {
	Slice [][]byte `protobuf:"bytes,1"`
}

func (*EquivSliceMarshalerMsg) ProtoMessage()    {}
func (m *EquivSliceMarshalerMsg) String() string { return fmt.Sprintf("%+v", *m) }
func (m *EquivSliceMarshalerMsg) Reset()         { *m = EquivSliceMarshalerMsg{} }

func TestSliceMarshlerMsg(t *testing.T) {
	m := SliceMarshalerMsg{
		Slice: []TestMarshaler{[4]byte{1, 2, 3, 4}, [4]byte{5, 6, 7, 8}},
	}

	o := EquivSliceMarshalerMsg{
		Slice: [][]byte{[]byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}},
	}

	check(&o, &o, t)
	check(&m, &o, t)

	var mb SliceMarshalerMsg
	var mc EquivSliceMarshalerMsg
	uncheck(&m, &mb, &mc, t)
	eq("mb", m, mb, t)
	eq("mc", o, mc, t)
}

type StructArrayMsg struct {
	Str string `protobuf:"bytes,1"`
	Sub struct {
		Flos [3]float32 `protobuf:"fixed32,2"`
		Num  uint32     `protobuf:"varint,3"`
	} `protobuf:"bytes,2"`
	Str2 string `protobuf:"bytes,3"`
}

func TestStructArrayMsg(t *testing.T) {
	var m, m2 StructArrayMsg
	m.Str = "hello"
	m.Sub.Flos[0] = 0
	m.Sub.Flos[1] = 0
	m.Sub.Flos[2] = 0
	m.Sub.Num = 4
	m.Str2 = "goodbye"

	pb, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error("ERROR ", err)
	}
	t.Logf("StructArrayMsg = %+v\n", m)
	t.Logf("StructArrayMsg = % x\n", pb)

	err = protobuf3.Unmarshal(pb, &m2)
	if err != nil {
		t.Error("ERROR ", err)
	}
	t.Logf("StructArrayMsg = %+v\n", m2)
	if !reflect.DeepEqual(m, m2) {
		t.Error("ERROR results are !=")
	}
}

type BadZigZagTagMsg struct {
	x int `protobuf:"zigzag,1"` // such an encoding does not exist (it's "zigzag32" and "zigzag64")
}

func TestBadZigZagTagMsg(t *testing.T) {
	_, err := protobuf3.Marshal(&BadZigZagTagMsg{})
	if err == nil {
		t.Error("ERROR marshaling a BadZigZagTagMsg should have failed")
	}
}

type MissingTagMsg struct {
	x int32 // no protobuf tag
}

type MissingInnerTagMsg struct {
	m *MissingTagMsg `protobuf:"bytes,1"`
}

func TestMissingTagMsg(t *testing.T) {
	_, err := protobuf3.Marshal(&MissingTagMsg{})
	if err == nil {
		t.Error("ERROR marshaling a MissingTagMsg should have failed")
	}

	// make sure 2nd use of the same broken type also fails
	_, err = protobuf3.Marshal(&MissingTagMsg{})
	if err == nil {
		t.Error("ERROR marshaling a MissingTagMsg a 2nd time should also have failed")
	}

	// and same for a type which contains a broken type
	_, err = protobuf3.Marshal(&MissingInnerTagMsg{})
	if err == nil {
		t.Error("ERROR marshaling a MissingInnerTagMsg should have failed")
	}

	// and again
	_, err = protobuf3.Marshal(&MissingInnerTagMsg{})
	if err == nil {
		t.Error("ERROR marshaling a MissingInnerTagMsg a 2nd time should also have failed")
	}
}

// make sure we fail if someone passes in &(&msg)
func TestPtoPtoS(t *testing.T) {
	var msg = struct {
		I int `protobuf:"varint,1"`
	}{
		I: 99,
	}
	data, err := protobuf3.Marshal(&msg)
	if err != nil {
		t.Error(err)
		return
	}
	err = protobuf3.Unmarshal(data, &msg)
	if err != nil {
		t.Error(err)
		return
	}

	ptr := &msg
	err = protobuf3.Unmarshal(data, &ptr)
	if err != protobuf3.ErrNotPointerToStruct {
		t.Errorf("should have failed with ErrNotPointerToStruct. Got %v", err)
	}
}

func TestFind(t *testing.T) {
	var b = []byte{1, 2, 3, 4, 5}
	var msg = struct {
		B []byte  `protobuf:"bytes,1"`
		I int     `protobuf:"varint,2"`
		J int     `protobuf:"varint,3"`
		F float64 `protobuf:"fixed64,4"`
	}{
		B: b, I: 2, J: 3, F: 5.6,
	}

	pb, err := protobuf3.Marshal(&msg)
	if err != nil { // we don't expect any errors
		t.Error(err)
		return
	}

	buf := protobuf3.NewBuffer(pb)
	pos, full, val, wt, err := buf.Find(1, false)
	if err != nil {
		t.Error(err)
		return
	}
	if wt != protobuf3.WireBytes {
		t.Errorf("wt %v", wt)
	} else if !bytes.Equal(full, []byte{2 + 1<<3, 5, 1, 2, 3, 4, 5}) {
		t.Errorf("full % x", full)
	} else if !bytes.Equal(val, b) {
		t.Errorf("val % x", full)
	} else if pos != 0 {
		t.Errorf("pos %d", pos)
	}

	// next find should fail
	pos, full, val, wt, err = buf.Find(1, false)
	if err != protobuf3.ErrNotFound {
		t.Errorf("didn't fail properly: %v", err)
	}

	// go back and find the I and J values
	buf.Rewind()

	for i := byte(2); i <= 3; i++ {
		pos, full, val, wt, err = buf.Find(uint(i), false)
		if err != nil {
			t.Error(err)
		} else if wt != protobuf3.WireVarint {
			t.Errorf("wrong wt %v", wt)
		} else if !bytes.Equal(full, []byte{i<<3 + 0, i}) {
			t.Errorf("full % x", full)
		} else if !bytes.Equal(val, []byte{i}) {
			t.Errorf("val % x", full)
		} else if pos != 2+len(b)+2*int(i-2) {
			t.Errorf("pos %d != %d", pos, 2+len(b)+2*int(i-2))
		}
	}
}

func TestNext(t *testing.T) {
	var b = []byte{1, 2, 3, 4, 5}
	var msg = struct {
		B []byte  `protobuf:"bytes,1"`
		I int     `protobuf:"varint,2"`
		J int     `protobuf:"varint,3"`
		F float64 `protobuf:"fixed64,4"`
	}{
		B: b, I: 2, J: 3, F: 5.6,
	}

	pb, err := protobuf3.Marshal(&msg)
	if err != nil { // we don't expect any errors
		t.Error(err)
		return
	}

	buf := protobuf3.NewBuffer(pb)

	for i := 0; i < 4; i++ {
		id, full, val, wt, err := buf.Next()
		t.Logf("id %d, full % x, val % x, %v, %v", id, full, val, wt, err)
		if err != nil {
			t.Error(err)
			return
		}
		if id != i+1 {
			t.Errorf("id %v", id)
		}
	}
	id, full, val, wt, err := buf.Next()
	if id != 0 || full != nil || val != nil || wt != 0 || err != nil {
		t.Error("should have returned zero-values")
	}
}

type InappropriateWiretypeMsg struct {
	F1 struct {
		f float64 `protobuf:"bytes,1"` // should cause an error; you should use fixed64 for float64
	}
	F2 struct {
		f float32 `protobuf:"fixed64,1"` // should cause an error; you must use fixed32 for float32
	}
	D1 struct {
		d float64 `protobuf:"fixed32,1"` // should cause an error; you must use fixed64 for float64
	}
	I1 struct {
		i int32 `protobuf:"bytes,2"` // should cause an error
	}
	I2 struct {
		i int32 `protobuf:"fixed64,2"` // should *not* cause an error. we consider it peculiar but permitted
	}
}

func TestInappropriateWiretypes(t *testing.T) {
	var m InappropriateWiretypeMsg

	_, err := protobuf3.Marshal(&m.F1)
	t.Log(err)
	if err == nil {
		t.Error("InappropriateWiretypeMsg.F1 should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.F2)
	t.Log(err)
	if err == nil {
		t.Error("InappropriateWiretypeMsg.F2 should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.D1)
	t.Log(err)
	if err == nil {
		t.Error("InappropriateWiretypeMsg.D1 should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.I1)
	t.Log(err)
	if err == nil {
		t.Error("InappropriateWiretypeMsg.I1 should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.I2)
	t.Log(err)
	if err != nil {
		t.Error("InappropriateWiretypeMsg.I2 should not have caused the error ", err)
	}
}

type DuplicateIdMsg struct {
	x int `protobuf:"varint,1"`
	y int `protobuf:"zigzag32,1"` // should cause an error b/c it has the same ID as field 'x', even though the wiretype is different
}

func TestDuplicateId(t *testing.T) {
	var m DuplicateIdMsg

	_, err := protobuf3.Marshal(&m)
	t.Log(err)
	if err == nil {
		t.Error("DuplicateIdMsg should have caused an error")
	}
}

type EmbeddedMsg struct {
	X                uint32                `protobuf:"varint,2"`
	InnerEmbeddedMsg `protobuf:"embedded"` // marshals as part of the outer struct's fields
}

type NestedEmbeddedMsg struct {
	InnerEmbeddedMsg `protobuf:"bytes,3"` // marshals as if it were named, nested in a bytes,3
	X                uint32               `protobuf:"varint,1"`
}

type BadEmbeddedMsg struct {
	X                uint32                `protobuf:"varint,1"` // collides with InnerEmbeddedMsg.S
	F                float64               `protobuf:"fixed64,4"`
	InnerEmbeddedMsg `protobuf:"embedded"` // marshals as part of the outer struct's fields
}

type InnerEmbeddedMsg struct {
	S string `protobuf:"bytes,1"` // don't collide with tags in EmbeddedMsg
}

func TestEmbeddedMsg(t *testing.T) {
	{
		var m1 EmbeddedMsg
		m1.S = "abc"
		b, err := protobuf3.Marshal(&m1)
		t.Logf("[% x], %v", b, err)
		if err != nil {
			t.Error("EmbeddedMsg", err)
		}
		if !bytes.Equal(b, []byte{0x0a, 0x03, 0x61, 0x62, 0x63}) {
			t.Errorf("EmbeddedMsg marshaling %x incorrect", b)
		}
	}

	{
		var m2 NestedEmbeddedMsg
		m2.S = "abc"
		b, err := protobuf3.Marshal(&m2)
		t.Logf("[% x], %v", b, err)
		if err != nil {
			t.Error("NestedEmbeddedMsg", err)
		}
		if !bytes.Equal(b, []byte{0x1a, 0x05, 0x0a, 0x03, 0x61, 0x62, 0x63}) {
			t.Errorf("NestedEmbeddedMsg protobuf [% x] incorrect", b)
		}
	}

	{
		var m3 BadEmbeddedMsg
		_, err := protobuf3.Marshal(&m3)
		t.Log(err)
		if err == nil {
			t.Error("BadEmbeddedMsg should have caused an error")
		}
	}
}

type BadMapMsg struct {
	A struct {
		m map[string]int32 `protobuf:"varint,1" protobuf_key:"bytes,1" protobuf_val:"varint,2"` // must use bytes wiretype for maps
	}
	B struct {
		m map[string]int32 `protobuf:"bytes,1" protobuf_key:"-" protobuf_val:"varint,2"` // can't skip keys
	}
	C struct {
		m map[string]int32 `protobuf:"bytes,1" protobuf_key:"bytes,3" protobuf_val:"varint,2"` // keys must use tag 1
	}
	D struct {
		m map[string]int32 `protobuf:"bytes,1" protobuf_key:"bytes,1" protobuf_val:"-"` // can't skip values (we could support it if we ever needed it. for now if all the values are zero-values we're effectively skipping them)
	}
	E struct {
		m map[string]int32 `protobuf:"bytes,1" protobuf_key:"bytes,1" protobuf_val:"varint,3"` // values must use tag 2
	}
}

func TestBadMapMsg(t *testing.T) {
	var m BadMapMsg
	_, err := protobuf3.Marshal(&m.A)
	t.Log(err)
	if err == nil {
		t.Error("BadMapMsg.A should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.B)
	t.Log(err)
	if err == nil {
		t.Error("BadMapMsg.B should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.C)
	t.Log(err)
	if err == nil {
		t.Error("BadMapMsg.C should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.D)
	t.Log(err)
	if err == nil {
		t.Error("BadMapMsg.D should have caused an error")
	}

	_, err = protobuf3.Marshal(&m.E)
	t.Log(err)
	if err == nil {
		t.Error("BadMapMsg.E should have caused an error")
	}
}

type MapOfSliceOfStruct struct {
	m map[int][]StructForMap `protobuf:"bytes,1" protobuf_key:"varint,1" protobuf_val:"bytes,2"`
}

type StructForMap struct {
	s string `protobuf:"bytes,1"`
	t bool   `protobuf:"varint,2"`
}

func TestMapOfSliceOfStruct(t *testing.T) {
	var m = MapOfSliceOfStruct{
		m: make(map[int][]StructForMap),
	}
	m.m[0] = nil
	m.m[1] = []StructForMap{StructForMap{s: "one.0", t: false}, StructForMap{s: "one.1", t: true}, StructForMap{s: "one.2", t: false}}
	m.m[2] = []StructForMap{StructForMap{s: "two.3", t: true}}

	b, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error(err)
	}

	var m2 MapOfSliceOfStruct
	err = protobuf3.Unmarshal(b, &m2)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(&m, &m2) {
		t.Error("unmarshal(marshal(x)) != x")
		t.Errorf("x = %+v", &m)
		t.Errorf("unmarshal(marshal(x)) = %+v", &m2)
	}
}

type MapOfPtrToStruct struct {
	m map[int]*StructForMap `protobuf:"bytes,1" protobuf_key:"varint,1" protobuf_val:"bytes,2"`
}

func TestMapOfPtrToStruct(t *testing.T) {
	var m = MapOfPtrToStruct{
		m: make(map[int]*StructForMap),
	}
	m.m[0] = nil
	m.m[10] = &StructForMap{s: "one.0", t: true}
	m.m[11] = &StructForMap{s: "one.1", t: false}
	m.m[12] = &StructForMap{s: "one.2", t: true}
	m.m[23] = &StructForMap{s: "two.3", t: false}

	b, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error(err)
	}

	var m2 MapOfPtrToStruct
	err = protobuf3.Unmarshal(b, &m2)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(&m, &m2) {
		t.Error("unmarshal(marshal(x)) != x")
		t.Errorf("x = %+v", &m)
		t.Errorf("unmarshal(marshal(x)) = %+v", &m2)
	}
}

type MapOfStruct struct {
	m map[int]StructForMap `protobuf:"bytes,1" protobuf_key:"varint,1" protobuf_val:"bytes,2"`
}

func TestMapOfStruct(t *testing.T) {
	var m = MapOfStruct{
		m: make(map[int]StructForMap),
	}
	m.m[0] = StructForMap{}
	m.m[10] = StructForMap{s: "one.0", t: true}
	m.m[11] = StructForMap{s: "one.1", t: false}
	m.m[12] = StructForMap{s: "one.2", t: true}
	m.m[23] = StructForMap{s: "two.3", t: false}

	b, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error(err)
	}

	var m2 MapOfStruct
	err = protobuf3.Unmarshal(b, &m2)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(&m, &m2) {
		t.Error("unmarshal(marshal(x)) != x")
		t.Errorf("x = %+v", &m)
		t.Errorf("unmarshal(marshal(x)) = %+v", &m2)
	}
}

type MapOfString struct {
	m map[string]string `protobuf:"bytes,1" protobuf_key:"varint,1" protobuf_val:"bytes,2"`
}

func TestMapOfString(t *testing.T) {
	var m = MapOfString{
		m: make(map[string]string),
	}
	m.m["0"] = ""
	m.m["10"] = "one.0"
	m.m["11"] = "one.1"
	m.m["12"] = "one.2"
	m.m["23"] = "two.3"

	b, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error(err)
	}

	var m2 MapOfString
	err = protobuf3.Unmarshal(b, &m2)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(&m, &m2) {
		t.Error("unmarshal(marshal(x)) != x")
		t.Errorf("x = %+v", &m)
		t.Errorf("unmarshal(marshal(x)) = %+v", &m2)
	}
}

// integers smaller than protobuf natively supports
type SmallVarIntMsg struct {
	i8  int8   `protobuf:"varint,30"`
	u8  uint8  `protobuf:"varint,31"`
	i16 int16  `protobuf:"varint,32"`
	u16 uint16 `protobuf:"varint,33"`

	pi8  *int8   `protobuf:"varint,34"`
	pu8  *uint8  `protobuf:"varint,35"`
	pi16 *int16  `protobuf:"varint,36"`
	pu16 *uint16 `protobuf:"varint,37"`

	si8  []int8   `protobuf:"varint,38,packed"`
	su8  []uint8  `protobuf:"varint,39,packed"`
	si16 []int16  `protobuf:"varint,40,packed"`
	su16 []uint16 `protobuf:"varint,41,packed"`

	ai8  [2]int8   `protobuf:"varint,42,packed"`
	au8  [3]uint8  `protobuf:"varint,43,packed"`
	ai16 [1]int16  `protobuf:"varint,44,packed"`
	au16 [1]uint16 `protobuf:"varint,45,packed"`

	// try a zero-length array type too
	zu16 [0]uint16 `protobuf:"varint,46,packed"`
}

func TestSmallVarIntMsg(t *testing.T) {
	i8 := int8(-4)
	u8 := uint8(4)
	i16 := int16(4567)
	u16 := uint16(55555)

	m := SmallVarIntMsg{
		i8:  -8,
		u8:  9,
		i16: -10,
		u16: 11,

		pi8:  &i8,
		pu8:  &u8,
		pi16: &i16,
		pu16: &u16,

		si8:  []int8{-1, 2, -3},
		su8:  []uint8{1, 127, 128, 255},
		si16: []int16{-10000, 20000, -30000},
		su16: []uint16{0, 30000, 60000},

		ai8:  [2]int8{3, -4},
		au8:  [3]uint8{0, 0, 99},
		ai16: [1]int16{-0x7770},
		au16: [1]uint16{16},

		zu16: [0]uint16{},
	}

	b, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error(err)
	}

	var m2 SmallVarIntMsg
	err = protobuf3.Unmarshal(b, &m2)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(&m, &m2) {
		t.Error("unmarshal(marshal(x)) != x")
	}
}

type AnEnum uint16

const (
	AnEnum_0 = AnEnum(iota)
	AnEnum_1
	AnEnum_2
)

func (*AnEnum) AsProtobuf3() (string, string) {
	return "AnEnum", `enum AnEnum {
  AnEnum_0 = 0;
  AnEnum_1 = 1;
  AnEnum_2 = 2;
}`
}

type EnumMsg struct {
	E AnEnum `protobuf:"varint,1"`
}

func TestCustomEnum(t *testing.T) {
	m := EnumMsg{
		E: AnEnum_1,
	}
	b, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error(err)
	}

	// b should be varint(1)
	buf := protobuf3.NewBuffer(nil)
	buf.EncodeVarint(1 << 3)
	buf.EncodeVarint(1)
	b2 := buf.Bytes()

	if !bytes.Equal(b, b2) {
		t.Errorf("unexpectd encoding %x != %x", b, b2)
	}

	var m2 EnumMsg
	err = protobuf3.Unmarshal(b, &m2)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(&m, &m2) {
		t.Error("unmarshal(marshal(x)) != x")
	}

	s := protobuf3.AsProtobuf(reflect.TypeOf(m))
	t.Log(s)
	nm, ap := m.E.AsProtobuf3()
	s2 := "message EnumMsg {\n  " + nm + " e = 1;\n}"
	if s != s2 {
		t.Errorf("AsProtobuf unexpected: %q != %q", s, s2)
	}

	f := protobuf3.AsProtobufFull(reflect.TypeOf(m))
	t.Log(f)
	if !strings.Contains(f, ap) {
		t.Errorf("AsProtobufFull doesn't define type AnEnum:\n%s", f)
	}
}

func TestVarint(t *testing.T) {
	var pb, pba []byte
	var err error
	var x uint64

	for _, pad := range [][]byte{nil, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}} {
		// exhaustively test the first 2M varints because we can, and it doesn't take long at all
		wb := protobuf3.NewBuffer(make([]byte, 0, 10))
		for i := uint64(0); i < 128; i++ {
			pb = []byte{byte(i)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128); i < 128*128; i++ {
			pb = []byte{byte(i&0x7f | 0x80), byte(i >> 7)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128 * 128); i < 128*128*128; i++ {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte(i >> 14)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		// spotcheck some larger varints
		for i := uint64(128 * 128 * 128); i < 128*128*128*128; i += 3 * 127 {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte((i>>14)&0x7f | 0x80), byte(i >> 21)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128 * 128 * 128 * 128); i < 128*128*128*128*128; i += 3 * 127 * 127 {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte((i>>14)&0x7f | 0x80), byte((i>>21)&0x7f | 0x80), byte(i >> 28)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128 * 128 * 128 * 128 * 128); i < 128*128*128*128*128*128; i += 3 * 127 * 127 * 127 {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte((i>>14)&0x7f | 0x80), byte((i>>21)&0x7f | 0x80),
				byte((i>>28)&0x7f | 0x80), byte(i >> 35)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128 * 128 * 128 * 128 * 128 * 128); i < 128*128*128*128*128*128*128; i += 3 * 127 * 127 * 127 * 127 {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte((i>>14)&0x7f | 0x80), byte((i>>21)&0x7f | 0x80),
				byte((i>>28)&0x7f | 0x80), byte((i>>35)&0x7f | 0x80), byte(i >> 42)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128 * 128 * 128 * 128 * 128 * 128 * 128); i < 128*128*128*128*128*128*128*128; i += 3 * 127 * 127 * 127 * 127 * 127 {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte((i>>14)&0x7f | 0x80), byte((i>>21)&0x7f | 0x80),
				byte((i>>28)&0x7f | 0x80), byte((i>>35)&0x7f | 0x80), byte((i>>42)&0x7f | 0x80), byte(i >> 49)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128 * 128 * 128 * 128 * 128 * 128 * 128 * 128); i < 128*128*128*128*128*128*128*128*128; i += 3 * 127 * 127 * 127 * 127 * 127 * 127 {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte((i>>14)&0x7f | 0x80), byte((i>>21)&0x7f | 0x80),
				byte((i>>28)&0x7f | 0x80), byte((i>>35)&0x7f | 0x80), byte((i>>42)&0x7f | 0x80), byte((i>>49)&0x7f | 0x80), byte(i >> 56)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		for i := uint64(128 * 128 * 128 * 128 * 128 * 128 * 128 * 128 * 128); i+1 > 0x7fffffffffffffff; i += 3 * 127 * 127 * 127 * 127 * 127 * 127 * 127 {
			pb = []byte{byte(i&0x7f | 0x80), byte((i>>7)&0x7f | 0x80), byte((i>>14)&0x7f | 0x80), byte((i>>21)&0x7f | 0x80),
				byte((i>>28)&0x7f | 0x80), byte((i>>35)&0x7f | 0x80), byte((i>>42)&0x7f | 0x80), byte((i>>49)&0x7f | 0x80),
				byte((i>>56)&0x7f | 0x80), byte(i >> 63)}
			pba = append(pb, pad...)
			b := protobuf3.NewBuffer(pba)
			x, err = b.DecodeVarint()
			if err != nil {
				t.Error(err)
				return
			}
			if x != i {
				t.Errorf("DecodeVarint(% x (i=%d)) => %d", pba, i, x)
				return
			}

			wb.Reset()
			wb.EncodeVarint(i)
			if !bytes.Equal(wb.Bytes(), pb) {
				t.Errorf("EncodeVarint(%d) => % x; expected % x", i, wb.Bytes(), pb)
				return
			}
		}

		// check 1<<63
		pb = []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01}
		pba = append(pb, pad...)
		x, err = protobuf3.NewBuffer(pba).DecodeVarint()
		if err != nil {
			t.Error(err)
			return
		} else if x != 1<<63 {
			t.Errorf("DecodeVarint(% x) => 0x%x; expected 1<<63", pba, x)
			return
		}

		// check overflow is caught
		pb = []byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x02}
		pba = append(pb, pad...)
		_, err = protobuf3.NewBuffer(pba).DecodeVarint()
		if err == nil {
			t.Errorf("DecodeVarint(% x) didn't detect overflow", pba)
			return
		}

		pb = []byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8a, 0x0b}
		pba = append(pb, pad...)
		_, err = protobuf3.NewBuffer(pba).DecodeVarint()
		if err == nil {
			t.Errorf("DecodeVarint(% x) didn't detect overflow", pba)
			return
		}
	}

	// check truncation is caught
	pb = []byte{0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8a, 0x8b}
	for i := range pb {
		_, err = protobuf3.NewBuffer(pb[:i]).DecodeVarint()
		if err == nil {
			t.Errorf("DecodeVarint(% x) didn't detect truncation", pb[:i])
			return
		}
	}
}

type ReservedMsg struct {
	protobuf3.Reserved `protobuf:"2,3"`
	X                  uint32             `protobuf:"varint,4"`
	_                  protobuf3.Reserved `protobuf:"5"`
	MyReserved         protobuf3.Reserved `protobuf:"1,6"`
	Y                  uint32             `protobuf:"varint,7"`
}

func TestReserved(t *testing.T) {
	ox := unsafe.Offsetof(ReservedMsg{}.X)
	oy := unsafe.Offsetof(ReservedMsg{}.Y)
	sz := unsafe.Sizeof(ReservedMsg{})
	if ox != 0 || oy != 4 || sz != 8 {
		t.Errorf("unexpected offsets or sizeof struct with protobuf3.Reserved fields: %d, %d, %d", ox, oy, sz)
	}

	var m = ReservedMsg{
		X: 0,
		Y: 1,
	}
	b, err := protobuf3.Marshal(&m)
	if err != nil {
		t.Error(err)
	}

	// we should have encoded only Y
	buf := protobuf3.NewBuffer(nil)
	buf.EncodeVarint(7 << 3)
	buf.EncodeVarint(1)
	b2 := buf.Bytes()

	if !bytes.Equal(b, b2) {
		t.Errorf("unexpectd encoding %x != %x", b, b2)
	}

	var m2 ReservedMsg
	err = protobuf3.Unmarshal(b, &m2)
	if err != nil {
		t.Error(err)
	}

	s := protobuf3.AsProtobuf(reflect.TypeOf(m))
	t.Log(s)
	if s != `message ReservedMsg {
  uint32 x = 4;
  uint32 y = 7;
  reserved 1, 2, 3, 5, 6;
}` {
		t.Errorf("unexpected AsProtobuf result with reserved fields:\n%s\n", s)

	}
}

type MsgWithOptionalFields struct {
	s    *string  `protobuf:"bytes,1,optional"`
	b    *bool    `protobuf:"varint,2,optional"`

	vi32 *int32   `protobuf:"varint,3,optional"`
	vu32 *uint32  `protobuf:"varint,4,optional"`
	vi64 *int32   `protobuf:"varint,5,optional"`
	vu64 *uint32  `protobuf:"varint,6,optional"`

	fi32 *int32   `protobuf:"fixed32,7,optional"`
	fu32 *uint32  `protobuf:"fixed32,8,optional"`
	fi64 *int64   `protobuf:"fixed64,9,optional"`
	fu64 *uint64  `protobuf:"fixed64,10,optional"`
	ff32 *float32 `protobuf:"fixed32,11,optional"`
	ff64 *float64 `protobuf:"fixed64,12,optional"`
}

func (*MsgWithOptionalFields) ProtoMessage()    {}
func (m *MsgWithOptionalFields) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgWithOptionalFields) Reset()         { *m = MsgWithOptionalFields{} }

type MsgWithoutOptionalFields struct {
	s    string  `protobuf:"bytes,1"`
	b    bool    `protobuf:"varint,2"`

	vi32 int32   `protobuf:"varint,3"`
	vu32 uint32  `protobuf:"varint,4"`
	vi64 int32   `protobuf:"varint,5"`
	vu64 uint32  `protobuf:"varint,6"`

	fi32 int32   `protobuf:"fixed32,7"`
	fu32 uint32  `protobuf:"fixed32,8"`
	fi64 int64   `protobuf:"fixed64,9"`
	fu64 uint64  `protobuf:"fixed64,10"`
	ff32 float32 `protobuf:"fixed32,11"`
	ff64 float64 `protobuf:"fixed64,12"`
}

func (*MsgWithoutOptionalFields) ProtoMessage()    {}
func (m *MsgWithoutOptionalFields) String() string { return fmt.Sprintf("%+v", *m) }
func (m *MsgWithoutOptionalFields) Reset()         { *m = MsgWithoutOptionalFields{} }

func TestOptionalField(t *testing.T) {
	m := MsgWithOptionalFields{}
	f := protobuf3.AsProtobufFull(reflect.TypeOf(m))
	t.Log("\n" + f)
	r, _ := regexp.Compile(` = \d+;$`)
	for _, line := range strings.Split(f, "\n") {
		if r.MatchString(line) {
			if !strings.Contains(line, "optional ") {
				t.Errorf("Missing `optional` keyword: %s", line)
			}
		}
	}

	check(&m, &m, t)
	var mb MsgWithOptionalFields
	uncheck(&m, &mb, nil, t)
	eq("mb", mb, m, t)

	b, _ := protobuf3.Marshal(&m)
	if len(b) != 0 {
		t.Errorf("Marshalling optional fields pointing to nil: expect 0 bytes, got %x", b)
	}

	if true {
		m1 := MsgWithoutOptionalFields{}

		m.s = &m1.s
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.s = nil

		m.b = &m1.b
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.b = nil

		m.vi32 = &m1.vi32
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.vi32 = nil

		m.vu32 = &m1.vu32
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.vu32 = nil

		m.vi64 = &m1.vi64
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.vi64 = nil

		m.vu64 = &m1.vu64
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.vu64 = nil

		m.fi32 = &m1.fi32
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.fi32 = nil

		m.fu32 = &m1.fu32
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.fu32 = nil

		m.fi64 = &m1.fi64
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.fi64 = nil

		m.fu64 = &m1.fu64
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.fu64 = nil

		m.ff32 = &m1.ff32
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.ff32 = nil

		m.ff64 = &m1.ff64
		b, _ = protobuf3.Marshal(&m)
		if len(b) == 0 {
			t.Error("Marshalling optional field pointing to zero value: expect some bytes, got 0 bytes")
		}
		m.ff64 = nil
	}

	m2 := MsgWithoutOptionalFields{
		s   : "abc",
		b   : true,
		vi32: 1,
		vu32: 1,
		vi64: 1,
		vu64: 1,
		fi32: 1,
		fu32: 1,
		fi64: 1,
		fu64: 1,
		ff32: 1.0,
		ff64: 1.0,
	}

	if true {
		m.s    = &m2.s
		m.b    = &m2.b
		m.vi32 = &m2.vi32
		m.vu32 = &m2.vu32
		m.vi64 = &m2.vi64
		m.vu64 = &m2.vu64
		m.fi32 = &m2.fi32
		m.fu32 = &m2.fu32
		m.fi64 = &m2.fi64
		m.fu64 = &m2.fu64
		m.ff32 = &m2.ff32
		m.ff64 = &m2.ff64

		check(&m, &m, t)
		var mc MsgWithOptionalFields
		uncheck(&m, &mc, nil, t)
		m2_ := MsgWithoutOptionalFields{
			s   : *mc.s,
			b   : *mc.b,
			vi32: *mc.vi32,
			vu32: *mc.vu32,
			vi64: *mc.vi64,
			vu64: *mc.vu64,
			fi32: *mc.fi32,
			fu32: *mc.fu32,
			fi64: *mc.fi64,
			fu64: *mc.fu64,
			ff32: *mc.ff32,
			ff64: *mc.ff64,
		}
		eq("m2_", m2_, m2, t)

		// Test backward compatibility -- message generated from struct with optional fields, can be unmarshal-ed as struct without optional fields
		var m3 MsgWithoutOptionalFields
		uncheck(&m, &m3, nil, t)
		eq("m3", m3, m2, t)
	}


	if true {
		check(&m2, &m2, t)
		var m3 MsgWithoutOptionalFields
		uncheck(&m2, &m3, nil, t)
		eq("m3", m3, m2, t)

		// Test backward compatibility -- message generated from struct without optional fields, can be unmarshal-ed as struct with optional fields
		var mc MsgWithOptionalFields
		uncheck(&m2, &mc, nil, t)
		m2_ := MsgWithoutOptionalFields{
			s   : *mc.s,
			b   : *mc.b,
			vi32: *mc.vi32,
			vu32: *mc.vu32,
			vi64: *mc.vi64,
			vu64: *mc.vu64,
			fi32: *mc.fi32,
			fu32: *mc.fu32,
			fi64: *mc.fi64,
			fu64: *mc.fu64,
			ff32: *mc.ff32,
			ff64: *mc.ff64,
		}
		eq("m2_", m2_, m2, t)
	}
}
