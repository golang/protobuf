// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"flag"
	"fmt"
	"reflect"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/proto"
)

// The results of these microbenchmarks are unlikely to correspond well
// to real world peformance. They are mainly useful as a quick check to
// detect unexpected regressions and for profiling specific cases.

var (
	benchV1      = flag.Bool("v1", false, "benchmark the v1 implementation")
	allowPartial = flag.Bool("allow_partial", false, "set AllowPartial")
)

// BenchmarkEncode benchmarks encoding all the test messages.
func BenchmarkEncode(b *testing.B) {
	for _, test := range testProtos {
		for _, want := range test.decodeTo {
			v1 := want.(protoV1.Message)
			opts := proto.MarshalOptions{AllowPartial: *allowPartial}
			b.Run(fmt.Sprintf("%s (%T)", test.desc, want), func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						var err error
						if *benchV1 {
							_, err = protoV1.Marshal(v1)
						} else {
							_, err = opts.Marshal(want)
						}
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			})
		}
	}
}

// BenchmarkDecode benchmarks decoding all the test messages.
func BenchmarkDecode(b *testing.B) {
	for _, test := range testProtos {
		for _, want := range test.decodeTo {
			opts := proto.UnmarshalOptions{AllowPartial: *allowPartial}
			b.Run(fmt.Sprintf("%s (%T)", test.desc, want), func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					m := reflect.New(reflect.TypeOf(want).Elem()).Interface().(proto.Message)
					v1 := m.(protoV1.Message)
					for pb.Next() {
						var err error
						if *benchV1 {
							err = protoV1.Unmarshal(test.wire, v1)
						} else {
							err = opts.Unmarshal(test.wire, m)
						}
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			})
		}
	}
}
