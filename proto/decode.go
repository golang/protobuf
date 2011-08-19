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
 * Routines for decoding protocol buffer data to construct in-memory representations.
 */

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"unsafe"
)

// ErrWrongType occurs when the wire encoding for the field disagrees with
// that specified in the type being decoded.  This is usually caused by attempting
// to convert an encoded protocol buffer into a struct of the wrong type.
var ErrWrongType = os.NewError("field/encoding mismatch: wrong type for field")

// The fundamental decoders that interpret bytes on the wire.
// Those that take integer types all return uint64 and are
// therefore of type valueDecoder.

// DecodeVarint reads a varint-encoded integer from the slice.
// It returns the integer and the number of bytes consumed, or
// zero if there is not enough.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func DecodeVarint(buf []byte) (x uint64, n int) {
	// x, n already 0
	for shift := uint(0); ; shift += 7 {
		if n >= len(buf) {
			return 0, 0
		}
		b := uint64(buf[n])
		n++
		x |= (b & 0x7F) << shift
		if (b & 0x80) == 0 {
			break
		}
	}
	return x, n
}

// DecodeVarint reads a varint-encoded integer from the Buffer.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func (p *Buffer) DecodeVarint() (x uint64, err os.Error) {
	// x, err already 0

	i := p.index
	l := len(p.buf)

	for shift := uint(0); ; shift += 7 {
		if i >= l {
			err = io.ErrUnexpectedEOF
			return
		}
		b := p.buf[i]
		i++
		x |= (uint64(b) & 0x7F) << shift
		if b < 0x80 {
			break
		}
	}
	p.index = i
	return
}

// DecodeFixed64 reads a 64-bit integer from the Buffer.
// This is the format for the
// fixed64, sfixed64, and double protocol buffer types.
func (p *Buffer) DecodeFixed64() (x uint64, err os.Error) {
	// x, err already 0
	i := p.index + 8
	if i > len(p.buf) {
		err = io.ErrUnexpectedEOF
		return
	}
	p.index = i

	x = uint64(p.buf[i-8])
	x |= uint64(p.buf[i-7]) << 8
	x |= uint64(p.buf[i-6]) << 16
	x |= uint64(p.buf[i-5]) << 24
	x |= uint64(p.buf[i-4]) << 32
	x |= uint64(p.buf[i-3]) << 40
	x |= uint64(p.buf[i-2]) << 48
	x |= uint64(p.buf[i-1]) << 56
	return
}

// DecodeFixed32 reads a 32-bit integer from the Buffer.
// This is the format for the
// fixed32, sfixed32, and float protocol buffer types.
func (p *Buffer) DecodeFixed32() (x uint64, err os.Error) {
	// x, err already 0
	i := p.index + 4
	if i > len(p.buf) {
		err = io.ErrUnexpectedEOF
		return
	}
	p.index = i

	x = uint64(p.buf[i-4])
	x |= uint64(p.buf[i-3]) << 8
	x |= uint64(p.buf[i-2]) << 16
	x |= uint64(p.buf[i-1]) << 24
	return
}

// DecodeZigzag64 reads a zigzag-encoded 64-bit integer
// from the Buffer.
// This is the format used for the sint64 protocol buffer type.
func (p *Buffer) DecodeZigzag64() (x uint64, err os.Error) {
	x, err = p.DecodeVarint()
	if err != nil {
		return
	}
	x = (x >> 1) ^ uint64((int64(x&1)<<63)>>63)
	return
}

// DecodeZigzag32 reads a zigzag-encoded 32-bit integer
// from  the Buffer.
// This is the format used for the sint32 protocol buffer type.
func (p *Buffer) DecodeZigzag32() (x uint64, err os.Error) {
	x, err = p.DecodeVarint()
	if err != nil {
		return
	}
	x = uint64((uint32(x) >> 1) ^ uint32((int32(x&1)<<31)>>31))
	return
}

// These are not ValueDecoders: they produce an array of bytes or a string.
// bytes, embedded messages

// DecodeRawBytes reads a count-delimited byte buffer from the Buffer.
// This is the format used for the bytes protocol buffer
// type and for embedded messages.
func (p *Buffer) DecodeRawBytes(alloc bool) (buf []byte, err os.Error) {
	n, err := p.DecodeVarint()
	if err != nil {
		return
	}

	nb := int(n)
	if p.index+nb > len(p.buf) {
		err = io.ErrUnexpectedEOF
		return
	}

	if !alloc {
		// todo: check if can get more uses of alloc=false
		buf = p.buf[p.index : p.index+nb]
		p.index += nb
		return
	}

	buf = make([]byte, nb)
	copy(buf, p.buf[p.index:])
	p.index += nb
	return
}

