// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"sync"
	"testing"

	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

type (
	MessageA struct {
		A1 *MessageA `protobuf:"bytes,1,req,name=a1"`
		A2 *MessageB `protobuf:"bytes,2,req,name=a2"`
		A3 Enum      `protobuf:"varint,3,opt,name=a3,enum=legacy.Enum"`
	}
	MessageB struct {
		B1 *MessageA `protobuf:"bytes,1,req,name=b1"`
		B2 *MessageB `protobuf:"bytes,2,req,name=b2"`
		B3 Enum      `protobuf:"varint,3,opt,name=b3,enum=legacy.Enum"`
	}
	Enum int32
)

// TestConcurrentInit tests that concurrent wrapping of multiple legacy types
// results in the exact same descriptor being created.
func TestConcurrentInit(t *testing.T) {
	const numParallel = 5
	var messageATypes [numParallel]protoreflect.MessageType
	var messageBTypes [numParallel]protoreflect.MessageType
	var enumTypes [numParallel]protoreflect.EnumType

	// Concurrently load message and enum types.
	var wg sync.WaitGroup
	for i := 0; i < numParallel; i++ {
		i := i
		wg.Add(3)
		go func() {
			defer wg.Done()
			messageATypes[i] = Export{}.MessageTypeOf((*MessageA)(nil))
		}()
		go func() {
			defer wg.Done()
			messageBTypes[i] = Export{}.MessageTypeOf((*MessageB)(nil))
		}()
		go func() {
			defer wg.Done()
			enumTypes[i] = Export{}.EnumTypeOf(Enum(0))
		}()
	}
	wg.Wait()

	var (
		wantMTA = messageATypes[0]
		wantMDA = messageATypes[0].Fields().ByNumber(1).MessageType()
		wantMTB = messageBTypes[0]
		wantMDB = messageBTypes[0].Fields().ByNumber(2).MessageType()
		wantET  = enumTypes[0]
		wantED  = messageATypes[0].Fields().ByNumber(3).EnumType()
	)

	for _, gotMT := range messageATypes[1:] {
		if gotMT != wantMTA {
			t.Error("MessageType(MessageA) mismatch")
		}
		if gotMDA := gotMT.Fields().ByNumber(1).MessageType(); gotMDA != wantMDA {
			t.Error("MessageDescriptor(MessageA) mismatch")
		}
		if gotMDB := gotMT.Fields().ByNumber(2).MessageType(); gotMDB != wantMDB {
			t.Error("MessageDescriptor(MessageB) mismatch")
		}
		if gotED := gotMT.Fields().ByNumber(3).EnumType(); gotED != wantED {
			t.Error("EnumDescriptor(Enum) mismatch")
		}
	}
	for _, gotMT := range messageBTypes[1:] {
		if gotMT != wantMTB {
			t.Error("MessageType(MessageB) mismatch")
		}
		if gotMDA := gotMT.Fields().ByNumber(1).MessageType(); gotMDA != wantMDA {
			t.Error("MessageDescriptor(MessageA) mismatch")
		}
		if gotMDB := gotMT.Fields().ByNumber(2).MessageType(); gotMDB != wantMDB {
			t.Error("MessageDescriptor(MessageB) mismatch")
		}
		if gotED := gotMT.Fields().ByNumber(3).EnumType(); gotED != wantED {
			t.Error("EnumDescriptor(Enum) mismatch")
		}
	}
	for _, gotET := range enumTypes[1:] {
		if gotET != wantET {
			t.Error("EnumType(Enum) mismatch")
		}
	}
}
