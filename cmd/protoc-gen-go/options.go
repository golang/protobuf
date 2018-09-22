// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains functions for fetching the options for a protoreflect descriptor.
//
// TODO: Replace this with the appropriate protoreflect API, once it exists.

package main

import (
	"github.com/golang/protobuf/proto"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/v2/protogen"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// messageOptions returns the MessageOptions for a message.
func messageOptions(gen *protogen.Plugin, message *protogen.Message) *descpb.MessageOptions {
	d := getDescriptorProto(gen, message.Desc, message.Path)
	if d == nil {
		return nil
	}
	return d.(*descpb.DescriptorProto).GetOptions()
}

// fieldOptions returns the FieldOptions for a message.
func fieldOptions(gen *protogen.Plugin, field *protogen.Field) *descpb.FieldOptions {
	d := getDescriptorProto(gen, field.Desc, field.Path)
	if d == nil {
		return nil
	}
	return d.(*descpb.FieldDescriptorProto).GetOptions()
}

// enumOptions returns the EnumOptions for an enum
func enumOptions(gen *protogen.Plugin, enum *protogen.Enum) *descpb.EnumOptions {
	d := getDescriptorProto(gen, enum.Desc, enum.Path)
	if d == nil {
		return nil
	}
	return d.(*descpb.EnumDescriptorProto).GetOptions()
}

// enumValueOptions returns the EnumValueOptions for an enum value
func enumValueOptions(gen *protogen.Plugin, value *protogen.EnumValue) *descpb.EnumValueOptions {
	d := getDescriptorProto(gen, value.Desc, value.Path)
	if d == nil {
		return nil
	}
	return d.(*descpb.EnumValueDescriptorProto).GetOptions()
}

func getDescriptorProto(gen *protogen.Plugin, desc protoreflect.Descriptor, path []int32) proto.Message {
	var p proto.Message
	// Look up the FileDescriptorProto.
	for {
		if fdesc, ok := desc.(protoreflect.FileDescriptor); ok {
			file, ok := gen.FileByName(fdesc.Path())
			if !ok {
				return nil
			}
			p = file.Proto
			break
		}
		var ok bool
		desc, ok = desc.Parent()
		if !ok {
			return nil
		}
	}
	const (
		// field numbers in FileDescriptorProto
		filePackageField   = 2 // package
		fileMessageField   = 4 // message_type
		fileEnumField      = 5 // enum_type
		fileExtensionField = 7 // extension
		// field numbers in DescriptorProto
		messageFieldField     = 2 // field
		messageMessageField   = 3 // nested_type
		messageEnumField      = 4 // enum_type
		messageExtensionField = 6 // extension
		messageOneofField     = 8 // oneof_decl
		// field numbers in EnumDescriptorProto
		enumValueField = 2 // value
	)
	for len(path) > 1 {
		switch d := p.(type) {
		case *descpb.FileDescriptorProto:
			switch path[0] {
			case fileMessageField:
				p = d.MessageType[path[1]]
			case fileEnumField:
				p = d.EnumType[path[1]]
			default:
				return nil
			}
		case *descpb.DescriptorProto:
			switch path[0] {
			case messageFieldField:
				p = d.Field[path[1]]
			case messageMessageField:
				p = d.NestedType[path[1]]
			case messageEnumField:
				p = d.EnumType[path[1]]
			default:
				return nil
			}
		case *descpb.EnumDescriptorProto:
			switch path[0] {
			case enumValueField:
				p = d.Value[path[1]]
			default:
				return nil
			}
		default:
			return nil
		}
		path = path[2:]
	}
	return p
}
