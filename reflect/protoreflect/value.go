// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoreflect

import "google.golang.org/proto/internal/encoding/wire"

// TODO: Document the behavior of each Range operation when a mutation
// occurs while ranging. Also document the order.

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

	// ProtoMutable is a marker method to implement the Mutable interface.
	ProtoMutable()
}

// KnownFields provides accessor and mutator methods for known fields.
//
// Each field Value can either be a scalar, Message, Vector, or Map.
// The field is a Vector or Map if FieldDescriptor.Cardinality is Repeated and
// a Map if and only if FieldDescriptor.IsMap is true. The scalar type or
// underlying repeated element type is determined by the FieldDescriptor.Kind.
// See Value for a list of Go types associated with each Kind.
//
// Field extensions are handled as known fields once the extension type has been
// registered with KnownFields.ExtensionTypes.
//
// List, Len, Has, Get, Range, and ExtensionTypes are safe for concurrent use.
type KnownFields interface {
	// List returns a new, unordered list of all populated fields.
	List() []FieldNumber

	// Len reports the number of fields that are populated.
	//
	// Invariant: f.Len() == len(f.List())
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
	// It returns the default value for unpopulated fields.
	// It returns an invalid value for unknown fields.
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
	//
	// It panics if the field number does not correspond with a known field
	// in MessageDescriptor.Fields or an extension field in ExtensionTypes.
	Clear(FieldNumber)

	// Mutable returns a reference value for a field with a given field number.
	// If the field is unset, Mutable implicitly initializes the field with
	// a zero value instance of the Go type for that field.
	//
	// The returned Mutable reference is never nil, and is only valid until the
	// next Set or Mutable call.
	//
	// It panics if the field number does not correspond with a known field
	// in MessageDescriptor.Fields or an extension field in ExtensionTypes.
	Mutable(FieldNumber) Mutable

	// Range calls f sequentially for each known field that is populated.
	// If f returns false, range stops the iteration.
	Range(f func(FieldNumber, Value) bool)

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
// List, Len, Get, and Range are safe for concurrent use.
type UnknownFields interface {
	// List returns a new, unordered list of all fields that are set.
	List() []FieldNumber

	// Len reports the number of fields that are set.
	//
	// Invariant: f.Len() == len(f.List())
	Len() int

	// Get retrieves the raw bytes of fields with the given field number.
	// It returns an empty RawFields if there are no set fields.
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

	// Range calls f sequentially for each unknown field that is populated.
	// If f returns false, range stops the iteration.
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
// (e.g., raw[:len(raw)] is immutable, but raw[len(raw):cap(raw)] is mutable).
// Thus, appending to RawFields (with valid wire data) is permitted.
type RawFields []byte

// IsValid reports whether RawFields is syntactically correct wire format.
func (b RawFields) IsValid() bool {
	for len(b) > 0 {
		_, _, n := wire.ConsumeField(b)
		if n < 0 {
			return false
		}
		b = b[n:]
	}
	return true
}

// ExtensionFieldTypes are the extension field types that this message instance
// has been extended with.
//
// List, Len, Get, and Range are safe for concurrent use.
type ExtensionFieldTypes interface {
	// List returns a new, unordered list of known extension field types.
	List() []ExtensionType

	// Len reports the number of field extensions.
	//
	// Invariant: f.Len() == len(f.List())
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
	Remove(ExtensionType)

	// ByNumber looks up an extension by field number.
	// It returns nil if not found.
	ByNumber(FieldNumber) ExtensionType

	// ByName looks up an extension field by full name.
	// It returns nil if not found.
	ByName(FullName) ExtensionType

	// Range calls f sequentially for each registered extension field type.
	// If f returns false, range stops the iteration.
	Range(f func(ExtensionType) bool)
}

// Vector is an ordered list. Every element is considered populated
// (i.e., Get never provides and Set never accepts invalid Values).
// The element Value type is determined by the associated FieldDescriptor.Kind
// and cannot be a Map or Vector.
//
// Len and Get are safe for concurrent use.
type Vector interface {
	// Len reports the number of entries in the Vector.
	// Get, Set, Mutable, and Truncate panic with out of bound indexes.
	Len() int

	// Get retrieves the value at the given index.
	Get(int) Value

	// Set stores a value for the given index.
	//
	// When setting a composite type, it is unspecified whether the set
	// value aliases the source's memory in any way.
	Set(int, Value)

	// Append appends the provided value to the end of the vector.
	//
	// When appending a composite type, it is unspecified whether the appended
	// value aliases the source's memory in any way.
	Append(Value)

	// Mutable returns a Mutable reference for the element with a given index.
	//
	// The returned reference is never nil, and is only valid until the
	// next Set, Mutable, Append, MutableAppend, or Truncate call.
	Mutable(int) Mutable

	// MutableAppend appends a new element and returns a mutable reference.
	//
	// The returned reference is never nil, and is only valid until the
	// next Set, Mutable, Append, MutableAppend, or Truncate call.
	MutableAppend() Mutable

	// TODO: Should truncate accept two indexes similar to slicing?M

	// Truncate truncates the vector to a smaller length.
	Truncate(int)

	// ProtoMutable is a marker method to implement the Mutable interface.
	ProtoMutable()
}

// Map is an unordered, associative map. Only elements within the map
// is considered populated. The entry Value type is determined by the associated
// FieldDescripto.Kind and cannot be a Map or Vector.
//
// List, Len, Has, Get, and Range are safe for concurrent use.
type Map interface {
	// List returns an unordered list of keys for all entries in the map.
	List() []MapKey

	// Len reports the number of elements in the map.
	//
	// Invariant: f.Len() == len(f.List())
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
	Clear(MapKey)

	// Mutable returns a Mutable reference for the element with a given key,
	// allocating a new entry if necessary.
	//
	// The returned Mutable reference is never nil, and is only valid until the
	// next Set or Mutable call.
	Mutable(MapKey) Mutable

	// Range calls f sequentially for each key and value present in the map.
	// The Value provided is guaranteed to be valid.
	// If f returns false, range stops the iteration.
	Range(f func(MapKey, Value) bool)

	// ProtoMutable is a marker method to implement the Mutable interface.
	ProtoMutable()
}

// Mutable is a mutable reference, where mutate operations also affect
// the backing store. Possible Mutable types: Vector, Map, or Message.
type Mutable interface {
	ProtoMutable()
}
