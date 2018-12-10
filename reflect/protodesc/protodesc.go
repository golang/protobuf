// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protodesc provides for converting descriptorpb.FileDescriptorProto
// to/from the reflective protoreflect.FileDescriptor.
package protodesc

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/v2/internal/encoding/defval"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"
	"github.com/golang/protobuf/v2/reflect/prototype"

	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

// TODO: Should we be responsible for validating other parts of the descriptor
// that we don't directly use?
//
// For example:
//	* That field numbers don't overlap with reserved numbers.
//	* That field names don't overlap with reserved names.
//	* That enum numbers don't overlap with reserved numbers.
//	* That enum names don't overlap with reserved names.
//	* That "extendee" is not set for a message field.
//	* That "oneof_index" is not set for an extension field.
//	* That "json_name" is not set for an extension field. Maybe, maybe not.
//	* That "type_name" is not set on a field for non-enums and non-messages.
//	* That "weak" is not set for an extension field (double check this).

// TODO: Store the input file descriptor to implement:
//	* protoreflect.Descriptor.DescriptorProto
//	* protoreflect.Descriptor.DescriptorOptions

// TODO: Should we return a File instead of protoreflect.FileDescriptor?
// This would allow users to mutate the File before converting it.
// However, this will complicate future work for validation since File may now
// diverge from the stored descriptor proto (see above TODO).

// NewFile creates a new protoreflect.FileDescriptor from the provided
// file descriptor message. The file must represent a valid proto file according
// to protobuf semantics.
//
// Any import files, enum types, or message types referenced in the file are
// resolved using the provided registry. When looking up an import file path,
// the path must be unique. The newly created file descriptor is not registered
// back into the provided file registry.
//
// The caller must relinquish full ownership of the input fd and must not
// access or mutate any fields.
func NewFile(fd *descriptorpb.FileDescriptorProto, r *protoregistry.Files) (protoreflect.FileDescriptor, error) {
	var f prototype.File
	switch fd.GetSyntax() {
	case "proto2", "":
		f.Syntax = protoreflect.Proto2
	case "proto3":
		f.Syntax = protoreflect.Proto3
	default:
		return nil, errors.New("invalid syntax: %v", fd.GetSyntax())
	}
	f.Path = fd.GetName()
	f.Package = protoreflect.FullName(fd.GetPackage())
	f.Options = fd.GetOptions()

	f.Imports = make([]protoreflect.FileImport, len(fd.GetDependency()))
	for _, i := range fd.GetPublicDependency() {
		if int(i) >= len(f.Imports) || f.Imports[i].IsPublic {
			return nil, errors.New("invalid or duplicate public import index: %d", i)
		}
		f.Imports[i].IsPublic = true
	}
	for _, i := range fd.GetWeakDependency() {
		if int(i) >= len(f.Imports) || f.Imports[i].IsWeak {
			return nil, errors.New("invalid or duplicate weak import index: %d", i)
		}
		f.Imports[i].IsWeak = true
	}
	for i, path := range fd.GetDependency() {
		var n int
		imp := &f.Imports[i]
		r.RangeFilesByPath(path, func(fd protoreflect.FileDescriptor) bool {
			imp.FileDescriptor = fd
			n++
			return true
		})
		if n > 1 {
			return nil, errors.New("duplicate files for import %q", path)
		}
		if imp.IsWeak || imp.FileDescriptor == nil {
			imp.FileDescriptor = prototype.PlaceholderFile(path, "")
		}
	}

	var err error
	f.Messages, err = messagesFromDescriptorProto(fd.GetMessageType(), f.Syntax, r)
	if err != nil {
		return nil, err
	}
	f.Enums, err = enumsFromDescriptorProto(fd.GetEnumType(), r)
	if err != nil {
		return nil, err
	}
	f.Extensions, err = extensionsFromDescriptorProto(fd.GetExtension(), r)
	if err != nil {
		return nil, err
	}
	f.Services, err = servicesFromDescriptorProto(fd.GetService(), r)
	if err != nil {
		return nil, err
	}

	return prototype.NewFile(&f)
}

