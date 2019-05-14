// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto_test

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"

	descriptorpb "google.golang.org/protobuf/types/descriptor"
)

func TestRegistry(t *testing.T) {
	if got := proto.FileDescriptor("google/protobuf/descriptor.proto"); len(got) == 0 {
		t.Errorf(`FileDescriptor("google/protobuf/descriptor.proto") = empty, want non-empty`)
	}
	if got := proto.EnumValueMap("google.protobuf.FieldDescriptorProto_Label"); len(got) == 0 {
		t.Errorf(`EnumValueMap("google.protobuf.FieldDescriptorProto_Label") = empty, want non-empty`)
	}
	wantType := reflect.TypeOf(new(descriptorpb.EnumDescriptorProto_EnumReservedRange))
	gotType := proto.MessageType("google.protobuf.EnumDescriptorProto.EnumReservedRange")
	if gotType != wantType {
		t.Errorf(`MessageType("google.protobuf.EnumDescriptorProto.EnumReservedRange") = %v, want %v`, gotType, wantType)
	}
}
