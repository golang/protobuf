// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileinit

// Constants for field numbers of messages declared in descriptor.proto.
const (
	// FileDescriptorProto field numbers
	fileDesc_Syntax        = 12 // optional string
	fileDesc_Name          = 1  // optional string
	fileDesc_Package       = 2  // optional string
	fileDesc_Imports       = 3  // repeated string
	fileDesc_PublicImports = 10 // repeated int32
	fileDesc_WeakImports   = 11 // repeated int32
	fileDesc_Enums         = 5  // repeated EnumDescriptorProto
	fileDesc_Messages      = 4  // repeated DescriptorProto
	fileDesc_Extensions    = 7  // repeated FieldDescriptorProto
	fileDesc_Services      = 6  // repeated ServiceDescriptorProto
	fileDesc_Options       = 8  // optional FileOptions

	// EnumDescriptorProto field numbers
	enumDesc_Name           = 1 // optional string
	enumDesc_Values         = 2 // repeated EnumValueDescriptorProto
	enumDesc_ReservedNames  = 5 // repeated string
	enumDesc_ReservedRanges = 4 // repeated EnumReservedRange
	enumDesc_Options        = 3 // optional EnumOptions

	// EnumReservedRange field numbers
	enumReservedRange_Start = 1 // optional int32
	enumReservedRange_End   = 2 // optional int32

	// EnumValueDescriptorProto field numbers
	enumValueDesc_Name    = 1 // optional string
	enumValueDesc_Number  = 2 // optional int32
	enumValueDesc_Options = 3 // optional EnumValueOptions

	// DescriptorProto field numbers
	messageDesc_Name            = 1  // optional string
	messageDesc_Fields          = 2  // repeated FieldDescriptorProto
	messageDesc_Oneofs          = 8  // repeated OneofDescriptorProto
	messageDesc_ReservedNames   = 10 // repeated string
	messageDesc_ReservedRanges  = 9  // repeated ReservedRange
	messageDesc_ExtensionRanges = 5  // repeated ExtensionRange
	messageDesc_Enums           = 4  // repeated EnumDescriptorProto
	messageDesc_Messages        = 3  // repeated DescriptorProto
	messageDesc_Extensions      = 6  // repeated FieldDescriptorProto
	messageDesc_Options         = 7  // optional MessageOptions

	// ReservedRange field numbers
	messageReservedRange_Start = 1 // optional int32
	messageReservedRange_End   = 2 // optional int32

	// ExtensionRange field numbers
	messageExtensionRange_Start   = 1 // optional int32
	messageExtensionRange_End     = 2 // optional int32
	messageExtensionRange_Options = 3 // optional ExtensionRangeOptions

	// MessageOptions field numbers
	messageOptions_IsMapEntry = 7 // optional bool

	// FieldDescriptorProto field numbers
	fieldDesc_Name         = 1  // optional string
	fieldDesc_Number       = 3  // optional int32
	fieldDesc_Cardinality  = 4  // optional Label
	fieldDesc_Kind         = 5  // optional Type
	fieldDesc_JSONName     = 10 // optional string
	fieldDesc_Default      = 7  // optional string
	fieldDesc_OneofIndex   = 9  // optional int32
	fieldDesc_TypeName     = 6  // optional string
	fieldDesc_ExtendedType = 2  // optional string
	fieldDesc_Options      = 8  // optional FieldOptions

	// FieldOptions field numbers
	fieldOptions_IsPacked = 2  // optional bool
	fieldOptions_IsWeak   = 10 // optional bool

	// OneofDescriptorProto field numbers
	oneofDesc_Name    = 1 // optional string
	oneofDesc_Options = 2 // optional OneofOptions

	// ServiceDescriptorProto field numbers
	serviceDesc_Name    = 1 // optional string
	serviceDesc_Methods = 2 // repeated MethodDescriptorProto
	serviceDesc_Options = 3 // optional ServiceOptions

	// MethodDescriptorProto field numbers
	methodDesc_Name              = 1 // optional string
	methodDesc_InputType         = 2 // optional string
	methodDesc_OutputType        = 3 // optional string
	methodDesc_IsStreamingClient = 5 // optional bool
	methodDesc_IsStreamingServer = 6 // optional bool
	methodDesc_Options           = 4 // optional MethodOptions
)
