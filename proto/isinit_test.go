// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto_test

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/v2/internal/scalar"
	"github.com/golang/protobuf/v2/proto"

	testpb "github.com/golang/protobuf/v2/internal/testprotos/test"
)

func TestIsInitializedErrors(t *testing.T) {
	for _, test := range []struct {
		m    proto.Message
		want string
	}{
		{
			&testpb.TestRequired{},
			`proto: required field required_field not set`,
		},
		{
			&testpb.TestRequiredForeign{
				OptionalMessage: &testpb.TestRequired{},
			},
			`proto: required field optional_message.required_field not set`,
		},
		{
			&testpb.TestRequiredForeign{
				RepeatedMessage: []*testpb.TestRequired{
					{RequiredField: scalar.Int32(1)},
					{},
				},
			},
			`proto: required field repeated_message[1].required_field not set`,
		},
		{
			&testpb.TestRequiredForeign{
				MapMessage: map[int32]*testpb.TestRequired{
					1: {},
				},
			},
			`proto: required field map_message[1].required_field not set`,
		},
	} {
		err := proto.IsInitialized(test.m)
		got := "<nil>"
		if err != nil {
			got = fmt.Sprintf("%q", err)
		}
		want := fmt.Sprintf("%q", test.want)
		if got != want {
			t.Errorf("IsInitialized(m):\n got: %v\nwant: %v\nMessage:\n%v", got, want, marshalText(test.m))
		}
	}
}
