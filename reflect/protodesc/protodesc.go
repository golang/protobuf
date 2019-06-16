// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protodesc provides for converting descriptorpb.FileDescriptorProto
// to/from the reflective protoreflect.FileDescriptor.
package protodesc

import (
	"strings"

	"google.golang.org/protobuf/internal/encoding/defval"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/prototype"
	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Resolver is the resolver used by NewFile to resolve dependencies.
// It is implemented by protoregistry.Files.
type Resolver interface {
	FindFileByPath(string) (protoreflect.FileDescriptor, error)
	FindEnumByName(protoreflect.FullName) (protoreflect.EnumDescriptor, error)
	FindMessageByName(protoreflect.FullName) (protoreflect.MessageDescriptor, error)
}

// TODO: Should we be responsible for validating other parts of the descriptor
// that we don't directly use?
//
// For example:
//	* That "json_name" is not set for an extension field. Maybe, maybe not.
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
func NewFile(fd *descriptorpb.FileDescriptorProto, r Resolver) (protoreflect.FileDescriptor, error) {
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
		imp := &f.Imports[i]
		fd, err := r.FindFileByPath(path)
		if err != nil {
			fd = prototype.PlaceholderFile(path, "")
		}
		imp.FileDescriptor = fd
	}

	imps := importedFiles(f.Imports)

	var err error
	f.Messages, err = messagesFromDescriptorProto(fd.GetMessageType(), imps, r)
	if err != nil {
		return nil, err
	}
	f.Enums, err = enumsFromDescriptorProto(fd.GetEnumType(), r)
	if err != nil {
		return nil, err
	}
	f.Extensions, err = extensionsFromDescriptorProto(fd.GetExtension(), imps, r)
	if err != nil {
		return nil, err
	}
	f.Services, err = servicesFromDescriptorProto(fd.GetService(), imps, r)
	if err != nil {
		return nil, err
	}

	return prototype.NewFile(&f)
}

type importSet map[string]bool

func importedFiles(imps []protoreflect.FileImport) importSet {
	ret := make(importSet)
	for _, imp := range imps {
		ret[imp.Path()] = true
		addPublicImports(imp, ret)
	}
	return ret
}

func addPublicImports(fd protoreflect.FileDescriptor, out importSet) {
	imps := fd.Imports()
	for i := 0; i < imps.Len(); i++ {
		imp := imps.Get(i)
		if imp.IsPublic {
			out[imp.Path()] = true
			addPublicImports(imp, out)
		}
	}
}

