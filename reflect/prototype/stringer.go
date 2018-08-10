// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"

	"google.golang.org/proto/internal/pragma"
	pref "google.golang.org/proto/reflect/protoreflect"
)

// TODO: This is useful for print descriptor types in a human readable way.
// This is not strictly necessary.

// list is an interface that matches any of the list interfaces defined in the
// protoreflect package.
type list interface {
	Len() int
	pragma.DoNotImplement
}

func formatList(s fmt.State, r rune, vs list) {}

func formatDesc(s fmt.State, r rune, t pref.Descriptor) {}
