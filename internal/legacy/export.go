// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"reflect"

	papi "github.com/golang/protobuf/protoapi"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
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

func (Export) MessageOf(m interface{}) pvalue.LegacyMessage {
	return wrapMessage(reflect.ValueOf(m)).ProtoReflect().(pvalue.LegacyMessage)
}

func (Export) MessageTypeOf(m interface{}) pref.MessageType {
	return loadMessageType(reflect.TypeOf(m)).PBType
}

func (Export) ExtensionTypeOf(d pref.ExtensionDescriptor, t interface{}) pref.ExtensionType {
	return extensionTypeOf(d, reflect.TypeOf(t))
}

func (Export) ExtensionDescFromType(t pref.ExtensionType) *papi.ExtensionDesc {
	return extensionDescFromType(t)
}

func (Export) ExtensionTypeFromDesc(d *papi.ExtensionDesc) pref.ExtensionType {
	return extensionTypeFromDesc(d)
}

func init() {
	pimpl.RegisterLegacyWrapper(Export{})
}
