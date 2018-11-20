// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package proto

/*
 * Types and routines for supporting protocol buffer extensions.
 */

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"sync"

	"github.com/golang/protobuf/protoapi"
)

// ErrMissingExtension is the error returned by GetExtension if the named extension is not in the message.
var ErrMissingExtension = errors.New("proto: missing extension")

func extendable(p interface{}) (protoapi.ExtensionFields, error) {
	type extendableProto interface {
		Message
		ExtensionRangeArray() []ExtensionRange
	}
	if _, ok := p.(extendableProto); ok {
		v := reflect.ValueOf(p)
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			v = v.Elem()
			if v := v.FieldByName("XXX_InternalExtensions"); v.IsValid() {
				return protoapi.ExtensionFieldsOf(v.Addr().Interface()), nil
			}
			if v := v.FieldByName("XXX_extensions"); v.IsValid() {
				return protoapi.ExtensionFieldsOf(v.Addr().Interface()), nil
			}
		}
	}
	// Don't allocate a specific error containing %T:
	// this is the hot path for Clone and MarshalText.
	return nil, errNotExtendable
}

var errNotExtendable = errors.New("proto: not an extendable proto.Message")

type (
	ExtensionRange         = protoapi.ExtensionRange
	ExtensionDesc          = protoapi.ExtensionDesc
	Extension              = protoapi.ExtensionField
	XXX_InternalExtensions = protoapi.InternalExtensions
)

func isRepeatedExtension(ed *ExtensionDesc) bool {
	t := reflect.TypeOf(ed.ExtensionType)
	return t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
}

// SetRawExtension is for testing only.
func SetRawExtension(base Message, id int32, b []byte) {
	epb, err := extendable(base)
	if err != nil {
		return
	}
	epb.Set(id, Extension{Raw: b})
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
	if err != nil {
		return false
	}
	if !epb.HasInit() {
		return false
	}
	epb.Lock()
	defer epb.Unlock()
	return epb.Has(extension.Field)
}

// ClearExtension removes the given extension from pb.
func ClearExtension(pb Message, extension *ExtensionDesc) {
	epb, err := extendable(pb)
	if err != nil {
		return
	}
	// TODO: Check types, field numbers, etc.?
	epb.Clear(extension.Field)
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

	if !epb.HasInit() {
		return defaultExtensionValue(pb, extension)
	}
	epb.Lock()
	defer epb.Unlock()
	if !epb.Has(extension.Field) {
		// defaultExtensionValue returns the default value or
		// ErrMissingExtension if there is no default.
		return defaultExtensionValue(pb, extension)
	}
	e := epb.Get(extension.Field)

	if e.Value != nil {
		// Already decoded. Check the descriptor, though.
		if e.Desc != extension {
			// This shouldn't happen. If it does, it means that
			// GetExtension was called twice with two different
			// descriptors with the same field number.
			return nil, errors.New("proto: descriptor conflict")
		}
		return extensionAsLegacyType(e.Value), nil
	}

	if extension.ExtensionType == nil {
		// incomplete descriptor
		return e.Raw, nil
	}

	v, err := decodeExtension(e.Raw, extension)
	if err != nil {
		return nil, err
	}

	// Remember the decoded version and drop the encoded version.
	// That way it is safe to mutate what we return.
	e.Value = extensionAsStorageType(v)
	e.Desc = extension
	e.Raw = nil
	epb.Set(extension.Field, e)
	return extensionAsLegacyType(e.Value), nil
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

	if !epb.HasInit() {
		return nil, nil
	}
	epb.Lock()
	defer epb.Unlock()
	extensions := make([]*ExtensionDesc, 0, epb.Len())
	epb.Range(func(extid int32, e Extension) bool {
		desc := e.Desc
		if desc == nil {
			desc = registeredExtensions[extid]
			if desc == nil {
				desc = &ExtensionDesc{Field: extid}
			}
		}

		extensions = append(extensions, desc)
		return true
	})
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

	epb.Set(extension.Field, Extension{Desc: extension, Value: extensionAsStorageType(value)})
	return nil
}

// ClearAllExtensions clears all extensions from pb.
func ClearAllExtensions(pb Message) {
	epb, err := extendable(pb)
	if err != nil {
		return
	}
	epb.Range(func(k int32, _ Extension) bool {
		epb.Clear(k)
		return true
	})
}

// A global registry of extensions.
// The generated code will register the generated descriptors by calling RegisterExtension.

var extensionMaps = make(map[reflect.Type]map[int32]*ExtensionDesc)

// RegisterExtension is called from the generated code.
func RegisterExtension(desc *ExtensionDesc) {
	st := reflect.TypeOf(desc.ExtendedType).Elem()
	m := extensionMaps[st]
	if m == nil {
		m = make(map[int32]*ExtensionDesc)
		extensionMaps[st] = m
	}
	if _, ok := m[desc.Field]; ok {
		panic("proto: duplicate extension registered: " + st.String() + " " + strconv.Itoa(int(desc.Field)))
	}
	m[desc.Field] = desc
}

// RegisteredExtensions returns a map of the registered extensions of a
// protocol buffer struct, indexed by the extension number.
// The argument pb should be a nil pointer to the struct type.
func RegisteredExtensions(pb Message) map[int32]*ExtensionDesc {
	return extensionMaps[reflect.TypeOf(pb).Elem()]
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
