// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"sync"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
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
	var enumDescs [numParallel]protoreflect.EnumDescriptor

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
			enumDescs[i] = Export{}.EnumDescriptorOf(Enum(0))
		}()
	}
	wg.Wait()

	var (
		wantMTA = messageATypes[0]
		wantMDA = messageATypes[0].Descriptor().Fields().ByNumber(1).Message()
		wantMTB = messageBTypes[0]
		wantMDB = messageBTypes[0].Descriptor().Fields().ByNumber(2).Message()
		wantED  = messageATypes[0].Descriptor().Fields().ByNumber(3).Enum()
	)

	for _, gotMT := range messageATypes[1:] {
		if gotMT != wantMTA {
			t.Error("MessageType(MessageA) mismatch")
		}
		if gotMDA := gotMT.Descriptor().Fields().ByNumber(1).Message(); gotMDA != wantMDA {
			t.Error("MessageDescriptor(MessageA) mismatch")
		}
		if gotMDB := gotMT.Descriptor().Fields().ByNumber(2).Message(); gotMDB != wantMDB {
			t.Error("MessageDescriptor(MessageB) mismatch")
		}
		if gotED := gotMT.Descriptor().Fields().ByNumber(3).Enum(); gotED != wantED {
			t.Error("EnumDescriptor(Enum) mismatch")
		}
	}
	for _, gotMT := range messageBTypes[1:] {
		if gotMT != wantMTB {
			t.Error("MessageType(MessageB) mismatch")
		}
		if gotMDA := gotMT.Descriptor().Fields().ByNumber(1).Message(); gotMDA != wantMDA {
			t.Error("MessageDescriptor(MessageA) mismatch")
		}
		if gotMDB := gotMT.Descriptor().Fields().ByNumber(2).Message(); gotMDB != wantMDB {
			t.Error("MessageDescriptor(MessageB) mismatch")
		}
		if gotED := gotMT.Descriptor().Fields().ByNumber(3).Enum(); gotED != wantED {
			t.Error("EnumDescriptor(Enum) mismatch")
		}
	}
	for _, gotED := range enumDescs[1:] {
		if gotED != wantED {
			t.Error("EnumType(Enum) mismatch")
		}
	}
}
