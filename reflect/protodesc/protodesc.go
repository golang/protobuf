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
	"google.golang.org/protobuf/internal/filedesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Resolver is the resolver used by NewFile to resolve dependencies.
// It is implemented by protoregistry.Files.
type Resolver interface {
	FindFileByPath(string) (protoreflect.FileDescriptor, error)
	FindDescriptorByName(protoreflect.FullName) (protoreflect.Descriptor, error)
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

// TODO: This is important to prevent users from creating invalid types,
// but is not functionality needed now.
//
// Things to verify:
//	* Weak fields are only used if flags.Proto1Legacy is set
//	* Weak fields can only reference singular messages
//	(check if this the case for oneof fields)
//	* FieldDescriptor.MessageType cannot reference a remote type when the
//	remote name is a type within the local file.
//	* Default enum identifiers resolve to a declared number.
//	* Default values are only allowed in proto2.
//	* Default strings are valid UTF-8? Note that protoc does not check this.
//	* Field extensions are only valid in proto2, except when extending the
//	descriptor options.
//	* Remote enum and message types are actually found in imported files.
//	* Placeholder messages and types may only be for weak fields.
//	* Placeholder full names must be valid.
//	* The name of each descriptor must be valid.
//	* Options are of the correct Go type (e.g. *descriptorpb.MessageOptions).
// 	* len(ExtensionRangeOptions) <= len(ExtensionRanges)

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
	if r == nil {
		r = (*protoregistry.Files)(nil) // empty resolver
	}
	f := &filedesc.File{L2: &filedesc.FileL2{}}
	switch fd.GetSyntax() {
	case "proto2", "":
		f.L1.Syntax = protoreflect.Proto2
	case "proto3":
		f.L1.Syntax = protoreflect.Proto3
	default:
		return nil, errors.New("invalid syntax: %v", fd.GetSyntax())
	}
	f.L1.Path = fd.GetName()
	f.L1.Package = protoreflect.FullName(fd.GetPackage())
	if opts := fd.GetOptions(); opts != nil {
		f.L2.Options = func() protoreflect.ProtoMessage { return opts }
	}

	f.L2.Imports = make(filedesc.FileImports, len(fd.GetDependency()))
	for _, i := range fd.GetPublicDependency() {
		if int(i) >= len(f.L2.Imports) || f.L2.Imports[i].IsPublic {
			return nil, errors.New("invalid or duplicate public import index: %d", i)
		}
		f.L2.Imports[i].IsPublic = true
	}
	for _, i := range fd.GetWeakDependency() {
		if int(i) >= len(f.L2.Imports) || f.L2.Imports[i].IsWeak {
			return nil, errors.New("invalid or duplicate weak import index: %d", i)
		}
		f.L2.Imports[i].IsWeak = true
	}
	for i, path := range fd.GetDependency() {
		imp := &f.L2.Imports[i]
		f, err := r.FindFileByPath(path)
		if err != nil {
			// TODO: This should be an error.
			f = filedesc.PlaceholderFile(path)
		}
		imp.FileDescriptor = f
	}

	// Step 1: Pre-initialize all declared enums and messages.
	// This enables step 2 to properly resolve references to locally defined
	// enums and messages.
	rl := make(descsByName)
	f.L1.Enums.List = rl.initEnumsFromDescriptorProto(fd.GetEnumType(), f)
	f.L1.Messages.List = rl.initMessagesFromDescriptorProto(fd.GetMessageType(), f)
	f.L1.Extensions.List = rl.initExtensionsFromDescriptorProto(fd.GetExtension(), f)
	f.L1.Services.List = rl.initServicesFromDescriptorProto(fd.GetService(), f)

	// Step 2: Handle every enum, message, extension, or service declaration.
	r = resolver{local: rl, remote: r} // wrap remote resolver with local declarations
	imps := importSet{f.L1.Path: true}
	imps.importFiles(f.L2.Imports)
	if err := enumsFromDescriptorProto(f.L1.Enums.List, fd.GetEnumType(), imps, r); err != nil {
		return nil, err
	}
	if err := messagesFromDescriptorProto(f.L1.Messages.List, fd.GetMessageType(), imps, r); err != nil {
		return nil, err
	}
	if err := extensionsFromDescriptorProto(f.L1.Extensions.List, fd.GetExtension(), imps, r); err != nil {
		return nil, err
	}
	if err := servicesFromDescriptorProto(f.L1.Services.List, fd.GetService(), imps, r); err != nil {
		return nil, err
	}

	return f, nil
}

