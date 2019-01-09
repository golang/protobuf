// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoreflect

import "github.com/golang/protobuf/v2/internal/encoding/wire"

// Enum is a reflection interface for a concrete enum value,
// which provides type information and a getter for the enum number.
// Enum does not provide a mutable API since enums are commonly backed by
// Go constants, which are not addressable.
type Enum interface {
	Type() EnumType

	// Number returns the enum value as an integer.
	Number() EnumNumber
}

// Message is a reflective interface for a concrete message value,
// which provides type information and getters/setters for individual fields.
//
// Concrete types may implement interfaces defined in proto/protoiface,
// which provide specialized, performant implementations of high-level
// operations such as Marshal and Unmarshal.
type Message interface {
	Type() MessageType

	// KnownFields returns an interface to access/mutate known fields.
	KnownFields() KnownFields

	// UnknownFields returns an interface to access/mutate unknown fields.
	UnknownFields() UnknownFields

	// Interface unwraps the message reflection interface and
	// returns the underlying proto.Message interface.
	Interface() ProtoMessage
}

// KnownFields provides accessor and mutator methods for known fields.
//
// Each field Value can either be a scalar, Message, List, or Map.
// The field is a List or Map if FieldDescriptor.Cardinality is Repeated and
// a Map if and only if FieldDescriptor.IsMap is true. The scalar type or
// underlying repeated element type is determined by the FieldDescriptor.Kind.
// See Value for a list of Go types associated with each Kind.
//
// Field extensions are handled as known fields once the extension type has been
// registered with KnownFields.ExtensionTypes.
//
// Len, Has, Get, Range, and ExtensionTypes are safe for concurrent use.
type KnownFields interface {
	// Len reports the number of fields that are populated.
	Len() int

	// Has reports whether a field is populated.
	//
	// Some fields have the property of nullability where it is possible to
	// distinguish between the default value of a field and whether the field
	// was explicitly populated with the default value. Only scalars in proto2,
	// member fields of a oneof, and singular messages are nullable.
	//
	// A nullable field is populated only if explicitly set.
	// A scalar field in proto3 is populated if it contains a non-zero value.
	// A repeated field is populated only if it is non-empty.
	Has(FieldNumber) bool

	// Get retrieves the value for a field with the given field number.
	// If the field is unpopulated, it returns the default value for scalars,
	// a mutable empty List for empty repeated fields, a mutable empty Map for
	// empty map fields, and an invalid value for message fields.
	// If the field is unknown (does not appear in MessageDescriptor.Fields
	// or ExtensionFieldTypes), it returns an invalid value.
	Get(FieldNumber) Value

	// Set stores the value for a field with the given field number.
	// Setting a field belonging to a oneof implicitly clears any other field
	// that may be currently set by the same oneof.
	//
	// When setting a composite type, it is unspecified whether the set
	// value aliases the source's memory in any way.
	//
	// It panics if the field number does not correspond with a known field
	// in MessageDescriptor.Fields or an extension field in ExtensionTypes.
	Set(FieldNumber, Value)

	// TODO: Document memory aliasing behavior when a field is cleared?
	// For example, if Mutable is called later, can it reuse memory?

	// Clear clears the field such that a subsequent call to Has reports false.
	// The operation does nothing if the field number does not correspond with
	// a known field or extension field.
	Clear(FieldNumber)

	// Range iterates over every populated field in an undefined order,
	// calling f for each field number and value encountered.
	// Range calls f Len times unless f returns false, which stops iteration.
	// While iterating, mutating operations through Set, Clear, or Mutable
	// may only be performed on the current field number.
	Range(f func(FieldNumber, Value) bool)

	// NewMessage returns a newly allocated empty message assignable to
	// the field of the given number.
	// It panics if the field is not a singular message.
	NewMessage(FieldNumber) Message

	// ExtensionTypes are extension field types that are known by this
	// specific message instance.
	ExtensionTypes() ExtensionFieldTypes
}

// UnknownFields are a list of unknown or unparsed fields and may contain
// field numbers corresponding with defined fields or extension fields.
// The ordering of fields is maintained for fields of the same field number.
// However, the relative ordering of fields with different field numbers
// is undefined.
//
// Len, Get, and Range are safe for concurrent use.
type UnknownFields interface {
	// Len reports the number of fields that are populated.
	Len() int

	// Get retrieves the raw bytes of fields with the given field number.
	// It returns an empty RawFields if there are no populated fields.
	//
	// The caller must not mutate the content of the retrieved RawFields.
	Get(FieldNumber) RawFields

	// Set stores the raw bytes of fields with the given field number.
	// The RawFields must be valid and correspond with the given field number;
	// an implementation may panic if the fields are invalid.
	// An empty RawFields may be passed to clear the fields.
	//
	// The caller must not mutate the content of the RawFields being stored.
	Set(FieldNumber, RawFields)

	// Range iterates over every populated field in an undefined order,
	// calling f for each field number and raw field value encountered.
	// Range calls f Len times unless f returns false, which stops iteration.
	// While iterating, mutating operations through Set may only be performed
	// on the current field number.
	//
	// While the iteration order is undefined, it is deterministic.
	// It is recommended, but not required, that fields be presented
	// in the order that they were encountered in the wire data.
	Range(f func(FieldNumber, RawFields) bool)

	// TODO: Should IsSupported be renamed as ReadOnly?
	// TODO: Should IsSupported panic on Set instead of silently ignore?

	// IsSupported reports whether this message supports unknown fields.
	// If false, UnknownFields ignores all Set operations.
	IsSupported() bool
}

