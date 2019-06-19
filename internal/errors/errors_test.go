// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package errors

import (
	"strings"
	"testing"
)

func TestNewPrefix(t *testing.T) {
	e1 := New("abc")
	got := e1.Error()
	if !strings.HasPrefix(got, "proto:") {
		t.Errorf("missing \"proto:\" prefix in %q", got)
	}
	if !strings.Contains(got, "abc") {
		t.Errorf("missing text \"abc\" in %q", got)
	}

	e2 := New("%v", e1)
	got = e2.Error()
	if !strings.HasPrefix(got, "proto:") {
		t.Errorf("missing \"proto:\" prefix in %q", got)
	}
	// Test to make sure prefix is removed from the embedded error.
	if strings.Contains(strings.TrimPrefix(got, "proto:"), "proto:") {
		t.Errorf("prefix \"proto:\" not elided in embedded error: %q", got)
	}
}
