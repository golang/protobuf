// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"reflect"

	"github.com/golang/protobuf/protoapi"
)

// TODO: Registration should be written in terms of v2 registries.

// RegisterEnum is called from the generated code to install the enum descriptor
// maps into the global table to aid parsing text format protocol buffers.
func RegisterEnum(typeName string, unusedNameMap map[int32]string, valueMap map[string]int32) {
	protoapi.RegisterEnum(typeName, unusedNameMap, valueMap)
}

// EnumValueMap returns the mapping from names to integers of the
// enum type enumType, or a nil if not found.
func EnumValueMap(enumType string) map[string]int32 {
	return protoapi.EnumValueMap(enumType)
}

// RegisterType is called from generated code and maps from the fully qualified
// proto name to the type (pointer to struct) of the protocol buffer.
func RegisterType(x Message, name string) {
	protoapi.RegisterType(x, name)
}

// RegisterMapType is called from generated code and maps from the fully qualified
// proto name to the native map type of the proto map definition.
func RegisterMapType(x interface{}, name string) {
	protoapi.RegisterMapType(x, name)
}

// MessageName returns the fully-qualified proto name for the given message type.
func MessageName(x Message) string {
	return protoapi.MessageName(x)
}

// MessageType returns the message type (pointer to struct) for a named message.
// The type is not guaranteed to implement proto.Message if the name refers to a
// map entry.
func MessageType(name string) reflect.Type {
	return protoapi.MessageType(name)
}

// RegisterFile is called from generated code and maps from the
// full file name of a .proto file to its compressed FileDescriptorProto.
func RegisterFile(filename string, fileDescriptor []byte) {
	protoapi.RegisterFile(filename, fileDescriptor)
}

// FileDescriptor returns the compressed FileDescriptorProto for a .proto file.
func FileDescriptor(filename string) []byte {
	return protoapi.FileDescriptor(filename)
}

// RegisterExtension is called from the generated code.
func RegisterExtension(desc *ExtensionDesc) {
	protoapi.RegisterExtension(desc)
}

// RegisteredExtensions returns a map of the registered extensions of a
// protocol buffer struct, indexed by the extension number.
// The argument pb should be a nil pointer to the struct type.
func RegisteredExtensions(pb Message) map[int32]*ExtensionDesc {
	return protoapi.RegisteredExtensions(pb)
}
