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
	"fmt"
	"reflect"
	"time"
	"unsafe"
)

var (
	// errRepeatedHasNil is the error returned if Marshal is called with
	// a struct with a repeated field containing a nil element.
	errRepeatedHasNil = errors.New("protobuf3: repeated field has nil element")

	// ErrNil is the error returned if Marshal is called with nil.
	ErrNil = errors.New("protobuf3: Marshal called with nil")
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
			o.noteError(err)
		}
		o.buf = append(o.buf, data...)
		return o.err
	}

	// unpack the interface and sanity check
	if pb == nil {
		return ErrNil // don't pass in nil interfaces. we need types
	}
	v := reflect.ValueOf(pb)
	t := v.Type()
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("protobuf3: can't Marshal(%s): not a *struct type", t)
	}
	base := structPointer(unsafe.Pointer(v.Pointer()))
	if base == nil {
		return ErrNil // don't pass in nil pointers. we need values
	}

	o.enc_struct(GetProperties(t.Elem()), base)

	return o.err
}

// Individual type encoders.

// Encode a *bool.
func (o *Buffer) enc_ptr_bool(p *Properties, base structPointer) {
	v := *(**bool)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if v == nil {
		return
	}
	x := 0
	if *v {
		x = 1
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode a bool.
func (o *Buffer) enc_bool(p *Properties, base structPointer) {
	v := *(*bool)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if !v {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, 1)
}

// Encode an int
func (o *Buffer) enc_int(p *Properties, base structPointer) {
	x := *(*int)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode a uint
func (o *Buffer) enc_uint(p *Properties, base structPointer) {
	x := *(*uint)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode an int8
func (o *Buffer) enc_int8(p *Properties, base structPointer) {
	x := *(*int8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode a uint8
func (o *Buffer) enc_uint8(p *Properties, base structPointer) {
	x := *(*uint8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode an int16
func (o *Buffer) enc_int16(p *Properties, base structPointer) {
	x := *(*int16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode a uint16
func (o *Buffer) enc_uint16(p *Properties, base structPointer) {
	x := *(*uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode an *int32.
func (o *Buffer) enc_ptr_int32(p *Properties, base structPointer) {
	v := *(**int32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if v == nil {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(*v))
}

// Encode an int32.
func (o *Buffer) enc_int32(p *Properties, base structPointer) {
	x := *(*int32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode a *uint32.
// Exactly the same as int32, except for no sign extension.
func (o *Buffer) enc_ptr_uint32(p *Properties, base structPointer) {
	v := *(**uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if v == nil {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(*v))
}

// Encode a uint32.
func (o *Buffer) enc_uint32(p *Properties, base structPointer) {
	x := *(*uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, uint64(x))
}

// Encode an *int64.
func (o *Buffer) enc_ptr_int64(p *Properties, base structPointer) {
	v := *(**uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if v == nil {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, *v)
}

// Encode an int64.
func (o *Buffer) enc_int64(p *Properties, base structPointer) {
	x := *(*uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	p.valEnc(o, x)
}

// Encode a *string.
func (o *Buffer) enc_ptr_string(p *Properties, base structPointer) {
	v := *(**string)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if v == nil {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeStringBytes(*v)
}

// Encode a string.
func (o *Buffer) enc_string(p *Properties, base structPointer) {
	x := *(*string)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if x == "" {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeStringBytes(x)
}

// Encode an message struct field which implements the Marshaler interface
func (o *Buffer) enc_marshaler(p *Properties, base structPointer) {
	ptr := (unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	// note *ptr is embedded in base, so pointer cannot be nil

	m := reflect.NewAt(p.stype, ptr).Interface().(Marshaler)
	data, err := m.MarshalProtobuf3()
	if err != nil {
		o.noteError(err)
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeRawBytes(data)
}

// Encode an message struct field of a message struct.
func (o *Buffer) enc_struct_message(p *Properties, base structPointer) {
	structp := (structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	// note struct is embedded in base, so pointer cannot be nil

	o.buf = append(o.buf, p.tagcode...)
	o.enc_len_struct(p.sprop, structp)
}

// Encode a *Marshaler.
func (o *Buffer) enc_ptr_marshaler(p *Properties, base structPointer) {
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if ptr == nil {
		return
	}

	m := reflect.NewAt(p.stype, ptr).Interface().(Marshaler)
	data, err := m.MarshalProtobuf3()
	if err != nil {
		o.noteError(err)
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeRawBytes(data)
}

// Encode a *message struct.
func (o *Buffer) enc_ptr_struct_message(p *Properties, base structPointer) {
	structp := *(*structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if structp == nil {
		return
	}

	o.buf = append(o.buf, p.tagcode...)
	o.enc_len_struct(p.sprop, structp)
}

// Encode a slice of bools ([]bool) in packed format.
func (o *Buffer) enc_slice_packed_bool(p *Properties, base structPointer) {
	s := *(*[]bool)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
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
}

// Encode an array of bools ([N]bool) in packed format.
func (o *Buffer) enc_array_packed_bool(p *Properties, base structPointer) {
	n := p.length
	s := ((*[maxLen]bool)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(n)) // each bool takes exactly one byte
	for _, x := range s {
		v := uint64(0)
		if x {
			v = 1
		}
		p.valEnc(o, v)
	}
}

// Encode a slice of bytes ([]byte).
func (o *Buffer) enc_slice_byte(p *Properties, base structPointer) {
	s := *(*[]byte)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	if len(s) == 0 {
		return
	}
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeRawBytes(s)
}

// Encode an array of bytes ([n]byte).
func (o *Buffer) enc_array_byte(p *Properties, base structPointer) {
	n := p.length
	s := ((*[maxLen]byte)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]
	o.buf = append(o.buf, p.tagcode...)
	o.EncodeRawBytes(s)
}

// Encode a slice of int ([]int) in packed format.
func (o *Buffer) enc_slice_packed_int(p *Properties, base structPointer) {
	s := *(*[]int)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of uint ([]uint) in packed format.
func (o *Buffer) enc_slice_packed_uint(p *Properties, base structPointer) {
	s := *(*[]uint)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of int8s ([]int8) in packed format.
func (o *Buffer) enc_slice_packed_int8(p *Properties, base structPointer) {
	s := *(*[]int8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of int16s ([]int16) in packed format.
func (o *Buffer) enc_slice_packed_int16(p *Properties, base structPointer) {
	s := *(*[]int16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of uint16s ([]uint16) in packed format.
func (o *Buffer) enc_slice_packed_uint16(p *Properties, base structPointer) {
	s := *(*[]uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of int32s ([]int32) in packed format.
func (o *Buffer) enc_slice_packed_int32(p *Properties, base structPointer) {
	s := *(*[]int32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode an array of int32s ([length]int32) in packed format.
func (o *Buffer) enc_array_packed_int32(p *Properties, base structPointer) {
	n := p.length
	s := ((*[maxLen / 4]int32)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]

	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of uint32s ([]uint32) in packed format.
// Exactly the same as int32, except for no sign extension.
func (o *Buffer) enc_slice_packed_uint32(p *Properties, base structPointer) {
	s := *(*[]uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode an array of uint32s ([length]uint32) in packed format.
func (o *Buffer) enc_array_packed_uint32(p *Properties, base structPointer) {
	n := p.length
	s := ((*[maxLen / 4]uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]

	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, uint64(x))
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of int64s or uint64s ([](u)int64) in packed format.
func (o *Buffer) enc_slice_packed_int64(p *Properties, base structPointer) {
	s := *(*[]uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	l := len(s)
	if l == 0 {
		return
	}
	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, x)
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode an array of int64s ([n]int64) in packed format.
func (o *Buffer) enc_array_packed_int64(p *Properties, base structPointer) {
	n := p.length
	s := ((*[maxLen / 8]uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]

	buf := NewBuffer(nil)
	for _, x := range s {
		p.valEnc(buf, x)
	}

	o.buf = append(o.buf, p.tagcode...)
	o.EncodeVarint(uint64(len(buf.buf)))
	o.buf = append(o.buf, buf.buf...)
}

// Encode a slice of slice of bytes ([][]byte).
func (o *Buffer) enc_slice_slice_byte(p *Properties, base structPointer) {
	ss := *(*[][]byte)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	for _, s := range ss {
		o.buf = append(o.buf, p.tagcode...)
		o.EncodeRawBytes(s)
	}
}

// Encode a slice of strings ([]string).
func (o *Buffer) enc_slice_string(p *Properties, base structPointer) {
	ss := *(*[]string)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	for _, x := range ss {
		o.buf = append(o.buf, p.tagcode...)
		o.EncodeStringBytes(x)
	}
}

// Encode an array of strings ([n]string).
func (o *Buffer) enc_array_string(p *Properties, base structPointer) {
	n := p.length
	s := ((*[maxLen / 8 / 2]string)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]

	for _, x := range s {
		o.buf = append(o.buf, p.tagcode...)
		o.EncodeStringBytes(x)
	}
}

// Encode a slice of *message structs ([]*struct).
func (o *Buffer) enc_slice_ptr_struct_message(p *Properties, base structPointer) {
	s := *(*[]structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	// Can the object marshal itself?
	if p.isMarshaler {
		for _, structp := range s {
			if structp == nil {
				o.noteError(errRepeatedHasNil)
				return
			}

			m := reflect.NewAt(p.stype, unsafe.Pointer(structp)).Interface().(Marshaler)
			data, err := m.MarshalProtobuf3()
			if err != nil {
				o.noteError(err)
				return
			}
			o.buf = append(o.buf, p.tagcode...)
			o.EncodeRawBytes(data)
		}
		return
	}

	for _, structp := range s {
		if structp == nil {
			o.noteError(errRepeatedHasNil)
			return
		}

		o.buf = append(o.buf, p.tagcode...)
		o.enc_len_struct(p.sprop, structp)
	}
}

// Encode an array of *message structs ([n]*struct).
func (o *Buffer) enc_array_ptr_struct_message(p *Properties, base structPointer) {
	n := p.length
	s := ((*[maxLen / 8]structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]

	// Can the object marshal itself?
	if p.isMarshaler {
		for _, structp := range s {
			if structp == nil {
				o.noteError(errRepeatedHasNil)
				return
			}

			m := reflect.NewAt(p.stype, unsafe.Pointer(structp)).Interface().(Marshaler)
			data, err := m.MarshalProtobuf3()
			if err != nil {
				o.noteError(err)
				return
			}
			o.buf = append(o.buf, p.tagcode...)
			o.EncodeRawBytes(data)
		}
		return
	}

	for _, structp := range s {
		if structp == nil {
			o.noteError(errRepeatedHasNil)
			return
		}

		o.buf = append(o.buf, p.tagcode...)
		o.enc_len_struct(p.sprop, structp)
	}
}

// Encode a slice of message structs ([]struct).
func (o *Buffer) enc_slice_struct_message(p *Properties, base structPointer) {
	s := *(*[]byte)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	n := len(s) // note this is the byte size of the slice's array, not # of elements
	sz := int(p.stype.Size())

	// Can the object marshal itself?
	if p.isMarshaler {
		for i := 0; i < n; i += sz {
			structp := unsafe.Pointer(&s[i])

			m := reflect.NewAt(p.stype, structp).Interface().(Marshaler)
			data, err := m.MarshalProtobuf3()
			if err != nil {
				o.noteError(err)
				return
			}
			o.buf = append(o.buf, p.tagcode...)
			o.EncodeRawBytes(data)
		}
		return
	}

	for i := 0; i < n; i += sz {
		structp := structPointer(unsafe.Pointer(&s[i]))

		o.buf = append(o.buf, p.tagcode...)
		o.enc_len_struct(p.sprop, structp)
	}
}

// Encode an array of message structs ([n]struct).
func (o *Buffer) enc_array_struct_message(p *Properties, base structPointer) {
	sz := p.stype.Size()
	n := uintptr(p.length) * sz // note this is the byte size of the slice's array, not # of elements
	s := ((*[maxLen]byte)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))[0:n:n]

	// Can the object marshal itself?
	if p.isMarshaler {
		for i := uintptr(0); i < n; i += sz {
			structp := structPointer(unsafe.Pointer(&s[i]))

			m := reflect.NewAt(p.stype, unsafe.Pointer(structp)).Interface().(Marshaler)
			data, err := m.MarshalProtobuf3()
			if err != nil {
				o.noteError(err)
				return
			}
			o.buf = append(o.buf, p.tagcode...)
			o.EncodeRawBytes(data)
		}
		return
	}

	for i := uintptr(0); i < n; i += sz {
		structp := structPointer(unsafe.Pointer(&s[i]))

		o.buf = append(o.buf, p.tagcode...)
		o.enc_len_struct(p.sprop, structp)
	}
}

// Encode a map field.
func (o *Buffer) enc_new_map(p *Properties, base structPointer) {
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

	v := reflect.NewAt(p.mtype, unsafe.Pointer(uintptr(base)+uintptr(p.field))).Elem() // map[K]V
	if v.Len() == 0 {
		return
	}

	keycopy, valcopy, keybase, valbase := mapEncodeScratch(p.mtype)

	enc := func() {
		p.mkeyprop.enc(o, p.mkeyprop, keybase)
		p.mvalprop.enc(o, p.mvalprop, valbase)
	}

	// Don't sort map keys. It is not required by the spec, and C++ doesn't do it.
	for _, key := range v.MapKeys() {
		val := v.MapIndex(key)

		keycopy.Set(key)
		valcopy.Set(val)

		o.buf = append(o.buf, p.tagcode...)
		o.enc_len_thing(enc)
	}
}

// mapEncodeScratch returns a new reflect.Value matching the map's value type,
// and a structPointer suitable for passing to an encoder or sizer.
func mapEncodeScratch(mapType reflect.Type) (keycopy, valcopy reflect.Value, keybase, valbase structPointer) {
	// Prepare addressable doubly-indirect placeholders for the key and value types.
	// This is needed because the element-type encoders expect **T, but the map iteration produces T.

	keycopy = reflect.New(mapType.Key()).Elem()                      // addressable K
	keyptr := reflect.New(reflect.PtrTo(keycopy.Type())).Elem()      // addressable *K
	keyptr.Set(keycopy.Addr())                                       //
	keybase = structPointer(unsafe.Pointer(keyptr.Addr().Pointer())) // **K

	// Value types are more varied and require special handling.
	switch mapType.Elem().Kind() {
	case reflect.Slice:
		// []byte
		var dummy []byte
		valcopy = reflect.ValueOf(&dummy).Elem() // addressable []byte
		valbase = structPointer(unsafe.Pointer(valcopy.Addr().Pointer()))
	case reflect.Ptr:
		// message; the generated field type is map[K]*Msg (so V is *Msg),
		// so we only need one level of indirection.
		valcopy = reflect.New(mapType.Elem()).Elem() // addressable V
		valbase = structPointer(unsafe.Pointer(valcopy.Addr().Pointer()))
	default:
		// everything else
		valcopy = reflect.New(mapType.Elem()).Elem()                     // addressable V
		valptr := reflect.New(reflect.PtrTo(valcopy.Type())).Elem()      // addressable *V
		valptr.Set(valcopy.Addr())                                       //
		valbase = structPointer(unsafe.Pointer(valptr.Addr().Pointer())) // **V
	}
	return
}

// Encode a struct.
func (o *Buffer) enc_struct(prop *StructProperties, base structPointer) {
	// Encode fields in tag order so that decoders may use optimizations
	// that depend on the ordering.
	// https://developers.google.com/protocol-buffers/docs/encoding#order
	for _, i := range prop.order {
		p := &prop.Prop[i]
		p.enc(o, p, base)
	}
}

var zeroes [20]byte // longer than any conceivable SizeVarint

// Encode a struct, preceded by its encoded length (as a varint).
func (o *Buffer) enc_len_struct(prop *StructProperties, base structPointer) {
	o.enc_len_thing(func() { o.enc_struct(prop, base) })
}

// Encode something, preceded by its encoded length (as a varint).
func (o *Buffer) enc_len_thing(enc func()) {
	iLen := len(o.buf)
	o.buf = append(o.buf, 0, 0, 0, 0) // reserve four bytes for length
	iMsg := len(o.buf)
	enc()
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
}

// dummy no-op encoder used for encoding 0-length array types
func (o *Buffer) enc_nothing(p *Properties, base structPointer) {
	return
}

// custom encoder for time.Time, encpding it into the protobuf3 standard Timestamp
func (o *Buffer) enc_time_Time(p *Properties, base structPointer) {
	t := (*time.Time)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	// protobuf Timestamp uses its own encoding, different from time.Time
	// we have to convert.
	// don't blame me, the algo comes from ptypes/timestamp.go
	secs := t.Unix()
	nanos := int32(t.Sub(time.Unix(secs, 0))) // abuses the implementation detail that time.Duration is in nanoseconds

	o.buf = append(o.buf, 1<<3|byte(WireVarint))
	o.EncodeVarint(uint64(secs))
	o.buf = append(o.buf, 2<<3|byte(WireVarint))
	o.EncodeVarint(uint64(nanos))
}
