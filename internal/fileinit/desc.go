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

	pragma "github.com/golang/protobuf/v2/internal/pragma"
	pfmt "github.com/golang/protobuf/v2/internal/typefmt"
	"github.com/golang/protobuf/v2/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
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

	// EnumOutputTypes is where Init stores all initialized enum types
	// in "flattened ordering".
	EnumOutputTypes []pref.EnumType
	// MessageOutputTypes is where Init stores all initialized message types
	// in "flattened ordering"; this includes map entry types.
	MessageOutputTypes []pref.MessageType
	// ExtensionOutputTypes is where Init stores all initialized extension types
	// in "flattened ordering".
	ExtensionOutputTypes []pref.ExtensionType

	// TODO: Provide ability for FileBuilder to handle registration?
	// FilesRegistry *pref.Files
	// TypesRegistry *pref.Types
}

// Init constructs a FileDescriptor given the parameters set in FileBuilder.
// It assumes that the inputs are well-formed and panics if any inconsistencies
// are encountered.
func (fb FileBuilder) Init() pref.FileDescriptor {
	fd := newFileDesc(fb)

	for i := range fd.allEnums {
		fb.EnumOutputTypes[i] = &fd.allEnums[i]
	}
	for i := range fd.allMessages {
		fb.MessageOutputTypes[i] = &fd.allMessages[i]
	}
	for i := range fd.allExtensions {
		fb.ExtensionOutputTypes[i] = &fd.allExtensions[i]
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
func (fd *fileDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.FileOptions(), fd.lazyInit().options)
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
func (ed *enumDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.EnumOptions(), ed.lazyInit().options)
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

func (ed *enumValueDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.EnumValueOptions(), ed.options)
}
func (ed *enumValueDesc) Number() pref.EnumNumber            { return ed.number }
func (ed *enumValueDesc) Format(s fmt.State, r rune)         { pfmt.FormatDesc(s, r, ed) }
func (ed *enumValueDesc) ProtoType(pref.EnumValueDescriptor) {}

type (
	messageDesc struct {
		baseDesc

		enums      enumDescs
		messages   messageDescs
		extensions extensionDescs

		lazy *messageLazy // protected by fileDesc.once
	}
	messageLazy struct {
		typ reflect.Type
		new func() pref.Message

		isMapEntry      bool
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

func (md *messageDesc) GoType() reflect.Type { return md.lazyInit().typ }
func (md *messageDesc) New() pref.Message    { return md.lazyInit().new() }
func (md *messageDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.MessageOptions(), md.lazyInit().options)
}
func (md *messageDesc) IsMapEntry() bool                   { return md.lazyInit().isMapEntry }
func (md *messageDesc) Fields() pref.FieldDescriptors      { return &md.lazyInit().fields }
func (md *messageDesc) Oneofs() pref.OneofDescriptors      { return &md.lazyInit().oneofs }
func (md *messageDesc) ReservedNames() pref.Names          { return &md.lazyInit().resvNames }
func (md *messageDesc) ReservedRanges() pref.FieldRanges   { return &md.lazyInit().resvRanges }
func (md *messageDesc) RequiredNumbers() pref.FieldNumbers { return &md.lazyInit().reqNumbers }
func (md *messageDesc) ExtensionRanges() pref.FieldRanges  { return &md.lazyInit().extRanges }
func (md *messageDesc) ExtensionRangeOptions(i int) pref.OptionsMessage {
	return unmarshalOptions(ptype.X.ExtensionRangeOptions(), md.lazyInit().extRangeOptions[i])
}
func (md *messageDesc) Enums() pref.EnumDescriptors           { return &md.enums }
func (md *messageDesc) Messages() pref.MessageDescriptors     { return &md.messages }
func (md *messageDesc) Extensions() pref.ExtensionDescriptors { return &md.extensions }
func (md *messageDesc) Format(s fmt.State, r rune)            { pfmt.FormatDesc(s, r, md) }
func (md *messageDesc) ProtoType(pref.MessageDescriptor)      {}
func (md *messageDesc) lazyInit() *messageLazy {
	md.parentFile.lazyInit() // implicitly initializes messageLazy
	return md.lazy
}

func (fd *fieldDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.FieldOptions(), fd.options)
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

func (od *oneofDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.OneofOptions(), od.options)
}
func (od *oneofDesc) Fields() pref.FieldDescriptors  { return &od.fields }
func (od *oneofDesc) Format(s fmt.State, r rune)     { pfmt.FormatDesc(s, r, od) }
func (od *oneofDesc) ProtoType(pref.OneofDescriptor) {}

type (
	extensionDesc struct {
		baseDesc

		number       pref.FieldNumber
		extendedType pref.MessageDescriptor

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
func (xd *extensionDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.FieldOptions(), xd.lazyInit().options)
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

func (sd *serviceDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.ServiceOptions(), sd.lazyInit().options)
}
func (sd *serviceDesc) Methods() pref.MethodDescriptors     { return &sd.lazyInit().methods }
func (sd *serviceDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, sd) }
func (sd *serviceDesc) ProtoType(pref.ServiceDescriptor)    {}
func (sd *serviceDesc) ProtoInternal(pragma.DoNotImplement) {}
func (sd *serviceDesc) lazyInit() *serviceLazy {
	sd.parentFile.lazyInit() // implicitly initializes serviceLazy
	return sd.lazy
}

func (md *methodDesc) Options() pref.OptionsMessage {
	return unmarshalOptions(ptype.X.MethodOptions(), md.options)
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

func unmarshalOptions(p pref.OptionsMessage, b []byte) pref.OptionsMessage {
	if b != nil {
		// TODO: Consider caching the unmarshaled options message.
		p = reflect.New(reflect.TypeOf(p).Elem()).Interface().(pref.OptionsMessage)
		if err := proto.Unmarshal(b, p.(proto.Message)); err != nil {
			panic(err)
		}
	}
	return p.(proto.Message)
}
