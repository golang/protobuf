// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb_test

import (
	"testing"

	"github.com/golang/protobuf/v2/encoding/jsonpb"
	knownpb "github.com/golang/protobuf/v2/types/known"
)

func BenchmarkUnmarshal_Duration(b *testing.B) {
	input := []byte(`"-123456789.123456789s"`)

	for i := 0; i < b.N; i++ {
		err := jsonpb.Unmarshal(&knownpb.Duration{}, input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
