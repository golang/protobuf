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

package protobuf3

/*
 * Routines for encoding data into the wire format for protocol buffers.
 */

import (
	"errors"
	"reflect"
)

var (
	// errRepeatedHasNil is the error returned if Marshal is called with
	// a struct with a repeated field containing a nil element.
	errRepeatedHasNil = errors.New("proto: repeated field has nil element")

	// errOneofHasNil is the error returned if Marshal is called with
	// a struct with a oneof field containing a nil element.
	errOneofHasNil = errors.New("proto: oneof field has nil value")

	// ErrNil is the error returned if Marshal is called with nil.
	ErrNil = errors.New("proto: Marshal called with nil")
)

// The fundamental encoders that put bytes on the wire.
// Those that take integer types all accept uint64 and are
// therefore of type valueEncoder.

// EncodeVarint writes a varint-encoded integer to the Buffer.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func (p *Buffer) EncodeVarint(x uint64) {
	for x >= 1<<7 {
		p.buf = append(p.buf, uint8(x&0x7f|0x80))
		x >>= 7
	}
	p.buf = append(p.buf, uint8(x))
}

// SizeVarint returns the varint encoding size of an integer.
func SizeVarint(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}

// EncodeFixed64 writes a 64-bit integer to the Buffer.
// This is the format for the
// fixed64, sfixed64, and double protocol buffer types.
func (p *Buffer) EncodeFixed64(x uint64) {
	p.buf = append(p.buf,
		uint8(x),
		uint8(x>>8),
		uint8(x>>16),
		uint8(x>>24),
		uint8(x>>32),
		uint8(x>>40),
		uint8(x>>48),
		uint8(x>>56))
}

// EncodeFixed32 writes a 32-bit integer to the Buffer.
// This is the format for the
// fixed32, sfixed32, and float protocol buffer types.
func (p *Buffer) EncodeFixed32(x uint64) {
	p.buf = append(p.buf,
		uint8(x),
		uint8(x>>8),
		uint8(x>>16),
		uint8(x>>24))
}

// EncodeZigzag64 writes a zigzag-encoded 64-bit integer
// to the Buffer.
// This is the format used for the sint64 protocol buffer type.
func (p *Buffer) EncodeZigzag64(x uint64) {
	// use signed number to get arithmetic right shift.
	p.EncodeVarint(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}

// EncodeZigzag32 writes a zigzag-encoded 32-bit integer
// to the Buffer.
// This is the format used for the sint32 protocol buffer type.
func (p *Buffer) EncodeZigzag32(x uint64) {
	// use signed number to get arithmetic right shift.
	p.EncodeVarint(uint64((uint32(x) << 1) ^ uint32((int32(x) >> 31))))
}

// EncodeRawBytes writes a count-delimited byte buffer to the Buffer.
// This is the format used for the bytes protocol buffer
// type and for embedded messages.
func (p *Buffer) EncodeRawBytes(b []byte) {
	p.EncodeVarint(uint64(len(b)))
	p.buf = append(p.buf, b...)
}

// EncodeStringBytes writes an encoded string to the Buffer.
// This is the format used for the proto2 string type.
func (p *Buffer) EncodeStringBytes(s string) {
	p.EncodeVarint(uint64(len(s)))
	p.buf = append(p.buf, s...)
}

// Marshaler is the interface representing objects that can marshal themselves.
type Marshaler interface {
	MarshalProtobuf3() ([]byte, error)
}

// Marshal takes the protocol buffer
// and encodes it into the wire format, returning the data.
func Marshal(pb Message) ([]byte, error) {
	p := NewBuffer(nil)
	err := p.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return p.buf, nil
}

// Marshal takes the protocol buffer
// and encodes it into the wire format, writing the result to the
// Buffer.
func (o *Buffer) Marshal(pb Message) error {
	// Can it marshal itself?
	if m, ok := pb.(Marshaler); ok {
		data, err := m.MarshalProtobuf3()
		if err != nil {
			return err
		}
		o.buf = append(o.buf, data...)
		return nil
	}

	// unpack the interface and sanity check
	if pb == nil {
		return ErrNil // don't pass in nil interfaces. we need types
	}
	v := reflect.ValueOf(pb)
	t := v.Type()
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("protobuf3: can't Marshal(%s): not a *struct", t)
	}
	base := toStructPointer(v)
	if structPointer_IsNil(base) {
		return ErrNil // don't pass in nil pointers. we need values
	}

	return o.enc_struct(GetProperties(t.Elem()), base)
}

// Individual type encoders.

