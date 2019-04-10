// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fileinit constructs protoreflect.FileDescriptors from the encoded
// file descriptor proto messages. This package uses a custom proto unmarshaler
// 1) to avoid a dependency on the descriptor proto 2) for performance to keep
// the initialization cost as low as possible.
package fileinit

import (
	"fmt"
	"reflect"
	"sync"

	descopts "github.com/golang/protobuf/v2/internal/descopts"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	pragma "github.com/golang/protobuf/v2/internal/pragma"
	pfmt "github.com/golang/protobuf/v2/internal/typefmt"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	preg "github.com/golang/protobuf/v2/reflect/protoregistry"
	piface "github.com/golang/protobuf/v2/runtime/protoiface"
)

// FileBuilder construct a protoreflect.FileDescriptor from the
// raw file descriptor and the Go types for declarations and dependencies.
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
type FileBuilder struct {
	// RawDescriptor is the wire-encoded bytes of FileDescriptorProto.
	RawDescriptor []byte

	// GoTypes is a unique set of the Go types for all declarations and
	// dependencies. Each type is represented as a zero value of the Go type.
	//
	// Declarations are Go types generated for enums and messages directly
	// declared (not publicly imported) in the proto source file.
	// Messages for map entries are included, but represented by nil.
	// Enum declarations in "flattened ordering" come first, followed by
	// message declarations in "flattened ordering". The length of each sub-list
	// is len(EnumOutputTypes) and len(MessageOutputTypes), respectively.
	//
	// Dependencies are Go types for enums or messages referenced by
	// message fields (excluding weak fields), for parent extended messages of
	// extension fields, for enums or messages referenced by extension fields,
	// and for input and output messages referenced by service methods.
	// Dependencies must come after declarations, but the ordering of
	// dependencies themselves is unspecified.
	GoTypes []interface{}

	// DependencyIndexes is an ordered list of indexes into GoTypes for the
	// dependencies of messages, extensions, or services. There are 4 sub-lists
	// each in "flattened ordering" concatenated back-to-back:
	//	* Extension field targets: list of the extended parent message of
	//	every extension. Length is len(ExtensionOutputTypes).
	//	* Message field dependencies: list of the enum or message type
	//	referred to by every message field.
	//	* Extension field dependencies: list of the enum or message type
	//	referred to by every extension field.
	//	* Service method dependencies: list of the input and output message type
	//	referred to by every service method.
	DependencyIndexes []int32

	// TODO: Provide a list of imported files.
	// FileDependencies []pref.FileDescriptor

	// TODO: Provide a list of extension types for options extensions.
	// OptionDependencies []pref.ExtensionType

	// LegacyExtensions are a list of legacy extension descriptors.
	// If provided, the pointer to the v1 ExtensionDesc will be stored into the
	// associated v2 ExtensionType and accessible via a pseudo-internal API.
	// Also, the v2 ExtensionType will be stored into each v1 ExtensionDesc.
	// If non-nil, len(LegacyExtensions) must equal len(ExtensionOutputTypes).
	LegacyExtensions []piface.ExtensionDescV1

	// EnumOutputTypes is where Init stores all initialized enum types
	// in "flattened ordering".
	EnumOutputTypes []pref.EnumType
	// MessageOutputTypes is where Init stores all initialized message types
	// in "flattened ordering". This includes slots for map entry messages,
	// which are skipped over.
	MessageOutputTypes []pimpl.MessageType
	// ExtensionOutputTypes is where Init stores all initialized extension types
	// in "flattened ordering".
	ExtensionOutputTypes []pref.ExtensionType

	// FilesRegistry is the file registry to register the file descriptor.
	// If nil, no registration occurs.
	FilesRegistry *preg.Files
	// TypesRegistry is the types registry to register each type descriptor.
	// If nil, no registration occurs.
	TypesRegistry *preg.Types
}

