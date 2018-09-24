// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The protoc-gen-go binary is a protoc plugin to generate a Go protocol
// buffer package.
package main

import (
	"github.com/golang/protobuf/v2/cmd/protoc-gen-go/internal_gengo"
)

func main() {
	internal_gengo.Main()
}
