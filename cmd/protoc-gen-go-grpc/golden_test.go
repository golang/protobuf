// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !race

package main

import (
	"flag"
	"testing"

	"github.com/golang/protobuf/v2/internal/protogen/goldentest"
)

// Set --regenerate to regenerate the golden files.
var regenerate = flag.Bool("regenerate", false, "regenerate golden files")

func init() {
	goldentest.Plugin(main)
}

func TestGolden(t *testing.T) {
	goldentest.Run(t, *regenerate)
}
