// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// MessageType provides protobuf related functionality for a given Go type
// that represents a message. A given instance of MessageType is tied to
// exactly one Go type, which must be a pointer to a struct type.
type MessageType struct {
	// GoType is the underlying message Go type and must be populated.
	// Once set, this field must never be mutated.
	GoType reflect.Type // pointer to struct

	// PBType is the underlying message descriptor type and must be populated.
	// Once set, this field must never be mutated.
	PBType pref.MessageType

	once sync.Once // protects all unexported fields

	// TODO: Split fields into dense and sparse maps similar to the current
	// table-driven implementation in v1?
	fields map[pref.FieldNumber]*fieldInfo

	unknownFields   func(*messageDataType) pref.UnknownFields
	extensionFields func(*messageDataType) pref.KnownFields
}

func (mi *MessageType) init() {
	mi.once.Do(func() {
		t := mi.GoType
		if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
			panic(fmt.Sprintf("got %v, want *struct kind", t))
		}

		mi.makeKnownFieldsFunc(t.Elem())
		mi.makeUnknownFieldsFunc(t.Elem())
		mi.makeExtensionFieldsFunc(t.Elem())
	})
}

// makeKnownFieldsFunc generates functions for operations that can be performed
// on each protobuf message field. It takes in a reflect.Type representing the
// Go struct and matches message fields with struct fields.
//
// This code assumes that the struct is well-formed and panics if there are
// any discrepancies.
func (mi *MessageType) makeKnownFieldsFunc(t reflect.Type) {
	// Generate a mapping of field numbers and names to Go struct field or type.
	fields := map[pref.FieldNumber]reflect.StructField{}
	oneofs := map[pref.Name]reflect.StructField{}
	oneofFields := map[pref.FieldNumber]reflect.Type{}
	special := map[string]reflect.StructField{}
fieldLoop:
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
			if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
				n, _ := strconv.ParseUint(s, 10, 64)
				fields[pref.FieldNumber(n)] = f
				continue fieldLoop
			}
		}
		if s := f.Tag.Get("protobuf_oneof"); len(s) > 0 {
			oneofs[pref.Name(s)] = f
			continue fieldLoop
		}
		switch f.Name {
		case "XXX_weak", "XXX_unrecognized", "XXX_sizecache", "XXX_extensions", "XXX_InternalExtensions":
			special[f.Name] = f
			continue fieldLoop
		}
	}
	var oneofWrappers []interface{}
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofFuncs"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[3].Interface().([]interface{})
	}
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofWrappers"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0].Interface().([]interface{})
	}
	for _, v := range oneofWrappers {
		tf := reflect.TypeOf(v).Elem()
		f := tf.Field(0)
		for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
			if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
				n, _ := strconv.ParseUint(s, 10, 64)
				oneofFields[pref.FieldNumber(n)] = tf
				break
			}
		}
	}

	mi.fields = map[pref.FieldNumber]*fieldInfo{}
	for i := 0; i < mi.PBType.Fields().Len(); i++ {
		fd := mi.PBType.Fields().Get(i)
		fs := fields[fd.Number()]
		var fi fieldInfo
		switch {
		case fd.IsWeak():
			fi = fieldInfoForWeak(fd, special["XXX_weak"])
		case fd.OneofType() != nil:
			fi = fieldInfoForOneof(fd, oneofs[fd.OneofType().Name()], oneofFields[fd.Number()])
		case fd.IsMap():
			fi = fieldInfoForMap(fd, fs)
		case fd.Cardinality() == pref.Repeated:
			fi = fieldInfoForList(fd, fs)
		case fd.Kind() == pref.MessageKind || fd.Kind() == pref.GroupKind:
			fi = fieldInfoForMessage(fd, fs)
		default:
			fi = fieldInfoForScalar(fd, fs)
		}
		mi.fields[fd.Number()] = &fi
	}
}

