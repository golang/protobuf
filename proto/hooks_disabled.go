// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build use_golang_protobuf_v1

package proto

import (
	"reflect"

	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

var (
	// Hooks for lib.go.
	setDefaultsAlt func(Message)

	// Hooks for discard.go.
	discardUnknownAlt func(Message)

	// Hooks for registry.go.
	registerEnumAlt         func(string, map[int32]string, map[string]int32)
	enumValueMapAlt         func(string) map[string]int32
	registerTypeAlt         func(Message, string)
	registerMapTypeAlt      func(interface{}, string)
	messageNameAlt          func(Message) string
	messageTypeAlt          func(string) reflect.Type
	registerFileAlt         func(string, []byte)
	fileDescriptorAlt       func(string) []byte
	registerExtensionAlt    func(*ExtensionDesc)
	registeredExtensionsAlt func(Message) map[int32]*ExtensionDesc
)

// The v2 descriptor no longer registers with v1.
// If we're only relying on the v1 registry, we need to manually register the
// types in descriptor.
func init() {
	// TODO: This should be eventually deleted once the v1 repository is fully
	// switched over to wrap the v2 repository.
	rawDesc, _ := (*descriptorpb.DescriptorProto)(nil).Descriptor()
	RegisterFile("google/protobuf/descriptor.proto", rawDesc)
	RegisterEnum("google.protobuf.FieldDescriptorProto_Type", descriptorpb.FieldDescriptorProto_Type_name, descriptorpb.FieldDescriptorProto_Type_value)
	RegisterEnum("google.protobuf.FieldDescriptorProto_Label", descriptorpb.FieldDescriptorProto_Label_name, descriptorpb.FieldDescriptorProto_Label_value)
	RegisterEnum("google.protobuf.FileOptions_OptimizeMode", descriptorpb.FileOptions_OptimizeMode_name, descriptorpb.FileOptions_OptimizeMode_value)
	RegisterEnum("google.protobuf.FieldOptions_CType", descriptorpb.FieldOptions_CType_name, descriptorpb.FieldOptions_CType_value)
	RegisterEnum("google.protobuf.FieldOptions_JSType", descriptorpb.FieldOptions_JSType_name, descriptorpb.FieldOptions_JSType_value)
	RegisterEnum("google.protobuf.MethodOptions_IdempotencyLevel", descriptorpb.MethodOptions_IdempotencyLevel_name, descriptorpb.MethodOptions_IdempotencyLevel_value)
	RegisterType((*descriptorpb.FileDescriptorSet)(nil), "google.protobuf.FileDescriptorSet")
	RegisterType((*descriptorpb.FileDescriptorProto)(nil), "google.protobuf.FileDescriptorProto")
	RegisterType((*descriptorpb.DescriptorProto)(nil), "google.protobuf.DescriptorProto")
	RegisterType((*descriptorpb.ExtensionRangeOptions)(nil), "google.protobuf.ExtensionRangeOptions")
	RegisterType((*descriptorpb.FieldDescriptorProto)(nil), "google.protobuf.FieldDescriptorProto")
	RegisterType((*descriptorpb.OneofDescriptorProto)(nil), "google.protobuf.OneofDescriptorProto")
	RegisterType((*descriptorpb.EnumDescriptorProto)(nil), "google.protobuf.EnumDescriptorProto")
	RegisterType((*descriptorpb.EnumValueDescriptorProto)(nil), "google.protobuf.EnumValueDescriptorProto")
	RegisterType((*descriptorpb.ServiceDescriptorProto)(nil), "google.protobuf.ServiceDescriptorProto")
	RegisterType((*descriptorpb.MethodDescriptorProto)(nil), "google.protobuf.MethodDescriptorProto")
	RegisterType((*descriptorpb.FileOptions)(nil), "google.protobuf.FileOptions")
	RegisterType((*descriptorpb.MessageOptions)(nil), "google.protobuf.MessageOptions")
	RegisterType((*descriptorpb.FieldOptions)(nil), "google.protobuf.FieldOptions")
	RegisterType((*descriptorpb.OneofOptions)(nil), "google.protobuf.OneofOptions")
	RegisterType((*descriptorpb.EnumOptions)(nil), "google.protobuf.EnumOptions")
	RegisterType((*descriptorpb.EnumValueOptions)(nil), "google.protobuf.EnumValueOptions")
	RegisterType((*descriptorpb.ServiceOptions)(nil), "google.protobuf.ServiceOptions")
	RegisterType((*descriptorpb.MethodOptions)(nil), "google.protobuf.MethodOptions")
	RegisterType((*descriptorpb.UninterpretedOption)(nil), "google.protobuf.UninterpretedOption")
	RegisterType((*descriptorpb.SourceCodeInfo)(nil), "google.protobuf.SourceCodeInfo")
	RegisterType((*descriptorpb.GeneratedCodeInfo)(nil), "google.protobuf.GeneratedCodeInfo")
	RegisterType((*descriptorpb.DescriptorProto_ExtensionRange)(nil), "google.protobuf.DescriptorProto.ExtensionRange")
	RegisterType((*descriptorpb.DescriptorProto_ReservedRange)(nil), "google.protobuf.DescriptorProto.ReservedRange")
	RegisterType((*descriptorpb.EnumDescriptorProto_EnumReservedRange)(nil), "google.protobuf.EnumDescriptorProto.EnumReservedRange")
	RegisterType((*descriptorpb.UninterpretedOption_NamePart)(nil), "google.protobuf.UninterpretedOption.NamePart")
	RegisterType((*descriptorpb.SourceCodeInfo_Location)(nil), "google.protobuf.SourceCodeInfo.Location")
	RegisterType((*descriptorpb.GeneratedCodeInfo_Annotation)(nil), "google.protobuf.GeneratedCodeInfo.Annotation")
}
