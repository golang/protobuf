// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	. "github.com/golang/protobuf/proto/test_proto"
)

func TestUnmarshalMessageSetWithDuplicate(t *testing.T) {
	/*
		Message{
			Tag{1, StartGroup},
			Message{
				Tag{2, Varint}, Uvarint(12345),
				Tag{3, Bytes}, Bytes("hoo"),
			},
			Tag{1, EndGroup},
			Tag{1, StartGroup},
			Message{
				Tag{2, Varint}, Uvarint(12345),
				Tag{3, Bytes}, Bytes("hah"),
			},
			Tag{1, EndGroup},
		}
	*/
	var in []byte
	fmt.Sscanf("0b10b9601a03686f6f0c0b10b9601a036861680c", "%x", &in)

	/*
		Message{
			Tag{1, StartGroup},
			Message{
				Tag{2, Varint}, Uvarint(12345),
				Tag{3, Bytes}, Bytes("hoohah"),
			},
			Tag{1, EndGroup},
		}
	*/
	var want []byte
	fmt.Sscanf("0b10b9601a06686f6f6861680c", "%x", &want)

	var m MyMessageSet
	if err := proto.Unmarshal(in, &m); err != nil {
		t.Fatalf("unexpected Unmarshal error: %v", err)
	}
	got, err := proto.Marshal(&m)
	if err != nil {
		t.Fatalf("unexpected Marshal error: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("output mismatch:\ngot  %x\nwant %x", got, want)
	}
}
