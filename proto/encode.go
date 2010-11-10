// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 Google Inc.  All rights reserved.
// http://code.google.com/p/goprotobuf/
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
 * Routines for encoding data into the wire format for protocol buffers.
 */

import (
	"bytes"
	"os"
	"reflect"
	"runtime"
	"unsafe"
)

// ErrRequiredNotSet is the error returned if Marshal is called with
// a protocol buffer struct whose required fields have not
// all been initialized. It is also the error returned if Unmarshal is
// called with an encoded protocol buffer that does not include all the
// required fields.
var ErrRequiredNotSet = os.NewError("required fields not set")

// ErrRepeatedHasNil is the error returned if Marshal is called with
// a protocol buffer struct with a repeated field containing a nil element.
var ErrRepeatedHasNil = os.NewError("repeated field has nil")

// ErrNil is the error returned if Marshal is called with nil.
var ErrNil = os.NewError("marshal called with nil")

// The fundamental encoders that put bytes on the wire.
// Those that take integer types all accept uint64 and are
// therefore of type valueEncoder.

const maxVarintBytes = 10 // maximum length of a varint

// EncodeVarint returns the varint encoding of x.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
// Not used by the package itself, but helpful to clients
// wishing to use the same encoding.
func EncodeVarint(x uint64) []byte {
	var buf [maxVarintBytes]byte
	var n int
	for n = 0; x > 127; n++ {
		buf[n] = 0x80 | uint8(x&0x7F)
		x >>= 7
	}
	buf[n] = uint8(x)
	n++
	return buf[0:n]
}

var emptyBytes [maxVarintBytes]byte

// EncodeVarint writes a varint-encoded integer to the Buffer.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func (p *Buffer) EncodeVarint(x uint64) os.Error {
	l := len(p.buf)
	if l+maxVarintBytes > cap(p.buf) { // not necessary except for performance
		p.buf = append(p.buf, emptyBytes[:]...)
	} else {
		p.buf = p.buf[:l+maxVarintBytes]
	}

	for x >= 1<<7 {
		p.buf[l] = uint8(x&0x7f | 0x80)
		l++
		x >>= 7
	}
	p.buf[l] = uint8(x)
	p.buf = p.buf[:l+1]
	return nil
}

// EncodeFixed64 writes a 64-bit integer to the Buffer.
// This is the format for the
// fixed64, sfixed64, and double protocol buffer types.
func (p *Buffer) EncodeFixed64(x uint64) os.Error {
	const fixed64Bytes = 8
	l := len(p.buf)
	if l+fixed64Bytes > cap(p.buf) { // not necessary except for performance
		p.buf = append(p.buf, emptyBytes[:fixed64Bytes]...)
	} else {
		p.buf = p.buf[:l+fixed64Bytes]
	}

	p.buf[l] = uint8(x)
	p.buf[l+1] = uint8(x >> 8)
	p.buf[l+2] = uint8(x >> 16)
	p.buf[l+3] = uint8(x >> 24)
	p.buf[l+4] = uint8(x >> 32)
	p.buf[l+5] = uint8(x >> 40)
	p.buf[l+6] = uint8(x >> 48)
	p.buf[l+7] = uint8(x >> 56)
	return nil
}

// EncodeFixed32 writes a 32-bit integer to the Buffer.
// This is the format for the
// fixed32, sfixed32, and float protocol buffer types.
func (p *Buffer) EncodeFixed32(x uint64) os.Error {
	const fixed32Bytes = 4
	l := len(p.buf)
	if l+fixed32Bytes > cap(p.buf) { // not necessary except for performance
		p.buf = append(p.buf, emptyBytes[:fixed32Bytes]...)
	} else {
		p.buf = p.buf[:l+fixed32Bytes]
	}

	p.buf[l] = uint8(x)
	p.buf[l+1] = uint8(x >> 8)
	p.buf[l+2] = uint8(x >> 16)
	p.buf[l+3] = uint8(x >> 24)
	return nil
}