func messagesFromDescriptorProto(mds []*descriptorpb.DescriptorProto, syntax protoreflect.Syntax, r *protoregistry.Files) (ms []prototype.Message, err error) {
	for _, md := range mds {
		var m prototype.Message
		m.Name = protoreflect.Name(md.GetName())
		m.Options = md.GetOptions()
		m.IsMapEntry = md.GetOptions().GetMapEntry()
		for _, fd := range md.GetField() {
			var f prototype.Field
			f.Name = protoreflect.Name(fd.GetName())
			f.Number = protoreflect.FieldNumber(fd.GetNumber())
			f.Cardinality = protoreflect.Cardinality(fd.GetLabel())
			f.Kind = protoreflect.Kind(fd.GetType())
			opts := fd.GetOptions()
			f.Options = opts
			if opts != nil && opts.Packed != nil {
				if *opts.Packed {
					f.IsPacked = prototype.True
				} else {
					f.IsPacked = prototype.False
				}
			}
			f.IsWeak = opts.GetWeak()
			f.JSONName = fd.GetJsonName()
			if fd.DefaultValue != nil {
				f.Default, err = defval.Unmarshal(fd.GetDefaultValue(), f.Kind, defval.Descriptor)
				if err != nil {
					return nil, err
				}
			}
			if fd.OneofIndex != nil {
				i := int(fd.GetOneofIndex())
				if i >= len(md.GetOneofDecl()) {
					return nil, errors.New("invalid oneof index: %d", i)
				}
				f.OneofName = protoreflect.Name(md.GetOneofDecl()[i].GetName())
			}
			switch f.Kind {
			case protoreflect.EnumKind:
				f.EnumType, err = findEnumDescriptor(fd.GetTypeName(), r)
				if err != nil {
					return nil, err
				}
				if opts.GetWeak() && !f.EnumType.IsPlaceholder() {
					f.EnumType = prototype.PlaceholderEnum(f.EnumType.FullName())
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				f.MessageType, err = findMessageDescriptor(fd.GetTypeName(), r)
				if err != nil {
					return nil, err
				}
				if opts.GetWeak() && !f.MessageType.IsPlaceholder() {
					f.MessageType = prototype.PlaceholderMessage(f.MessageType.FullName())
				}
			}
			m.Fields = append(m.Fields, f)
		}
		for _, od := range md.GetOneofDecl() {
			m.Oneofs = append(m.Oneofs, prototype.Oneof{
				Name:    protoreflect.Name(od.GetName()),
				Options: od.Options,
			})
		}
		for _, s := range md.GetReservedName() {
			m.ReservedNames = append(m.ReservedNames, protoreflect.Name(s))
		}
		for _, rr := range md.GetReservedRange() {
			m.ReservedRanges = append(m.ReservedRanges, [2]protoreflect.FieldNumber{
				protoreflect.FieldNumber(rr.GetStart()),
				protoreflect.FieldNumber(rr.GetEnd()),
			})
		}
		for _, xr := range md.GetExtensionRange() {
			m.ExtensionRanges = append(m.ExtensionRanges, [2]protoreflect.FieldNumber{
				protoreflect.FieldNumber(xr.GetStart()),
				protoreflect.FieldNumber(xr.GetEnd()),
			})
			m.ExtensionRangeOptions = append(m.ExtensionRangeOptions, xr.GetOptions())
		}

		m.Messages, err = messagesFromDescriptorProto(md.GetNestedType(), syntax, r)
		if err != nil {
			return nil, err
		}
		m.Enums, err = enumsFromDescriptorProto(md.GetEnumType(), r)
		if err != nil {
			return nil, err
		}
		m.Extensions, err = extensionsFromDescriptorProto(md.GetExtension(), r)
		if err != nil {
			return nil, err
		}

		ms = append(ms, m)
	}
	return ms, nil
}

func enumsFromDescriptorProto(eds []*descriptorpb.EnumDescriptorProto, r *protoregistry.Files) (es []prototype.Enum, err error) {
	for _, ed := range eds {
		var e prototype.Enum
		e.Name = protoreflect.Name(ed.GetName())
		e.Options = ed.GetOptions()
		for _, vd := range ed.GetValue() {
			e.Values = append(e.Values, prototype.EnumValue{
				Name:    protoreflect.Name(vd.GetName()),
				Number:  protoreflect.EnumNumber(vd.GetNumber()),
				Options: vd.Options,
			})
		}
		for _, s := range ed.GetReservedName() {
			e.ReservedNames = append(e.ReservedNames, protoreflect.Name(s))
		}
		for _, rr := range ed.GetReservedRange() {
			e.ReservedRanges = append(e.ReservedRanges, [2]protoreflect.EnumNumber{
				protoreflect.EnumNumber(rr.GetStart()),
				protoreflect.EnumNumber(rr.GetEnd()),
			})
		}
		es = append(es, e)
	}
	return es, nil
}

func extensionsFromDescriptorProto(xds []*descriptorpb.FieldDescriptorProto, r *protoregistry.Files) (xs []prototype.Extension, err error) {
	for _, xd := range xds {
		var x prototype.Extension
		x.Name = protoreflect.Name(xd.GetName())
		x.Number = protoreflect.FieldNumber(xd.GetNumber())
		x.Cardinality = protoreflect.Cardinality(xd.GetLabel())
		x.Kind = protoreflect.Kind(xd.GetType())
		x.Options = xd.GetOptions()
		if xd.DefaultValue != nil {
			x.Default, err = defval.Unmarshal(xd.GetDefaultValue(), x.Kind, defval.Descriptor)
			if err != nil {
				return nil, err
			}
		}
		switch x.Kind {
		case protoreflect.EnumKind:
			x.EnumType, err = findEnumDescriptor(xd.GetTypeName(), r)
			if err != nil {
				return nil, err
			}
		case protoreflect.MessageKind, protoreflect.GroupKind:
			x.MessageType, err = findMessageDescriptor(xd.GetTypeName(), r)
			if err != nil {
				return nil, err
			}
		}
		x.ExtendedType, err = findMessageDescriptor(xd.GetExtendee(), r)
		if err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	return xs, nil
}

func servicesFromDescriptorProto(sds []*descriptorpb.ServiceDescriptorProto, r *protoregistry.Files) (ss []prototype.Service, err error) {
	for _, sd := range sds {
		var s prototype.Service
		s.Name = protoreflect.Name(sd.GetName())
		s.Options = sd.GetOptions()
		for _, md := range sd.GetMethod() {
			var m prototype.Method
			m.Name = protoreflect.Name(md.GetName())
			m.Options = md.GetOptions()
			m.InputType, err = findMessageDescriptor(md.GetInputType(), r)
			if err != nil {
				return nil, err
			}
			m.OutputType, err = findMessageDescriptor(md.GetOutputType(), r)
			if err != nil {
				return nil, err
			}
			m.IsStreamingClient = md.GetClientStreaming()
			m.IsStreamingServer = md.GetServerStreaming()
			s.Methods = append(s.Methods, m)
		}
		ss = append(ss, s)
	}
	return ss, nil
}

// TODO: Should we allow relative names? The protoc compiler has emitted
// absolute names for some time now. Requiring absolute names as an input
// simplifies our implementation as we won't need to implement C++'s namespace
// scoping rules.

func findMessageDescriptor(s string, r *protoregistry.Files) (protoreflect.MessageDescriptor, error) {
	if !strings.HasPrefix(s, ".") {
		return nil, errors.New("identifier name must be fully qualified with a leading dot: %v", s)
	}
	name := protoreflect.FullName(strings.TrimPrefix(s, "."))
	switch m, err := r.FindDescriptorByName(name); {
	case err == nil:
		m, ok := m.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, errors.New("resolved wrong type: got %v, want message", typeName(m))
		}
		return m, nil
	case err == protoregistry.NotFound:
		return prototype.PlaceholderMessage(name), nil
	default:
		return nil, err
	}
}

func findEnumDescriptor(s string, r *protoregistry.Files) (protoreflect.EnumDescriptor, error) {
	if !strings.HasPrefix(s, ".") {
		return nil, errors.New("identifier name must be fully qualified with a leading dot: %v", s)
	}
	name := protoreflect.FullName(strings.TrimPrefix(s, "."))
	switch e, err := r.FindDescriptorByName(name); {
	case err == nil:
		e, ok := e.(protoreflect.EnumDescriptor)
		if !ok {
			return nil, errors.New("resolved wrong type: got %T, want enum", typeName(e))
		}
		return e, nil
	case err == protoregistry.NotFound:
		return prototype.PlaceholderEnum(name), nil
	default:
		return nil, err
	}
}

func typeName(t protoreflect.Descriptor) string {
	switch t.(type) {
	case protoreflect.EnumType:
		return "enum"
	case protoreflect.MessageType:
		return "message"
	default:
		return fmt.Sprintf("%T", t)
	}
}
