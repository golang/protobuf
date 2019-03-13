// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy_test

import (
	"testing"

	papi "github.com/golang/protobuf/protoapi"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	ptype "github.com/golang/protobuf/v2/internal/prototype"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"

	// The legacy package must be imported prior to use of any legacy messages.
	// TODO: Remove this when protoV1 registers these hooks for you.
	plegacy "github.com/golang/protobuf/v2/internal/legacy"

	proto2_20180125 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v1.0.0-20180125-92554152"
)

type legacyTestMessage struct {
	XXX_unrecognized []byte
	papi.XXX_InternalExtensions
}

func (*legacyTestMessage) Reset()         {}
func (*legacyTestMessage) String() string { return "" }
func (*legacyTestMessage) ProtoMessage()  {}
func (*legacyTestMessage) ExtensionRangeArray() []papi.ExtensionRange {
	return []papi.ExtensionRange{{Start: 10000, End: 20000}}
}

func mustMakeExtensionType(x *ptype.StandaloneExtension, v interface{}) pref.ExtensionType {
	xd, err := ptype.NewExtension(x)
	if err != nil {
		panic(xd)
	}
	return pimpl.Export{}.ExtensionTypeOf(xd, v)
}

var (
	parentType    = pimpl.Export{}.MessageTypeOf((*legacyTestMessage)(nil))
	messageV1Type = pimpl.Export{}.MessageTypeOf((*proto2_20180125.Message_ChildMessage)(nil))

	wantType = mustMakeExtensionType(&ptype.StandaloneExtension{
		FullName:     "fizz.buzz.optional_message_v1",
		Number:       10007,
		Cardinality:  pref.Optional,
		Kind:         pref.MessageKind,
		MessageType:  messageV1Type,
		ExtendedType: parentType,
	}, (*proto2_20180125.Message_ChildMessage)(nil))
	wantDesc = &papi.ExtensionDesc{
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*proto2_20180125.Message_ChildMessage)(nil),
		Field:         10007,
		Name:          "fizz.buzz.optional_message_v1",
		Tag:           "bytes,10007,opt,name=optional_message_v1",
	}
)

func BenchmarkConvert(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		xd := plegacy.Export{}.ExtensionDescFromType(wantType)
		gotType := plegacy.Export{}.ExtensionTypeFromDesc(xd)
		if gotType != wantType {
			b.Fatalf("ExtensionType mismatch: got %p, want %p", gotType, wantType)
		}

		xt := plegacy.Export{}.ExtensionTypeFromDesc(wantDesc)
		gotDesc := plegacy.Export{}.ExtensionDescFromType(xt)
		if gotDesc != wantDesc {
			b.Fatalf("ExtensionDesc mismatch: got %p, want %p", gotDesc, wantDesc)
		}
	}
}
