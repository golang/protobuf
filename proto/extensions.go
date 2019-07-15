// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

/*
 * Types and routines for supporting protocol buffer extensions.
 */

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/golang/protobuf/internal/wire"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoiface"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// ErrMissingExtension is the error returned by GetExtension if the named extension is not in the message.
var ErrMissingExtension = errors.New("proto: missing extension")

func extensionFieldsOf(p interface{}) *extensionMap {
	if p, ok := p.(*map[int32]Extension); ok {
		return (*extensionMap)(p)
	}
	panic(fmt.Sprintf("invalid extension fields type: %T", p))
}

type extensionMap map[int32]Extension

func (m extensionMap) Len() int {
	return len(m)
}
func (m extensionMap) Has(n protoreflect.FieldNumber) bool {
	_, ok := m[int32(n)]
	return ok
}
func (m extensionMap) Get(n protoreflect.FieldNumber) Extension {
	return m[int32(n)]
}
func (m *extensionMap) Set(n protoreflect.FieldNumber, x Extension) {
	if *m == nil {
		*m = make(map[int32]Extension)
	}
	(*m)[int32(n)] = x
}
func (m *extensionMap) Clear(n protoreflect.FieldNumber) {
	delete(*m, int32(n))
}
func (m extensionMap) Range(f func(protoreflect.FieldNumber, Extension) bool) {
	for n, x := range m {
		if !f(protoreflect.FieldNumber(n), x) {
			return
		}
	}
}

func extendable(p interface{}) (*extensionMap, error) {
	type extendableProto interface {
		Message
		ExtensionRangeArray() []ExtensionRange
	}
	if _, ok := p.(extendableProto); ok {
		v := reflect.ValueOf(p)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
			if vf := extensionFieldsValue(v); vf.IsValid() {
				return extensionFieldsOf(vf.Addr().Interface()), nil
			}
		}
	}
	// Don't allocate a specific error containing %T:
	// this is the hot path for Clone and MarshalText.
	return nil, errNotExtendable
}

var errNotExtendable = errors.New("proto: not an extendable proto.Message")

type (
	ExtensionRange         = protoiface.ExtensionRangeV1
	ExtensionDesc          = protoiface.ExtensionDescV1
	Extension              = protoimpl.ExtensionFieldV1
	XXX_InternalExtensions = protoimpl.ExtensionFields
)

func isRepeatedExtension(ed *ExtensionDesc) bool {
	t := reflect.TypeOf(ed.ExtensionType)
	return t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
}

// SetRawExtension is for testing only.
func SetRawExtension(base Message, id int32, b []byte) {
	v := reflect.ValueOf(base)
	if !v.IsValid() || v.Kind() != reflect.Ptr || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return
	}
	v = unknownFieldsValue(v.Elem())
	if !v.IsValid() {
		return
	}

	// Verify that the raw field is valid.
	for b0 := b; len(b0) > 0; {
		fieldNum, _, n := wire.ConsumeField(b0)
		if int32(fieldNum) != id {
			panic(fmt.Sprintf("mismatching field number: got %d, want %d", fieldNum, id))
		}
		b0 = b0[n:]
	}

	fnum := protoreflect.FieldNumber(id)
	v.SetBytes(append(removeRawFields(v.Bytes(), fnum), b...))
}

func removeRawFields(b []byte, fnum protoreflect.FieldNumber) []byte {
	out := b[:0]
	for len(b) > 0 {
		got, _, n := wire.ConsumeField(b)
		if got != fnum {
			out = append(out, b[:n]...)
		}
		b = b[n:]
	}
	return out
}

// isExtensionField returns true iff the given field number is in an extension range.
func isExtensionField(pb Message, field int32) bool {
	m, ok := pb.(interface{ ExtensionRangeArray() []ExtensionRange })
	if ok {
		for _, er := range m.ExtensionRangeArray() {
			if er.Start <= field && field <= er.End {
				return true
			}
		}
	}
	return false
}

