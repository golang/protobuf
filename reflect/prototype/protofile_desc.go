// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	descriptorV1 "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/golang/protobuf/v2/internal/encoding/text"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"
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

// NewFileFromDescriptorProto creates a new protoreflect.FileDescriptor from
// the provided descriptor message. The file must represent a valid proto file
// according to protobuf semantics.
//
// Any import files, enum types, or message types referenced in the file are
// resolved using the provided registry. When looking up an import file path,
// the path must be unique. The newly created file descriptor is not registered
// back into the provided file registry.
//
// The caller must relinquish full ownership of the input fd and must not
// access or mutate any fields.
func NewFileFromDescriptorProto(fd *descriptorV1.FileDescriptorProto, r *protoregistry.Files) (protoreflect.FileDescriptor, error) {
	var f File
	switch fd.GetSyntax() {
	case "", "proto2":
		f.Syntax = protoreflect.Proto2
	case "proto3":
		f.Syntax = protoreflect.Proto3
	default:
		return nil, errors.New("invalid syntax: %v", fd.GetSyntax())
	}
	f.Path = fd.GetName()
	f.Package = protoreflect.FullName(fd.GetPackage())

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
			imp.FileDescriptor = PlaceholderFile(path, "")
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

	return NewFile(&f)
}

func messagesFromDescriptorProto(mds []*descriptorV1.DescriptorProto, syntax protoreflect.Syntax, r *protoregistry.Files) (ms []Message, err error) {
	for _, md := range mds {
		var m Message
		m.Name = protoreflect.Name(md.GetName())
		m.IsMapEntry = md.GetOptions().GetMapEntry()
		for _, fd := range md.GetField() {
			var f Field
			f.Name = protoreflect.Name(fd.GetName())
			f.Number = protoreflect.FieldNumber(fd.GetNumber())
			f.Cardinality = protoreflect.Cardinality(fd.GetLabel())
			f.Kind = protoreflect.Kind(fd.GetType())
			f.JSONName = fd.GetJsonName()
			if opts := fd.GetOptions(); opts != nil && opts.Packed != nil {
				f.IsPacked = *opts.Packed
			} else {
				// https://developers.google.com/protocol-buffers/docs/proto3:
				// "In proto3, repeated fields of scalar numeric types use packed
				// encoding by default."
				f.IsPacked = (syntax == protoreflect.Proto3 &&
					f.Cardinality == protoreflect.Repeated &&
					isScalarNumeric[f.Kind])
			}
			f.IsWeak = fd.GetOptions().GetWeak()
			if fd.DefaultValue != nil {
				f.Default, err = parseDefault(fd.GetDefaultValue(), f.Kind)
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
				if f.IsWeak && !f.EnumType.IsPlaceholder() {
					f.EnumType = PlaceholderEnum(f.EnumType.FullName())
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				f.MessageType, err = findMessageDescriptor(fd.GetTypeName(), r)
				if err != nil {
					return nil, err
				}
				if f.IsWeak && !f.MessageType.IsPlaceholder() {
					f.MessageType = PlaceholderMessage(f.MessageType.FullName())
				}
			}
			m.Fields = append(m.Fields, f)
		}
		for _, od := range md.GetOneofDecl() {
			m.Oneofs = append(m.Oneofs, Oneof{Name: protoreflect.Name(od.GetName())})
		}
		for _, xr := range md.GetExtensionRange() {
			m.ExtensionRanges = append(m.ExtensionRanges, [2]protoreflect.FieldNumber{
				protoreflect.FieldNumber(xr.GetStart()),
				protoreflect.FieldNumber(xr.GetEnd()),
			})
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

var isScalarNumeric = map[protoreflect.Kind]bool{
	protoreflect.BoolKind:     true,
	protoreflect.EnumKind:     true,
	protoreflect.Int32Kind:    true,
	protoreflect.Sint32Kind:   true,
	protoreflect.Uint32Kind:   true,
	protoreflect.Int64Kind:    true,
	protoreflect.Sint64Kind:   true,
	protoreflect.Uint64Kind:   true,
	protoreflect.Sfixed32Kind: true,
	protoreflect.Fixed32Kind:  true,
	protoreflect.FloatKind:    true,
	protoreflect.Sfixed64Kind: true,
	protoreflect.Fixed64Kind:  true,
	protoreflect.DoubleKind:   true,
}

func enumsFromDescriptorProto(eds []*descriptorV1.EnumDescriptorProto, r *protoregistry.Files) (es []Enum, err error) {
	for _, ed := range eds {
		var e Enum
		e.Name = protoreflect.Name(ed.GetName())
		for _, vd := range ed.GetValue() {
			e.Values = append(e.Values, EnumValue{
				Name:   protoreflect.Name(vd.GetName()),
				Number: protoreflect.EnumNumber(vd.GetNumber()),
			})
		}
		es = append(es, e)
	}
	return es, nil
}

func extensionsFromDescriptorProto(xds []*descriptorV1.FieldDescriptorProto, r *protoregistry.Files) (xs []Extension, err error) {
	for _, xd := range xds {
		var x Extension
		x.Name = protoreflect.Name(xd.GetName())
		x.Number = protoreflect.FieldNumber(xd.GetNumber())
		x.Cardinality = protoreflect.Cardinality(xd.GetLabel())
		x.Kind = protoreflect.Kind(xd.GetType())
		// TODO: When a proto3 file extends a proto2 message (permitted only for
		// extending descriptor options), does the extension have proto2 or proto3
		// semantics? If the latter, repeated, scalar, numeric, proto3 extension
		// fields should default to packed. If the former, perhaps the extension syntax
		// should be protoreflect.Proto2.
		x.IsPacked = xd.GetOptions().GetPacked()
		if xd.DefaultValue != nil {
			x.Default, err = parseDefault(xd.GetDefaultValue(), x.Kind)
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

func servicesFromDescriptorProto(sds []*descriptorV1.ServiceDescriptorProto, r *protoregistry.Files) (ss []Service, err error) {
	for _, sd := range sds {
		var s Service
		s.Name = protoreflect.Name(sd.GetName())
		for _, md := range sd.GetMethod() {
			var m Method
			m.Name = protoreflect.Name(md.GetName())
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
		return PlaceholderMessage(name), nil
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
		return PlaceholderEnum(name), nil
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

func parseDefault(s string, k protoreflect.Kind) (protoreflect.Value, error) {
	switch k {
	case protoreflect.BoolKind:
		switch s {
		case "true":
			return protoreflect.ValueOf(true), nil
		case "false":
			return protoreflect.ValueOf(false), nil
		}
	case protoreflect.EnumKind:
		// For enums, we are supposed to return a protoreflect.EnumNumber type.
		// However, default values record the name instead of the number.
		// We are unable to resolve the name into a number without additional
		// type information. Thus, we temporarily return the name identifier
		// for now and rely on logic in defaultValue.lazyInit to resolve it.
		return protoreflect.ValueOf(s), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(s, 0, 32)
		if err == nil {
			return protoreflect.ValueOf(int32(v)), nil
		}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(s, 0, 64)
		if err == nil {
			return protoreflect.ValueOf(int64(v)), nil
		}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(s, 0, 32)
		if err == nil {
			return protoreflect.ValueOf(uint64(v)), nil
		}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(s, 0, 64)
		if err == nil {
			return protoreflect.ValueOf(uint64(v)), nil
		}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		var v float64
		var err error
		switch s {
		case "nan":
			v = math.NaN()
		case "inf":
			v = math.Inf(+1)
		case "-inf":
			v = math.Inf(-1)
		default:
			v, err = strconv.ParseFloat(s, 64)
		}
		if err == nil {
			if k == protoreflect.FloatKind {
				return protoreflect.ValueOf(float32(v)), nil
			}
			return protoreflect.ValueOf(float64(v)), nil
		}
	case protoreflect.StringKind:
		// String values are already unescaped and can be used as is.
		return protoreflect.ValueOf(s), nil
	case protoreflect.BytesKind:
		// Bytes values use the same escaping as the text format,
		// however they lack the surrounding double quotes.
		// TODO: Export unmarshalString in the text package to avoid this hack.
		v, err := text.Unmarshal([]byte(`["` + s + `"]:0`))
		if err == nil && len(v.Message()) == 1 {
			s := v.Message()[0][0].String()
			return protoreflect.ValueOf([]byte(s)), nil
		}
	}
	return protoreflect.Value{}, errors.New("invalid default value for %v: %q", k, s)
}
