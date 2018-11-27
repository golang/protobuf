// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"reflect"

	protoV1 "github.com/golang/protobuf/proto"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// Export is a zero-length named type that exists only to export a set of
// functions that we do not want to appear in godoc.
type Export struct{}

func (Export) EnumOf(e interface{}) pvalue.LegacyEnum {
	return legacyWrapEnum(reflect.ValueOf(e)).ProtoReflect().(pvalue.LegacyEnum)
}

func (Export) EnumTypeOf(e interface{}) pref.EnumType {
	return legacyLoadEnumType(reflect.TypeOf(e))
}

func (Export) MessageOf(m interface{}) pvalue.LegacyMessage {
	return legacyWrapMessage(reflect.ValueOf(m)).ProtoReflect().(pvalue.LegacyMessage)
}

func (Export) MessageTypeOf(m interface{}) pref.MessageType {
	return legacyLoadMessageType(reflect.TypeOf(m)).Type
}

func (Export) ExtensionTypeOf(d pref.ExtensionDescriptor, t interface{}) pref.ExtensionType {
	return legacyExtensionTypeOf(d, reflect.TypeOf(t))
}

func (Export) ExtensionDescFromType(t pref.ExtensionType) *protoV1.ExtensionDesc {
	return legacyExtensionDescFromType(t)
}

func (Export) ExtensionTypeFromDesc(d *protoV1.ExtensionDesc) pref.ExtensionType {
	return legacyExtensionTypeFromDesc(d)
}

func init() {
	pimpl.RegisterLegacyWrapper(Export{})
}