func messagesFromDescriptorProto(mds []*descriptorpb.DescriptorProto, imps importSet, r Resolver) (ms []prototype.Message, err error) {
	for _, md := range mds {
		var m prototype.Message
		m.Name = protoreflect.Name(md.GetName())
		m.Options = md.GetOptions()
		m.IsMapEntry = md.GetOptions().GetMapEntry()

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
		resNames := prototype.Names(m.ReservedNames)
		resRanges := prototype.FieldRanges(m.ReservedRanges)
		extRanges := prototype.FieldRanges(m.ExtensionRanges)

		for _, fd := range md.GetField() {
			if fd.GetExtendee() != "" {
				return nil, errors.New("message field may not have extendee")
			}
			var f prototype.Field
			f.Name = protoreflect.Name(fd.GetName())
			if resNames.Has(f.Name) {
				return nil, errors.New("%v contains field with reserved name %q", m.Name, f.Name)
			}
			f.Number = protoreflect.FieldNumber(fd.GetNumber())
			if resRanges.Has(f.Number) {
				return nil, errors.New("%v contains field with reserved number %d", m.Name, f.Number)
			}
			if extRanges.Has(f.Number) {
				return nil, errors.New("%v contains field with number %d in extension range", m.Name, f.Number)
			}
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
				f.EnumType, err = findEnumDescriptor(fd.GetTypeName(), imps, r)
				if err != nil {
					return nil, err
				}
				if opts.GetWeak() && !f.EnumType.IsPlaceholder() {
					f.EnumType = prototype.PlaceholderEnum(f.EnumType.FullName())
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				f.MessageType, err = findMessageDescriptor(fd.GetTypeName(), imps, r)
				if err != nil {
					return nil, err
				}
				if opts.GetWeak() && !f.MessageType.IsPlaceholder() {
					f.MessageType = prototype.PlaceholderMessage(f.MessageType.FullName())
				}
			default:
				if fd.GetTypeName() != "" {
					return nil, errors.New("field of kind %v has type_name set", f.Kind)
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

		m.Messages, err = messagesFromDescriptorProto(md.GetNestedType(), imps, r)
		if err != nil {
			return nil, err
		}
		m.Enums, err = enumsFromDescriptorProto(md.GetEnumType(), r)
		if err != nil {
			return nil, err
		}
		m.Extensions, err = extensionsFromDescriptorProto(md.GetExtension(), imps, r)
		if err != nil {
			return nil, err
		}

		ms = append(ms, m)
	}
	return ms, nil
}

func enumsFromDescriptorProto(eds []*descriptorpb.EnumDescriptorProto, r Resolver) (es []prototype.Enum, err error) {
	for _, ed := range eds {
		var e prototype.Enum
		e.Name = protoreflect.Name(ed.GetName())
		e.Options = ed.GetOptions()
		for _, s := range ed.GetReservedName() {
			e.ReservedNames = append(e.ReservedNames, protoreflect.Name(s))
		}
		for _, rr := range ed.GetReservedRange() {
			e.ReservedRanges = append(e.ReservedRanges, [2]protoreflect.EnumNumber{
				protoreflect.EnumNumber(rr.GetStart()),
				protoreflect.EnumNumber(rr.GetEnd()),
			})
		}
		resNames := prototype.Names(e.ReservedNames)
		resRanges := prototype.EnumRanges(e.ReservedRanges)

		for _, vd := range ed.GetValue() {
			v := prototype.EnumValue{
				Name:    protoreflect.Name(vd.GetName()),
				Number:  protoreflect.EnumNumber(vd.GetNumber()),
				Options: vd.Options,
			}
			if resNames.Has(v.Name) {
				return nil, errors.New("enum %v contains value with reserved name %q", e.Name, v.Name)
			}
			if resRanges.Has(v.Number) {
				return nil, errors.New("enum %v contains value with reserved number %d", e.Name, v.Number)
			}
			e.Values = append(e.Values, v)
		}
		es = append(es, e)
	}
	return es, nil
}

func extensionsFromDescriptorProto(xds []*descriptorpb.FieldDescriptorProto, imps importSet, r Resolver) (xs []prototype.Extension, err error) {
	for _, xd := range xds {
		if xd.OneofIndex != nil {
			return nil, errors.New("extension may not have oneof_index")
		}
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
			x.EnumType, err = findEnumDescriptor(xd.GetTypeName(), imps, r)
			if err != nil {
				return nil, err
			}
		case protoreflect.MessageKind, protoreflect.GroupKind:
			x.MessageType, err = findMessageDescriptor(xd.GetTypeName(), imps, r)
			if err != nil {
				return nil, err
			}
		default:
			if xd.GetTypeName() != "" {
				return nil, errors.New("extension of kind %v has type_name set", x.Kind)
			}
		}
		x.ExtendedType, err = findMessageDescriptor(xd.GetExtendee(), imps, r)
		if err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	return xs, nil
}

func servicesFromDescriptorProto(sds []*descriptorpb.ServiceDescriptorProto, imps importSet, r Resolver) (ss []prototype.Service, err error) {
	for _, sd := range sds {
		var s prototype.Service
		s.Name = protoreflect.Name(sd.GetName())
		s.Options = sd.GetOptions()
		for _, md := range sd.GetMethod() {
			var m prototype.Method
			m.Name = protoreflect.Name(md.GetName())
			m.Options = md.GetOptions()
			m.InputType, err = findMessageDescriptor(md.GetInputType(), imps, r)
			if err != nil {
				return nil, err
			}
			m.OutputType, err = findMessageDescriptor(md.GetOutputType(), imps, r)
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

func findMessageDescriptor(s string, imps importSet, r Resolver) (protoreflect.MessageDescriptor, error) {
	if !strings.HasPrefix(s, ".") {
		return nil, errors.New("identifier name must be fully qualified with a leading dot: %v", s)
	}
	name := protoreflect.FullName(strings.TrimPrefix(s, "."))
	md, err := r.FindMessageByName(name)
	if err != nil {
		return prototype.PlaceholderMessage(name), nil
	}
	if err := validateFileInImports(md, imps); err != nil {
		return nil, err
	}
	return md, nil
}

func findEnumDescriptor(s string, imps importSet, r Resolver) (protoreflect.EnumDescriptor, error) {
	if !strings.HasPrefix(s, ".") {
		return nil, errors.New("identifier name must be fully qualified with a leading dot: %v", s)
	}
	name := protoreflect.FullName(strings.TrimPrefix(s, "."))
	ed, err := r.FindEnumByName(name)
	if err != nil {
		return prototype.PlaceholderEnum(name), nil
	}
	if err := validateFileInImports(ed, imps); err != nil {
		return nil, err
	}
	return ed, nil
}

func validateFileInImports(d protoreflect.Descriptor, imps importSet) error {
	fd := d.ParentFile()
	if fd == nil {
		return errors.New("%v has no parent FileDescriptor", d.FullName())
	}
	if !imps[fd.Path()] {
		return errors.New("reference to type %v without import of %v", d.FullName(), fd.Path())
	}
	return nil
}
