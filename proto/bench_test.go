// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"

	benchpb "google.golang.org/protobuf/internal/testprotos/benchmarks"
	_ "google.golang.org/protobuf/internal/testprotos/benchmarks/datasets/google_message1/proto2"
	_ "google.golang.org/protobuf/internal/testprotos/benchmarks/datasets/google_message1/proto3"
	_ "google.golang.org/protobuf/internal/testprotos/benchmarks/datasets/google_message2"
	_ "google.golang.org/protobuf/internal/testprotos/benchmarks/datasets/google_message3"
	_ "google.golang.org/protobuf/internal/testprotos/benchmarks/datasets/google_message4"
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

// BenchmarkData runs various benchmarks using the general-purpose protocol buffer
// benchmarking dataset:
// https://github.com/protocolbuffers/protobuf/tree/master/benchmarks
func BenchmarkData(b *testing.B) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		b.Fatal(err)
	}
	repoRoot := strings.TrimSpace(string(out))
	dataDir := filepath.Join(repoRoot, ".cache", "benchdata")

	var datasets []string
	filepath.Walk(dataDir, func(path string, _ os.FileInfo, _ error) error {
		if filepath.Ext(path) == ".pb" {
			datasets = append(datasets, path)
		}
		return nil
	})

	for _, data := range datasets {
		raw, err := ioutil.ReadFile(data)
		if err != nil {
			b.Fatal(err)
		}
		ds := &benchpb.BenchmarkDataset{}
		if err := proto.Unmarshal(raw, ds); err != nil {
			b.Fatal(err)
		}
		mt, err := preg.GlobalTypes.FindMessageByName(pref.FullName(ds.MessageName))
		if err != nil {
			b.Fatal(err)
		}
		var messages []proto.Message
		for _, payload := range ds.Payload {
			m := mt.New().Interface()
			if err := proto.Unmarshal(payload, m); err != nil {
				b.Fatal(err)
			}
			messages = append(messages, m)
		}

		var (
			unmarshal = proto.Unmarshal
			marshal   = proto.Marshal
			size      = proto.Size
		)
		if *benchV1 {
			unmarshal = func(b []byte, m proto.Message) error {
				return protoV1.Unmarshal(b, m.(protoV1.Message))
			}
			marshal = func(m proto.Message) ([]byte, error) {
				return protoV1.Marshal(m.(protoV1.Message))
			}
			size = func(m proto.Message) int {
				return protoV1.Size(m.(protoV1.Message))
			}
		}

		b.Run(filepath.Base(data), func(b *testing.B) {
			b.Run("Unmarshal", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						for _, p := range ds.Payload {
							m := mt.New().Interface()
							if err := unmarshal(p, m); err != nil {
								b.Fatal(err)
							}
						}
					}
				})
			})
			b.Run("Marshal", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						for _, m := range messages {
							if _, err := marshal(m); err != nil {
								b.Fatal(err)
							}
						}
					}
				})
			})
			b.Run("Size", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						for _, m := range messages {
							size(m)
						}
					}
				})
			})
		})
	}
}