type descsByName map[protoreflect.FullName]protoreflect.Descriptor

func (r descsByName) initEnumsFromDescriptorProto(eds []*descriptorpb.EnumDescriptorProto, parent protoreflect.Descriptor) []filedesc.Enum {
	es := make([]filedesc.Enum, len(eds)) // allocate up-front to ensure stable pointers
	for i, ed := range eds {
		e := &es[i]
		e.L0 = makeBase(parent, ed.GetName(), i)
		r[e.FullName()] = e
	}
	return es
}

func (r descsByName) initMessagesFromDescriptorProto(mds []*descriptorpb.DescriptorProto, parent protoreflect.Descriptor) []filedesc.Message {
	ms := make([]filedesc.Message, len(mds)) // allocate up-front to ensure stable pointers
	for i, md := range mds {
		m := &ms[i]
		m.L0 = makeBase(parent, md.GetName(), i)
		m.L1.Enums.List = r.initEnumsFromDescriptorProto(md.GetEnumType(), m)
		m.L1.Messages.List = r.initMessagesFromDescriptorProto(md.GetNestedType(), m)
		m.L1.Extensions.List = r.initExtensionsFromDescriptorProto(md.GetExtension(), m)
		r[m.FullName()] = m
	}
	return ms
}

func (r descsByName) initExtensionsFromDescriptorProto(xds []*descriptorpb.FieldDescriptorProto, parent protoreflect.Descriptor) []filedesc.Extension {
	xs := make([]filedesc.Extension, len(xds)) // allocate up-front to ensure stable pointers
	for i, xd := range xds {
		x := &xs[i]
		x.L0 = makeBase(parent, xd.GetName(), i)
		r[x.FullName()] = x
	}
	return xs
}

func (r descsByName) initServicesFromDescriptorProto(sds []*descriptorpb.ServiceDescriptorProto, parent protoreflect.Descriptor) []filedesc.Service {
	ss := make([]filedesc.Service, len(sds)) // allocate up-front to ensure stable pointers
	for i, sd := range sds {
		s := &ss[i]
		s.L0 = makeBase(parent, sd.GetName(), i)
		r[s.FullName()] = s
	}
	return ss
}

func makeBase(parent protoreflect.Descriptor, name string, idx int) filedesc.BaseL0 {
	return filedesc.BaseL0{
		FullName:   parent.FullName().Append(protoreflect.Name(name)),
		ParentFile: parent.ParentFile().(*filedesc.File),
		Parent:     parent,
		Index:      idx,
	}
}

func enumsFromDescriptorProto(es []filedesc.Enum, eds []*descriptorpb.EnumDescriptorProto, imps importSet, r Resolver) error {
	for i, ed := range eds {
		e := &es[i]
		e.L2 = new(filedesc.EnumL2)
		if opts := ed.GetOptions(); opts != nil {
			e.L2.Options = func() protoreflect.ProtoMessage { return opts }
		}
		for _, s := range ed.GetReservedName() {
			e.L2.ReservedNames.List = append(e.L2.ReservedNames.List, protoreflect.Name(s))
		}
		for _, rr := range ed.GetReservedRange() {
			e.L2.ReservedRanges.List = append(e.L2.ReservedRanges.List, [2]protoreflect.EnumNumber{
				protoreflect.EnumNumber(rr.GetStart()),
				protoreflect.EnumNumber(rr.GetEnd()),
			})
		}
		e.L2.Values.List = make([]filedesc.EnumValue, len(ed.GetValue()))
		for j, vd := range ed.GetValue() {
			v := &e.L2.Values.List[j]
			v.L0 = makeBase(e, vd.GetName(), j)
			v.L0.FullName = e.L0.Parent.FullName().Append(protoreflect.Name(vd.GetName())) // enum values are in the same scope as the enum itself
			if opts := vd.GetOptions(); opts != nil {
				v.L1.Options = func() protoreflect.ProtoMessage { return opts }
			}
			v.L1.Number = protoreflect.EnumNumber(vd.GetNumber())
			if e.L2.ReservedNames.Has(v.Name()) {
				return errors.New("enum %v contains value with reserved name %q", e.Name(), v.Name())
			}
			if e.L2.ReservedRanges.Has(v.Number()) {
				return errors.New("enum %v contains value with reserved number %d", e.Name(), v.Number())
			}
		}
	}
	return nil
}

