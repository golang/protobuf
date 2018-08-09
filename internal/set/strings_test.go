// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set

import (
	"strconv"
	"testing"
)

func TestStrings(t *testing.T) {
	var ss Strings

	// Check that set starts empty.
	wantLen := 0
	if ss.Len() != wantLen {
		t.Errorf("init: Len() = %d, want %d", ss.Len(), wantLen)
	}
	for i := 0; i < maxLimit; i++ {
		if ss.Has(strconv.Itoa(i)) {
			t.Errorf("init: Has(%d) = true, want false", i)
		}
	}

	// Set some strings.
	for i, b := range toSet[:maxLimit] {
		if b {
			ss.Set(strconv.Itoa(i))
			wantLen++
		}
	}

	// Check that strings were set.
	if ss.Len() != wantLen {
		t.Errorf("after Set: Len() = %d, want %d", ss.Len(), wantLen)
	}
	for i := 0; i < maxLimit; i++ {
		if got := ss.Has(strconv.Itoa(i)); got != toSet[i] {
			t.Errorf("after Set: Has(%d) = %v, want %v", i, got, !got)
		}
	}

	// Clear some strings.
	for i, b := range toClear[:maxLimit] {
		if b {
			ss.Clear(strconv.Itoa(i))
			if toSet[i] {
				wantLen--
			}
		}
	}

	// Check that strings were cleared.
	if ss.Len() != wantLen {
		t.Errorf("after Clear: Len() = %d, want %d", ss.Len(), wantLen)
	}
	for i := 0; i < maxLimit; i++ {
		if got := ss.Has(strconv.Itoa(i)); got != toSet[i] && !toClear[i] {
			t.Errorf("after Clear: Has(%d) = %v, want %v", i, got, !got)
		}
	}
}
