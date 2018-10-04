// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains functions for fetching the options for a protoreflect descriptor.
//
// TODO: Replace this with the appropriate protoreflect API, once it exists.

package internal_gengogrpc

import (
	"github.com/golang/protobuf/proto"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/v2/protogen"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// serviceOptions returns the options for a service.
func serviceOptions(gen *protogen.Plugin, service *protogen.Service) *descpb.ServiceOptions {
	d := getDescriptorProto(gen, service.Desc, service.Location.Path)
	if d == nil {
		return nil
	}
	return d.(*descpb.ServiceDescriptorProto).GetOptions()
}

// methodOptions returns the options for a method.
func methodOptions(gen *protogen.Plugin, method *protogen.Method) *descpb.MethodOptions {
	d := getDescriptorProto(gen, method.Desc, method.Location.Path)
	if d == nil {
		return nil
	}
	return d.(*descpb.MethodDescriptorProto).GetOptions()
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
		fileServiceField   = 6 // service
		fileExtensionField = 7 // extension
		// field numbers in DescriptorProto
		messageFieldField     = 2 // field
		messageMessageField   = 3 // nested_type
		messageEnumField      = 4 // enum_type
		messageExtensionField = 6 // extension
		messageOneofField     = 8 // oneof_decl
		// field numbers in EnumDescriptorProto
		enumValueField = 2 // value
		// field numbers in ServiceDescriptorProto
		serviceMethodField = 2 // method
	)
	for len(path) > 1 {
		switch d := p.(type) {
		case *descpb.FileDescriptorProto:
			switch path[0] {
			case fileMessageField:
				p = d.MessageType[path[1]]
			case fileEnumField:
				p = d.EnumType[path[1]]
			case fileServiceField:
				p = d.Service[path[1]]
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
		case *descpb.ServiceDescriptorProto:
			switch path[0] {
			case serviceMethodField:
				p = d.Method[path[1]]
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