func messagesFromDescriptorProto(ms []filedesc.Message, mds []*descriptorpb.DescriptorProto, imps importSet, r Resolver) error {
	for i, md := range mds {
		m := &ms[i]

		// Handle nested declarations. All enums must be handled before handling
		// any messages since an enum field may have a default value that is
		// specified in the same file being constructed.
		if err := enumsFromDescriptorProto(m.L1.Enums.List, md.GetEnumType(), imps, r); err != nil {
			return err
		}
		if err := messagesFromDescriptorProto(m.L1.Messages.List, md.GetNestedType(), imps, r); err != nil {
			return err
		}
		if err := extensionsFromDescriptorProto(m.L1.Extensions.List, md.GetExtension(), imps, r); err != nil {
			return err
		}

		// Handle the message descriptor itself.
		m.L2 = new(filedesc.MessageL2)
		if opts := md.GetOptions(); opts != nil {
			m.L2.Options = func() protoreflect.ProtoMessage { return opts }
			m.L2.IsMapEntry = opts.GetMapEntry()
			m.L2.IsMessageSet = opts.GetMessageSetWireFormat()
		}
		for _, s := range md.GetReservedName() {
			m.L2.ReservedNames.List = append(m.L2.ReservedNames.List, protoreflect.Name(s))
		}
		for _, rr := range md.GetReservedRange() {
			m.L2.ReservedRanges.List = append(m.L2.ReservedRanges.List, [2]protoreflect.FieldNumber{
				protoreflect.FieldNumber(rr.GetStart()),
				protoreflect.FieldNumber(rr.GetEnd()),
			})
		}
		for _, xr := range md.GetExtensionRange() {
			m.L2.ExtensionRanges.List = append(m.L2.ExtensionRanges.List, [2]protoreflect.FieldNumber{
				protoreflect.FieldNumber(xr.GetStart()),
				protoreflect.FieldNumber(xr.GetEnd()),
			})
			var optsFunc func() protoreflect.ProtoMessage
			if opts := xr.GetOptions(); opts != nil {
				optsFunc = func() protoreflect.ProtoMessage { return opts }
			}
			m.L2.ExtensionRangeOptions = append(m.L2.ExtensionRangeOptions, optsFunc)
		}
		m.L2.Fields.List = make([]filedesc.Field, len(md.GetField()))
		m.L2.Oneofs.List = make([]filedesc.Oneof, len(md.GetOneofDecl()))
		for j, fd := range md.GetField() {
			f := &m.L2.Fields.List[j]
			f.L0 = makeBase(m, fd.GetName(), j)
			if opts := fd.GetOptions(); opts != nil {
				f.L1.Options = func() protoreflect.ProtoMessage { return opts }
				f.L1.IsWeak = opts.GetWeak()
				f.L1.HasPacked = opts.Packed != nil
				f.L1.IsPacked = opts.GetPacked()
			}
			if m.L2.ReservedNames.Has(f.Name()) {
				return errors.New("%v contains field with reserved name %q", m.Name(), f.Name())
			}
			f.L1.Number = protoreflect.FieldNumber(fd.GetNumber())
			if m.L2.ReservedRanges.Has(f.Number()) {
				return errors.New("%v contains field with reserved number %d", m.Name(), f.Number())
			}
			if m.L2.ExtensionRanges.Has(f.Number()) {
				return errors.New("%v contains field with number %d in extension range", m.Name(), f.Number())
			}
			f.L1.Cardinality = protoreflect.Cardinality(fd.GetLabel())
			if f.L1.Cardinality == protoreflect.Required {
				m.L2.RequiredNumbers.List = append(m.L2.RequiredNumbers.List, f.L1.Number)
			}
			f.L1.Kind = protoreflect.Kind(fd.GetType())
			if fd.JsonName != nil {
				f.L1.JSONName = filedesc.JSONName(fd.GetJsonName())
			}
			if fd.OneofIndex != nil {
				k := int(fd.GetOneofIndex())
				if k >= len(md.GetOneofDecl()) {
					return errors.New("invalid oneof index: %d", k)
				}
				o := &m.L2.Oneofs.List[k]
				f.L1.ContainingOneof = o
				o.L1.Fields.List = append(o.L1.Fields.List, f)
			}
			if fd.GetExtendee() != "" {
				return errors.New("message field may not have extendee")
			}
			switch f.L1.Kind {
			case protoreflect.EnumKind:
				ed, err := findEnumDescriptor(fd.GetTypeName(), f.L1.IsWeak, imps, r)
				if err != nil {
					return err
				}
				f.L1.Enum = ed
			case protoreflect.MessageKind, protoreflect.GroupKind:
				md, err := findMessageDescriptor(fd.GetTypeName(), f.L1.IsWeak, imps, r)
				if err != nil {
					return err
				}
				f.L1.Message = md
			default:
				if fd.GetTypeName() != "" {
					return errors.New("field of kind %v has type_name set", f.L1.Kind)
				}
			}
			if fd.DefaultValue != nil {
				// Handle default value after resolving the enum since the
				// list of enum values is needed to resolve enums by name.
				var evs protoreflect.EnumValueDescriptors
				if f.L1.Kind == protoreflect.EnumKind {
					evs = f.L1.Enum.Values()
				}
				v, ev, err := defval.Unmarshal(fd.GetDefaultValue(), f.L1.Kind, evs, defval.Descriptor)
				if err != nil {
					return err
				}
				f.L1.Default = filedesc.DefaultValue(v, ev)
			}
		}
		for j, od := range md.GetOneofDecl() {
			o := &m.L2.Oneofs.List[j]
			o.L0 = makeBase(m, od.GetName(), j)
			if opts := od.GetOptions(); opts != nil {
				o.L1.Options = func() protoreflect.ProtoMessage { return opts }
			}
		}
	}
	return nil
}

