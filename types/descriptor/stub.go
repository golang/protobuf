// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO: This file exists to have the minimum number of forwarding declarations
// to keep v1 working. This will be deleted in the near future.
package descriptor_proto

import "google.golang.org/protobuf/types/descriptorpb"

type (
	DescriptorProto                       = descriptorpb.DescriptorProto
	DescriptorProto_ExtensionRange        = descriptorpb.DescriptorProto_ExtensionRange
	DescriptorProto_ReservedRange         = descriptorpb.DescriptorProto_ReservedRange
	EnumDescriptorProto                   = descriptorpb.EnumDescriptorProto
	EnumDescriptorProto_EnumReservedRange = descriptorpb.EnumDescriptorProto_EnumReservedRange
	EnumOptions                           = descriptorpb.EnumOptions
	EnumValueDescriptorProto              = descriptorpb.EnumValueDescriptorProto
	EnumValueOptions                      = descriptorpb.EnumValueOptions
	ExtensionRangeOptions                 = descriptorpb.ExtensionRangeOptions
	FieldDescriptorProto                  = descriptorpb.FieldDescriptorProto
	FieldDescriptorProto_Label            = descriptorpb.FieldDescriptorProto_Label
	FieldDescriptorProto_Type             = descriptorpb.FieldDescriptorProto_Type
	FieldOptions                          = descriptorpb.FieldOptions
	FieldOptions_CType                    = descriptorpb.FieldOptions_CType
	FieldOptions_JSType                   = descriptorpb.FieldOptions_JSType
	FileDescriptorProto                   = descriptorpb.FileDescriptorProto
	FileDescriptorSet                     = descriptorpb.FileDescriptorSet
	FileOptions                           = descriptorpb.FileOptions
	FileOptions_OptimizeMode              = descriptorpb.FileOptions_OptimizeMode
	GeneratedCodeInfo                     = descriptorpb.GeneratedCodeInfo
	GeneratedCodeInfo_Annotation          = descriptorpb.GeneratedCodeInfo_Annotation
	MessageOptions                        = descriptorpb.MessageOptions
	MethodDescriptorProto                 = descriptorpb.MethodDescriptorProto
	MethodOptions                         = descriptorpb.MethodOptions
	MethodOptions_IdempotencyLevel        = descriptorpb.MethodOptions_IdempotencyLevel
	OneofDescriptorProto                  = descriptorpb.OneofDescriptorProto
	OneofOptions                          = descriptorpb.OneofOptions
	ServiceDescriptorProto                = descriptorpb.ServiceDescriptorProto
	ServiceOptions                        = descriptorpb.ServiceOptions
	SourceCodeInfo                        = descriptorpb.SourceCodeInfo
	SourceCodeInfo_Location               = descriptorpb.SourceCodeInfo_Location
	UninterpretedOption                   = descriptorpb.UninterpretedOption
	UninterpretedOption_NamePart          = descriptorpb.UninterpretedOption_NamePart
)

