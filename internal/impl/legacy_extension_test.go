// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl_test

import (
	"reflect"
	"testing"

	pimpl "google.golang.org/protobuf/internal/impl"
	ptype "google.golang.org/protobuf/internal/prototype"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"

	proto2_20180125 "google.golang.org/protobuf/internal/testprotos/legacy/proto2.v1.0.0-20180125-92554152"
)

type legacyExtendedMessage struct {
	XXX_unrecognized       []byte
	XXX_InternalExtensions map[int32]pimpl.ExtensionFieldV1
}

func (*legacyExtendedMessage) Reset()         {}
func (*legacyExtendedMessage) String() string { return "" }
func (*legacyExtendedMessage) ProtoMessage()  {}
func (*legacyExtendedMessage) ExtensionRangeArray() []piface.ExtensionRangeV1 {
	return []piface.ExtensionRangeV1{{Start: 10000, End: 20000}}
}

func mustMakeExtensionType(x *ptype.StandaloneExtension, v interface{}) pref.ExtensionType {
	xd, err := ptype.NewExtension(x)
	if err != nil {
		panic(err)
	}
	return pimpl.LegacyExtensionTypeOf(xd, reflect.TypeOf(v))
}

var (
	extParentDesc    = pimpl.Export{}.MessageDescriptorOf((*legacyExtendedMessage)(nil))
	extMessageV1Desc = pimpl.Export{}.MessageDescriptorOf((*proto2_20180125.Message_ChildMessage)(nil))

	wantType = mustMakeExtensionType(&ptype.StandaloneExtension{
		FullName:     "fizz.buzz.optional_message_v1",
		Number:       10007,
		Cardinality:  pref.Optional,
		Kind:         pref.MessageKind,
		MessageType:  extMessageV1Desc,
		ExtendedType: extParentDesc,
	}, (*proto2_20180125.Message_ChildMessage)(nil))
	wantDesc = &piface.ExtensionDescV1{
		ExtendedType:  (*legacyExtendedMessage)(nil),
		ExtensionType: (*proto2_20180125.Message_ChildMessage)(nil),
		Field:         10007,
		Name:          "fizz.buzz.optional_message_v1",
		Tag:           "bytes,10007,opt,name=optional_message_v1",
	}
)

func BenchmarkConvert(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		xd := pimpl.Export{}.ExtensionDescFromType(wantType)
		gotType := pimpl.Export{}.ExtensionTypeFromDesc(xd)
		if gotType != wantType {
			b.Fatalf("ExtensionType mismatch: got %p, want %p", gotType, wantType)
		}

		xt := pimpl.Export{}.ExtensionTypeFromDesc(wantDesc)
		gotDesc := pimpl.Export{}.ExtensionDescFromType(xt)
		if gotDesc != wantDesc {
			b.Fatalf("ExtensionDesc mismatch: got %p, want %p", gotDesc, wantDesc)
		}
	}
}