func extensionsFromDescriptorProto(xs []filedesc.Extension, xds []*descriptorpb.FieldDescriptorProto, imps importSet, r Resolver) error {
	for i, xd := range xds {
		x := &xs[i]
		x.L2 = new(filedesc.ExtensionL2)
		if opts := xd.GetOptions(); opts != nil {
			x.L2.Options = func() protoreflect.ProtoMessage { return opts }
			x.L2.IsPacked = opts.GetPacked()
		}
		x.L1.Number = protoreflect.FieldNumber(xd.GetNumber())
		x.L2.Cardinality = protoreflect.Cardinality(xd.GetLabel())
		x.L1.Kind = protoreflect.Kind(xd.GetType())
		if xd.JsonName != nil {
			x.L2.JSONName = filedesc.JSONName(xd.GetJsonName())
		}
		if xd.OneofIndex != nil {
			return errors.New("extension may not have oneof_index")
		}
		md, err := findMessageDescriptor(xd.GetExtendee(), false, imps, r)
		if err != nil {
			return err
		}
		x.L1.Extendee = md
		switch x.L1.Kind {
		case protoreflect.EnumKind:
			ed, err := findEnumDescriptor(xd.GetTypeName(), false, imps, r)
			if err != nil {
				return err
			}
			x.L2.Enum = ed
		case protoreflect.MessageKind, protoreflect.GroupKind:
			md, err := findMessageDescriptor(xd.GetTypeName(), false, imps, r)
			if err != nil {
				return err
			}
			x.L2.Message = md
		default:
			if xd.GetTypeName() != "" {
				return errors.New("field of kind %v has type_name set", x.L1.Kind)
			}
		}
		if xd.DefaultValue != nil {
			// Handle default value after resolving the enum since the
			// list of enum values is needed to resolve enums by name.
			var evs protoreflect.EnumValueDescriptors
			if x.L1.Kind == protoreflect.EnumKind {
				evs = x.L2.Enum.Values()
			}
			v, ev, err := defval.Unmarshal(xd.GetDefaultValue(), x.L1.Kind, evs, defval.Descriptor)
			if err != nil {
				return err
			}
			x.L2.Default = filedesc.DefaultValue(v, ev)
		}
	}
	return nil
}