// checkExtensionTypeAndRanges checks that the given extension is valid for pb.
func checkExtensionTypeAndRanges(pb Message, extension *ExtensionDesc) error {
	// Check the extended type.
	if extension.ExtendedType != nil {
		if a, b := reflect.TypeOf(pb), reflect.TypeOf(extension.ExtendedType); a != b {
			return fmt.Errorf("proto: bad extended type; %v does not extend %v", b, a)
		}
	}
	// Check the range.
	if !isExtensionField(pb, extension.Field) {
		return errors.New("proto: bad extension number; not in declared ranges")
	}
	return nil
}

// extPropKey is sufficient to uniquely identify an extension.
type extPropKey struct {
	base  reflect.Type
	field int32
}

var extProp = struct {
	sync.RWMutex
	m map[extPropKey]*Properties
}{
	m: make(map[extPropKey]*Properties),
}

func extensionProperties(pb Message, ed *ExtensionDesc) *Properties {
	key := extPropKey{base: reflect.TypeOf(pb), field: ed.Field}

	extProp.RLock()
	if prop, ok := extProp.m[key]; ok {
		extProp.RUnlock()
		return prop
	}
	extProp.RUnlock()

	extProp.Lock()
	defer extProp.Unlock()
	// Check again.
	if prop, ok := extProp.m[key]; ok {
		return prop
	}

	prop := new(Properties)
	prop.Init(reflect.TypeOf(ed.ExtensionType), "unknown_name", ed.Tag, nil)
	extProp.m[key] = prop
	return prop
}

// HasExtension returns whether the given extension is present in pb.
func HasExtension(pb Message, extension *ExtensionDesc) bool {
	// TODO: Check types, field numbers, etc.?
	epb, err := extendable(pb)
	if err != nil || epb == nil {
		return false
	}
	if epb.Has(protoreflect.FieldNumber(extension.Field)) {
		return true
	}

	// Check whether this field exists in raw form.
	unrecognized := unknownFieldsValue(reflect.ValueOf(pb).Elem())
	fnum := protoreflect.FieldNumber(extension.Field)
	for b := unrecognized.Bytes(); len(b) > 0; {
		got, _, n := wire.ConsumeField(b)
		if got == fnum {
			return true
		}
		b = b[n:]
	}
	return false
}

// ClearExtension removes the given extension from pb.
func ClearExtension(pb Message, extension *ExtensionDesc) {
	epb, err := extendable(pb)
	if err != nil {
		return
	}
	// TODO: Check types, field numbers, etc.?
	epb.Clear(protoreflect.FieldNumber(extension.Field))
}

// GetExtension retrieves a proto2 extended field from pb.
//
// If the descriptor is type complete (i.e., ExtensionDesc.ExtensionType is non-nil),
// then GetExtension parses the encoded field and returns a Go value of the specified type.
// If the field is not present, then the default value is returned (if one is specified),
// otherwise ErrMissingExtension is reported.
//
// If the descriptor is not type complete (i.e., ExtensionDesc.ExtensionType is nil),
// then GetExtension returns the raw encoded bytes of the field extension.
func GetExtension(pb Message, extension *ExtensionDesc) (interface{}, error) {
	epb, err := extendable(pb)
	if err != nil {
		return nil, err
	}

	// can only check type if this is a complete descriptor
	if err := checkExtensionTypeAndRanges(pb, extension); err != nil {
		return nil, err
	}

	unrecognized := unknownFieldsValue(reflect.ValueOf(pb).Elem())
	var out []byte
	fnum := protoreflect.FieldNumber(extension.Field)
	for b := unrecognized.Bytes(); len(b) > 0; {
		got, _, n := wire.ConsumeField(b)
		if got == fnum {
			out = append(out, b[:n]...)
		}
		b = b[n:]
	}

	if !epb.Has(protoreflect.FieldNumber(extension.Field)) && len(out) == 0 {
		// defaultExtensionValue returns the default value or
		// ErrMissingExtension if there is no default.
		return defaultExtensionValue(pb, extension)
	}

	e := epb.Get(protoreflect.FieldNumber(extension.Field))
	if e.HasValue() {
		// Already decoded. Check the descriptor, though.
		if protoimpl.X.ExtensionDescFromType(e.GetType()) != extension {
			// This shouldn't happen. If it does, it means that
			// GetExtension was called twice with two different
			// descriptors with the same field number.
			return nil, errors.New("proto: descriptor conflict")
		}
		return extensionAsLegacyType(e.GetValue()), nil
	}

	// Descriptor without type information.
	if extension.ExtensionType == nil {
		return out, nil
	}

	// TODO: Remove this logic for automatically unmarshaling the unknown fields.
	v, err := decodeExtension(out, extension)
	if err != nil {
		return nil, err
	}

	// Remember the decoded version and drop the encoded version.
	// That way it is safe to mutate what we return.
	e.SetType(protoimpl.X.ExtensionTypeFromDesc(extension))
	e.SetEagerValue(extensionAsStorageType(v))
	unrecognized.SetBytes(removeRawFields(unrecognized.Bytes(), fnum))
	epb.Set(protoreflect.FieldNumber(extension.Field), e)
	return extensionAsLegacyType(e.GetValue()), nil
}