// EncodeZigzag64 writes a zigzag-encoded 64-bit integer
// to the Buffer.
// This is the format used for the sint64 protocol buffer type.
func (p *Buffer) EncodeZigzag64(x uint64) os.Error {
	// use signed number to get arithmetic right shift.
	return p.EncodeVarint(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}

// EncodeZigzag32 writes a zigzag-encoded 32-bit integer
// to the Buffer.
// This is the format used for the sint32 protocol buffer type.
func (p *Buffer) EncodeZigzag32(x uint64) os.Error {
	// use signed number to get arithmetic right shift.
	return p.EncodeVarint(uint64((uint32(x) << 1) ^ uint32((int32(x) >> 31))))
}

// EncodeRawBytes writes a count-delimited byte buffer to the Buffer.
// This is the format used for the bytes protocol buffer
// type and for embedded messages.
func (p *Buffer) EncodeRawBytes(b []byte) os.Error {
	lb := len(b)
	p.EncodeVarint(uint64(lb))
	p.buf = bytes.Add(p.buf, b)
	return nil
}

// EncodeStringBytes writes an encoded string to the Buffer.
// This is the format used for the proto2 string type.
func (p *Buffer) EncodeStringBytes(s string) os.Error {

	// this works because strings and slices are the same.
	y := *(*[]byte)(unsafe.Pointer(&s))
	p.EncodeRawBytes(y)
	return nil
}

// Marshaler is the interface representing objects that can marshal themselves.
type Marshaler interface {
	Marshal() ([]byte, os.Error)
}

// Marshal takes the protocol buffer struct represented by pb
// and encodes it into the wire format, returning the data.
func Marshal(pb interface{}) ([]byte, os.Error) {
	// Can the object marshal itself?
	if m, ok := pb.(Marshaler); ok {
		return m.Marshal()
	}
	p := NewBuffer(nil)
	err := p.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return p.buf, err
}

// Marshal takes the protocol buffer struct represented by pb
// and encodes it into the wire format, writing the result to the
// Buffer.
func (p *Buffer) Marshal(pb interface{}) os.Error {
	// Can the object marshal itself?
	if m, ok := pb.(Marshaler); ok {
		data, err := m.Marshal()
		if err != nil {
			return err
		}
		p.buf = bytes.Add(p.buf, data)
		return nil
	}

	mstat := runtime.MemStats.Mallocs

	t, b, err := getbase(pb)
	if err == nil {
		err = p.enc_struct(t.Elem().(*reflect.StructType), b)
	}

	mstat = runtime.MemStats.Mallocs - mstat
	stats.Emalloc += mstat
	stats.Encode++

	return err
}

// Individual type encoders.

// Encode a bool.
func (o *Buffer) enc_bool(p *Properties, base uintptr) os.Error {
	v := *(**uint8)(unsafe.Pointer(base + p.offset))
	if v == nil {
		return ErrNil
	}
	x := *v
	if x != 0 {
		x = 1
	}
	o.buf = bytes.Add(o.buf, p.tagcode)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode an int32.
func (o *Buffer) enc_int32(p *Properties, base uintptr) os.Error {
	v := *(**uint32)(unsafe.Pointer(base + p.offset))
	if v == nil {
		return ErrNil
	}
	x := *v
	o.buf = bytes.Add(o.buf, p.tagcode)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode an int64.
func (o *Buffer) enc_int64(p *Properties, base uintptr) os.Error {
	v := *(**uint64)(unsafe.Pointer(base + p.offset))
	if v == nil {
		return ErrNil
	}
	x := *v
	o.buf = bytes.Add(o.buf, p.tagcode)
	p.valEnc(o, uint64(x))
	return nil
}

// Encode a string.
func (o *Buffer) enc_string(p *Properties, base uintptr) os.Error {
	v := *(**string)(unsafe.Pointer(base + p.offset))
	if v == nil {
		return ErrNil
	}
	x := *v
	o.buf = bytes.Add(o.buf, p.tagcode)
	o.EncodeStringBytes(x)
	return nil
}

// Encode a message struct.
func (o *Buffer) enc_struct_message(p *Properties, base uintptr) os.Error {
	// Can the object marshal itself?
	iv := unsafe.Unreflect(p.stype, unsafe.Pointer(base+p.offset))
	if m, ok := iv.(Marshaler); ok {
		if n, ok := reflect.NewValue(iv).(nillable); ok && n.IsNil() {
			return ErrNil
		}
		data, err := m.Marshal()
		if err != nil {
			return err
		}
		o.buf = bytes.Add(o.buf, p.tagcode)
		o.EncodeRawBytes(data)
		return nil
	}
	v := *(**struct{})(unsafe.Pointer(base + p.offset))
	if v == nil {
		return ErrNil
	}

	// need the length before we can write out the message itself,
	// so marshal into a separate byte buffer first.
	obuf := o.buf
	o.buf = o.bufalloc()

	b := uintptr(unsafe.Pointer(v))
	typ := p.stype.Elem().(*reflect.StructType)
	err := o.enc_struct(typ, b)

	nbuf := o.buf
	o.buf = obuf
	if err != nil {
		o.buffree(nbuf)
		return err
	}
	o.buf = bytes.Add(o.buf, p.tagcode)
	o.EncodeRawBytes(nbuf)
	o.buffree(nbuf)
	return nil
}

// Encode a group struct.
func (o *Buffer) enc_struct_group(p *Properties, base uintptr) os.Error {
	v := *(**struct{})(unsafe.Pointer(base + p.offset))
	if v == nil {
		return ErrNil
	}

	o.EncodeVarint(uint64((p.Tag << 3) | WireStartGroup))
	b := uintptr(unsafe.Pointer(v))
	typ := p.stype.Elem().(*reflect.StructType)
	err := o.enc_struct(typ, b)
	if err != nil {
		return err
	}
	o.EncodeVarint(uint64((p.Tag << 3) | WireEndGroup))
	return nil
}

// Encode a slice of bools ([]bool).
func (o *Buffer) enc_slice_bool(p *Properties, base uintptr) os.Error {
	s := *(*[]uint8)(unsafe.Pointer(base + p.offset))
	l := len(s)
	if l == 0 {
		return ErrNil
	}
	for _, x := range s {
		o.buf = bytes.Add(o.buf, p.tagcode)
		if x != 0 {
			x = 1
		}
		p.valEnc(o, uint64(x))
	}
	return nil
}

// Encode a slice of bytes ([]byte).
func (o *Buffer) enc_slice_byte(p *Properties, base uintptr) os.Error {
	s := *(*[]uint8)(unsafe.Pointer(base + p.offset))
	// if the field is required, we must send something, even if it's an empty array.
	if !p.Required {
		l := len(s)
		if l == 0 {
			return ErrNil
		}
		// check default
		if l == len(p.Default) {
			same := true
			for i := 0; i < len(p.Default); i++ {
				if p.Default[i] != s[i] {
					same = false
					break
				}
			}
			if same {
				return ErrNil
			}
		}
	}
	o.buf = bytes.Add(o.buf, p.tagcode)
	o.EncodeRawBytes(s)
	return nil
}

// Encode a slice of int32s ([]int32).
func (o *Buffer) enc_slice_int32(p *Properties, base uintptr) os.Error {
	s := *(*[]uint32)(unsafe.Pointer(base + p.offset))
	l := len(s)
	if l == 0 {
		return ErrNil
	}
	for i := 0; i < l; i++ {
		o.buf = bytes.Add(o.buf, p.tagcode)
		x := s[i]
		p.valEnc(o, uint64(x))
	}
	return nil
}

// Encode a slice of int64s ([]int64).
func (o *Buffer) enc_slice_int64(p *Properties, base uintptr) os.Error {
	s := *(*[]uint64)(unsafe.Pointer(base + p.offset))
	l := len(s)
	if l == 0 {
		return ErrNil
	}
	for i := 0; i < l; i++ {
		o.buf = bytes.Add(o.buf, p.tagcode)
		x := s[i]
		p.valEnc(o, uint64(x))
	}
	return nil
}

// Encode a slice of slice of bytes ([][]byte).
func (o *Buffer) enc_slice_slice_byte(p *Properties, base uintptr) os.Error {
	ss := *(*[][]uint8)(unsafe.Pointer(base + p.offset))
	l := len(ss)
	if l == 0 {
		return ErrNil
	}
	for i := 0; i < l; i++ {
		o.buf = bytes.Add(o.buf, p.tagcode)
		s := ss[i]
		o.EncodeRawBytes(s)
	}
	return nil
}

// Encode a slice of strings ([]string).
func (o *Buffer) enc_slice_string(p *Properties, base uintptr) os.Error {
	ss := *(*[]string)(unsafe.Pointer(base + p.offset))
	l := len(ss)
	for i := 0; i < l; i++ {
		o.buf = bytes.Add(o.buf, p.tagcode)
		s := ss[i]
		o.EncodeStringBytes(s)
	}
	return nil
}

// Encode a slice of message structs ([]*struct).
func (o *Buffer) enc_slice_struct_message(p *Properties, base uintptr) os.Error {
	s := *(*[]*struct{})(unsafe.Pointer(base + p.offset))
	l := len(s)
	typ := p.stype.Elem().(*reflect.StructType)

	for i := 0; i < l; i++ {
		v := s[i]
		if v == nil {
			return ErrRepeatedHasNil
		}

		// Can the object marshal itself?
		iv := unsafe.Unreflect(p.stype, unsafe.Pointer(&s[i]))
		if m, ok := iv.(Marshaler); ok {
			if n, ok := reflect.NewValue(iv).(nillable); ok && n.IsNil() {
				return ErrNil
			}
			data, err := m.Marshal()
			if err != nil {
				return err
			}
			o.buf = bytes.Add(o.buf, p.tagcode)
			o.EncodeRawBytes(data)
			continue
		}

		obuf := o.buf
		o.buf = o.bufalloc()

		b := uintptr(unsafe.Pointer(v))
		err := o.enc_struct(typ, b)

		nbuf := o.buf
		o.buf = obuf
		if err != nil {
			o.buffree(nbuf)
			if err == ErrNil {
				return ErrRepeatedHasNil
			}
			return err
		}
		o.buf = bytes.Add(o.buf, p.tagcode)
		o.EncodeRawBytes(nbuf)

		o.buffree(nbuf)
	}
	return nil
}

// Encode a slice of group structs ([]*struct).
func (o *Buffer) enc_slice_struct_group(p *Properties, base uintptr) os.Error {
	s := *(*[]*struct{})(unsafe.Pointer(base + p.offset))
	l := len(s)
	typ := p.stype.Elem().(*reflect.StructType)

	for i := 0; i < l; i++ {
		v := s[i]
		if v == nil {
			return ErrRepeatedHasNil
		}

		o.EncodeVarint(uint64((p.Tag << 3) | WireStartGroup))

		b := uintptr(unsafe.Pointer(v))
		err := o.enc_struct(typ, b)

		if err != nil {
			if err == ErrNil {
				return ErrRepeatedHasNil
			}
			return err
		}

		o.EncodeVarint(uint64((p.Tag << 3) | WireEndGroup))
	}
	return nil
}

// Encode an extension map.
func (o *Buffer) enc_map(p *Properties, base uintptr) os.Error {
	v := *(*map[int32][]byte)(unsafe.Pointer(base + p.offset))
	for _, b := range v {
		o.buf = bytes.Add(o.buf, b)
	}
	return nil
}

// Encode a struct.
func (o *Buffer) enc_struct(t *reflect.StructType, base uintptr) os.Error {
	prop := GetProperties(t)
	required := prop.reqCount
	for _, p := range prop.Prop {
		if p.enc != nil {
			err := p.enc(o, p, base)
			if err != nil {
				if err != ErrNil {
					return err
				}
			} else if p.Required {
				required--
			}
		}
	}
	// See if we encoded all required fields.
	if required > 0 {
		return ErrRequiredNotSet
	}

	return nil
}