func servicesFromDescriptorProto(ss []filedesc.Service, sds []*descriptorpb.ServiceDescriptorProto, imps importSet, r Resolver) (err error) {
	for i, sd := range sds {
		s := &ss[i]
		s.L2 = new(filedesc.ServiceL2)
		if opts := sd.GetOptions(); opts != nil {
			s.L2.Options = func() protoreflect.ProtoMessage { return opts }
		}
		s.L2.Methods.List = make([]filedesc.Method, len(sd.GetMethod()))
		for j, md := range sd.GetMethod() {
			m := &s.L2.Methods.List[j]
			m.L0 = makeBase(s, md.GetName(), j)
			if opts := md.GetOptions(); opts != nil {
				m.L1.Options = func() protoreflect.ProtoMessage { return opts }
			}
			m.L1.Input, err = findMessageDescriptor(md.GetInputType(), false, imps, r)
			if err != nil {
				return err
			}
			m.L1.Output, err = findMessageDescriptor(md.GetOutputType(), false, imps, r)
			if err != nil {
				return err
			}
			m.L1.IsStreamingClient = md.GetClientStreaming()
			m.L1.IsStreamingServer = md.GetServerStreaming()
		}
	}
	return nil
}

type resolver struct {
	local  descsByName
	remote Resolver
}

func (r resolver) FindFileByPath(s string) (protoreflect.FileDescriptor, error) {
	return r.remote.FindFileByPath(s)
}

func (r resolver) FindDescriptorByName(s protoreflect.FullName) (protoreflect.Descriptor, error) {
	if d, ok := r.local[s]; ok {
		return d, nil
	}
	return r.remote.FindDescriptorByName(s)
}

type importSet map[string]bool

func (is importSet) importFiles(files []protoreflect.FileImport) {
	for _, imp := range files {
		is[imp.Path()] = true
		is.importPublic(imp)
	}
}

func (is importSet) importPublic(fd protoreflect.FileDescriptor) {
	imps := fd.Imports()
	for i := 0; i < imps.Len(); i++ {
		if imp := imps.Get(i); imp.IsPublic {
			is[imp.Path()] = true
			is.importPublic(imp)
		}
	}
}

// check returns an error if d does not belong to a currently imported file.
func (is importSet) check(d protoreflect.Descriptor) error {
	if !is[d.ParentFile().Path()] {
		return errors.New("reference to type %v without import of %v", d.FullName(), d.ParentFile().Path())
	}
	return nil
}

// TODO: Should we allow relative names? The protoc compiler has emitted
// absolute names for some time now. Requiring absolute names as an input
// simplifies our implementation as we won't need to implement C++'s namespace
// scoping rules.

func findEnumDescriptor(s string, isWeak bool, imps importSet, r Resolver) (protoreflect.EnumDescriptor, error) {
	d, err := findDescriptor(s, isWeak, imps, r)
	if err != nil {
		return nil, err
	}
	if ed, ok := d.(protoreflect.EnumDescriptor); ok {
		if err == protoregistry.NotFound {
			if isWeak {
				return filedesc.PlaceholderEnum(protoreflect.FullName(s[1:])), nil
			}
			// TODO: This should be an error.
			return filedesc.PlaceholderEnum(protoreflect.FullName(s[1:])), nil
			// return nil, errors.New("could not resolve enum: %v", name)
		}
		return ed, nil
	}
	return nil, errors.New("invalid descriptor type")
}

func findMessageDescriptor(s string, isWeak bool, imps importSet, r Resolver) (protoreflect.MessageDescriptor, error) {
	d, err := findDescriptor(s, isWeak, imps, r)
	if err != nil {
		if err == protoregistry.NotFound {
			if isWeak {
				return filedesc.PlaceholderMessage(protoreflect.FullName(s[1:])), nil
			}
			// TODO: This should be an error.
			return filedesc.PlaceholderMessage(protoreflect.FullName(s[1:])), nil
			// return nil, errors.New("could not resolve enum: %v", name)
		}
		return nil, err
	}
	if md, ok := d.(protoreflect.MessageDescriptor); ok {
		return md, nil
	}
	return nil, errors.New("invalid descriptor type")
}

func findDescriptor(s string, isWeak bool, imps importSet, r Resolver) (protoreflect.Descriptor, error) {
	if !strings.HasPrefix(s, ".") {
		return nil, errors.New("identifier name must be fully qualified with a leading dot: %v", s)
	}
	name := protoreflect.FullName(strings.TrimPrefix(s, "."))
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	if err := imps.check(d); err != nil {
		return nil, err
	}
	return d, nil
}
