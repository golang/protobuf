// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2016 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package generator

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func TestProto2RepeatedPacked(t *testing.T) {
	g := &Generator{}
	msgDesc := &Descriptor{
		common: common{
			&descriptor.FileDescriptorProto{
				Syntax: proto.String("proto2"),
			},
		},
	}
	fieldType := descriptor.FieldDescriptorProto_TYPE_INT64
	fieldLabel := descriptor.FieldDescriptorProto_LABEL_REPEATED
	result := g.goTag(msgDesc, &descriptor.FieldDescriptorProto{
		Type:  &fieldType,
		Label: &fieldLabel,
	}, "int64")
	if strings.Contains(result, ",packed") {
		t.Errorf("repeated primitives should not be packed as default in proto2")
	}
}

func TestProto3RepeatedPacked(t *testing.T) {
	g := &Generator{}
	msgDesc := &Descriptor{
		common: common{
			&descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
		},
	}
	fieldType := descriptor.FieldDescriptorProto_TYPE_INT64
	fieldLabel := descriptor.FieldDescriptorProto_LABEL_REPEATED
	result := g.goTag(msgDesc, &descriptor.FieldDescriptorProto{
		Type:  &fieldType,
		Label: &fieldLabel,
	}, "int64")
	if !strings.Contains(result, ",packed") {
		t.Errorf("repeated primitives should be packed as default in proto3")
	}
}

func TestProto3RepeatedPackedOptions(t *testing.T) {
	g := &Generator{}
	msgDesc := &Descriptor{
		common: common{
			&descriptor.FileDescriptorProto{
				Syntax: proto.String("proto3"),
			},
		},
	}
	fieldType := descriptor.FieldDescriptorProto_TYPE_INT64
	fieldLabel := descriptor.FieldDescriptorProto_LABEL_REPEATED
	result := g.goTag(msgDesc, &descriptor.FieldDescriptorProto{
		Type:  &fieldType,
		Label: &fieldLabel,
		Options: &descriptor.FieldOptions{
			Packed: proto.Bool(false),
		},
	}, "int64")
	if strings.Contains(result, ",packed") {
		t.Errorf("got %s, expected without packed", result)
	}
}
