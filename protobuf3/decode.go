// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2016 Mist Systems. All rights reserved.
//
// This code is derived from earlier code which was itself:
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
 * Routines for decoding protocol buffer data to construct in-memory representations.
 */

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"time"
	"unsafe"
)

// errOverflow is returned when an integer is too large to be represented.
var errOverflow = errors.New("protobuf3: integer overflow")

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
	for shift := uint(0); shift < 64; shift += 7 {
		if n >= len(buf) {
			return 0, 0
		}
		b := uint64(buf[n])
		n++
		x |= (b & 0x7F) << shift
		if (b & 0x80) == 0 {
			return x, n
		}
	}

	// The number is too large to represent in a 64-bit value.
	return 0, 0
}

// DecodeVarint reads a varint-encoded integer from the Buffer.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func (p *Buffer) decodeVarintSlow() (x uint64, err error) {
	// x, err already 0

	i := p.index
	l := len(p.buf)

	for shift := uint(0); shift < 64; shift += 7 {
		if i >= l {
			err = io.ErrUnexpectedEOF
			return
		}
		b := p.buf[i]
		i++
		x |= (uint64(b) & 0x7F) << shift
		if b < 0x80 {
			p.index = i
			return
		}
	}

	// The number is too large to represent in a 64-bit value.
	err = errOverflow
	return
}

// DecodeVarint reads a varint-encoded integer from the Buffer.
// This is the format for the
// int32, int64, uint32, uint64, bool, and enum
// protocol buffer types.
func (p *Buffer) DecodeVarint() (x uint64, err error) {
	i := p.index
	buf := p.buf

	if i >= len(buf) {
		return 0, io.ErrUnexpectedEOF
	} else if buf[i] < 0x80 {
		p.index++
		return uint64(buf[i]), nil
	} else if len(buf)-i < 10 {
		return p.decodeVarintSlow()
	}

	var b uint64

	// we already checked the first byte
	x = uint64(buf[i]) - 0x80
	i++

	b = uint64(buf[i])
	i++
	x += b << 7
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 7

	b = uint64(buf[i])
	i++
	x += b << 14
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 14

	b = uint64(buf[i])
	i++
	x += b << 21
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 21

	b = uint64(buf[i])
	i++
	x += b << 28
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 28

	b = uint64(buf[i])
	i++
	x += b << 35
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 35

	b = uint64(buf[i])
	i++
	x += b << 42
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 42

	b = uint64(buf[i])
	i++
	x += b << 49
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 49

	b = uint64(buf[i])
	i++
	x += b << 56
	if b&0x80 == 0 {
		goto done
	}
	x -= 0x80 << 56

	b = uint64(buf[i])
	i++
	x += b << 63
	if b&0x80 == 0 {
		goto done
	}
	// x -= 0x80 << 63 // Always zero.

	return 0, errOverflow

done:
	p.index = i
	return x, nil
}

