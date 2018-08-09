// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set

import (
	"math/rand"
	"testing"
)

const maxLimit = 1024

var toSet, toClear [maxLimit]bool

func init() {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < maxLimit; i++ {
		toSet[i] = r.Intn(2) == 0
		toClear[i] = r.Intn(2) == 0
	}
}

func TestInts(t *testing.T) {
	type set interface {
		Len() int
		Has(n uint64) bool
		Set(n uint64)
		Clear(n uint64)
	}

	tests := []struct {
		label   string
		makeSet func() set
		limit   int
	}{
		{label: "Int32s", makeSet: func() set { return new(Int32s) }, limit: 32},
		{label: "Int64s", makeSet: func() set { return new(Int64s) }, limit: 64},
		{label: "Ints", makeSet: func() set { return new(Ints) }, limit: maxLimit},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			ns := tt.makeSet()

			// Check that set starts empty.
			wantLen := 0
			if ns.Len() != wantLen {
				t.Errorf("init: Len() = %d, want %d", ns.Len(), wantLen)
			}
			for i := 0; i < tt.limit; i++ {
				if ns.Has(uint64(i)) {
					t.Errorf("init: Has(%d) = true, want false", i)
				}
			}

			// Set some numbers.
			for i, b := range toSet[:tt.limit] {
				if b {
					ns.Set(uint64(i))
					wantLen++
				}
			}

			// Check that integers were set.
			if ns.Len() != wantLen {
				t.Errorf("after Set: Len() = %d, want %d", ns.Len(), wantLen)
			}
			for i := 0; i < tt.limit; i++ {
				if got := ns.Has(uint64(i)); got != toSet[i] {
					t.Errorf("after Set: Has(%d) = %v, want %v", i, got, !got)
				}
			}

			// Clear some numbers.
			for i, b := range toClear[:tt.limit] {
				if b {
					ns.Clear(uint64(i))
					if toSet[i] {
						wantLen--
					}
				}
			}

			// Check that integers were cleared.
			if ns.Len() != wantLen {
				t.Errorf("after Clear: Len() = %d, want %d", ns.Len(), wantLen)
			}
			for i := 0; i < tt.limit; i++ {
				if got := ns.Has(uint64(i)); got != toSet[i] && !toClear[i] {
					t.Errorf("after Clear: Has(%d) = %v, want %v", i, got, !got)
				}
			}
		})
	}
}