// defaultExtensionValue returns the default value for extension.
// If no default for an extension is defined ErrMissingExtension is returned.
func defaultExtensionValue(pb Message, extension *ExtensionDesc) (interface{}, error) {
	if extension.ExtensionType == nil {
		// incomplete descriptor, so no default
		return nil, ErrMissingExtension
	}

	t := reflect.TypeOf(extension.ExtensionType)
	props := extensionProperties(pb, extension)

	sf, _, err := fieldDefault(t, props)
	if err != nil {
		return nil, err
	}

	if sf == nil || sf.value == nil {
		// There is no default value.
		return nil, ErrMissingExtension
	}

	if t.Kind() != reflect.Ptr {
		// We do not need to return a Ptr, we can directly return sf.value.
		return sf.value, nil
	}

	// We need to return an interface{} that is a pointer to sf.value.
	value := reflect.New(t).Elem()
	value.Set(reflect.New(value.Type().Elem()))
	if sf.kind == reflect.Int32 {
		// We may have an int32 or an enum, but the underlying data is int32.
		// Since we can't set an int32 into a non int32 reflect.Value directly
		// set it as a int32.
		value.Elem().SetInt(int64(sf.value.(int32)))
	} else {
		value.Elem().Set(reflect.ValueOf(sf.value))
	}
	return value.Interface(), nil
}

// decodeExtension decodes an extension encoded in b.
func decodeExtension(b []byte, extension *ExtensionDesc) (interface{}, error) {
	t := reflect.TypeOf(extension.ExtensionType)
	unmarshal := typeUnmarshaler(t, extension.Tag)

	// t is a pointer to a struct, pointer to basic type or a slice.
	// Allocate space to store the pointer/slice.
	value := reflect.New(t).Elem()

	var err error
	for {
		x, n := decodeVarint(b)
		if n == 0 {
			return nil, io.ErrUnexpectedEOF
		}
		b = b[n:]
		wire := int(x) & 7

		b, err = unmarshal(b, valToPointer(value.Addr()), wire)
		if err != nil {
			return nil, err
		}

		if len(b) == 0 {
			break
		}
	}
	return value.Interface(), nil
}

// GetExtensions returns a slice of the extensions present in pb that are also listed in es.
// The returned slice has the same length as es; missing extensions will appear as nil elements.
func GetExtensions(pb Message, es []*ExtensionDesc) (extensions []interface{}, err error) {
	_, err = extendable(pb)
	if err != nil {
		return nil, err
	}
	extensions = make([]interface{}, len(es))
	for i, e := range es {
		extensions[i], err = GetExtension(pb, e)
		if err == ErrMissingExtension {
			err = nil
		}
		if err != nil {
			return
		}
	}
	return
}

