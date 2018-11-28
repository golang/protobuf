// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"strconv"
	"testing"

	"github.com/golang/protobuf/proto"
	tpb "github.com/golang/protobuf/proto/proto3_proto"
	"github.com/golang/protobuf/ptypes"
)

var (
	blackhole []byte
)

// BenchmarkAny creates increasingly large arbitrary Any messages.  The type is always the
// same.
func BenchmarkAny(b *testing.B) {
	data := make([]byte, 1<<20)
	quantum := 1 << 10
	for i := uint(0); i <= 10; i++ {
		b.Run(strconv.Itoa(quantum<<i), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				inner := &tpb.Message{
					Data: data[:quantum<<i],
				}
				outer, err := ptypes.MarshalAny(inner)
				if err != nil {
					b.Error("wrong encode", err)
				}
				raw, err := proto.Marshal(&tpb.Message{
					Anything: outer,
				})
				if err != nil {
					b.Error("wrong encode", err)
				}
				blackhole = raw
			}
		})
	}
}

// BenchmarkEmpy measures the overhead of doing the minimal possible encode.
func BenchmarkEmpy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		raw, err := proto.Marshal(&tpb.Message{})
		if err != nil {
			b.Error("wrong encode", err)
		}
		blackhole = raw
	}
}
