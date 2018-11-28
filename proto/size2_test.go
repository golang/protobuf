// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"math"
	"testing"
)

// This is a separate file and package from size_test.go because that one uses
// generated messages and thus may not be in package proto without having a circular
// dependency, whereas this file tests unexported details of size.go.

func TestVarintSize(t *testing.T) {
	// Check the edge cases carefully.
	testCases := []struct {
		n    uint64
		size int
	}{
		{0, 1},
		{1, 1},
		{127, 1},
		{128, 2},
		{16383, 2},
		{16384, 3},
		{math.MaxInt64, 9},
		{math.MaxInt64 + 1, 10},
	}
	for _, tc := range testCases {
		size := SizeVarint(tc.n)
		if size != tc.size {
			t.Errorf("sizeVarint(%d) = %d, want %d", tc.n, size, tc.size)
		}
	}
}
