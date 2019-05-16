// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build use_golang_protobuf_v1

package proto

import (
	"io"
	"reflect"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

	// Hooks for text.go
	marshalTextAlt       func(io.Writer, Message) error
	marshalTextStringAlt func(Message) string
	compactTextAlt       func(io.Writer, Message) error
	compactTextStringAlt func(Message) string

	// Hooks for text_parser.go
	unmarshalTextAlt func(string, Message) error
)

// Hooks for lib.go.
type RequiredNotSetError = requiredNotSetError

// Hooks for text.go
type TextMarshaler = textMarshaler

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

	// any.proto
	RegisterType((*anypb.Any)(nil), "google.protobuf.Any")

	// api.proto
	RegisterType((*apipb.Api)(nil), "google.protobuf.Api")
	RegisterType((*apipb.Method)(nil), "google.protobuf.Method")
	RegisterType((*apipb.Mixin)(nil), "google.protobuf.Mixin")

	// duration.proto
	RegisterType((*durationpb.Duration)(nil), "google.protobuf.Duration")

	// empty.proto
	RegisterType((*emptypb.Empty)(nil), "google.protobuf.Empty")

	// field_mask.proto
	RegisterType((*fieldmaskpb.FieldMask)(nil), "google.protobuf.FieldMask")

	// source_context.proto
	RegisterType((*sourcecontextpb.SourceContext)(nil), "google.protobuf.SourceContext")

	// struct.proto
	RegisterEnum("google.protobuf.NullValue", structpb.NullValue_name, structpb.NullValue_value)
	RegisterType((*structpb.Struct)(nil), "google.protobuf.Struct")
	RegisterType((*structpb.Value)(nil), "google.protobuf.Value")
	RegisterType((*structpb.ListValue)(nil), "google.protobuf.ListValue")

	// timestamp.proto
	RegisterType((*timestamppb.Timestamp)(nil), "google.protobuf.Timestamp")

	// type.proto
	RegisterEnum("google.protobuf.Syntax", typepb.Syntax_name, typepb.Syntax_value)
	RegisterEnum("google.protobuf.Field_Kind", typepb.Field_Kind_name, typepb.Field_Kind_value)
	RegisterEnum("google.protobuf.Field_Cardinality", typepb.Field_Cardinality_name, typepb.Field_Cardinality_value)
	RegisterType((*typepb.Type)(nil), "google.protobuf.Type")
	RegisterType((*typepb.Field)(nil), "google.protobuf.Field")
	RegisterType((*typepb.Enum)(nil), "google.protobuf.Enum")
	RegisterType((*typepb.EnumValue)(nil), "google.protobuf.EnumValue")
	RegisterType((*typepb.Option)(nil), "google.protobuf.Option")

	// wrapper.proto
	RegisterType((*wrapperspb.DoubleValue)(nil), "google.protobuf.DoubleValue")
	RegisterType((*wrapperspb.FloatValue)(nil), "google.protobuf.FloatValue")
	RegisterType((*wrapperspb.Int64Value)(nil), "google.protobuf.Int64Value")
	RegisterType((*wrapperspb.UInt64Value)(nil), "google.protobuf.UInt64Value")
	RegisterType((*wrapperspb.Int32Value)(nil), "google.protobuf.Int32Value")
	RegisterType((*wrapperspb.UInt32Value)(nil), "google.protobuf.UInt32Value")
	RegisterType((*wrapperspb.BoolValue)(nil), "google.protobuf.BoolValue")
	RegisterType((*wrapperspb.StringValue)(nil), "google.protobuf.StringValue")
	RegisterType((*wrapperspb.BytesValue)(nil), "google.protobuf.BytesValue")
}