// DecodeFixed64 reads a 64-bit integer from the Buffer.
// This is the format for the
// fixed64, sfixed64, and double protocol buffer types.
func (p *Buffer) DecodeFixed64() (x uint64, err error) {
	// x, err already 0
	i := p.index + 8
	if i < 0 || i > len(p.buf) {
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
func (p *Buffer) DecodeFixed32() (x uint64, err error) {
	// x, err already 0
	i := p.index + 4
	if i < 0 || i > len(p.buf) {
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
func (p *Buffer) DecodeZigzag64() (x uint64, err error) {
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
func (p *Buffer) DecodeZigzag32() (x uint64, err error) {
	x, err = p.DecodeVarint()
	if err != nil {
		return
	}
	x = uint64((uint32(x) >> 1) ^ uint32(((int32(x)&1)<<31)>>31))
	return
}

// These are not ValueDecoders: they produce an array of bytes or a string.
// bytes, embedded messages

// DecodeRawBytes reads a count-delimited byte buffer from the Buffer.
// This is the format used for the bytes protocol buffer
// type and for embedded messages.
func (p *Buffer) DecodeRawBytes(alloc bool) (buf []byte, err error) {
	n, err := p.DecodeVarint()
	if err != nil {
		return nil, err
	}

	nb := int(n)
	if nb < 0 || uint64(nb) != n {
		return nil, fmt.Errorf("protobuf3: bad byte length %d", n)
	}
	end := p.index + nb
	if end < p.index || end > len(p.buf) {
		return nil, io.ErrUnexpectedEOF
	}

	if !alloc {
		// todo: check if can get more uses of alloc=false
		buf = p.buf[p.index:end:end]
		p.index = end
		return
	}

	buf = make([]byte, nb)
	copy(buf, p.buf[p.index:])
	p.index = end
	return
}

// DecodeStringBytes reads an encoded string from the Buffer.
// This is the format used for the proto3 string type.
func (p *Buffer) DecodeStringBytes() (string, error) {
	buf, err := p.DecodeRawBytes(false)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// SkipVarint skips over a varint-encoded integer from the Buffer.
// Functionally it is identical to calling DecodeVarint and ignoring the
// value returned. In practice it runs much faster.
func (p *Buffer) SkipVarint() error {
	i := p.index
	l := len(p.buf)

	for shift := uint(0); shift < 64; shift += 7 {
		if i >= l {
			return io.ErrUnexpectedEOF
		}
		b := p.buf[i]
		i++
		if b < 0x80 {
			p.index = i
			return nil
		}
	}

	// The number is too large to represent in a 64-bit value.
	return errOverflow
}

// SkipFixed skips over n bytes. Useful for skipping over Fixed32 and Fixed64 with proper arguments
func (p *Buffer) SkipFixed(n int) error {
	p.index += n
	if p.index > len(p.buf) {
		p.index = len(p.buf)
		return io.ErrUnexpectedEOF
	}
	return nil
}

// SkipRawBytes skips over a count-delimited byte buffer from the Buffer.
// Functionally it is identical to calling DecodeRawBytes(false) and ignoring
// the value returned.
func (p *Buffer) SkipRawBytes() error {
	n, err := p.DecodeVarint()
	if err != nil {
		return err
	}

	nb := int(n)
	if nb < 0 || uint64(nb) != n {
		return fmt.Errorf("protobuf3: bad byte length %d", n)
	}
	end := p.index + nb
	if end < p.index || end > len(p.buf) {
		return io.ErrUnexpectedEOF
	}

	p.index = end
	return nil
}

// Unmarshaler is the interface representing objects that can
// unmarshal themselves.  The method should reset the receiver before
// decoding starts.  The argument points to data that may be
// overwritten, so implementations should not keep references to the
// buffer.
type Unmarshaler interface {
	UnmarshalProtobuf3([]byte) error
}

// Unmarshal parses the protocol buffer representation in buf and
// writes the decoded result to pb.  If the struct underlying pb does not match
// the data in buf, the results can be unpredictable.
//
// Unmarshal merges into existing data in pb. If that's not what you wanted then
// you ought to zero pb before calling Unmarshal. NOTE WELL this differs from the
// behavior of the golang/proto.Unmarshal(), but matches the standard go encoding/json.Unmarshal()
// Since we're used to json, and since having the caller do the zeroing is more efficient
// (both because they know the type (making it more efficient for the CPU), and it avoids forcing
// everyone to define a Reset() method for the Message interface (making it more efficient for
// the developer, me!)), our Unmarshal() matches the behavior of encoding/json.Unmarshal()
func Unmarshal(buf []byte, pb Message) error {
	typ := reflect.TypeOf(pb)
	// pb must be a pointer type, since it must be addressable so we can fill it in
	if typ.Kind() != reflect.Ptr {
		return ErrNotAddressable
	}
	return NewBuffer(buf).Unmarshal(pb)
}

// Unmarshal parses the protocol buffer representation in the
// Buffer and places the decoded result in pb.  If the struct
// underlying pb does not match the data in the buffer, the results can be
// unpredictable.
func (p *Buffer) Unmarshal(pb Message) error {
	if pb == nil { // we need a non-nil interface or this won't work
		return ErrNil // NOTE this could almost qualify for a panic(), because the calling code is clearly quite confused
	}

	// If the object can unmarshal itself, let it.
	if u, ok := pb.(Unmarshaler); ok {
		err := u.UnmarshalProtobuf3(p.buf[p.index:])
		p.index = len(p.buf)
		return err
	}

	// the caller already checked that pb is a pointer-to-struct type
	t := reflect.TypeOf(pb).Elem()
	base := structPointer(unsafe.Pointer(reflect.ValueOf(pb).Pointer()))

	err := p.unmarshalType(t, GetProperties(t), base)

	return err
}

// unmarshalType does the work of unmarshaling a structure.
func (o *Buffer) unmarshalType(st reflect.Type, prop *StructProperties, base structPointer) error {
	var err error
	for err == nil && o.index < len(o.buf) {
		var u uint64
		u, err = o.DecodeVarint()
		if err != nil {
			break
		}
		wire := WireType(u & 0x7)
		tag := int(u >> 3)
		if tag <= 0 {
			return fmt.Errorf("proto: %s: illegal tag %d (wire type %d)", st, tag, wire)
		}
		i := sort.Search(len(prop.order), func(i int) bool {
			return prop.Prop[prop.order[i]].Tag >= uint32(tag)
		})
		if i >= len(prop.order) || prop.Prop[prop.order[i]].Tag != uint32(tag) {
			err = o.skip(st, wire)
			continue
		}
		fieldnum := prop.order[i]
		p := &prop.Prop[fieldnum]

		if p.dec == nil {
			fmt.Fprintf(os.Stderr, "proto: no protobuf decoder for %s.%s\n", st, st.Field(fieldnum).Name)
			continue
		}
		dec := p.dec
		if wire != p.WireType && wire != WireBytes { // packed encoding, which is used in protobuf v3, wraps repeated numeric types in WireBytes
			err = fmt.Errorf("proto: bad wiretype for field %s.%s: got wiretype %d, want %d", st, st.Field(fieldnum).Name, wire, p.WireType)
			continue
		}
		decErr := dec(o, p, base)
		if decErr != nil {
			err = decErr
		}
	}
	return err
}

// Skip the next item in the buffer. Its wire type is decoded and presented as an argument.
func (o *Buffer) skip(t reflect.Type, wire WireType) error {
	var err error

	switch wire {
	case WireVarint:
		err = o.SkipVarint()
	case WireBytes:
		err = o.SkipRawBytes()
	case WireFixed64:
		err = o.SkipFixed(8)
	case WireFixed32:
		err = o.SkipFixed(4)
	default:
		err = fmt.Errorf("proto: can't skip unknown wire type %d for %v", wire, t)
	}
	return err
}

// Individual type decoders
// For each,
//	u is the decoded value,
//	v is a pointer to the field (pointer) in the struct

// Decode a *bool.
func (o *Buffer) dec_ptr_bool(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := u != 0
	*(**bool)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &x
	return nil
}

// Decode a bool.
func (o *Buffer) dec_bool(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	*(*bool)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = u != 0
	return nil
}

// Decode an *int8.
func (o *Buffer) dec_ptr_int8(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := uint8(u)
	*(**uint8)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &x
	return nil
}

// Decode an int8.
func (o *Buffer) dec_int8(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	*(*uint8)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = uint8(u)
	return nil
}

// Decode an *int16.
func (o *Buffer) dec_ptr_int16(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := uint16(u)
	*(**uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &x
	return nil
}

// Decode an int16.
func (o *Buffer) dec_int16(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	*(*uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = uint16(u)
	return nil
}

// Decode an *int32.
func (o *Buffer) dec_ptr_int32(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := uint32(u)
	*(**uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &x
	return nil
}

// Decode an int32.
func (o *Buffer) dec_int32(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	*(*uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = uint32(u)
	return nil
}

// Decode an *int.
func (o *Buffer) dec_ptr_int(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	x := uint(u)
	*(**uint)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &x
	return nil
}

// Decode an int.
func (o *Buffer) dec_int(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	*(*uint)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = uint(u)
	return nil
}

// Decode an *int64.
func (o *Buffer) dec_ptr_int64(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	*(**uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &u
	return nil
}

// Decode an int64.
func (o *Buffer) dec_int64(p *Properties, base structPointer) error {
	u, err := p.valDec(o)
	if err != nil {
		return err
	}
	*(*uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = u
	return nil
}

// Decode a *string.
func (o *Buffer) dec_ptr_string(p *Properties, base structPointer) error {
	s, err := o.DecodeStringBytes()
	if err != nil {
		return err
	}
	*(**string)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &s
	return nil
}

// Decode a string.
func (o *Buffer) dec_string(p *Properties, base structPointer) error {
	s, err := o.DecodeStringBytes()
	if err != nil {
		return err
	}
	*(*string)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = s
	return nil
}

// custom decoder for protobuf3 standard Timestamp, decoding it into the standard go time.Time
func (o *Buffer) dec_time_Time(p *Properties, base structPointer) error {
	var secs, nanos uint64
	for {
		tag, err := o.DecodeVarint()
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				break
			}
			return err
		}
		switch tag {
		case 1<<3 | uint64(WireVarint): // seconds
			secs, err = o.DecodeVarint()
		case 2<<3 | uint64(WireVarint): // nanoseconds
			nanos, err = o.DecodeVarint()
		default:
			// do the protobuf thing and ignore unknown tags
		}
		if err != nil {
			return err
		}
	}

	t := time.Unix(int64(secs), int64(nanos))

	*(*time.Time)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = t
	return nil
}

// custom decoder for protobuf3 standard Duration, decoding it into the go standard time.Duration
func (o *Buffer) dec_time_Duration(p *Properties, base structPointer) error {
	d, err := o.dec_Duration(p)
	if err != nil {
		return err
	}
	*(*time.Duration)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = d
	return nil
}

// helper function to decode a protobuf3 Duration value into a time.Duration
func (o *Buffer) dec_Duration(p *Properties) (time.Duration, error) {
	// time.Duration is not a struct. it is a int64. So it does not translate
	// readily to a protobuf message. We had to prepend the tag and length,
	// and here we need to remove it.
	tag, err := o.DecodeVarint()
	if err != nil {
		return 0, err
	}
	// sanity check that the tag's wiretype is bytes
	if WireType(tag&7) != WireBytes {
		return 0, fmt.Errorf("protobuf3: Wiretype not Bytes when decoding Duration tag 0x%x", tag)
	}
	n, err := o.DecodeVarint()
	if err != nil {
		return 0, err
	}
	end := o.index + int(n)

	if end < o.index || end > len(o.buf) {
		return 0, io.ErrUnexpectedEOF
	}

	// restrict ourselves to p.index:end
	oo := NewBuffer(o.buf[o.index:end:end])

	var secs, nanos uint64
	for oo.index < len(oo.buf) {
		tag, err := oo.DecodeVarint()
		if err != nil {
			return 0, err
		}
		switch tag {
		case 1<<3 | uint64(WireVarint): // seconds
			secs, err = oo.DecodeVarint()
		case 2<<3 | uint64(WireVarint): // nanoseconds
			nanos, err = oo.DecodeVarint()
		default:
			// do the protobuf thing and ignore unknown tags
			oo.skip(nil, WireType(tag)&7)
		}
		if err != nil {
			return 0, err
		}
	}

	o.index = end

	d := time.Duration(secs)*time.Second + time.Duration(nanos)*time.Nanosecond

	return d, nil
}

// custom encoder for *time.Duration, ... protobuf Duration message
func (o *Buffer) dec_ptr_time_Duration(p *Properties, base structPointer) error {
	d, err := o.dec_Duration(p)
	if err != nil {
		return err
	}
	*(**time.Duration)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &d
	return nil
}
