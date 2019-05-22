// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"fmt"
	"reflect"

	pimpl "google.golang.org/protobuf/internal/impl"
	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// Export is a zero-length named type that exists only to export a set of
// functions that we do not want to appear in godoc.
type Export struct{}

func (Export) EnumOf(e interface{}) pref.Enum {
	return wrapEnum(reflect.ValueOf(e))
}

func (Export) EnumTypeOf(e interface{}) pref.EnumType {
	return loadEnumType(reflect.TypeOf(e))
}

func (Export) EnumDescriptorOf(e interface{}) pref.EnumDescriptor {
	return LoadEnumDesc(reflect.TypeOf(e))
}

func (Export) MessageOf(m interface{}) pref.Message {
	return wrapMessage(reflect.ValueOf(m)).ProtoReflect()
}

func (Export) MessageTypeOf(m interface{}) pref.MessageType {
	return loadMessageInfo(reflect.TypeOf(m)).PBType
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

var (
	enumIfaceV2    = reflect.TypeOf((*pref.Enum)(nil)).Elem()
	messageIfaceV1 = reflect.TypeOf((*piface.MessageV1)(nil)).Elem()
	messageIfaceV2 = reflect.TypeOf((*pref.ProtoMessage)(nil)).Elem()
)

func (Export) NewConverter(t reflect.Type, k pref.Kind) pvalue.Converter {
	c, _ := newConverter(t, k)
	return c
}

func newConverter(t reflect.Type, k pref.Kind) (pvalue.Converter, bool) {
	switch k {
	case pref.EnumKind:
		if t.Kind() == reflect.Int32 && !t.Implements(enumIfaceV2) {
			return pvalue.Converter{
				PBValueOf: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					return pref.ValueOf(pref.EnumNumber(v.Int()))
				},
				GoValueOf: func(v pref.Value) reflect.Value {
					return reflect.ValueOf(v.Enum()).Convert(t)
				},
				NewEnum: func(n pref.EnumNumber) pref.Enum {
					return wrapEnum(reflect.ValueOf(n).Convert(t))
				},
			}, true
		}
	case pref.MessageKind, pref.GroupKind:
		if t.Kind() == reflect.Ptr && t.Implements(messageIfaceV1) && !t.Implements(messageIfaceV2) {
			return pvalue.Converter{
				PBValueOf: func(v reflect.Value) pref.Value {
					if v.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", v.Type(), t))
					}
					return pref.ValueOf(Export{}.MessageOf(v.Interface()))
				},
				GoValueOf: func(v pref.Value) reflect.Value {
					rv := reflect.ValueOf(v.Message().(pvalue.Unwrapper).ProtoUnwrap())
					if rv.Type() != t {
						panic(fmt.Sprintf("invalid type: got %v, want %v", rv.Type(), t))
					}
					return rv
				},
				NewMessage: func() pref.Message {
					return wrapMessage(reflect.New(t.Elem())).ProtoReflect()
				},
			}, true
		}
	}
	return pvalue.NewConverter(t, k), false
}

func init() {
	pimpl.RegisterLegacyWrapper(Export{})
}