func (mi *MessageType) makeUnknownFieldsFunc(t reflect.Type) {
	if f := makeLegacyUnknownFieldsFunc(t); f != nil {
		mi.unknownFields = f
		return
	}
	mi.unknownFields = func(*messageDataType) pref.UnknownFields {
		return emptyUnknownFields{}
	}
}

func (mi *MessageType) makeExtensionFieldsFunc(t reflect.Type) {
	if f := makeLegacyExtensionFieldsFunc(t); f != nil {
		mi.extensionFields = f
		return
	}
	mi.extensionFields = func(*messageDataType) pref.KnownFields {
		return emptyExtensionFields{}
	}
}

func (mi *MessageType) MessageOf(p interface{}) pref.Message {
	return (*messageReflectWrapper)(mi.dataTypeOf(p))
}

func (mi *MessageType) dataTypeOf(p interface{}) *messageDataType {
	// TODO: Remove this check? This API is primarily used by generated code,
	// and should not violate this assumption. Leave this check in for now to
	// provide some sanity checks during development. This can be removed if
	// it proves to be detrimental to performance.
	if reflect.TypeOf(p) != mi.GoType {
		panic(fmt.Sprintf("type mismatch: got %T, want %v", p, mi.GoType))
	}
	return &messageDataType{pointerOfIface(p), mi}
}

// messageDataType is a tuple of a pointer to the message data and
// a pointer to the message type.
//
// TODO: Unfortunately, we need to close over a pointer and MessageType,
// which incurs an an allocation. This pair is similar to a Go interface,
// which is essentially a tuple of the same thing. We can make this efficient
// with reflect.NamedOf (see https://golang.org/issues/16522).
//
// With that hypothetical API, we could dynamically create a new named type
// that has the same underlying type as MessageType.GoType, and
// dynamically create methods that close over MessageType.
// Since the new type would have the same underlying type, we could directly
// convert between pointers of those types, giving us an efficient way to swap
// out the method set.
//
// Barring the ability to dynamically create named types, the workaround is
//	1. either to accept the cost of an allocation for this wrapper struct or
//	2. generate more types and methods, at the expense of binary size increase.
type messageDataType struct {
	p  pointer
	mi *MessageType
}

type messageReflectWrapper messageDataType

func (m *messageReflectWrapper) Type() pref.MessageType {
	return m.mi.PBType
}
func (m *messageReflectWrapper) KnownFields() pref.KnownFields {
	m.mi.init()
	return (*knownFields)(m)
}
func (m *messageReflectWrapper) UnknownFields() pref.UnknownFields {
	m.mi.init()
	return m.mi.unknownFields((*messageDataType)(m))
}
func (m *messageReflectWrapper) Interface() pref.ProtoMessage {
	if m, ok := m.ProtoUnwrap().(pref.ProtoMessage); ok {
		return m
	}
	return (*messageIfaceWrapper)(m)
}
func (m *messageReflectWrapper) ProtoUnwrap() interface{} {
	return m.p.AsIfaceOf(m.mi.GoType.Elem())
}
func (m *messageReflectWrapper) ProtoMutable() {}

var _ pvalue.Unwrapper = (*messageReflectWrapper)(nil)

type messageIfaceWrapper messageDataType

func (m *messageIfaceWrapper) ProtoReflect() pref.Message {
	return (*messageReflectWrapper)(m)
}
func (m *messageIfaceWrapper) ProtoUnwrap() interface{} {
	return m.p.AsIfaceOf(m.mi.GoType.Elem())
}

type knownFields messageDataType