// Init constructs a FileDescriptor given the parameters set in FileBuilder.
// It assumes that the inputs are well-formed and panics if any inconsistencies
// are encountered.
func (fb FileBuilder) Init() pref.FileDescriptor {
	fd := newFileDesc(fb)

	// Keep v1 and v2 extension descriptors in sync.
	if fb.LegacyExtensions != nil {
		for i := range fd.allExtensions {
			fd.allExtensions[i].legacyDesc = &fb.LegacyExtensions[i]
			fb.LegacyExtensions[i].Type = &fd.allExtensions[i]
		}
	}

	// Copy type descriptors to the output.
	//
	// While iterating over the messages, we also determine whether the message
	// is a map entry type.
	messageGoTypes := fb.GoTypes[len(fd.allEnums):][:len(fd.allMessages)]
	for i := range fd.allEnums {
		fb.EnumOutputTypes[i] = &fd.allEnums[i]
	}
	for i := range fd.allMessages {
		if messageGoTypes[i] == nil {
			fd.allMessages[i].isMapEntry = true
		} else {
			fb.MessageOutputTypes[i].GoType = reflect.TypeOf(messageGoTypes[i])
			fb.MessageOutputTypes[i].PBType = fd.allMessages[i].asDesc().(pref.MessageType)
		}
	}
	for i := range fd.allExtensions {
		fb.ExtensionOutputTypes[i] = &fd.allExtensions[i]
	}

	// As a special-case for descriptor.proto,
	// locally register concrete message type for the options.
	if fd.Path() == "google/protobuf/descriptor.proto" && fd.Package() == "google.protobuf" {
		for i := range fd.allMessages {
			switch fd.allMessages[i].Name() {
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

	// Register file and type descriptors.
	if fb.FilesRegistry != nil {
		if err := fb.FilesRegistry.Register(fd); err != nil {
			panic(err)
		}
	}
	if fb.TypesRegistry != nil {
		for i := range fd.allEnums {
			if err := fb.TypesRegistry.Register(&fd.allEnums[i]); err != nil {
				panic(err)
			}
		}
		for i := range fd.allMessages {
			if mt, _ := fd.allMessages[i].asDesc().(pref.MessageType); mt != nil {
				if err := fb.TypesRegistry.Register(mt); err != nil {
					panic(err)
				}
			}
		}
		for i := range fd.allExtensions {
			if err := fb.TypesRegistry.Register(&fd.allExtensions[i]); err != nil {
				panic(err)
			}
		}
	}

	return fd
}

type (
	// fileInit contains a copy of certain fields in FileBuilder for use during
	// lazy initialization upon first use.
	fileInit struct {
		RawDescriptor     []byte
		GoTypes           []interface{}
		DependencyIndexes []int32
	}
	fileDesc struct {
		fileInit

		path         string
		protoPackage pref.FullName

		fileDecls

		enums      enumDescs
		messages   messageDescs
		extensions extensionDescs
		services   serviceDescs

		once sync.Once
		lazy *fileLazy // protected by once
	}
	fileDecls struct {
		allEnums      []enumDesc
		allMessages   []messageDesc
		allExtensions []extensionDesc
	}
	fileLazy struct {
		syntax  pref.Syntax
		imports fileImports
		byName  map[pref.FullName]pref.Descriptor
		options []byte
	}
)

func (fd *fileDesc) Parent() (pref.Descriptor, bool) { return nil, false }
func (fd *fileDesc) Index() int                      { return 0 }
func (fd *fileDesc) Syntax() pref.Syntax             { return fd.lazyInit().syntax }
func (fd *fileDesc) Name() pref.Name                 { return fd.Package().Name() }
func (fd *fileDesc) FullName() pref.FullName         { return fd.Package() }
func (fd *fileDesc) IsPlaceholder() bool             { return false }
func (fd *fileDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.File, fd.lazyInit().options)
}
func (fd *fileDesc) Path() string                                     { return fd.path }
func (fd *fileDesc) Package() pref.FullName                           { return fd.protoPackage }
func (fd *fileDesc) Imports() pref.FileImports                        { return &fd.lazyInit().imports }
func (fd *fileDesc) Enums() pref.EnumDescriptors                      { return &fd.enums }
func (fd *fileDesc) Messages() pref.MessageDescriptors                { return &fd.messages }
func (fd *fileDesc) Extensions() pref.ExtensionDescriptors            { return &fd.extensions }
func (fd *fileDesc) Services() pref.ServiceDescriptors                { return &fd.services }
func (fd *fileDesc) DescriptorByName(s pref.FullName) pref.Descriptor { return fd.lazyInit().byName[s] }
func (fd *fileDesc) Format(s fmt.State, r rune)                       { pfmt.FormatDesc(s, r, fd) }
func (fd *fileDesc) ProtoType(pref.FileDescriptor)                    {}
func (fd *fileDesc) ProtoInternal(pragma.DoNotImplement)              {}

type (
	enumDesc struct {
		baseDesc

		lazy *enumLazy // protected by fileDesc.once
	}
	enumLazy struct {
		typ reflect.Type
		new func(pref.EnumNumber) pref.Enum

		values     enumValueDescs
		resvNames  names
		resvRanges enumRanges
		options    []byte
	}
	enumValueDesc struct {
		baseDesc

		number  pref.EnumNumber
		options []byte
	}
)

func (ed *enumDesc) GoType() reflect.Type            { return ed.lazyInit().typ }
func (ed *enumDesc) New(n pref.EnumNumber) pref.Enum { return ed.lazyInit().new(n) }
func (ed *enumDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.Enum, ed.lazyInit().options)
}
func (ed *enumDesc) Values() pref.EnumValueDescriptors { return &ed.lazyInit().values }
func (ed *enumDesc) ReservedNames() pref.Names         { return &ed.lazyInit().resvNames }
func (ed *enumDesc) ReservedRanges() pref.EnumRanges   { return &ed.lazyInit().resvRanges }
func (ed *enumDesc) Format(s fmt.State, r rune)        { pfmt.FormatDesc(s, r, ed) }
func (ed *enumDesc) ProtoType(pref.EnumDescriptor)     {}
func (ed *enumDesc) lazyInit() *enumLazy {
	ed.parentFile.lazyInit() // implicitly initializes enumLazy
	return ed.lazy
}

func (ed *enumValueDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.EnumValue, ed.options)
}
func (ed *enumValueDesc) Number() pref.EnumNumber            { return ed.number }
func (ed *enumValueDesc) Format(s fmt.State, r rune)         { pfmt.FormatDesc(s, r, ed) }
func (ed *enumValueDesc) ProtoType(pref.EnumValueDescriptor) {}

