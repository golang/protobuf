// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoapi

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
)

// TODO: This entire file should not exist and will eventually be deleted.
// This is added for bootstrapping purposes so that a descriptor proto can be
// generated that supports v2 reflection, but still registers into the
// v1 registries for the time being.

var enumValueMaps = make(map[string]map[string]int32)

// RegisterEnum is called from the generated code to install the enum descriptor
// maps into the global table to aid parsing text format protocol buffers.
func RegisterEnum(typeName string, unusedNameMap map[int32]string, valueMap map[string]int32) {
	if _, ok := enumValueMaps[typeName]; ok {
		panic("proto: duplicate enum registered: " + typeName)
	}
	enumValueMaps[typeName] = valueMap
}

// EnumValueMap returns the mapping from names to integers of the
// enum type enumType, or a nil if not found.
func EnumValueMap(enumType string) map[string]int32 {
	return enumValueMaps[enumType]
}

// A registry of all linked message types.
// The string is a fully-qualified proto name ("pkg.Message").
var (
	protoTypedNils = make(map[string]Message)      // a map from proto names to typed nil pointers
	protoMapTypes  = make(map[string]reflect.Type) // a map from proto names to map types
	revProtoTypes  = make(map[reflect.Type]string)
)

// RegisterType is called from generated code and maps from the fully qualified
// proto name to the type (pointer to struct) of the protocol buffer.
func RegisterType(x Message, name string) {
	if _, ok := protoTypedNils[name]; ok {
		// TODO: Some day, make this a panic.
		log.Printf("proto: duplicate proto type registered: %s", name)
		return
	}
	t := reflect.TypeOf(x)
	if v := reflect.ValueOf(x); v.Kind() == reflect.Ptr && v.Pointer() == 0 {
		// Generated code always calls RegisterType with nil x.
		// This check is just for extra safety.
		protoTypedNils[name] = x
	} else {
		protoTypedNils[name] = reflect.Zero(t).Interface().(Message)
	}
	revProtoTypes[t] = name
}

// RegisterMapType is called from generated code and maps from the fully qualified
// proto name to the native map type of the proto map definition.
func RegisterMapType(x interface{}, name string) {
	if reflect.TypeOf(x).Kind() != reflect.Map {
		panic(fmt.Sprintf("RegisterMapType(%T, %q); want map", x, name))
	}
	if _, ok := protoMapTypes[name]; ok {
		log.Printf("proto: duplicate proto type registered: %s", name)
		return
	}
	t := reflect.TypeOf(x)
	protoMapTypes[name] = t
	revProtoTypes[t] = name
}

// MessageName returns the fully-qualified proto name for the given message type.
func MessageName(x Message) string {
	type xname interface {
		XXX_MessageName() string
	}
	if m, ok := x.(xname); ok {
		return m.XXX_MessageName()
	}
	return revProtoTypes[reflect.TypeOf(x)]
}

// MessageType returns the message type (pointer to struct) for a named message.
// The type is not guaranteed to implement proto.Message if the name refers to a
// map entry.
func MessageType(name string) reflect.Type {
	if t, ok := protoTypedNils[name]; ok {
		return reflect.TypeOf(t)
	}
	return protoMapTypes[name]
}

// A registry of all linked proto files.
var protoFiles = make(map[string][]byte) // file name => fileDescriptor

// RegisterFile is called from generated code and maps from the
// full file name of a .proto file to its compressed FileDescriptorProto.
func RegisterFile(filename string, fileDescriptor []byte) {
	protoFiles[filename] = fileDescriptor
}

// FileDescriptor returns the compressed FileDescriptorProto for a .proto file.
func FileDescriptor(filename string) []byte { return protoFiles[filename] }

var extensionMaps = make(map[reflect.Type]map[int32]*ExtensionDesc)

// RegisterExtension is called from the generated code.
func RegisterExtension(desc *ExtensionDesc) {
	st := reflect.TypeOf(desc.ExtendedType).Elem()
	m := extensionMaps[st]
	if m == nil {
		m = make(map[int32]*ExtensionDesc)
		extensionMaps[st] = m
	}
	if _, ok := m[desc.Field]; ok {
		panic("proto: duplicate extension registered: " + st.String() + " " + strconv.Itoa(int(desc.Field)))
	}
	m[desc.Field] = desc
}

// RegisteredExtensions returns a map of the registered extensions of a
// protocol buffer struct, indexed by the extension number.
// The argument pb should be a nil pointer to the struct type.
func RegisteredExtensions(pb Message) map[int32]*ExtensionDesc {
	return extensionMaps[reflect.TypeOf(pb).Elem()]
}
