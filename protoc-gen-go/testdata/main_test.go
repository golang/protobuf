// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// A simple binary to link together the protocol buffers in this test.

package testdata

import (
	"testing"

	importspb "github.com/golang/protobuf/protoc-gen-go/testdata/imports"
	multipb "github.com/golang/protobuf/protoc-gen-go/testdata/multi"
	mytestpb "github.com/golang/protobuf/protoc-gen-go/testdata/my_test"
)

func TestLink(t *testing.T) {
	_ = &multipb.Multi1{}
	_ = &mytestpb.Request{}
	_ = &importspb.All{}
}
