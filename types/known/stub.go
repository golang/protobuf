// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO: This file exists to have the minimum number of forwarding declarations
// to keep v1 working. This will be deleted in the near future.

package known_proto

import (
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type (
	Any               = anypb.Any
	BoolValue         = wrapperspb.BoolValue
	BytesValue        = wrapperspb.BytesValue
	DoubleValue       = wrapperspb.DoubleValue
	Duration          = durationpb.Duration
	Empty             = emptypb.Empty
	FloatValue        = wrapperspb.FloatValue
	Int32Value        = wrapperspb.Int32Value
	Int64Value        = wrapperspb.Int64Value
	ListValue         = structpb.ListValue
	NullValue         = structpb.NullValue
	StringValue       = wrapperspb.StringValue
	Struct            = structpb.Struct
	Timestamp         = timestamppb.Timestamp
	UInt32Value       = wrapperspb.UInt32Value
	UInt64Value       = wrapperspb.UInt64Value
	Value             = structpb.Value
	Value_BoolValue   = structpb.Value_BoolValue
	Value_ListValue   = structpb.Value_ListValue
	Value_NullValue   = structpb.Value_NullValue
	Value_NumberValue = structpb.Value_NumberValue
	Value_StringValue = structpb.Value_StringValue
	Value_StructValue = structpb.Value_StructValue
)

const (
	NullValue_NULL_VALUE = structpb.NullValue_NULL_VALUE
)

var (
	File_google_protobuf_any_proto       = anypb.File_google_protobuf_any_proto
	File_google_protobuf_duration_proto  = durationpb.File_google_protobuf_duration_proto
	File_google_protobuf_empty_proto     = emptypb.File_google_protobuf_empty_proto
	File_google_protobuf_struct_proto    = structpb.File_google_protobuf_struct_proto
	File_google_protobuf_timestamp_proto = timestamppb.File_google_protobuf_timestamp_proto
	File_google_protobuf_wrappers_proto  = wrapperspb.File_google_protobuf_wrappers_proto
	NullValue_name                       = structpb.NullValue_name
	NullValue_value                      = structpb.NullValue_value
)