func (fs *knownFields) Len() (cnt int) {
	for _, fi := range fs.mi.fields {
		if fi.has(fs.p) {
			cnt++
		}
	}
	return cnt + fs.extensionFields().Len()
}
func (fs *knownFields) Has(n pref.FieldNumber) bool {
	if fi := fs.mi.fields[n]; fi != nil {
		return fi.has(fs.p)
	}
	return fs.extensionFields().Has(n)
}
func (fs *knownFields) Get(n pref.FieldNumber) pref.Value {
	if fi := fs.mi.fields[n]; fi != nil {
		return fi.get(fs.p)
	}
	return fs.extensionFields().Get(n)
}
func (fs *knownFields) Set(n pref.FieldNumber, v pref.Value) {
	if fi := fs.mi.fields[n]; fi != nil {
		fi.set(fs.p, v)
		return
	}
	if fs.mi.PBType.ExtensionRanges().Has(n) {
		fs.extensionFields().Set(n, v)
		return
	}
	panic(fmt.Sprintf("invalid field: %d", n))
}
func (fs *knownFields) Clear(n pref.FieldNumber) {
	if fi := fs.mi.fields[n]; fi != nil {
		fi.clear(fs.p)
		return
	}
	if fs.mi.PBType.ExtensionRanges().Has(n) {
		fs.extensionFields().Clear(n)
		return
	}
}
func (fs *knownFields) Range(f func(pref.FieldNumber, pref.Value) bool) {
	for n, fi := range fs.mi.fields {
		if fi.has(fs.p) {
			if !f(n, fi.get(fs.p)) {
				return
			}
		}
	}
	fs.extensionFields().Range(f)
}
func (fs *knownFields) NewMessage(n pref.FieldNumber) pref.Message {
	if fi := fs.mi.fields[n]; fi != nil {
		return fi.newMessage()
	}
	if fs.mi.PBType.ExtensionRanges().Has(n) {
		return fs.extensionFields().NewMessage(n)
	}
	panic(fmt.Sprintf("invalid field: %d", n))
}
func (fs *knownFields) ExtensionTypes() pref.ExtensionFieldTypes {
	return fs.extensionFields().ExtensionTypes()
}
func (fs *knownFields) extensionFields() pref.KnownFields {
	return fs.mi.extensionFields((*messageDataType)(fs))
}

type emptyUnknownFields struct{}

func (emptyUnknownFields) Len() int                                          { return 0 }
func (emptyUnknownFields) Get(pref.FieldNumber) pref.RawFields               { return nil }
func (emptyUnknownFields) Set(pref.FieldNumber, pref.RawFields)              { return } // noop
func (emptyUnknownFields) Range(func(pref.FieldNumber, pref.RawFields) bool) { return }
func (emptyUnknownFields) IsSupported() bool                                 { return false }

type emptyExtensionFields struct{}

func (emptyExtensionFields) Len() int                                      { return 0 }
func (emptyExtensionFields) Has(pref.FieldNumber) bool                     { return false }
func (emptyExtensionFields) Get(pref.FieldNumber) pref.Value               { return pref.Value{} }
func (emptyExtensionFields) Set(pref.FieldNumber, pref.Value)              { panic("extensions not supported") }
func (emptyExtensionFields) Clear(pref.FieldNumber)                        { return } // noop
func (emptyExtensionFields) Range(func(pref.FieldNumber, pref.Value) bool) { return }
func (emptyExtensionFields) NewMessage(pref.FieldNumber) pref.Message {
	panic("extensions not supported")
}
func (emptyExtensionFields) ExtensionTypes() pref.ExtensionFieldTypes { return emptyExtensionTypes{} }

type emptyExtensionTypes struct{}

func (emptyExtensionTypes) Len() int                                     { return 0 }
func (emptyExtensionTypes) Register(pref.ExtensionType)                  { panic("extensions not supported") }
func (emptyExtensionTypes) Remove(pref.ExtensionType)                    { return } // noop
func (emptyExtensionTypes) ByNumber(pref.FieldNumber) pref.ExtensionType { return nil }
func (emptyExtensionTypes) ByName(pref.FullName) pref.ExtensionType      { return nil }
func (emptyExtensionTypes) Range(func(pref.ExtensionType) bool)          { return }
