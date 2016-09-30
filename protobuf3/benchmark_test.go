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
