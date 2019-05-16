// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO: This file exists to have the minimum number of forwarding declarations
// to keep v1 working. This will be deleted in the near future.

package plugin_proto

import "google.golang.org/protobuf/types/pluginpb"

type (
	CodeGeneratorRequest       = pluginpb.CodeGeneratorRequest
	CodeGeneratorResponse      = pluginpb.CodeGeneratorResponse
	CodeGeneratorResponse_File = pluginpb.CodeGeneratorResponse_File
	Version                    = pluginpb.Version
)

var (
	File_google_protobuf_compiler_plugin_proto = pluginpb.File_google_protobuf_compiler_plugin_proto
)
