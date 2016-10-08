// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2016 Mist Systems. All rights reserved.
//
// Unlike most files, this one is entirely by Mist, and not derived
// from any earlier code.
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
	"testing"

	"github.com/mistsys/protobuf3/proto"
	"github.com/mistsys/protobuf3/protobuf3"
)

func BenchmarkFixedMsg(b *testing.B) {
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

	_, err := protobuf3.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protobuf3.Marshal(&m)
	}
}

func BenchmarkOldFixedMsg(b *testing.B) {
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

	_, err := proto.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proto.Marshal(&m)
	}
}

func BenchmarkVarMsg(b *testing.B) {
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

	_, err := protobuf3.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protobuf3.Marshal(&m)
	}
}

func BenchmarkOldVarMsg(b *testing.B) {
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

	_, err := proto.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proto.Marshal(&m)
	}
}

func BenchmarkBytesMsg(b *testing.B) {
	s := "str"

	m := BytesMsg{
		s:  "test1",
		ps: &s,
		ss: []string{"test3", "test4"},
		sb: []byte{3, 2, 1, 0},
	}

	_, err := protobuf3.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protobuf3.Marshal(&m)
	}
}

func BenchmarkOldBytesMsg(b *testing.B) {
	s := "str"

	m := BytesMsg{
		s:  "test1",
		ps: &s,
		ss: []string{"test3", "test4"},
		sb: []byte{3, 2, 1, 0},
	}

	_, err := proto.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proto.Marshal(&m)
	}
}

func BenchmarkNestedPtrStructMsg(b *testing.B) {
	// note: value is chosen so it can be compared with the equivalent non-pointer nesting
	m := NestedPtrStructMsg{
		first:  &InnerMsg{0x11},
		second: &InnerMsg{0x22},
		many:   []*InnerMsg{&InnerMsg{0x33}},
		more:   []*InnerMsg{&InnerMsg{0x44}, &InnerMsg{0x55}, &InnerMsg{0x66}},
	}

	_, err := protobuf3.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protobuf3.Marshal(&m)
	}
}

func BenchmarkOldNestedPtrStructMsg(b *testing.B) {
	m := NestedPtrStructMsg{
		first:  &InnerMsg{0x11},
		second: &InnerMsg{0x22},
		many:   []*InnerMsg{&InnerMsg{0x33}},
		more:   []*InnerMsg{&InnerMsg{0x44}, &InnerMsg{0x55}, &InnerMsg{0x66}},
		some:   []*InnerMsg{&InnerMsg{0x77}},
	}

	_, err := proto.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proto.Marshal(&m)
	}
}

func BenchmarkNestedStructMsg(b *testing.B) {
	// note: value matches that of NestedPtrStructMsg above, for comparison between the
	// pointer-to-nested-struct and embedded-nested-struct cases
	m := NestedStructMsg{
		first:  InnerMsg{0x11},
		second: InnerMsg{0x22},
		many:   []InnerMsg{InnerMsg{0x33}},
		more:   [3]InnerMsg{InnerMsg{0x44}, InnerMsg{0x55}, InnerMsg{0x66}},
		some:   [1]*InnerMsg{&InnerMsg{0x77}},
	}

	_, err := protobuf3.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protobuf3.Marshal(&m)
	}
}

func BenchmarkMapMsg(b *testing.B) {
	m := MapMsg{
		m: map[string]int32{
			"Nic":     0,
			"Michele": 1,
		},
	}

	_, err := protobuf3.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		protobuf3.Marshal(&m)
	}
}

func BenchmarkOldMapMsg(b *testing.B) {
	m := MapMsg{
		m: map[string]int32{
			"Nic":     0,
			"Michele": 1,
		},
	}

	_, err := proto.Marshal(&m)
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proto.Marshal(&m)
	}
}