// DecodeStringBytes reads an encoded string from the Buffer.
// This is the format used for the proto2 string type.
func (p *Buffer) DecodeStringBytes() (s string, err os.Error) {
	buf, err := p.DecodeRawBytes(false)
	if err != nil {
		return
	}
	return string(buf), nil
}

// Skip the next item in the buffer. Its wire type is decoded and presented as an argument.
// If the protocol buffer has extensions, and the field matches, add it as an extension.
// Otherwise, if the XXX_unrecognized field exists, append the skipped data there.
func (o *Buffer) skipAndSave(t reflect.Type, tag, wire int, base uintptr) os.Error {

	oi := o.index

	err := o.skip(t, tag, wire)
	if err != nil {
		return err
	}

	x := fieldIndex(t, "XXX_unrecognized")
	if x == nil {
		return nil
	}

	p := propByIndex(t, x)
	ptr := (*[]byte)(unsafe.Pointer(base + p.offset))

	if *ptr == nil {
		// This is the first skipped element,
		// allocate a new buffer.
		*ptr = o.bufalloc()
	}

	// Add the skipped field to struct field
	obuf := o.buf

	o.buf = *ptr
	o.EncodeVarint(uint64(tag<<3 | wire))
	*ptr = append(o.buf, obuf[oi:o.index]...)

	o.buf = obuf

	return nil
}

// Skip the next item in the buffer. Its wire type is decoded and presented as an argument.
func (o *Buffer) skip(t reflect.Type, tag, wire int) os.Error {

	var u uint64
	var err os.Error

	switch wire {
	case WireVarint:
		_, err = o.DecodeVarint()
	case WireFixed64:
		_, err = o.DecodeFixed64()
	case WireBytes:
		_, err = o.DecodeRawBytes(false)
	case WireFixed32:
		_, err = o.DecodeFixed32()
	case WireStartGroup:
		for {
			u, err = o.DecodeVarint()
			if err != nil {
				break
			}
			fwire := int(u & 0x7)
			if fwire == WireEndGroup {
				break
			}
			ftag := int(u >> 3)
			err = o.skip(t, ftag, fwire)
			if err != nil {
				break
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "proto: can't skip wire type %d for %s\n", wire, t)
	}
	return err
}

// Unmarshaler is the interface representing objects that can unmarshal themselves.
type Unmarshaler interface {
	Unmarshal([]byte) os.Error
}

// Unmarshal parses the protocol buffer representation in buf and places the
// decoded result in pb.  If the struct underlying pb does not match
// the data in buf, the results can be unpredictable.
func Unmarshal(buf []byte, pb interface{}) os.Error {
	// If the object can unmarshal itself, let it.
	if u, ok := pb.(Unmarshaler); ok {
		return u.Unmarshal(buf)
	}

	return NewBuffer(buf).Unmarshal(pb)
}

// Unmarshal parses the protocol buffer representation in the
// Buffer and places the decoded result in pb.  If the struct
// underlying pb does not match the data in the buffer, the results can be
// unpredictable.
func (p *Buffer) Unmarshal(pb interface{}) os.Error {
	// If the object can unmarshal itself, let it.
	if u, ok := pb.(Unmarshaler); ok {
		err := u.Unmarshal(p.buf[p.index:])
		p.index = len(p.buf)
		return err
	}

	mstat := runtime.MemStats.Mallocs

	typ, base, err := getbase(pb)
	if err != nil {
		return err
	}

	err = p.unmarshalType(typ, false, base)

	mstat = runtime.MemStats.Mallocs - mstat
	stats.Dmalloc += mstat
	stats.Decode++

	return err
}

// unmarshalType does the work of unmarshaling a structure.
func (o *Buffer) unmarshalType(t reflect.Type, is_group bool, base uintptr) os.Error {
	st := t.Elem()
	prop := GetProperties(st)
	required, reqFields := prop.reqCount, uint64(0)
	sbase := getsbase(prop) // scratch area for data items

	var err os.Error
	for err == nil && o.index < len(o.buf) {
		oi := o.index
		var u uint64
		u, err = o.DecodeVarint()
		if err != nil {
			break
		}
		wire := int(u & 0x7)
		if wire == WireEndGroup {
			if is_group {
				return nil // input is satisfied
			}
			return ErrWrongType
		}
		tag := int(u >> 3)
		fieldnum, ok := prop.tags[tag]
		if !ok {
			// Maybe it's an extension?
			o.ptr = base
			iv := unsafe.Unreflect(t, unsafe.Pointer(&o.ptr))
			if e, ok := iv.(extendableProto); ok && isExtensionField(e, int32(tag)) {
				if err = o.skip(st, tag, wire); err == nil {
					e.ExtensionMap()[int32(tag)] = Extension{enc: append([]byte(nil), o.buf[oi:o.index]...)}
				}
				continue
			}
			err = o.skipAndSave(st, tag, wire, base)
			continue
		}
		p := prop.Prop[fieldnum]

		if p.dec == nil {
			fmt.Fprintf(os.Stderr, "no protobuf decoder for %s.%s\n", t, st.Field(fieldnum).Name)
			continue
		}
		dec := p.dec
		if wire != WireStartGroup && wire != p.WireType {
			if wire == WireBytes && p.packedDec != nil {
				// a packable field
				dec = p.packedDec
			} else {
				err = ErrWrongType
				continue
			}
		}
		err = dec(o, p, base, sbase)
		if err == nil && p.Required {
			// Successfully decoded a required field.
			if tag <= 64 {
				// use bitmap for fields 1-64 to catch field reuse.
				var mask uint64 = 1 << uint64(tag-1)
				if reqFields&mask == 0 {
					// new required field
					reqFields |= mask
					required--
				}
			} else {
				// This is imprecise. It can be fooled by a required field
				// with a tag > 64 that is encoded twice; that's very rare.
				// A fully correct implementation would require allocating
				// a data structure, which we would like to avoid.
				required--
			}
		}
	}
	if err == nil {
		if is_group {
			return io.ErrUnexpectedEOF
		}
		if required > 0 {
			return &ErrRequiredNotSet{st}
		}
	}
	return err
}

// Make *pslice have base address base, length 0, and capacity startSize.
func initSlice(pslice unsafe.Pointer, base uintptr) {
	sp := (*reflect.SliceHeader)(pslice)
	sp.Data = base
	sp.Len = 0
	sp.Cap = startSize
}

// Individual type decoders
// For each,
//	u is the decoded value,
//	v is a pointer to the field (pointer) in the struct
//	x is a pointer to the preallocated scratch space to hold the decoded value.

// Decode a bool.
func (o *Buffer) dec_bool(p *Properties, base uintptr, sbase uintptr) os.Error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	v := (**uint8)(unsafe.Pointer(base + p.offset))
	x := (*uint8)(unsafe.Pointer(sbase + p.scratch))
	*x = uint8(u)
	*v = x
	return nil
}