type (
	messageType       struct{ *messageDesc }
	messageDescriptor struct{ *messageDesc }

	// messageDesc does not implement protoreflect.Descriptor to avoid
	// accidental usages of it as such. Use the asDesc method to retrieve one.
	messageDesc struct {
		baseDesc

		enums      enumDescs
		messages   messageDescs
		extensions extensionDescs

		isMapEntry bool
		lazy       *messageLazy // protected by fileDesc.once
	}
	messageLazy struct {
		typ reflect.Type
		new func() pref.Message

		isMessageSet    bool
		fields          fieldDescs
		oneofs          oneofDescs
		resvNames       names
		resvRanges      fieldRanges
		reqNumbers      fieldNumbers
		extRanges       fieldRanges
		extRangeOptions [][]byte
		options         []byte
	}
	fieldDesc struct {
		baseDesc

		number      pref.FieldNumber
		cardinality pref.Cardinality
		kind        pref.Kind
		hasJSONName bool
		jsonName    string
		hasPacked   bool
		isPacked    bool
		isWeak      bool
		isMap       bool
		defVal      defaultValue
		oneofType   pref.OneofDescriptor
		enumType    pref.EnumDescriptor
		messageType pref.MessageDescriptor
		options     []byte
	}
	oneofDesc struct {
		baseDesc

		fields  oneofFields
		options []byte
	}
)

