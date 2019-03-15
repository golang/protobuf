// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proto marshals and unmarshals protocol buffer messages as the
// wire-format and text-format. For more information, see:
//	https://developers.google.com/protocol-buffers/docs/gotutorial
package proto

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strconv"
	"sync"

	// Add a bogus dependency on the v2 API to ensure the Go toolchain does not
	// remove our dependency from the go.mod file.
	_ "github.com/golang/protobuf/v2/reflect/protoreflect"

	"github.com/golang/protobuf/protoapi"
)

// RequiredNotSetError is an error type returned by either Marshal or Unmarshal.
// Marshal reports this when a required field is not initialized.
// Unmarshal reports this when a required field is missing from the wire data.
type RequiredNotSetError struct{ field string }

func (e *RequiredNotSetError) Error() string {
	if e.field == "" {
		return fmt.Sprintf("proto: required field not set")
	}
	return fmt.Sprintf("proto: required field %q not set", e.field)
}
func (e *RequiredNotSetError) RequiredNotSet() bool {
	return true
}

type invalidUTF8Error struct{ field string }

func (e *invalidUTF8Error) Error() string {
	if e.field == "" {
		return "proto: invalid UTF-8 detected"
	}
	return fmt.Sprintf("proto: field %q contains invalid UTF-8", e.field)
}
func (e *invalidUTF8Error) InvalidUTF8() bool {
	return true
}

// errInvalidUTF8 is a sentinel error to identify fields with invalid UTF-8.
// This error should not be exposed to the external API as such errors should
// be recreated with the field information.
var errInvalidUTF8 = &invalidUTF8Error{}

// isNonFatal reports whether the error is either a RequiredNotSet error
// or a InvalidUTF8 error.
func isNonFatal(err error) bool {
	if re, ok := err.(interface{ RequiredNotSet() bool }); ok && re.RequiredNotSet() {
		return true
	}
	if re, ok := err.(interface{ InvalidUTF8() bool }); ok && re.InvalidUTF8() {
		return true
	}
	return false
}

type nonFatal struct{ E error }

// Merge merges err into nf and reports whether it was successful.
// Otherwise it returns false for any fatal non-nil errors.
func (nf *nonFatal) Merge(err error) (ok bool) {
	if err == nil {
		return true // not an error
	}
	if !isNonFatal(err) {
		return false // fatal error
	}
	if nf.E == nil {
		nf.E = err // store first instance of non-fatal error
	}
	return true
}

// Message is implemented by generated protocol buffer messages.
type Message = protoapi.Message

// A Buffer is a buffer manager for marshaling and unmarshaling
// protocol buffers.  It may be reused between invocations to
// reduce memory usage.  It is not necessary to use a Buffer;
// the global functions Marshal and Unmarshal create a
// temporary Buffer and are fine for most applications.
type Buffer struct {
	buf   []byte // encode/decode byte stream
	index int    // read point

	deterministic bool
}

// NewBuffer allocates a new Buffer and initializes its internal data to
// the contents of the argument slice.
func NewBuffer(e []byte) *Buffer {
	return &Buffer{buf: e}
}

// Reset resets the Buffer, ready for marshaling a new protocol buffer.
func (p *Buffer) Reset() {
	p.buf = p.buf[0:0] // for reading/writing
	p.index = 0        // for reading
}

// SetBuf replaces the internal buffer with the slice,
// ready for unmarshaling the contents of the slice.
func (p *Buffer) SetBuf(s []byte) {
	p.buf = s
	p.index = 0
}

// Bytes returns the contents of the Buffer.
func (p *Buffer) Bytes() []byte { return p.buf }