// Encode a *bool.
func (o *Buffer) enc_ptr_bool(p *Properties, base structPointer) error {
	v := *structPointer_Bool(base, p.field)
	if v == nil {
		return ErrNil
	}
	x := 0
	if *v {
		x = 1
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode a bool.
func (o *Buffer) enc_bool(p *Properties, base structPointer) error {
	v := *structPointer_BoolVal(base, p.field)
	if !v {
		return ErrNil
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, 1)
	return nil
}

// Encode an *int32.
func (o *Buffer) enc_ptr_int32(p *Properties, base structPointer) error {
	v := structPointer_Word32(base, p.field)
	if word32_IsNil(v) {
		return ErrNil
	}
	x := int32(word32_Get(v)) // permit sign extension to use full 64-bit range
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode an int32.
func (o *Buffer) enc_int32(p *Properties, base structPointer) error {
	v := structPointer_Word32Val(base, p.field)
	x := int32(word32Val_Get(v)) // permit sign extension to use full 64-bit range
	if x == 0 {
		return ErrNil
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode a *uint32.
// Exactly the same as int32, except for no sign extension.
func (o *Buffer) enc_ptr_uint32(p *Properties, base structPointer) error {
	v := structPointer_Word32(base, p.field)
	if word32_IsNil(v) {
		return ErrNil
	}
	x := word32_Get(v)
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode a uint32.
func (o *Buffer) enc_uint32(p *Properties, base structPointer) error {
	v := structPointer_Word32Val(base, p.field)
	x := word32Val_Get(v)
	if x == 0 {
		return ErrNil
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode an *int64.
func (o *Buffer) enc_ptr_int64(p *Properties, base structPointer) error {
	v := structPointer_Word64(base, p.field)
	if word64_IsNil(v) {
		return ErrNil
	}
	x := word64_Get(v)
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, x)
	return nil
}

// Encode an int64.
func (o *Buffer) enc_int64(p *Properties, base structPointer) error {
	v := structPointer_Word64Val(base, p.field)
	x := word64Val_Get(v)
	if x == 0 {
		return ErrNil
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, x)
	return nil
}

// Encode a *string.
func (o *Buffer) enc_ptr_string(p *Properties, base structPointer) error {
	v := *structPointer_String(base, p.field)
	if v == nil {
		return ErrNil
	}
	x := *v
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeStringBytes(x)
	return nil
}

// Encode a string.
func (o *Buffer) enc_string(p *Properties, base structPointer) error {
	v := *structPointer_StringVal(base, p.field)
	if v == "" {
		return ErrNil
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeStringBytes(v)
	return nil
}

// All protocol buffer fields are nillable, but be careful.
func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// Encode an message struct field of a message struct.
func (o *Buffer) enc_struct_message(p *Properties, base structPointer) error {
	structp := structPointer_GetStructVal(base, p.field)
	// note struct is embedded in base, so pointer cannot be nil

	// Can the object marshal itself?
	if p.isMarshaler {
		m := structPointer_Interface(structp, p.stype).(Marshaler)
		data, err := m.MarshalProtobuf3()
		if err != nil {
			return err
		}
		o.buf = append(o.buf, p.tagcode...)
		o.EncodeRawBytes(data)
		return nil
	}

	o.buf = append(o.buf, p.tagcode...)
	return o.enc_len_struct(p.sprop, structp)

}

// Encode a *message struct.
func (o *Buffer) enc_ptr_struct_message(p *Properties, base structPointer) error {
	structp := structPointer_GetStructPointer(base, p.field)
	if structPointer_IsNil(structp) {
		return ErrNil
	}

	// Can the object marshal itself?
	if p.isMarshaler {
		m := structPointer_Interface(structp, p.stype).(Marshaler)
		data, err := m.MarshalProtobuf3()
		if err != nil {
			return err
		}
		o.buf = append(o.buf, p.tagcode...)
		o.EncodeRawBytes(data)
		return nil
	}

	o.buf = append(o.buf, p.tagcode...)
	return o.enc_len_struct(p.sprop, structp)
}

// Encode a slice of bools ([]bool) in packed format.
func (o *Buffer) enc_slice_packed_bool(p *Properties, base structPointer) error {
	s := *structPointer_BoolSlice(base, p.field)
	l := len(s)
	if l == 0 {
		return ErrNil
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(l)) // each bool takes exactly one byte
	for _, x := range s {
		v := uint64(0)
		if x {
			v = 1
		}
		p.valEnc(o, v)
	}
	return nil
}

// Encode a slice of bytes ([]byte).
func (o *Buffer) enc_slice_byte(p *Properties, base structPointer) error {
	s := *structPointer_Bytes(base, p.field)
	if len(s) == 0 {
		return ErrNil
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeRawBytes(s)
	return nil
}

// Encode a slice of int32s ([]int32) in packed format.
func (o *Buffer) enc_slice_packed_int32(p *Properties, base structPointer) error {
	s := structPointer_Word32Slice(base, p.field)
	l := s.Len()
	if l == 0 {
		return ErrNil
	}
	// TODO: Reuse a Buffer.
	buf := NewBuffer(nil)
	for i := 0; i < l; i++ {
		x := int32(s.Index(i)) // permit sign extension to use full 64-bit range
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
	return nil
}

// Encode a slice of uint32s ([]uint32) in packed format.
// Exactly the same as int32, except for no sign extension.
func (o *Buffer) enc_slice_packed_uint32(p *Properties, base structPointer) error {
	s := structPointer_Word32Slice(base, p.field)
	l := s.Len()
	if l == 0 {
		return ErrNil
	}
	// TODO: Reuse a Buffer.
	buf := NewBuffer(nil)
	for i := 0; i < l; i++ {
		p.valEnc(buf, uint64(s.Index(i)))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
	return nil
}

// Encode a slice of int64s ([]int64) in packed format.
func (o *Buffer) enc_slice_packed_int64(p *Properties, base structPointer) error {
	s := structPointer_Word64Slice(base, p.field)
	l := s.Len()
	if l == 0 {
		return ErrNil
	}
	// TODO: Reuse a Buffer.
	buf := NewBuffer(nil)
	for i := 0; i < l; i++ {
		p.valEnc(buf, s.Index(i))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
	return nil
}

// Encode a slice of slice of bytes ([][]byte).
func (o *Buffer) enc_slice_slice_byte(p *Properties, base structPointer) error {
	ss := *structPointer_BytesSlice(base, p.field)
	l := len(ss)
	if l == 0 {
		return ErrNil
	}
	for i := 0; i < l; i++ {
		o.buf = append(o.buf, p.tagcode...)
		o.EncodeRawBytes(ss[i])
	}
	return nil
}

// Encode a slice of strings ([]string).
func (o *Buffer) enc_slice_string(p *Properties, base structPointer) error {
	ss := *structPointer_StringSlice(base, p.field)
	l := len(ss)
	for i := 0; i < l; i++ {
		o.buf = append(o.buf, p.tagcode...)
		o.EncodeStringBytes(ss[i])
	}
	return nil
}

// Encode a slice of *message structs ([]*struct).
func (o *Buffer) enc_slice_ptr_struct_message(p *Properties, base structPointer) error {
	s := structPointer_StructPointerSlice(base, p.field)
	l := s.Len()

	for i := 0; i < l; i++ {
		structp := s.Index(i)
		if structPointer_IsNil(structp) {
			return errRepeatedHasNil
		}

		// Can the object marshal itself?
		if p.isMarshaler {
			m := structPointer_Interface(structp, p.stype).(Marshaler)
			data, err := m.MarshalProtobuf3()
			if err != nil {
				return err
			}
			o.buf = append(o.buf, p.tagcode...)
			o.EncodeRawBytes(data)
			continue
		}

		o.buf = append(o.buf, p.tagcode...)
		err := o.enc_len_struct(p.sprop, structp)
		if err != nil {
			if err == ErrNil {
				return errRepeatedHasNil
			}
			return err
		}
	}
	return nil
}

// Encode a slice of message structs ([]struct).
func (o *Buffer) enc_slice_struct_message(p *Properties, base structPointer) error {
	s := structPointer_StructSlice(base, p.field)
	l := s.Len() // note this is the byte size of the slice's array, not # of elements

	for i := uintptr(0); i < l; i += p.stype.Size() {
		structp := s.Index(i)

		// Can the object marshal itself?
		if p.isMarshaler {
			m := structPointer_Interface(structp, p.stype).(Marshaler)
			data, err := m.MarshalProtobuf3()
			if err != nil {
				return err
			}
			o.buf = append(o.buf, p.tagcode...)
			o.EncodeRawBytes(data)
			continue
		}

		o.buf = append(o.buf, p.tagcode...)
		err := o.enc_len_struct(p.sprop, structp)
		if err != nil {
			if err == ErrNil {
				return errRepeatedHasNil
			}
			return err
		}
	}
	return nil
}

// Encode a map field.
func (o *Buffer) enc_new_map(p *Properties, base structPointer) error {
	/*
		A map defined as
			map<key_type, value_type> map_field = N;
		is encoded in the same way as
			message MapFieldEntry {
				key_type key = 1;
				value_type value = 2;
			}
			repeated MapFieldEntry map_field = N;
	*/

	v := structPointer_NewAt(base, p.field, p.mtype).Elem() // map[K]V
	if v.Len() == 0 {
		return nil
	}

	keycopy, valcopy, keybase, valbase := mapEncodeScratch(p.mtype)

	enc := func() error {
		if err := p.mkeyprop.enc(o, p.mkeyprop, keybase); err != nil {
			return err
		}
		if err := p.mvalprop.enc(o, p.mvalprop, valbase); err != nil && err != ErrNil {
			return err
		}
		return nil
	}

	// Don't sort map keys. It is not required by the spec, and C++ doesn't do it.
	for _, key := range v.MapKeys() {
		val := v.MapIndex(key)

		keycopy.Set(key)
		valcopy.Set(val)

		o.buf = append(o.buf, p.tagcode...)
		if err := o.enc_len_thing(enc); err != nil {
			return err
		}
	}
	return nil
}

// mapEncodeScratch returns a new reflect.Value matching the map's value type,
// and a structPointer suitable for passing to an encoder or sizer.
func mapEncodeScratch(mapType reflect.Type) (keycopy, valcopy reflect.Value, keybase, valbase structPointer) {
	// Prepare addressable doubly-indirect placeholders for the key and value types.
	// This is needed because the element-type encoders expect **T, but the map iteration produces T.

	keycopy = reflect.New(mapType.Key()).Elem()                 // addressable K
	keyptr := reflect.New(reflect.PtrTo(keycopy.Type())).Elem() // addressable *K
	keyptr.Set(keycopy.Addr())                                  //
	keybase = toStructPointer(keyptr.Addr())                    // **K

	// Value types are more varied and require special handling.
	switch mapType.Elem().Kind() {
	case reflect.Slice:
		// []byte
		var dummy []byte
		valcopy = reflect.ValueOf(&dummy).Elem() // addressable []byte
		valbase = toStructPointer(valcopy.Addr())
	case reflect.Ptr:
		// message; the generated field type is map[K]*Msg (so V is *Msg),
		// so we only need one level of indirection.
		valcopy = reflect.New(mapType.Elem()).Elem() // addressable V
		valbase = toStructPointer(valcopy.Addr())
	default:
		// everything else
		valcopy = reflect.New(mapType.Elem()).Elem()                // addressable V
		valptr := reflect.New(reflect.PtrTo(valcopy.Type())).Elem() // addressable *V
		valptr.Set(valcopy.Addr())                                  //
		valbase = toStructPointer(valptr.Addr())                    // **V
	}
	return
}

// Encode a struct.
func (o *Buffer) enc_struct(prop *StructProperties, base structPointer) error {
	// Encode fields in tag order so that decoders may use optimizations
	// that depend on the ordering.
	// https://developers.google.com/protocol-buffers/docs/encoding#order
	for _, i := range prop.order {
		p := &prop.Prop[i]
		if p.enc != nil {
			err := p.enc(o, p, base)
			if err != nil {
				if err == errRepeatedHasNil {
					// Give more context to nil values in repeated fields.
					return errors.New("repeated field " + p.Name + " has nil element")
				} else {
					return err
				}
			}
		}
	}

	return nil
}

var zeroes [20]byte // longer than any conceivable SizeVarint

// Encode a struct, preceded by its encoded length (as a varint).
func (o *Buffer) enc_len_struct(prop *StructProperties, base structPointer) error {
	return o.enc_len_thing(func() error { return o.enc_struct(prop, base) })
}

// Encode something, preceded by its encoded length (as a varint).
func (o *Buffer) enc_len_thing(enc func() error) error {
	iLen := len(o.buf)
	o.buf = append(o.buf, 0, 0, 0, 0) // reserve four bytes for length
	iMsg := len(o.buf)
	err := enc()
	if err != nil {
		return err
	}
	lMsg := len(o.buf) - iMsg
	lLen := SizeVarint(uint64(lMsg))
	switch x := lLen - (iMsg - iLen); {
	case x > 0: // actual length is x bytes larger than the space we reserved
		// Move msg x bytes right.
		o.buf = append(o.buf, zeroes[:x]...)
		copy(o.buf[iMsg+x:], o.buf[iMsg:iMsg+lMsg])
	case x < 0: // actual length is x bytes smaller than the space we reserved
		// Move msg x bytes left.
		copy(o.buf[iMsg+x:], o.buf[iMsg:iMsg+lMsg])
		o.buf = o.buf[:len(o.buf)+x] // x is negative
	}
	// Encode the length in the reserved space.
	o.buf = o.buf[:iLen]
	o.EncodeVarint(uint64(lMsg))
	o.buf = o.buf[:len(o.buf)+lMsg]
	return nil
}