// Decode an int32.
func (o *Buffer) dec_int32(p *Properties, base uintptr, sbase uintptr) os.Error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	v := (**int32)(unsafe.Pointer(base + p.offset))
	x := (*int32)(unsafe.Pointer(sbase + p.scratch))
	*x = int32(u)
	*v = x
	return nil
}

// Decode an int64.
func (o *Buffer) dec_int64(p *Properties, base uintptr, sbase uintptr) os.Error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	v := (**int64)(unsafe.Pointer(base + p.offset))
	x := (*int64)(unsafe.Pointer(sbase + p.scratch))
	*x = int64(u)
	*v = x
	return nil
}

// Decode a string.
func (o *Buffer) dec_string(p *Properties, base uintptr, sbase uintptr) os.Error {
	s, err := o.DecodeStringBytes()
	if err != nil {
		return err
	}
	v := (**string)(unsafe.Pointer(base + p.offset))
	x := (*string)(unsafe.Pointer(sbase + p.scratch))
	*x = s
	*v = x
	return nil
}

// Decode a slice of bytes ([]byte).
func (o *Buffer) dec_slice_byte(p *Properties, base uintptr, sbase uintptr) os.Error {
	b, err := o.DecodeRawBytes(false)
	if err != nil {
		return err
	}

	x := (*[]uint8)(unsafe.Pointer(base + p.offset))

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	*x = append(y, b...)
	return nil
}

// Decode a slice of bools ([]bool).
func (o *Buffer) dec_slice_bool(p *Properties, base uintptr, sbase uintptr) os.Error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := (*[]bool)(unsafe.Pointer(base + p.offset))

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	*x = append(y, u != 0)
	return nil
}

// Decode a slice of bools ([]bool) in packed format.
func (o *Buffer) dec_slice_packed_bool(p *Properties, base uintptr, sbase uintptr) os.Error {
	x := (*[]bool)(unsafe.Pointer(base + p.offset))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded bools

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	for i := 0; i < nb; i++ {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, u != 0)
	}

	*x = y
	return nil
}

// Decode a slice of int32s ([]int32).
func (o *Buffer) dec_slice_int32(p *Properties, base uintptr, sbase uintptr) os.Error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := (*[]int32)(unsafe.Pointer(base + p.offset))

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	*x = append(y, int32(u))
	return nil
}