// SetDeterministic sets whether to use deterministic serialization.
//
// Deterministic serialization guarantees that for a given binary, equal
// messages will always be serialized to the same bytes. This implies:
//
//   - Repeated serialization of a message will return the same bytes.
//   - Different processes of the same binary (which may be executing on
//     different machines) will serialize equal messages to the same bytes.
//
// Note that the deterministic serialization is NOT canonical across
// languages. It is not guaranteed to remain stable over time. It is unstable
// across different builds with schema changes due to unknown fields.
// Users who need canonical serialization (e.g., persistent storage in a
// canonical form, fingerprinting, etc.) should define their own
// canonicalization specification and implement their own serializer rather
// than relying on this API.
//
// If deterministic serialization is requested, map entries will be sorted
// by keys in lexographical order. This is an implementation detail and
// subject to change.
func (p *Buffer) SetDeterministic(deterministic bool) {
	p.deterministic = deterministic
}

/*
 * Helper routines for simplifying the creation of optional fields of basic type.
 */

// Bool is a helper routine that allocates a new bool value
// to store v and returns a pointer to it.
func Bool(v bool) *bool {
	return &v
}

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

// Int is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it, but unlike Int32
// its argument value is an int.
func Int(v int) *int32 {
	p := new(int32)
	*p = int32(v)
	return p
}

// Int64 is a helper routine that allocates a new int64 value
// to store v and returns a pointer to it.
func Int64(v int64) *int64 {
	return &v
}

// Float32 is a helper routine that allocates a new float32 value
// to store v and returns a pointer to it.
func Float32(v float32) *float32 {
	return &v
}

// Float64 is a helper routine that allocates a new float64 value
// to store v and returns a pointer to it.
func Float64(v float64) *float64 {
	return &v
}

// Uint32 is a helper routine that allocates a new uint32 value
// to store v and returns a pointer to it.
func Uint32(v uint32) *uint32 {
	return &v
}

// Uint64 is a helper routine that allocates a new uint64 value
// to store v and returns a pointer to it.
func Uint64(v uint64) *uint64 {
	return &v
}

// String is a helper routine that allocates a new string value
// to store v and returns a pointer to it.
func String(v string) *string {
	return &v
}

// DebugPrint dumps the encoded data in b in a debugging format with a header
// including the string s. Used in testing but made available for general debugging.
func (p *Buffer) DebugPrint(s string, b []byte) {
	var u uint64

	obuf := p.buf
	index := p.index
	p.buf = b
	p.index = 0
	depth := 0

	fmt.Printf("\n--- %s ---\n", s)

out:
	for {
		for i := 0; i < depth; i++ {
			fmt.Print("  ")
		}

		index := p.index
		if index == len(p.buf) {
			break
		}

		op, err := p.DecodeVarint()
		if err != nil {
			fmt.Printf("%3d: fetching op err %v\n", index, err)
			break out
		}
		tag := op >> 3
		wire := op & 7

		switch wire {
		default:
			fmt.Printf("%3d: t=%3d unknown wire=%d\n",
				index, tag, wire)
			break out

		case WireBytes:
			var r []byte

			r, err = p.DecodeRawBytes(false)
			if err != nil {
				break out
			}
			fmt.Printf("%3d: t=%3d bytes [%d]", index, tag, len(r))
			if len(r) <= 6 {
				for i := 0; i < len(r); i++ {
					fmt.Printf(" %.2x", r[i])
				}
			} else {
				for i := 0; i < 3; i++ {
					fmt.Printf(" %.2x", r[i])
				}
				fmt.Printf(" ..")
				for i := len(r) - 3; i < len(r); i++ {
					fmt.Printf(" %.2x", r[i])
				}
			}
			fmt.Printf("\n")

		case WireFixed32:
			u, err = p.DecodeFixed32()
			if err != nil {
				fmt.Printf("%3d: t=%3d fix32 err %v\n", index, tag, err)
				break out
			}
			fmt.Printf("%3d: t=%3d fix32 %d\n", index, tag, u)

		case WireFixed64:
			u, err = p.DecodeFixed64()
			if err != nil {
				fmt.Printf("%3d: t=%3d fix64 err %v\n", index, tag, err)
				break out
			}
			fmt.Printf("%3d: t=%3d fix64 %d\n", index, tag, u)

		case WireVarint:
			u, err = p.DecodeVarint()
			if err != nil {
				fmt.Printf("%3d: t=%3d varint err %v\n", index, tag, err)
				break out
			}
			fmt.Printf("%3d: t=%3d varint %d\n", index, tag, u)

		case WireStartGroup:
			fmt.Printf("%3d: t=%3d start\n", index, tag)
			depth++

		case WireEndGroup:
			depth--
			fmt.Printf("%3d: t=%3d end\n", index, tag)
		}
	}

	if depth != 0 {
		fmt.Printf("%3d: start-end not balanced %d\n", p.index, depth)
	}
	fmt.Printf("\n")

	p.buf = obuf
	p.index = index
}

