// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/prototype"
)

// legacyWrapMessage wraps v as a protoreflect.ProtoMessage,
// where v must be a *struct kind and not implement the v2 API already.
func legacyWrapMessage(v reflect.Value) pref.ProtoMessage {
	mt := legacyLoadMessageInfo(v.Type())
	return mt.MessageOf(v.Interface()).Interface()
}

var legacyMessageTypeCache sync.Map // map[reflect.Type]*MessageInfo

// legacyLoadMessageInfo dynamically loads a *MessageInfo for t,
// where t must be a *struct kind and not implement the v2 API already.
func legacyLoadMessageInfo(t reflect.Type) *MessageInfo {
	// Fast-path: check if a MessageInfo is cached for this concrete type.
	if mt, ok := legacyMessageTypeCache.Load(t); ok {
		return mt.(*MessageInfo)
	}

	// Slow-path: derive message descriptor and initialize MessageInfo.
	md := LegacyLoadMessageDesc(t)
	mt := new(MessageInfo)
	mt.GoType = t
	mt.PBType = &prototype.Message{
		MessageDescriptor: md,
		NewMessage: func() pref.Message {
			return mt.MessageOf(reflect.New(t.Elem()).Interface())
		},
	}
	if mt, ok := legacyMessageTypeCache.LoadOrStore(t, mt); ok {
		return mt.(*MessageInfo)
	}
	return mt
}

var legacyMessageDescCache sync.Map // map[reflect.Type]protoreflect.MessageDescriptor

// LegacyLoadMessageDesc returns an MessageDescriptor derived from the Go type,
// which must be a *struct kind and not implement the v2 API already.
//
// This is exported for testing purposes.
func LegacyLoadMessageDesc(t reflect.Type) pref.MessageDescriptor {
	// Fast-path: check if a MessageDescriptor is cached for this concrete type.
	if mi, ok := legacyMessageDescCache.Load(t); ok {
		return mi.(pref.MessageDescriptor)
	}

	// Slow-path: initialize MessageDescriptor from the raw descriptor.
	mv := reflect.New(t.Elem()).Interface()
	if _, ok := mv.(pref.ProtoMessage); ok {
		panic(fmt.Sprintf("%v already implements proto.Message", t))
	}
	mdV1, ok := mv.(messageV1)
	if !ok {
		panic(fmt.Sprintf("message %v is no longer supported; please regenerate", t))
	}
	b, idxs := mdV1.Descriptor()

	md := legacyLoadFileDesc(b).Messages().Get(idxs[0])
	for _, i := range idxs[1:] {
		md = md.Messages().Get(i)
	}
	if md, ok := legacyMessageDescCache.LoadOrStore(t, md); ok {
		return md.(protoreflect.MessageDescriptor)
	}
	return md
}