// Decode a slice of int32s ([]int32) in packed format.
func (o *Buffer) dec_slice_packed_int32(p *Properties, base uintptr, sbase uintptr) os.Error {
	x := (*[]int32)(unsafe.Pointer(base + p.offset))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded int32s

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	fin := o.index + nb
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, int32(u))
	}

	*x = y
	return nil
}

// Decode a slice of int64s ([]int64).
func (o *Buffer) dec_slice_int64(p *Properties, base uintptr, sbase uintptr) os.Error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := (*[]int64)(unsafe.Pointer(base + p.offset))

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	*x = append(y, int64(u))
	return nil
}

// Decode a slice of int64s ([]int64) in packed format.
func (o *Buffer) dec_slice_packed_int64(p *Properties, base uintptr, sbase uintptr) os.Error {
	x := (*[]int64)(unsafe.Pointer(base + p.offset))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded int64s

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	fin := o.index + nb
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, int64(u))
	}

	*x = y
	return nil
}

// Decode a slice of strings ([]string).
func (o *Buffer) dec_slice_string(p *Properties, base uintptr, sbase uintptr) os.Error {
	s, err := o.DecodeStringBytes()
	if err != nil {
		return err
	}
	x := (*[]string)(unsafe.Pointer(base + p.offset))

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	*x = append(y, s)
	return nil
}

// Decode a slice of slice of bytes ([][]byte).
func (o *Buffer) dec_slice_slice_byte(p *Properties, base uintptr, sbase uintptr) os.Error {
	b, err := o.DecodeRawBytes(true)
	if err != nil {
		return err
	}
	x := (*[][]byte)(unsafe.Pointer(base + p.offset))

	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	*x = append(y, b)
	return nil
}

// Decode a group.
func (o *Buffer) dec_struct_group(p *Properties, base uintptr, sbase uintptr) os.Error {
	ptr := (**struct{})(unsafe.Pointer(base + p.offset))
	typ := p.stype.Elem()
	structv := unsafe.New(typ)
	bas := uintptr(structv)
	*ptr = (*struct{})(structv)

	err := o.unmarshalType(p.stype, true, bas)

	return err
}

// Decode an embedded message.
func (o *Buffer) dec_struct_message(p *Properties, base uintptr, sbase uintptr) (err os.Error) {
	raw, e := o.DecodeRawBytes(false)
	if e != nil {
		return e
	}

	ptr := (**struct{})(unsafe.Pointer(base + p.offset))
	typ := p.stype.Elem()
	structv := unsafe.New(typ)
	bas := uintptr(structv)
	*ptr = (*struct{})(structv)

	// If the object can unmarshal itself, let it.
	iv := unsafe.Unreflect(p.stype, unsafe.Pointer(ptr))
	if u, ok := iv.(Unmarshaler); ok {
		return u.Unmarshal(raw)
	}

	obuf := o.buf
	oi := o.index
	o.buf = raw
	o.index = 0

	err = o.unmarshalType(p.stype, false, bas)
	o.buf = obuf
	o.index = oi

	return err
}

// Decode a slice of embedded messages.
func (o *Buffer) dec_slice_struct_message(p *Properties, base uintptr, sbase uintptr) os.Error {
	return o.dec_slice_struct(p, false, base, sbase)
}

// Decode a slice of embedded groups.
func (o *Buffer) dec_slice_struct_group(p *Properties, base uintptr, sbase uintptr) os.Error {
	return o.dec_slice_struct(p, true, base, sbase)
}

// Decode a slice of structs ([]*struct).
func (o *Buffer) dec_slice_struct(p *Properties, is_group bool, base uintptr, sbase uintptr) os.Error {

	x := (*[]*struct{})(unsafe.Pointer(base + p.offset))
	y := *x
	if cap(y) == 0 {
		initSlice(unsafe.Pointer(x), sbase+p.scratch)
		y = *x
	}

	typ := p.stype.Elem()
	structv := unsafe.New(typ)
	bas := uintptr(structv)
	y = append(y, (*struct{})(structv))
	*x = y

	if is_group {
		err := o.unmarshalType(p.stype, is_group, bas)
		return err
	}

	raw, err := o.DecodeRawBytes(true)
	if err != nil {
		return err
	}

	// If the object can unmarshal itself, let it.
	iv := unsafe.Unreflect(p.stype, unsafe.Pointer(&y[len(y)-1]))
	if u, ok := iv.(Unmarshaler); ok {
		return u.Unmarshal(raw)
	}

	obuf := o.buf
	oi := o.index
	o.buf = raw
	o.index = 0

	err = o.unmarshalType(p.stype, is_group, bas)

	o.buf = obuf
	o.index = oi

	return err
}