// RawFields is the raw bytes for an ordered sequence of fields.
// Each field contains both the tag (representing field number and wire type),
// and also the wire data itself.
//
// Once stored, the content of a RawFields must be treated as immutable.
// The capacity of RawFields may be treated as mutable only for the use-case of
// appending additional data to store back into UnknownFields.
type RawFields []byte

// IsValid reports whether RawFields is syntactically correct wire format.
// All fields must belong to the same field number.
func (b RawFields) IsValid() bool {
	var want FieldNumber
	for len(b) > 0 {
		got, _, n := wire.ConsumeField(b)
		if n < 0 || (want > 0 && got != want) {
			return false
		}
		want = got
		b = b[n:]
	}
	return true
}

// ExtensionFieldTypes are the extension field types that this message instance
// has been extended with.
//
// Len, Get, and Range are safe for concurrent use.
type ExtensionFieldTypes interface {
	// Len reports the number of field extensions.
	Len() int

	// Register stores an ExtensionType.
	// The ExtensionType.ExtendedType must match the containing message type
	// and the field number must be within the valid extension ranges
	// (see MessageDescriptor.ExtensionRanges).
	// It panics if the extension has already been registered (i.e.,
	// a conflict by number or by full name).
	Register(ExtensionType)

	// Remove removes the ExtensionType.
	// It panics if a value for this extension field is still populated.
	// The operation does nothing if there is no associated type to remove.
	Remove(ExtensionType)

	// ByNumber looks up an extension by field number.
	// It returns nil if not found.
	ByNumber(FieldNumber) ExtensionType

	// ByName looks up an extension field by full name.
	// It returns nil if not found.
	ByName(FullName) ExtensionType

	// Range iterates over every registered field in an undefined order,
	// calling f for each extension descriptor encountered.
	// Range calls f Len times unless f returns false, which stops iteration.
	// While iterating, mutating operations through Remove may only
	// be performed on the current descriptor.
	Range(f func(ExtensionType) bool)
}

// List is an ordered list. Every element is considered populated
// (i.e., Get never provides and Set never accepts invalid Values).
// The element Value type is determined by the associated FieldDescriptor.Kind
// and cannot be a Map or List.
//
// Len and Get are safe for concurrent use.
type List interface {
	// Len reports the number of entries in the List.
	// Get, Set, Mutable, and Truncate panic with out of bound indexes.
	Len() int

	// Get retrieves the value at the given index.
	Get(int) Value

	// Set stores a value for the given index.
	//
	// When setting a composite type, it is unspecified whether the set
	// value aliases the source's memory in any way.
	Set(int, Value)

	// Append appends the provided value to the end of the list.
	//
	// When appending a composite type, it is unspecified whether the appended
	// value aliases the source's memory in any way.
	Append(Value)

	// TODO: Should truncate accept two indexes similar to slicing?

	// Truncate truncates the list to a smaller length.
	Truncate(int)

	// NewMessage returns a newly allocated empty message assignable to a list entry.
	// It panics if the list entry type is not a message.
	NewMessage() Message
}

// Map is an unordered, associative map. Only elements within the map
// is considered populated. The entry Value type is determined by the associated
// FieldDescripto.Kind and cannot be a Map or List.
//
// Len, Has, Get, and Range are safe for concurrent use.
type Map interface {
	// Len reports the number of elements in the map.
	Len() int

	// Has reports whether an entry with the given key is in the map.
	Has(MapKey) bool

	// Get retrieves the value for an entry with the given key.
	// It returns an invalid value for non-existent entries.
	Get(MapKey) Value

	// Set stores the value for an entry with the given key.
	//
	// When setting a composite type, it is unspecified whether the set
	// value aliases the source's memory in any way.
	//
	// It panics if either the key or value are invalid.
	Set(MapKey, Value)

	// Clear clears the entry associated with they given key.
	// The operation does nothing if there is no entry associated with the key.
	Clear(MapKey)

	// Range iterates over every map entry in an undefined order,
	// calling f for each key and value encountered.
	// Range calls f Len times unless f returns false, which stops iteration.
	// While iterating, mutating operations through Set, Clear, or Mutable
	// may only be performed on the current map key.
	Range(f func(MapKey, Value) bool)

	// NewMessage returns a newly allocated empty message assignable to a map value.
	// It panics if the map value type is not a message.
	NewMessage() Message
}
