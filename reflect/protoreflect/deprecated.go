// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoreflect

// TODO: Remove this file.

type (
	deprecatedEnum interface {
		// Deprecated: Use Descriptor instead.
		Type() EnumType
	}

	deprecatedMessage interface {
		// Deprecated: Use Descriptor instead.
		Type() MessageType
		// Deprecated: Use methods on Message directly.
		KnownFields() KnownFields
		// Deprecated: Use methods on Message directly.
		UnknownFields() UnknownFields
	}
)

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
//
// Deprecated: Use direct methods on Message instead.
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

	// WhichOneof reports which field within the named oneof is populated.
	// It returns 0 if the oneof does not exist or no fields are populated.
	WhichOneof(Name) FieldNumber

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
//
// Deprecated: Use direct methods on Message instead.
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

// ExtensionFieldTypes are the extension field types that this message instance
// has been extended with.
//
// Len, Get, and Range are safe for concurrent use.
//
// Deprecated: Use direct methods on Message instead.
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
