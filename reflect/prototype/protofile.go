// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prototype provides builders to construct protobuf types that
// implement the interfaces defined in the protoreflect package.
//
// Protobuf types can either be constructed as standalone types
// (e.g., StandaloneMessage), or together as a batch of types in a single
// proto file (e.g., File). When creating standalone types, additional
// information must be provided such as the full type name and the proto syntax.
// When creating an entire file, the syntax and full name is derived from
// the parent type.
//
// Most types contain options, defined as messages in descriptor.proto.
// To avoid cyclic dependencies, the prototype package treats these options
// as opaque protoreflect.ProtoMessage values. In some cases where the option
// contains semantically important information (e.g.,
// google.protobuf.MessageOptions.map_entry), this information must be provided
// as a field of the corresponding type (e.g., prototype.Message.MapEntry).
package prototype

import "github.com/golang/protobuf/v2/reflect/protoreflect"

// Every struct has a "meta" struct embedded within it as a pointer.
// The meta type provides additional data structures for efficient lookup on
// certain methods (e.g., ByName) or derived information that can be
// derived from the parent (e.g., FullName). The meta type is lazily allocated
// and initialized. This architectural approach keeps the literal representation
// smaller, which then keeps the generated code size smaller.

// TODO: Instead of a top-down construction approach where internal references
// to message types use placeholder types, we could add a Reference method
// on Message and Enum that creates a MessageDescriptor or EnumDescriptor
// reference that only becomes valid after NewFile.
// However, that API approach is more error prone, as it causes more memory
// aliasing and provides more opportunity for misuse.
// Also, it requires that NewFile at least eagerly initialize all
// messages and enums list types. We can always add that API in the future.

// File is a constructor for protoreflect.FileDescriptor.
type File struct {
	Syntax  protoreflect.Syntax
	Path    string
	Package protoreflect.FullName
	Imports []protoreflect.FileImport
	Options protoreflect.OptionsMessage

	Enums      []Enum
	Messages   []Message
	Extensions []Extension
	Services   []Service

	*fileMeta
}

// NewFile creates a new protoreflect.FileDescriptor from the provided value.
// The file must represent a valid proto file according to protobuf semantics.
//
// Fields that reference an enum or message that is being declared within the
// same File can be represented using a placeholder descriptor. NewFile will
// automatically resolve the placeholder to point to a concrete descriptor.
// Alternatively, a reference descriptor obtained via Enum.Reference or
// Message.Reference can be used instead. The placeholder approach makes it
// possible to declare the file descriptor as a single File literal and
// is generally easier to use. The reference approach is more performant,
// but also more error prone.
//
// The caller must relinquish full ownership of the input t and must not
// access or mutate any fields. The input must not contain slices that are
// sub-slices of each other.
func NewFile(t *File) (protoreflect.FileDescriptor, error) {
	// TODO: Provide an unverified make that avoids validating the file.
	// This is useful for generated code since we know that protoc-gen-go
	// already validated the protobuf types.
	ft := newFile(t)
	if err := validateFile(ft); err != nil {
		return nil, err
	}

	// TODO: When using reference descriptors, it is vital that all enums and
	// messages are touched so that they are initialized before returning.
	// Otherwise, reference descriptors may still be invalid.
	//
	// We can remove this once validateFile is implemented, since it will
	// inherently touch all the necessary messages and enums.
	visitMessages(ft)

	return ft, nil
}

func visitMessages(d interface {
	Enums() protoreflect.EnumDescriptors
	Messages() protoreflect.MessageDescriptors
}) {
	d.Enums()
	ms := d.Messages()
	for i := 0; i < ms.Len(); i++ {
		visitMessages(ms.Get(i))
	}
}

// Message is a constructor for protoreflect.MessageDescriptor.
type Message struct {
	Name                  protoreflect.Name
	Fields                []Field
	Oneofs                []Oneof
	ReservedNames         []protoreflect.Name
	ReservedRanges        [][2]protoreflect.FieldNumber
	ExtensionRanges       [][2]protoreflect.FieldNumber
	ExtensionRangeOptions []protoreflect.OptionsMessage
	Options               protoreflect.OptionsMessage
	IsMapEntry            bool

	Enums      []Enum
	Messages   []Message
	Extensions []Extension

	*messageMeta
}

// Reference returns m as a reference protoreflect.MessageDescriptor,
// which can be used to satisfy internal dependencies within a proto file.
// Methods on the returned descriptor are not valid until the file that this
// message belongs to has been constructed via NewFile.
func (m *Message) Reference() protoreflect.MessageDescriptor {
	return messageDesc{m}
}

// Field is a constructor for protoreflect.FieldDescriptor.
type Field struct {
	Name        protoreflect.Name
	Number      protoreflect.FieldNumber
	Cardinality protoreflect.Cardinality
	Kind        protoreflect.Kind
	JSONName    string
	Default     protoreflect.Value
	OneofName   protoreflect.Name
	MessageType protoreflect.MessageDescriptor
	EnumType    protoreflect.EnumDescriptor
	Options     protoreflect.OptionsMessage
	IsPacked    OptionalBool
	IsWeak      bool

	*fieldMeta
}

// Oneof is a constructor for protoreflect.OneofDescriptor.
type Oneof struct {
	Name    protoreflect.Name
	Options protoreflect.OptionsMessage

	*oneofMeta
}

// Extension is a constructor for protoreflect.ExtensionDescriptor.
type Extension struct {
	Name         protoreflect.Name
	Number       protoreflect.FieldNumber
	Cardinality  protoreflect.Cardinality
	Kind         protoreflect.Kind
	Default      protoreflect.Value
	MessageType  protoreflect.MessageDescriptor
	EnumType     protoreflect.EnumDescriptor
	ExtendedType protoreflect.MessageDescriptor
	Options      protoreflect.OptionsMessage
	IsPacked     OptionalBool

	*extensionMeta
}

// Enum is a constructor for protoreflect.EnumDescriptor.
type Enum struct {
	Name           protoreflect.Name
	Values         []EnumValue
	ReservedNames  []protoreflect.Name
	ReservedRanges [][2]protoreflect.EnumNumber
	Options        protoreflect.OptionsMessage

	*enumMeta
}

// Reference returns e as a reference protoreflect.EnumDescriptor,
// which can be used to satisfy internal dependencies within a proto file.
// Methods on the returned descriptor are not valid until the file that this
// enum belongs to has been constructed via NewFile.
func (e *Enum) Reference() protoreflect.EnumDescriptor {
	return enumDesc{e}
}

// EnumValue is a constructor for protoreflect.EnumValueDescriptor.
type EnumValue struct {
	Name    protoreflect.Name
	Number  protoreflect.EnumNumber
	Options protoreflect.OptionsMessage

	*enumValueMeta
}

// Service is a constructor for protoreflect.ServiceDescriptor.
type Service struct {
	Name    protoreflect.Name
	Methods []Method
	Options protoreflect.OptionsMessage

	*serviceMeta
}

// Method is a constructor for protoreflect.MethodDescriptor.
type Method struct {
	Name              protoreflect.Name
	InputType         protoreflect.MessageDescriptor
	OutputType        protoreflect.MessageDescriptor
	IsStreamingClient bool
	IsStreamingServer bool
	Options           protoreflect.OptionsMessage

	*methodMeta
}

// OptionalBool is a tristate boolean.
type OptionalBool uint8

// Tristate boolean values.
const (
	DefaultBool OptionalBool = iota
	True
	False
)
