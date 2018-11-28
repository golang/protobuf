// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptypes

import (
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/any"
)

func TestMarshalUnmarshal(t *testing.T) {
	orig := &any.Any{Value: []byte("test")}

	packed, err := MarshalAny(orig)
	if err != nil {
		t.Errorf("MarshalAny(%+v): got: _, %v exp: _, nil", orig, err)
	}

	unpacked := &any.Any{}
	err = UnmarshalAny(packed, unpacked)
	if err != nil || !proto.Equal(unpacked, orig) {
		t.Errorf("got: %v, %+v; want nil, %+v", err, unpacked, orig)
	}
}

func TestIs(t *testing.T) {
	a, err := MarshalAny(&pb.FileDescriptorProto{})
	if err != nil {
		t.Fatal(err)
	}
	if Is(a, &pb.DescriptorProto{}) {
		// No spurious match for message names of different length.
		t.Error("FileDescriptorProto is not a DescriptorProto, but Is says it is")
	}
	if Is(a, &pb.EnumDescriptorProto{}) {
		// No spurious match for message names of equal length.
		t.Error("FileDescriptorProto is not an EnumDescriptorProto, but Is says it is")
	}
	if !Is(a, &pb.FileDescriptorProto{}) {
		t.Error("FileDescriptorProto is indeed a FileDescriptorProto, but Is says it is not")
	}
}

func TestIsDifferentUrlPrefixes(t *testing.T) {
	m := &pb.FileDescriptorProto{}
	a := &any.Any{TypeUrl: "foo/bar/" + proto.MessageName(m)}
	if !Is(a, m) {
		t.Errorf("message with type url %q didn't satisfy Is for type %q", a.TypeUrl, proto.MessageName(m))
	}
}

func TestIsCornerCases(t *testing.T) {
	m := &pb.FileDescriptorProto{}
	if Is(nil, m) {
		t.Errorf("message with nil type url incorrectly claimed to be %q", proto.MessageName(m))
	}
	noPrefix := &any.Any{TypeUrl: proto.MessageName(m)}
	if Is(noPrefix, m) {
		t.Errorf("message with type url %q incorrectly claimed to be %q", noPrefix.TypeUrl, proto.MessageName(m))
	}
	shortPrefix := &any.Any{TypeUrl: "/" + proto.MessageName(m)}
	if !Is(shortPrefix, m) {
		t.Errorf("message with type url %q didn't satisfy Is for type %q", shortPrefix.TypeUrl, proto.MessageName(m))
	}
}

func TestUnmarshalDynamic(t *testing.T) {
	want := &pb.FileDescriptorProto{Name: proto.String("foo")}
	a, err := MarshalAny(want)
	if err != nil {
		t.Fatal(err)
	}
	var got DynamicAny
	if err := UnmarshalAny(a, &got); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(got.Message, want) {
		t.Errorf("invalid result from UnmarshalAny, got %q want %q", got.Message, want)
	}
}

func TestEmpty(t *testing.T) {
	want := &pb.FileDescriptorProto{}
	a, err := MarshalAny(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Empty(a)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(got, want) {
		t.Errorf("unequal empty message, got %q, want %q", got, want)
	}

	// that's a valid type_url for a message which shouldn't be linked into this
	// test binary. We want an error.
	a.TypeUrl = "type.googleapis.com/google.protobuf.FieldMask"
	if _, err := Empty(a); err == nil {
		t.Errorf("got no error for an attempt to create a message of type %q, which shouldn't be linked in", a.TypeUrl)
	}
}

func TestEmptyCornerCases(t *testing.T) {
	_, err := Empty(nil)
	if err == nil {
		t.Error("expected Empty for nil to fail")
	}
	want := &pb.FileDescriptorProto{}
	noPrefix := &any.Any{TypeUrl: proto.MessageName(want)}
	_, err = Empty(noPrefix)
	if err == nil {
		t.Errorf("expected Empty for any type %q to fail", noPrefix.TypeUrl)
	}
	shortPrefix := &any.Any{TypeUrl: "/" + proto.MessageName(want)}
	got, err := Empty(shortPrefix)
	if err != nil {
		t.Errorf("Empty for any type %q failed: %s", shortPrefix.TypeUrl, err)
	}
	if !proto.Equal(got, want) {
		t.Errorf("Empty for any type %q differs, got %q, want %q", shortPrefix.TypeUrl, got, want)
	}
}