// ExtensionDescs returns a new slice containing pb's extension descriptors, in undefined order.
// For non-registered extensions, ExtensionDescs returns an incomplete descriptor containing
// just the Field field, which defines the extension's field number.
func ExtensionDescs(pb Message) ([]*ExtensionDesc, error) {
	epb, err := extendable(pb)
	if err != nil {
		return nil, err
	}
	registeredExtensions := RegisteredExtensions(pb)

	if epb == nil {
		return nil, nil
	}
	extensions := make([]*ExtensionDesc, 0, epb.Len())
	epb.Range(func(extid protoreflect.FieldNumber, e Extension) bool {
		desc := protoimpl.X.ExtensionDescFromType(e.GetType())
		if desc == nil {
			desc = registeredExtensions[int32(extid)]
			if desc == nil {
				desc = &ExtensionDesc{Field: int32(extid)}
			}
		}

		extensions = append(extensions, desc)
		return true
	})

	unrecognized := unknownFieldsValue(reflect.ValueOf(pb).Elem())
	if b := unrecognized.Bytes(); len(b) > 0 {
		fieldNums := make(map[int32]bool)
		for len(b) > 0 {
			fnum, _, n := wire.ConsumeField(b)
			if isExtensionField(pb, int32(fnum)) {
				fieldNums[int32(fnum)] = true
			}
			b = b[n:]
		}

		for id := range fieldNums {
			desc := registeredExtensions[id]
			if desc == nil {
				desc = &ExtensionDesc{Field: id}
			}
			extensions = append(extensions, desc)
		}
	}

	return extensions, nil
}

// SetExtension sets the specified extension of pb to the specified value.
func SetExtension(pb Message, extension *ExtensionDesc, value interface{}) error {
	epb, err := extendable(pb)
	if err != nil {
		return err
	}
	if err := checkExtensionTypeAndRanges(pb, extension); err != nil {
		return err
	}
	typ := reflect.TypeOf(extension.ExtensionType)
	if typ != reflect.TypeOf(value) {
		return fmt.Errorf("proto: bad extension value type. got: %T, want: %T", value, extension.ExtensionType)
	}
	// nil extension values need to be caught early, because the
	// encoder can't distinguish an ErrNil due to a nil extension
	// from an ErrNil due to a missing field. Extensions are
	// always optional, so the encoder would just swallow the error
	// and drop all the extensions from the encoded message.
	if reflect.ValueOf(value).IsNil() {
		return fmt.Errorf("proto: SetExtension called with nil value of type %T", value)
	}

	var x Extension
	x.SetType(protoimpl.X.ExtensionTypeFromDesc(extension))
	x.SetEagerValue(extensionAsStorageType(value))
	epb.Set(protoreflect.FieldNumber(extension.Field), x)
	return nil
}

// ClearAllExtensions clears all extensions from pb.
func ClearAllExtensions(pb Message) {
	epb, err := extendable(pb)
	if err != nil {
		return
	}
	epb.Range(func(k protoreflect.FieldNumber, _ Extension) bool {
		epb.Clear(k)
		return true
	})
}

// extensionAsLegacyType converts an value in the storage type as the API type.
// See Extension.Value.
func extensionAsLegacyType(v interface{}) interface{} {
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		// Represent primitive types as a pointer to the value.
		rv2 := reflect.New(rv.Type())
		rv2.Elem().Set(rv)
		v = rv2.Interface()
	case reflect.Ptr:
		// Represent slice types as the value itself.
		switch rv.Type().Elem().Kind() {
		case reflect.Slice:
			if rv.IsNil() {
				v = reflect.Zero(rv.Type().Elem()).Interface()
			} else {
				v = rv.Elem().Interface()
			}
		}
	}
	return v
}

// extensionAsStorageType converts an value in the API type as the storage type.
// See Extension.Value.
func extensionAsStorageType(v interface{}) interface{} {
	switch rv := reflect.ValueOf(v); rv.Kind() {
	case reflect.Ptr:
		// Represent slice types as the value itself.
		switch rv.Type().Elem().Kind() {
		case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
			if rv.IsNil() {
				v = reflect.Zero(rv.Type().Elem()).Interface()
			} else {
				v = rv.Elem().Interface()
			}
		}
	case reflect.Slice:
		// Represent slice types as a pointer to the value.
		if rv.Type().Elem().Kind() != reflect.Uint8 {
			rv2 := reflect.New(rv.Type())
			rv2.Elem().Set(rv)
			v = rv2.Interface()
		}
	}
	return v
}
