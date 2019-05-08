// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototest_test

import (
	"fmt"
	"testing"

	testpb "github.com/golang/protobuf/v2/internal/testprotos/test"
	test3pb "github.com/golang/protobuf/v2/internal/testprotos/test3"
	"github.com/golang/protobuf/v2/proto"
	"github.com/golang/protobuf/v2/testing/prototest"
)

func Test(t *testing.T) {
	for _, m := range []proto.Message{
		(*testpb.TestAllTypes)(nil),
		(*test3pb.TestAllTypes)(nil),
		(*testpb.TestRequired)(nil),
		(*testpb.TestWeak)(nil),
	} {
		t.Run(fmt.Sprintf("%T", m), func(t *testing.T) {
			prototest.TestMessage(t, m)
		})
	}
}