var setDefaultsAlt func(Message) // populated by hooks.go

// SetDefaults sets unset protocol buffer fields to their default values.
// It only modifies fields that are both unset and have defined defaults.
// It recursively sets default values in any non-nil sub-messages.
func SetDefaults(pb Message) {
	if setDefaultsAlt != nil {
		setDefaultsAlt(pb)
		return
	}
	setDefaults(reflect.ValueOf(pb), true, false)
}

// v is a pointer to a struct.
func setDefaults(v reflect.Value, recur, zeros bool) {
	v = v.Elem()

	defaultMu.RLock()
	dm, ok := defaults[v.Type()]
	defaultMu.RUnlock()
	if !ok {
		dm = buildDefaultMessage(v.Type())
		defaultMu.Lock()
		defaults[v.Type()] = dm
		defaultMu.Unlock()
	}

	for _, sf := range dm.scalars {
		f := v.Field(sf.index)
		if !f.IsNil() {
			// field already set
			continue
		}
		dv := sf.value
		if dv == nil && !zeros {
			// no explicit default, and don't want to set zeros
			continue
		}
		fptr := f.Addr().Interface() // **T
		// TODO: Consider batching the allocations we do here.
		switch sf.kind {
		case reflect.Bool:
			b := new(bool)
			if dv != nil {
				*b = dv.(bool)
			}
			*(fptr.(**bool)) = b
		case reflect.Float32:
			f := new(float32)
			if dv != nil {
				*f = dv.(float32)
			}
			*(fptr.(**float32)) = f
		case reflect.Float64:
			f := new(float64)
			if dv != nil {
				*f = dv.(float64)
			}
			*(fptr.(**float64)) = f
		case reflect.Int32:
			// might be an enum
			if ft := f.Type(); ft != int32PtrType {
				// enum
				f.Set(reflect.New(ft.Elem()))
				if dv != nil {
					f.Elem().SetInt(int64(dv.(int32)))
				}
			} else {
				// int32 field
				i := new(int32)
				if dv != nil {
					*i = dv.(int32)
				}
				*(fptr.(**int32)) = i
			}
		case reflect.Int64:
			i := new(int64)
			if dv != nil {
				*i = dv.(int64)
			}
			*(fptr.(**int64)) = i
		case reflect.String:
			s := new(string)
			if dv != nil {
				*s = dv.(string)
			}
			*(fptr.(**string)) = s
		case reflect.Uint8:
			// exceptional case: []byte
			var b []byte
			if dv != nil {
				db := dv.([]byte)
				b = make([]byte, len(db))
				copy(b, db)
			} else {
				b = []byte{}
			}
			*(fptr.(*[]byte)) = b
		case reflect.Uint32:
			u := new(uint32)
			if dv != nil {
				*u = dv.(uint32)
			}
			*(fptr.(**uint32)) = u
		case reflect.Uint64:
			u := new(uint64)
			if dv != nil {
				*u = dv.(uint64)
			}
			*(fptr.(**uint64)) = u
		default:
			log.Printf("proto: can't set default for field %v (sf.kind=%v)", f, sf.kind)
		}
	}

	for _, ni := range dm.nested {
		f := v.Field(ni)
		// f is *T or []*T or map[T]*T
		switch f.Kind() {
		case reflect.Ptr:
			if f.IsNil() {
				continue
			}
			setDefaults(f, recur, zeros)

		case reflect.Slice:
			for i := 0; i < f.Len(); i++ {
				e := f.Index(i)
				if e.IsNil() {
					continue
				}
				setDefaults(e, recur, zeros)
			}

		case reflect.Map:
			for _, k := range f.MapKeys() {
				e := f.MapIndex(k)
				if e.IsNil() {
					continue
				}
				setDefaults(e, recur, zeros)
			}
		}
	}
}

