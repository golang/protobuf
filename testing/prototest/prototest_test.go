// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototest_test

import (
	"fmt"
	"testing"

	irregularpb "google.golang.org/protobuf/internal/testprotos/irregular"
	testpb "google.golang.org/protobuf/internal/testprotos/test"
	test3pb "google.golang.org/protobuf/internal/testprotos/test3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/prototest"
)

func Test(t *testing.T) {
	for _, m := range []proto.Message{
		(*testpb.TestAllTypes)(nil),
		(*test3pb.TestAllTypes)(nil),
		(*testpb.TestRequired)(nil),
		(*testpb.TestWeak)(nil),
		(*irregularpb.Message)(nil),
	} {
		t.Run(fmt.Sprintf("%T", m), func(t *testing.T) {
			prototest.TestMessage(t, m)
		})
	}
}