const (
	Default_EnumOptions_Deprecated                      = descriptorpb.Default_EnumOptions_Deprecated
	Default_EnumValueOptions_Deprecated                 = descriptorpb.Default_EnumValueOptions_Deprecated
	Default_FieldOptions_Ctype                          = descriptorpb.Default_FieldOptions_Ctype
	Default_FieldOptions_Deprecated                     = descriptorpb.Default_FieldOptions_Deprecated
	Default_FieldOptions_Jstype                         = descriptorpb.Default_FieldOptions_Jstype
	Default_FieldOptions_Lazy                           = descriptorpb.Default_FieldOptions_Lazy
	Default_FieldOptions_Weak                           = descriptorpb.Default_FieldOptions_Weak
	Default_FileOptions_CcEnableArenas                  = descriptorpb.Default_FileOptions_CcEnableArenas
	Default_FileOptions_CcGenericServices               = descriptorpb.Default_FileOptions_CcGenericServices
	Default_FileOptions_Deprecated                      = descriptorpb.Default_FileOptions_Deprecated
	Default_FileOptions_JavaGenericServices             = descriptorpb.Default_FileOptions_JavaGenericServices
	Default_FileOptions_JavaMultipleFiles               = descriptorpb.Default_FileOptions_JavaMultipleFiles
	Default_FileOptions_JavaStringCheckUtf8             = descriptorpb.Default_FileOptions_JavaStringCheckUtf8
	Default_FileOptions_OptimizeFor                     = descriptorpb.Default_FileOptions_OptimizeFor
	Default_FileOptions_PhpGenericServices              = descriptorpb.Default_FileOptions_PhpGenericServices
	Default_FileOptions_PyGenericServices               = descriptorpb.Default_FileOptions_PyGenericServices
	Default_MessageOptions_Deprecated                   = descriptorpb.Default_MessageOptions_Deprecated
	Default_MessageOptions_MessageSetWireFormat         = descriptorpb.Default_MessageOptions_MessageSetWireFormat
	Default_MessageOptions_NoStandardDescriptorAccessor = descriptorpb.Default_MessageOptions_NoStandardDescriptorAccessor
	Default_MethodDescriptorProto_ClientStreaming       = descriptorpb.Default_MethodDescriptorProto_ClientStreaming
	Default_MethodDescriptorProto_ServerStreaming       = descriptorpb.Default_MethodDescriptorProto_ServerStreaming
	Default_MethodOptions_Deprecated                    = descriptorpb.Default_MethodOptions_Deprecated
	Default_MethodOptions_IdempotencyLevel              = descriptorpb.Default_MethodOptions_IdempotencyLevel
	Default_ServiceOptions_Deprecated                   = descriptorpb.Default_ServiceOptions_Deprecated
	FieldDescriptorProto_LABEL_OPTIONAL                 = descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	FieldDescriptorProto_LABEL_REPEATED                 = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	FieldDescriptorProto_LABEL_REQUIRED                 = descriptorpb.FieldDescriptorProto_LABEL_REQUIRED
	FieldDescriptorProto_TYPE_BOOL                      = descriptorpb.FieldDescriptorProto_TYPE_BOOL
	FieldDescriptorProto_TYPE_BYTES                     = descriptorpb.FieldDescriptorProto_TYPE_BYTES
	FieldDescriptorProto_TYPE_DOUBLE                    = descriptorpb.FieldDescriptorProto_TYPE_DOUBLE
	FieldDescriptorProto_TYPE_ENUM                      = descriptorpb.FieldDescriptorProto_TYPE_ENUM
	FieldDescriptorProto_TYPE_FIXED32                   = descriptorpb.FieldDescriptorProto_TYPE_FIXED32
	FieldDescriptorProto_TYPE_FIXED64                   = descriptorpb.FieldDescriptorProto_TYPE_FIXED64
	FieldDescriptorProto_TYPE_FLOAT                     = descriptorpb.FieldDescriptorProto_TYPE_FLOAT
	FieldDescriptorProto_TYPE_GROUP                     = descriptorpb.FieldDescriptorProto_TYPE_GROUP
	FieldDescriptorProto_TYPE_INT32                     = descriptorpb.FieldDescriptorProto_TYPE_INT32
	FieldDescriptorProto_TYPE_INT64                     = descriptorpb.FieldDescriptorProto_TYPE_INT64
	FieldDescriptorProto_TYPE_MESSAGE                   = descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	FieldDescriptorProto_TYPE_SFIXED32                  = descriptorpb.FieldDescriptorProto_TYPE_SFIXED32
	FieldDescriptorProto_TYPE_SFIXED64                  = descriptorpb.FieldDescriptorProto_TYPE_SFIXED64
	FieldDescriptorProto_TYPE_SINT32                    = descriptorpb.FieldDescriptorProto_TYPE_SINT32
	FieldDescriptorProto_TYPE_SINT64                    = descriptorpb.FieldDescriptorProto_TYPE_SINT64
	FieldDescriptorProto_TYPE_STRING                    = descriptorpb.FieldDescriptorProto_TYPE_STRING
	FieldDescriptorProto_TYPE_UINT32                    = descriptorpb.FieldDescriptorProto_TYPE_UINT32
	FieldDescriptorProto_TYPE_UINT64                    = descriptorpb.FieldDescriptorProto_TYPE_UINT64
	FieldOptions_CORD                                   = descriptorpb.FieldOptions_CORD
	FieldOptions_JS_NORMAL                              = descriptorpb.FieldOptions_JS_NORMAL
	FieldOptions_JS_NUMBER                              = descriptorpb.FieldOptions_JS_NUMBER
	FieldOptions_JS_STRING                              = descriptorpb.FieldOptions_JS_STRING
	FieldOptions_STRING                                 = descriptorpb.FieldOptions_STRING
	FieldOptions_STRING_PIECE                           = descriptorpb.FieldOptions_STRING_PIECE
	FileOptions_CODE_SIZE                               = descriptorpb.FileOptions_CODE_SIZE
	FileOptions_LITE_RUNTIME                            = descriptorpb.FileOptions_LITE_RUNTIME
	FileOptions_SPEED                                   = descriptorpb.FileOptions_SPEED
	MethodOptions_IDEMPOTENCY_UNKNOWN                   = descriptorpb.MethodOptions_IDEMPOTENCY_UNKNOWN
	MethodOptions_IDEMPOTENT                            = descriptorpb.MethodOptions_IDEMPOTENT
	MethodOptions_NO_SIDE_EFFECTS                       = descriptorpb.MethodOptions_NO_SIDE_EFFECTS
)

var (
	FieldDescriptorProto_Label_name       = descriptorpb.FieldDescriptorProto_Label_name
	FieldDescriptorProto_Label_value      = descriptorpb.FieldDescriptorProto_Label_value
	FieldDescriptorProto_Type_name        = descriptorpb.FieldDescriptorProto_Type_name
	FieldDescriptorProto_Type_value       = descriptorpb.FieldDescriptorProto_Type_value
	FieldOptions_CType_name               = descriptorpb.FieldOptions_CType_name
	FieldOptions_CType_value              = descriptorpb.FieldOptions_CType_value
	FieldOptions_JSType_name              = descriptorpb.FieldOptions_JSType_name
	FieldOptions_JSType_value             = descriptorpb.FieldOptions_JSType_value
	File_google_protobuf_descriptor_proto = descriptorpb.File_google_protobuf_descriptor_proto
	FileOptions_OptimizeMode_name         = descriptorpb.FileOptions_OptimizeMode_name
	FileOptions_OptimizeMode_value        = descriptorpb.FileOptions_OptimizeMode_value
	MethodOptions_IdempotencyLevel_name   = descriptorpb.MethodOptions_IdempotencyLevel_name
	MethodOptions_IdempotencyLevel_value  = descriptorpb.MethodOptions_IdempotencyLevel_value
)
