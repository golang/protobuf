// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	"testing"

	"github.com/golang/protobuf/v2/internal/encoding/json"
)

func BenchmarkFloat(b *testing.B) {
	input := []byte(`1.797693134862315708145274237317043567981e+308`)
	for i := 0; i < b.N; i++ {
		dec := json.NewDecoder(input)
		val, err := dec.Read()
		if err != nil {
			b.Fatal(err)
		}
		_, err = val.Float(64)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInt(b *testing.B) {
	input := []byte(`922337203.6854775807e+10`)
	for i := 0; i < b.N; i++ {
		dec := json.NewDecoder(input)
		val, err := dec.Read()
		if err != nil {
			b.Fatal(err)
		}
		_, err = val.Int(64)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkString(b *testing.B) {
	input := []byte(`"abcdefghijklmnopqrstuvwxyz0123456789\\n\\t"`)
	for i := 0; i < b.N; i++ {
		dec := json.NewDecoder(input)
		val, err := dec.Read()
		if err != nil {
			b.Fatal(err)
		}
		_ = val.String()
	}
}

func BenchmarkBool(b *testing.B) {
	input := []byte(`true`)
	for i := 0; i < b.N; i++ {
		dec := json.NewDecoder(input)
		val, err := dec.Read()
		if err != nil {
			b.Fatal(err)
		}
		_, err = val.Bool()
		if err != nil {
			b.Fatal(err)
		}
	}
}
