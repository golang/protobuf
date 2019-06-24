// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package filetype provides functionality for wrapping descriptors
// with Go type information.
package filetype

import (
	"reflect"

	"google.golang.org/protobuf/internal/descopts"
	fdesc "google.golang.org/protobuf/internal/filedesc"
	pimpl "google.golang.org/protobuf/internal/impl"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	ptype "google.golang.org/protobuf/reflect/prototype"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// TypeBuilder constructs type descriptors from a raw file descriptor
// and associated Go types for each enum and message declaration.
//
//
// Flattened Ordering
//
// The protobuf type system represents declarations as a tree. Certain nodes in
// the tree require us to either associate it with a concrete Go type or to
// resolve a dependency, which is information that must be provided separately
// since it cannot be derived from the file descriptor alone.
//
// However, representing a tree as Go literals is difficult to simply do in a
// space and time efficient way. Thus, we store them as a flattened list of
// objects where the serialization order from the tree-based form is important.
//
// The "flattened ordering" is defined as a tree traversal of all enum, message,
// extension, and service declarations using the following algorithm:
//
//	def VisitFileDecls(fd):
//		for e in fd.Enums:      yield e
//		for m in fd.Messages:   yield m
//		for x in fd.Extensions: yield x
//		for s in fd.Services:   yield s
//		for m in fd.Messages:   yield from VisitMessageDecls(m)
//
//	def VisitMessageDecls(md):
//		for e in md.Enums:      yield e
//		for m in md.Messages:   yield m
//		for x in md.Extensions: yield x
//		for m in md.Messages:   yield from VisitMessageDecls(m)
//
// The traversal starts at the root file descriptor and yields each direct
// declaration within each node before traversing into sub-declarations
// that children themselves may have.
type TypeBuilder struct {
	// File is the underlying file descriptor builder.
	File fdesc.DescBuilder

	// GoTypes is a unique set of the Go types for all declarations and
	// dependencies. Each type is represented as a zero value of the Go type.
	//
	// Declarations are Go types generated for enums and messages directly
	// declared (not publicly imported) in the proto source file.
	// Messages for map entries are accounted for, but represented by nil.
	// Enum declarations in "flattened ordering" come first, followed by
	// message declarations in "flattened ordering".
	//
	// Dependencies are Go types for enums or messages referenced by
	// message fields (excluding weak fields), for parent extended messages of
	// extension fields, for enums or messages referenced by extension fields,
	// and for input and output messages referenced by service methods.
	// Dependencies must come after declarations, but the ordering of
	// dependencies themselves is unspecified.
	GoTypes []interface{}

	// DependencyIndexes is an ordered list of indexes into GoTypes for the
	// dependencies of messages, extensions, or services.
	//
	// There are 5 sub-lists in "flattened ordering" concatenated back-to-back:
	//	0. Message field dependencies: list of the enum or message type
	//	referred to by every message field.
	//	1. Extension field targets: list of the extended parent message of
	//	every extension.
	//	2. Extension field dependencies: list of the enum or message type
	//	referred to by every extension field.
	//	3. Service method inputs: list of the input message type
	//	referred to by every service method.
	//	4. Service method outputs: list of the output message type
	//	referred to by every service method.
	//
	// The offset into DependencyIndexes for the start of each sub-list
	// is appended to the end in reverse order.
	DependencyIndexes []int32

	// MessageInfos is a list of message infos in "flattened ordering".
	// If provided, the GoType and PBType for each element is populated.
	//
	// Requirement: len(MessageInfos) == len(Build.Messages)
	MessageInfos []pimpl.MessageInfo

	// LegacyExtensions is a list of legacy extensions in "flattened ordering".
	// If provided, the pointer to the v1 ExtensionDesc will be stored into the
	// associated v2 ExtensionType and accessible via a pseudo-internal API.
	// Also, the v2 ExtensionType will be stored into each v1 ExtensionDesc.
	//
	// Requirement: len(LegacyExtensions) == len(Build.Extensions)
	LegacyExtensions []piface.ExtensionDescV1

	// TypeRegistry is the registry to register each type descriptor.
	// If nil, it uses protoregistry.GlobalTypes.
	TypeRegistry interface {
		Register(...preg.Type) error
	}
}

func (tb TypeBuilder) Build() (out struct {
	File pref.FileDescriptor

	// Enums is all enum types in "flattened ordering".
	Enums []Enum
	// Messages is all message types in "flattened ordering".
	// It includes a stub message type for map entries.
	Messages []Message
	// Extensions is all extension types in "flattened ordering".
	Extensions []Extension
}) {
	// Replace the resolver with one that resolves dependencies by index,
	// which is faster and more reliable than relying on the global registry.
	if tb.File.FileRegistry == nil {
		tb.File.FileRegistry = preg.GlobalFiles
	}
	tb.File.FileRegistry = &resolverByIndex{
		goTypes:      tb.GoTypes,
		depIdxs:      tb.DependencyIndexes,
		fileRegistry: tb.File.FileRegistry,
	}

	// Initialize registry if unpopulated.
	if tb.TypeRegistry == nil {
		tb.TypeRegistry = preg.GlobalTypes
	}

	fbOut := tb.File.Build()
	out.File = fbOut.File

	// Process enums.
	enumGoTypes := tb.GoTypes[:len(fbOut.Enums)]
	if len(fbOut.Enums) > 0 {
		out.Enums = make([]Enum, len(fbOut.Enums))
		for i := range fbOut.Enums {
			out.Enums[i] = Enum{
				EnumDescriptor: &fbOut.Enums[i],
				NewEnum:        enumMaker(reflect.TypeOf(enumGoTypes[i])),
			}

			// Register enum types.
			if err := tb.TypeRegistry.Register(&out.Enums[i]); err != nil {
				panic(err)
			}
		}
	}

	// Process messages.
	messageGoTypes := tb.GoTypes[len(fbOut.Enums):][:len(fbOut.Messages)]
	if tb.MessageInfos != nil && len(tb.MessageInfos) != len(fbOut.Messages) {
		panic("mismatching message lengths")
	}
	if len(fbOut.Messages) > 0 {
		out.Messages = make([]Message, len(fbOut.Messages))
		for i := range fbOut.Messages {
			if messageGoTypes[i] == nil {
				continue // skip map entry
			}
			out.Messages[i] = Message{
				MessageDescriptor: &fbOut.Messages[i],
				NewMessage:        messageMaker(reflect.TypeOf(messageGoTypes[i])),
			}

			if tb.MessageInfos != nil {
				tb.MessageInfos[i].GoType = reflect.TypeOf(messageGoTypes[i])
				tb.MessageInfos[i].PBType = &out.Messages[i]
			}

			// Register message types.
			if err := tb.TypeRegistry.Register(&out.Messages[i]); err != nil {
				panic(err)
			}
		}

		// As a special-case for descriptor.proto,
		// locally register concrete message type for the options.
		if out.File.Path() == "google/protobuf/descriptor.proto" && out.File.Package() == "google.protobuf" {
			for i := range fbOut.Messages {
				switch fbOut.Messages[i].Name() {
				case "FileOptions":
					descopts.File = messageGoTypes[i].(pref.ProtoMessage)
				case "EnumOptions":
					descopts.Enum = messageGoTypes[i].(pref.ProtoMessage)
				case "EnumValueOptions":
					descopts.EnumValue = messageGoTypes[i].(pref.ProtoMessage)
				case "MessageOptions":
					descopts.Message = messageGoTypes[i].(pref.ProtoMessage)
				case "FieldOptions":
					descopts.Field = messageGoTypes[i].(pref.ProtoMessage)
				case "OneofOptions":
					descopts.Oneof = messageGoTypes[i].(pref.ProtoMessage)
				case "ExtensionRangeOptions":
					descopts.ExtensionRange = messageGoTypes[i].(pref.ProtoMessage)
				case "ServiceOptions":
					descopts.Service = messageGoTypes[i].(pref.ProtoMessage)
				case "MethodOptions":
					descopts.Method = messageGoTypes[i].(pref.ProtoMessage)
				}
			}
		}
	}

	// Process extensions.
	if tb.LegacyExtensions != nil && len(tb.LegacyExtensions) != len(fbOut.Extensions) {
		panic("mismatching extension lengths")
	}
	if len(fbOut.Extensions) > 0 {
		var depIdx int32
		out.Extensions = make([]Extension, len(fbOut.Extensions))
		for i := range fbOut.Extensions {
			out.Extensions[i] = Extension{Extension: ptype.Extension{
				ExtensionDescriptor: &fbOut.Extensions[i],
			}}

			// For enum and message kinds, determine the referent Go type so
			// that we can construct their constructors.
			const listExtDeps = 2
			switch fbOut.Extensions[i].L1.Kind {
			case pref.EnumKind:
				j := depIdxs.Get(tb.DependencyIndexes, listExtDeps, depIdx)
				goType := reflect.TypeOf(tb.GoTypes[j])
				out.Extensions[i].NewEnum = enumMaker(goType)
				depIdx++
			case pref.MessageKind, pref.GroupKind:
				j := depIdxs.Get(tb.DependencyIndexes, listExtDeps, depIdx)
				goType := reflect.TypeOf(tb.GoTypes[j])
				out.Extensions[i].NewMessage = messageMaker(goType)
				depIdx++
			}

			// Keep v1 and v2 extensions in sync.
			if tb.LegacyExtensions != nil {
				out.Extensions[i].legacyDesc = &tb.LegacyExtensions[i]
				tb.LegacyExtensions[i].Type = &out.Extensions[i]
			}

			// Register extension types.
			if err := tb.TypeRegistry.Register(&out.Extensions[i]); err != nil {
				panic(err)
			}
		}
	}

	return out
}

type depIdxs []int32

// Get retrieves the jth element of the ith sub-list.
func (x depIdxs) Get(i, j int32) int32 {
	return x[x[int32(len(x))-i-1]+j]
}

type (
	resolverByIndex struct {
		goTypes []interface{}
		depIdxs depIdxs
		fileRegistry
	}
	fileRegistry interface {
		FindFileByPath(string) (pref.FileDescriptor, error)
		FindDescriptorByName(pref.FullName) (pref.Descriptor, error)
		Register(...pref.FileDescriptor) error
	}
)

func (r *resolverByIndex) FindEnumByIndex(i, j int32, es []fdesc.Enum, ms []fdesc.Message) pref.EnumDescriptor {
	if depIdx := int(r.depIdxs.Get(i, j)); int(depIdx) < len(es)+len(ms) {
		return &es[depIdx]
	} else {
		return pimpl.Export{}.EnumDescriptorOf(r.goTypes[depIdx])
	}
}

func (r *resolverByIndex) FindMessageByIndex(i, j int32, es []fdesc.Enum, ms []fdesc.Message) pref.MessageDescriptor {
	if depIdx := int(r.depIdxs.Get(i, j)); depIdx < len(es)+len(ms) {
		return &ms[depIdx-len(es)]
	} else {
		return pimpl.Export{}.MessageDescriptorOf(r.goTypes[depIdx])
	}
}

func enumMaker(t reflect.Type) func(pref.EnumNumber) pref.Enum {
	return func(n pref.EnumNumber) pref.Enum {
		v := reflect.New(t).Elem()
		v.SetInt(int64(n))
		return v.Interface().(pref.Enum)
	}
}

func messageMaker(t reflect.Type) func() pref.Message {
	return func() pref.Message {
		return reflect.New(t.Elem()).Interface().(pref.ProtoMessage).ProtoReflect()
	}
}

type (
	Enum      = ptype.Enum
	Message   = ptype.Message
	Extension struct {
		ptype.Extension
		legacyDesc *piface.ExtensionDescV1
	}
)

// ProtoLegacyExtensionDesc is a pseudo-internal API for allowing the v1 code
// to be able to retrieve a v1 ExtensionDesc.
//
// WARNING: This method is exempt from the compatibility promise and may be
// removed in the future without warning.
func (x *Extension) ProtoLegacyExtensionDesc() *piface.ExtensionDescV1 {
	return x.legacyDesc
}