func (md *messageDesc) options() pref.ProtoMessage {
	return unmarshalOptions(descopts.Message, md.lazyInit().options)
}
func (md *messageDesc) IsMapEntry() bool                   { return md.isMapEntry }
func (md *messageDesc) Fields() pref.FieldDescriptors      { return &md.lazyInit().fields }
func (md *messageDesc) Oneofs() pref.OneofDescriptors      { return &md.lazyInit().oneofs }
func (md *messageDesc) ReservedNames() pref.Names          { return &md.lazyInit().resvNames }
func (md *messageDesc) ReservedRanges() pref.FieldRanges   { return &md.lazyInit().resvRanges }
func (md *messageDesc) RequiredNumbers() pref.FieldNumbers { return &md.lazyInit().reqNumbers }
func (md *messageDesc) ExtensionRanges() pref.FieldRanges  { return &md.lazyInit().extRanges }
func (md *messageDesc) ExtensionRangeOptions(i int) pref.ProtoMessage {
	return unmarshalOptions(descopts.ExtensionRange, md.lazyInit().extRangeOptions[i])
}
func (md *messageDesc) Enums() pref.EnumDescriptors           { return &md.enums }
func (md *messageDesc) Messages() pref.MessageDescriptors     { return &md.messages }
func (md *messageDesc) Extensions() pref.ExtensionDescriptors { return &md.extensions }
func (md *messageDesc) ProtoType(pref.MessageDescriptor)      {}
func (md *messageDesc) Format(s fmt.State, r rune)            { pfmt.FormatDesc(s, r, md.asDesc()) }
func (md *messageDesc) lazyInit() *messageLazy {
	md.parentFile.lazyInit() // implicitly initializes messageLazy
	return md.lazy
}

// IsMessageSet is a pseudo-internal API for checking whether a message
// should serialize in the proto1 message format.
//
// WARNING: This method is exempt from the compatibility promise and may be
// removed in the future without warning.
func (md *messageDesc) IsMessageSet() bool {
	return md.lazyInit().isMessageSet
}

// asDesc returns a protoreflect.MessageDescriptor or protoreflect.MessageType
// depending on whether the message is a map entry or not.
func (mb *messageDesc) asDesc() pref.MessageDescriptor {
	if !mb.isMapEntry {
		return messageType{mb}
	}
	return messageDescriptor{mb}
}
func (mt messageType) GoType() reflect.Type             { return mt.lazyInit().typ }
func (mt messageType) New() pref.Message                { return mt.lazyInit().new() }
func (mt messageType) Options() pref.ProtoMessage       { return mt.options() }
func (md messageDescriptor) Options() pref.ProtoMessage { return md.options() }

func (fd *fieldDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.Field, fd.options)
}
func (fd *fieldDesc) Number() pref.FieldNumber                   { return fd.number }
func (fd *fieldDesc) Cardinality() pref.Cardinality              { return fd.cardinality }
func (fd *fieldDesc) Kind() pref.Kind                            { return fd.kind }
func (fd *fieldDesc) HasJSONName() bool                          { return fd.hasJSONName }
func (fd *fieldDesc) JSONName() string                           { return fd.jsonName }
func (fd *fieldDesc) IsPacked() bool                             { return fd.isPacked }
func (fd *fieldDesc) IsWeak() bool                               { return fd.isWeak }
func (fd *fieldDesc) IsMap() bool                                { return fd.isMap }
func (fd *fieldDesc) HasDefault() bool                           { return fd.defVal.has }
func (fd *fieldDesc) Default() pref.Value                        { return fd.defVal.get() }
func (fd *fieldDesc) DefaultEnumValue() pref.EnumValueDescriptor { return fd.defVal.enum }
func (fd *fieldDesc) OneofType() pref.OneofDescriptor            { return fd.oneofType }
func (fd *fieldDesc) ExtendedType() pref.MessageDescriptor       { return nil }
func (fd *fieldDesc) EnumType() pref.EnumDescriptor              { return fd.enumType }
func (fd *fieldDesc) MessageType() pref.MessageDescriptor        { return fd.messageType }
func (fd *fieldDesc) Format(s fmt.State, r rune)                 { pfmt.FormatDesc(s, r, fd) }
func (fd *fieldDesc) ProtoType(pref.FieldDescriptor)             {}