var (
	// defaults maps a protocol buffer struct type to a slice of the fields,
	// with its scalar fields set to their proto-declared non-zero default values.
	defaultMu sync.RWMutex
	defaults  = make(map[reflect.Type]defaultMessage)

	int32PtrType = reflect.TypeOf((*int32)(nil))
)

// defaultMessage represents information about the default values of a message.
type defaultMessage struct {
	scalars []scalarField
	nested  []int // struct field index of nested messages
}

type scalarField struct {
	index int          // struct field index
	kind  reflect.Kind // element type (the T in *T or []T)
	value interface{}  // the proto-declared default value, or nil
}

// t is a struct type.
func buildDefaultMessage(t reflect.Type) (dm defaultMessage) {
	sprop := GetProperties(t)
	for _, prop := range sprop.Prop {
		fi, ok := sprop.decoderTags.get(prop.Tag)
		if !ok {
			// XXX_unrecognized
			continue
		}
		ft := t.Field(fi).Type

		sf, nested, err := fieldDefault(ft, prop)
		switch {
		case err != nil:
			log.Print(err)
		case nested:
			dm.nested = append(dm.nested, fi)
		case sf != nil:
			sf.index = fi
			dm.scalars = append(dm.scalars, *sf)
		}
	}

	return dm
}

// fieldDefault returns the scalarField for field type ft.
// sf will be nil if the field can not have a default.
// nestedMessage will be true if this is a nested message.
// Note that sf.index is not set on return.
func fieldDefault(ft reflect.Type, prop *Properties) (sf *scalarField, nestedMessage bool, err error) {
	var canHaveDefault bool
	switch ft.Kind() {
	case reflect.Ptr:
		if ft.Elem().Kind() == reflect.Struct {
			nestedMessage = true
		} else {
			canHaveDefault = true // proto2 scalar field
		}

	case reflect.Slice:
		switch ft.Elem().Kind() {
		case reflect.Ptr:
			nestedMessage = true // repeated message
		case reflect.Uint8:
			canHaveDefault = true // bytes field
		}

	case reflect.Map:
		if ft.Elem().Kind() == reflect.Ptr {
			nestedMessage = true // map with message values
		}
	}

	if !canHaveDefault {
		if nestedMessage {
			return nil, true, nil
		}
		return nil, false, nil
	}

	// We now know that ft is a pointer or slice.
	sf = &scalarField{kind: ft.Elem().Kind()}

	// scalar fields without defaults
	if !prop.HasDefault {
		return sf, false, nil
	}

	// a scalar field: either *T or []byte
	switch ft.Elem().Kind() {
	case reflect.Bool:
		x, err := strconv.ParseBool(prop.Default)
		if err != nil {
			return nil, false, fmt.Errorf("proto: bad default bool %q: %v", prop.Default, err)
		}
		sf.value = x
	case reflect.Float32:
		x, err := strconv.ParseFloat(prop.Default, 32)
		if err != nil {
			return nil, false, fmt.Errorf("proto: bad default float32 %q: %v", prop.Default, err)
		}
		sf.value = float32(x)
	case reflect.Float64:
		x, err := strconv.ParseFloat(prop.Default, 64)
		if err != nil {
			return nil, false, fmt.Errorf("proto: bad default float64 %q: %v", prop.Default, err)
		}
		sf.value = x
	case reflect.Int32:
		x, err := strconv.ParseInt(prop.Default, 10, 32)
		if err != nil {
			return nil, false, fmt.Errorf("proto: bad default int32 %q: %v", prop.Default, err)
		}
		sf.value = int32(x)
	case reflect.Int64:
		x, err := strconv.ParseInt(prop.Default, 10, 64)
		if err != nil {
			return nil, false, fmt.Errorf("proto: bad default int64 %q: %v", prop.Default, err)
		}
		sf.value = x
	case reflect.String:
		sf.value = prop.Default
	case reflect.Uint8:
		// []byte (not *uint8)
		sf.value = []byte(prop.Default)
	case reflect.Uint32:
		x, err := strconv.ParseUint(prop.Default, 10, 32)
		if err != nil {
			return nil, false, fmt.Errorf("proto: bad default uint32 %q: %v", prop.Default, err)
		}
		sf.value = uint32(x)
	case reflect.Uint64:
		x, err := strconv.ParseUint(prop.Default, 10, 64)
		if err != nil {
			return nil, false, fmt.Errorf("proto: bad default uint64 %q: %v", prop.Default, err)
		}
		sf.value = x
	default:
		return nil, false, fmt.Errorf("proto: unhandled def kind %v", ft.Elem().Kind())
	}

	return sf, false, nil
}

