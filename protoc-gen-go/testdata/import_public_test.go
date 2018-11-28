// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testdata

import (
	"testing"

	mainpb "github.com/golang/protobuf/protoc-gen-go/testdata/import_public"
	subpb "github.com/golang/protobuf/protoc-gen-go/testdata/import_public/sub"
)

func TestImportPublicLink(t *testing.T) {
	// mainpb.[ME] should be interchangable with subpb.[ME].
	var _ mainpb.M = subpb.M{}
	var _ mainpb.E = subpb.E(0)
	_ = &mainpb.Public{
		M: &mainpb.M{},
		E: mainpb.E_ZERO.Enum(),
		Local: &mainpb.Local{
			M: &mainpb.M{},
			E: mainpb.E_ZERO.Enum(),
		},
	}
	_ = &mainpb.Public{
		M: &subpb.M{},
		E: subpb.E_ZERO.Enum(),
		Local: &mainpb.Local{
			M: &subpb.M{},
			E: subpb.E_ZERO.Enum(),
		},
	}
	_ = &mainpb.M{
		M2: &subpb.M2{},
	}
}