func (od *oneofDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.Oneof, od.options)
}
func (od *oneofDesc) Fields() pref.FieldDescriptors  { return &od.fields }
func (od *oneofDesc) Format(s fmt.State, r rune)     { pfmt.FormatDesc(s, r, od) }
func (od *oneofDesc) ProtoType(pref.OneofDescriptor) {}

type (
	extensionDesc struct {
		baseDesc

		number       pref.FieldNumber
		extendedType pref.MessageDescriptor

		legacyDesc *piface.ExtensionDescV1

		lazy *extensionLazy // protected by fileDesc.once
	}
	extensionLazy struct {
		typ         reflect.Type
		new         func() pref.Value
		valueOf     func(interface{}) pref.Value
		interfaceOf func(pref.Value) interface{}

		cardinality pref.Cardinality
		kind        pref.Kind
		// Extensions should not have JSON names, but older versions of protoc
		// used to set one on the descriptor. Preserve it for now to maintain
		// the property that protoc 3.6.1 descriptors can round-trip through
		// this package losslessly.
		//
		// TODO: Consider whether to drop JSONName parsing from extensions.
		hasJSONName bool
		jsonName    string
		isPacked    bool
		defVal      defaultValue
		enumType    pref.EnumType
		messageType pref.MessageType
		options     []byte
	}
)

func (xd *extensionDesc) GoType() reflect.Type                 { return xd.lazyInit().typ }
func (xd *extensionDesc) New() pref.Value                      { return xd.lazyInit().new() }
func (xd *extensionDesc) ValueOf(v interface{}) pref.Value     { return xd.lazyInit().valueOf(v) }
func (xd *extensionDesc) InterfaceOf(v pref.Value) interface{} { return xd.lazyInit().interfaceOf(v) }
func (xd *extensionDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.Field, xd.lazyInit().options)
}
func (xd *extensionDesc) Number() pref.FieldNumber                   { return xd.number }
func (xd *extensionDesc) Cardinality() pref.Cardinality              { return xd.lazyInit().cardinality }
func (xd *extensionDesc) Kind() pref.Kind                            { return xd.lazyInit().kind }
func (xd *extensionDesc) HasJSONName() bool                          { return xd.lazyInit().hasJSONName }
func (xd *extensionDesc) JSONName() string                           { return xd.lazyInit().jsonName }
func (xd *extensionDesc) IsPacked() bool                             { return xd.lazyInit().isPacked }
func (xd *extensionDesc) IsWeak() bool                               { return false }
func (xd *extensionDesc) IsMap() bool                                { return false }
func (xd *extensionDesc) HasDefault() bool                           { return xd.lazyInit().defVal.has }
func (xd *extensionDesc) Default() pref.Value                        { return xd.lazyInit().defVal.get() }
func (xd *extensionDesc) DefaultEnumValue() pref.EnumValueDescriptor { return xd.lazyInit().defVal.enum }
func (xd *extensionDesc) OneofType() pref.OneofDescriptor            { return nil }
func (xd *extensionDesc) ExtendedType() pref.MessageDescriptor       { return xd.extendedType }
func (xd *extensionDesc) EnumType() pref.EnumDescriptor              { return xd.lazyInit().enumType }
func (xd *extensionDesc) MessageType() pref.MessageDescriptor        { return xd.lazyInit().messageType }
func (xd *extensionDesc) Format(s fmt.State, r rune)                 { pfmt.FormatDesc(s, r, xd) }
func (xd *extensionDesc) ProtoType(pref.FieldDescriptor)             {}
func (xd *extensionDesc) ProtoInternal(pragma.DoNotImplement)        {}
func (xd *extensionDesc) lazyInit() *extensionLazy {
	xd.parentFile.lazyInit() // implicitly initializes extensionLazy
	return xd.lazy
}