// mapKeys returns a sort.Interface to be used for sorting the map keys.
// Map fields may have key types of non-float scalars, strings and enums.
func mapKeys(vs []reflect.Value) sort.Interface {
	s := mapKeySorter{vs: vs}

	// Type specialization per https://developers.google.com/protocol-buffers/docs/proto#maps.
	if len(vs) == 0 {
		return s
	}
	switch vs[0].Kind() {
	case reflect.Int32, reflect.Int64:
		s.less = func(a, b reflect.Value) bool { return a.Int() < b.Int() }
	case reflect.Uint32, reflect.Uint64:
		s.less = func(a, b reflect.Value) bool { return a.Uint() < b.Uint() }
	case reflect.Bool:
		s.less = func(a, b reflect.Value) bool { return !a.Bool() && b.Bool() } // false < true
	case reflect.String:
		s.less = func(a, b reflect.Value) bool { return a.String() < b.String() }
	default:
		panic(fmt.Sprintf("unsupported map key type: %v", vs[0].Kind()))
	}

	return s
}

type mapKeySorter struct {
	vs   []reflect.Value
	less func(a, b reflect.Value) bool
}

func (s mapKeySorter) Len() int      { return len(s.vs) }
func (s mapKeySorter) Swap(i, j int) { s.vs[i], s.vs[j] = s.vs[j], s.vs[i] }
func (s mapKeySorter) Less(i, j int) bool {
	return s.less(s.vs[i], s.vs[j])
}

// isProto3Zero reports whether v is a zero proto3 value.
func isProto3Zero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	}
	return false
}

const (
	// ProtoPackageIsVersion3 is referenced from generated protocol buffer files
	// to assert that that code is compatible with this version of the proto package.
	ProtoPackageIsVersion3 = true

	// ProtoPackageIsVersion2 is referenced from generated protocol buffer files
	// to assert that that code is compatible with this version of the proto package.
	ProtoPackageIsVersion2 = true

	// ProtoPackageIsVersion1 is referenced from generated protocol buffer files
	// to assert that that code is compatible with this version of the proto package.
	ProtoPackageIsVersion1 = true
)

// InternalMessageInfo is a type used internally by generated .pb.go files.
// This type is not intended to be used by non-generated code.
// This type is not subject to any compatibility guarantee.
type InternalMessageInfo struct {
	marshal   *marshalInfo
	unmarshal *unmarshalInfo
	merge     *mergeInfo
	discard   *discardInfo
}
