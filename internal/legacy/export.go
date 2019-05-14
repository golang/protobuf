// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"reflect"

	pimpl "google.golang.org/protobuf/internal/impl"
	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// Export is a zero-length named type that exists only to export a set of
// functions that we do not want to appear in godoc.
type Export struct{}

func (Export) EnumOf(e interface{}) pvalue.LegacyEnum {
	return wrapEnum(reflect.ValueOf(e)).(pvalue.LegacyEnum)
}

func (Export) EnumTypeOf(e interface{}) pref.EnumType {
	return loadEnumType(reflect.TypeOf(e))
}

func (Export) EnumDescriptorOf(e interface{}) pref.EnumDescriptor {
	return LoadEnumDesc(reflect.TypeOf(e))
}

func (Export) MessageOf(m interface{}) pvalue.LegacyMessage {
	return wrapMessage(reflect.ValueOf(m)).ProtoReflect().(pvalue.LegacyMessage)
}

func (Export) MessageTypeOf(m interface{}) pref.MessageType {
	return loadMessageType(reflect.TypeOf(m)).PBType
}

func (Export) MessageDescriptorOf(m interface{}) pref.MessageDescriptor {
	return LoadMessageDesc(reflect.TypeOf(m))
}

func (Export) ExtensionDescFromType(t pref.ExtensionType) *piface.ExtensionDescV1 {
	return extensionDescFromType(t)
}

func (Export) ExtensionTypeFromDesc(d *piface.ExtensionDescV1) pref.ExtensionType {
	return extensionTypeFromDesc(d)
}

func init() {
	pimpl.RegisterLegacyWrapper(Export{})
}