// ProtoLegacyExtensionDesc is a pseudo-internal API for allowing the v1 code
// to be able to retrieve a v1 ExtensionDesc.
//
// WARNING: This method is exempt from the compatibility promise and may be
// removed in the future without warning.
func (xd *extensionDesc) ProtoLegacyExtensionDesc() *piface.ExtensionDescV1 {
	return xd.legacyDesc
}

type (
	serviceDesc struct {
		baseDesc

		lazy *serviceLazy // protected by fileDesc.once
	}
	serviceLazy struct {
		methods methodDescs
		options []byte
	}
	methodDesc struct {
		baseDesc

		inputType         pref.MessageDescriptor
		outputType        pref.MessageDescriptor
		isStreamingClient bool
		isStreamingServer bool
		options           []byte
	}
)

func (sd *serviceDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.Service, sd.lazyInit().options)
}
func (sd *serviceDesc) Methods() pref.MethodDescriptors     { return &sd.lazyInit().methods }
func (sd *serviceDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, sd) }
func (sd *serviceDesc) ProtoType(pref.ServiceDescriptor)    {}
func (sd *serviceDesc) ProtoInternal(pragma.DoNotImplement) {}
func (sd *serviceDesc) lazyInit() *serviceLazy {
	sd.parentFile.lazyInit() // implicitly initializes serviceLazy
	return sd.lazy
}

func (md *methodDesc) Options() pref.ProtoMessage {
	return unmarshalOptions(descopts.Method, md.options)
}
func (md *methodDesc) InputType() pref.MessageDescriptor   { return md.inputType }
func (md *methodDesc) OutputType() pref.MessageDescriptor  { return md.outputType }
func (md *methodDesc) IsStreamingClient() bool             { return md.isStreamingClient }
func (md *methodDesc) IsStreamingServer() bool             { return md.isStreamingServer }
func (md *methodDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, md) }
func (md *methodDesc) ProtoType(pref.MethodDescriptor)     {}
func (md *methodDesc) ProtoInternal(pragma.DoNotImplement) {}

type baseDesc struct {
	parentFile *fileDesc
	parent     pref.Descriptor
	index      int
	fullName
}

func (d *baseDesc) Parent() (pref.Descriptor, bool)     { return d.parent, true }
func (d *baseDesc) Index() int                          { return d.index }
func (d *baseDesc) Syntax() pref.Syntax                 { return d.parentFile.Syntax() }
func (d *baseDesc) IsPlaceholder() bool                 { return false }
func (d *baseDesc) ProtoInternal(pragma.DoNotImplement) {}

type fullName struct {
	shortLen int
	fullName pref.FullName
}

func (s *fullName) Name() pref.Name         { return pref.Name(s.fullName[len(s.fullName)-s.shortLen:]) }
func (s *fullName) FullName() pref.FullName { return s.fullName }

func unmarshalOptions(p pref.ProtoMessage, b []byte) pref.ProtoMessage {
	if b != nil {
		// TODO: Consider caching the unmarshaled options message.
		p = reflect.New(reflect.TypeOf(p).Elem()).Interface().(pref.ProtoMessage)
		if err := proto.Unmarshal(b, p.(proto.Message)); err != nil {
			panic(err)
		}
	}
	return p.(proto.Message)
}
