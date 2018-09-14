// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains functions for fetching the options for a protoreflect descriptor.
//
// TODO: Replace this with the appropriate protoreflect API, once it exists.

package main

import (
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"google.golang.org/proto/protogen"
	"google.golang.org/proto/reflect/protoreflect"
)

// messageOptions returns the MessageOptions for a message.
func messageOptions(gen *protogen.Plugin, message *protogen.Message) *descpb.MessageOptions {
	file, ok := descriptorFile(gen, message.Desc)
	if !ok {
		return nil
	}
	desc := file.Proto.MessageType[message.Path[1]]
	for i := 3; i < len(message.Path); i += 2 {
		desc = desc.NestedType[message.Path[1]]
	}
	return desc.GetOptions()
}

func descriptorFile(gen *protogen.Plugin, desc protoreflect.Descriptor) (*protogen.File, bool) {
	for {
		if fdesc, ok := desc.(protoreflect.FileDescriptor); ok {
			return gen.FileByName(fdesc.Path())
		}
		var ok bool
		desc, ok = desc.Parent()
		if !ok {
			return nil, false
		}
	}
}
