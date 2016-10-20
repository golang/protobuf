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
// The returned slice points to shared memory. Treat as read-only.
func (p *Buffer) DecodeRawBytes() ([]byte, error) {
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

	buf := p.buf[p.index:end:end]
	p.index = end

	return buf, nil
}

// DecodeStringBytes reads an encoded string from the Buffer.
// This is the format used for the proto3 string type.
func (p *Buffer) DecodeStringBytes() (string, error) {
	buf, err := p.DecodeRawBytes()
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
// Functionally it is identical to calling DecodeRawBytes() and ignoring
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
	if m, ok := pb.(Marshaler); ok {
		err := m.UnmarshalProtobuf3(p.buf[p.index:])
		p.index = len(p.buf)
		return err
	}

	// the caller already checked that pb is a pointer-to-struct type
	t := reflect.TypeOf(pb).Elem()
	base := structPointer(unsafe.Pointer(reflect.ValueOf(pb).Pointer()))

	err := p.unmarshal_struct(t, GetProperties(t), base)

	return err
}

// unmarshal_struct does the work of unmarshaling a structure.
func (o *Buffer) unmarshal_struct(st reflect.Type, prop *StructProperties, base structPointer) error {
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
			return fmt.Errorf("protobuf3: %s: illegal tag %d (wire type %d)", st, tag, wire)
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
			fmt.Fprintf(os.Stderr, "protobuf3: no protobuf decoder for %s.%s\n", st, st.Field(fieldnum).Name)
			continue
		}
		if wire != p.WireType && wire != WireBytes { // packed encoding, which is used in protobuf v3, wraps repeated numeric types in WireBytes
			err = fmt.Errorf("protobuf3: bad wiretype for field %s.%s: got wiretype %d, want %d", st, st.Field(fieldnum).Name, wire, p.WireType)
			break
		}
		err = p.dec(o, p, base)
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
		err = fmt.Errorf("protobuf3: can't skip unknown wire type %d for %v", wire, t)
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

// Decode a slice of bytes ([]byte).
func (o *Buffer) dec_slice_byte(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	copied := make([]byte, len(raw))
	copy(copied, raw)

	*(*[]byte)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = copied
	return nil
}

// Decode a slice of bools ([]bool).
func (o *Buffer) dec_slice_packed_bool(p *Properties, base structPointer) error {
	v := (*[]bool)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded bools
	fin := o.index + nb
	if fin < o.index {
		return errOverflow
	}

	y := *v
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, u != 0)
	}

	*v = y
	return nil
}

// Decode a slice of int8s ([]int8) in packed format.
func (o *Buffer) dec_slice_packed_int8(p *Properties, base structPointer) error {
	v := (*[]int8)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded int8s

	fin := o.index + nb
	if fin < o.index {
		return errOverflow
	}
	y := *v
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, int8(u))
	}
	*v = y
	return nil
}

// Decode a slice of int16s ([]int16) in packed format.
func (o *Buffer) dec_slice_packed_int16(p *Properties, base structPointer) error {
	v := (*[]uint16)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded int16s

	fin := o.index + nb
	if fin < o.index {
		return errOverflow
	}
	y := *v
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, uint16(u))
	}
	*v = y
	return nil
}

// Decode a slice of int32s ([]int32) in packed format.
func (o *Buffer) dec_slice_packed_int32(p *Properties, base structPointer) error {
	v := (*[]uint32)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded int32s

	fin := o.index + nb
	if fin < o.index {
		return errOverflow
	}
	y := *v
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, uint32(u))
	}
	*v = y
	return nil
}

// Decode a slice of ints ([]int) in packed format.
func (o *Buffer) dec_slice_packed_int(p *Properties, base structPointer) error {
	v := (*[]uint)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded ints

	fin := o.index + nb
	if fin < o.index {
		return errOverflow
	}
	y := *v
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, uint(u))
	}
	*v = y
	return nil
}

// Decode a slice of int64s ([]int64) in packed format.
func (o *Buffer) dec_slice_packed_int64(p *Properties, base structPointer) error {
	v := (*[]uint64)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded int64s

	fin := o.index + nb
	if fin < o.index {
		return errOverflow
	}
	y := *v
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, u)
	}
	*v = y
	return nil
}

// Decode a slice of strings ([]string).
func (o *Buffer) dec_slice_string(p *Properties, base structPointer) error {
	s, err := o.DecodeStringBytes()
	if err != nil {
		return err
	}
	v := (*[]string)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	*v = append(*v, s)
	return nil
}

// Decode a slice of slice of bytes ([][]byte).
func (o *Buffer) dec_slice_slice_byte(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	copied := make([]byte, len(raw))
	copy(copied, raw)

	v := (*[][]byte)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	*v = append(*v, copied)
	return nil
}

// Decode a map field.
func (o *Buffer) dec_new_map(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}
	oi := o.index       // index at the end of this map entry
	o.index -= len(raw) // move buffer back to start of map entry

	mptr := reflect.NewAt(p.mtype, unsafe.Pointer(uintptr(base)+uintptr(p.field))) // *map[K]V
	if mptr.Elem().IsNil() {
		mptr.Elem().Set(reflect.MakeMap(mptr.Type().Elem()))
	}
	v := mptr.Elem() // map[K]V

	// Prepare addressable doubly-indirect placeholders for the key and value types.
	// See enc_new_map for why.
	keyptr := reflect.New(reflect.PtrTo(p.mtype.Key())).Elem()    // addressable *K
	keybase := structPointer(unsafe.Pointer(keyptr.UnsafeAddr())) // **K

	var valbase structPointer
	var valptr reflect.Value
	switch p.mtype.Elem().Kind() {
	case reflect.Slice:
		// []byte
		var dummy []byte
		valptr = reflect.ValueOf(&dummy)                          // *[]byte
		valbase = structPointer(unsafe.Pointer(valptr.Pointer())) // *[]byte
	case reflect.Ptr:
		// message; valptr is **Msg; need to allocate the intermediate pointer
		valptr = reflect.New(reflect.PtrTo(p.mtype.Elem())).Elem() // addressable *V
		valptr.Set(reflect.New(valptr.Type().Elem()))
		valbase = structPointer(unsafe.Pointer(valptr.Pointer()))
	default:
		// everything else
		valptr = reflect.New(reflect.PtrTo(p.mtype.Elem())).Elem()   // addressable *V
		valbase = structPointer(unsafe.Pointer(valptr.UnsafeAddr())) // **V
	}

	// Decode.
	// This parses a restricted wire format, namely the encoding of a message
	// with two fields. See enc_new_map for the format.
	for o.index < oi {
		// tagcode for key and value properties are always a single byte
		// because they have tags 1 and 2.
		tagcode := o.buf[o.index]
		o.index++
		switch tagcode {
		case p.mkeyprop.tagcode[0]:
			if err := p.mkeyprop.dec(o, p.mkeyprop, keybase); err != nil {
				return err
			}
		case p.mvalprop.tagcode[0]:
			if err := p.mvalprop.dec(o, p.mvalprop, valbase); err != nil {
				return err
			}
		default:
			// TODO: Should we silently skip this instead?
			return fmt.Errorf("protobuf3: bad map data tag %d", raw[0])
		}
	}
	keyelem, valelem := keyptr.Elem(), valptr.Elem()
	if !keyelem.IsValid() {
		keyelem = reflect.Zero(p.mtype.Key())
	}
	if !valelem.IsValid() {
		valelem = reflect.Zero(p.mtype.Elem())
	}

	v.SetMapIndex(keyelem, valelem)
	return nil
}

// Decode an embedded message.
func (o *Buffer) dec_struct_message(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	ptr := structPointer(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	// swizzle around and reuse the buffer. less gc
	obuf, oi := o.buf, o.index
	o.buf, o.index = raw, 0

	err = o.unmarshal_struct(p.stype, p.sprop, ptr)

	o.buf, o.index = obuf, oi
	return err
}

// Decode a pointer to an embedded message.
func (o *Buffer) dec_ptr_struct_message(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	pptr := (*structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	ptr := *pptr
	var val reflect.Value
	if ptr == nil {
		val = reflect.New(p.stype)
		ptr := structPointer(val.Pointer()) // Is this gc safe? it seems not to be to me, but I don't have a better solution, and it's what google's code does
		*pptr = ptr
	} // else the value is already allocated and we merge into it

	// swizzle around and reuse the buffer. less gc
	obuf, oi := o.buf, o.index
	o.buf, o.index = raw, 0

	err = o.unmarshal_struct(p.stype, p.sprop, ptr)

	o.buf, o.index = obuf, oi
	return err
}

//  Decode into a slice of messages ([]struct)
func (o *Buffer) dec_slice_struct_message(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	// build a reflect.Value of the slice
	ptr := unsafe.Pointer(uintptr(base) + uintptr(p.field))
	slice_type := reflect.SliceOf(p.stype)
	slice := reflect.NewAt(slice_type, ptr)

	// put a zero value at the end of the slice
	slice.Set(reflect.Append(slice, reflect.Zero(p.stype)))

	// and unmarshal into it
	val := slice.Index(slice.Len() - 1)
	if p.isMarshaler {
		return val.Interface().(Marshaler).UnmarshalProtobuf3(raw)
	}

	pval := structPointer(unsafe.Pointer(val.UnsafeAddr()))

	// unmarshal into pval
	obuf, oi := o.buf, o.index
	o.buf, o.index = raw, 0
	err = o.unmarshal_struct(p.stype, p.sprop, pval)
	o.buf, o.index = obuf, oi

	return err
}

//  Decode into a slice of pointers to messages ([]*struct)
func (o *Buffer) dec_slice_ptr_struct_message(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	// construct a new *struct
	v := reflect.New(p.stype)
	pv := structPointer(unsafe.Pointer(v.Pointer()))

	// unmarshal into the new struct
	if p.isMarshaler {
		err = v.Interface().(Marshaler).UnmarshalProtobuf3(raw)
	} else {
		obuf, oi := o.buf, o.index
		o.buf, o.index = raw, 0
		err = o.unmarshal_struct(p.stype, p.sprop, pv)
		o.buf, o.index = obuf, oi
	}
	if err != nil {
		return err
	}

	// append pv to the slice []*struct
	pslice := (*[]structPointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	*pslice = append(*pslice, pv)

	return nil
}

// Decode an embedded message that can unmarshal itself
func (o *Buffer) dec_marshaler(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	ptr := unsafe.Pointer(uintptr(base) + uintptr(p.field))
	iv := reflect.NewAt(p.stype, ptr).Interface()
	return iv.(Marshaler).UnmarshalProtobuf3(raw)
}

// Decode a pointer to an embedded message that can unmarshal itself
func (o *Buffer) dec_ptr_marshaler(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	pptr := (*unsafe.Pointer)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	var val reflect.Value
	if *pptr == nil {
		val = reflect.New(p.stype)
		*pptr = unsafe.Pointer(val.Pointer()) // Is this gc safe? it seems not to be to me, but I don't have a better solution, and it's what google's code does
	} else {
		// else the value is already allocated and we merge into it
		val = reflect.NewAt(p.stype, *pptr)
	}
	return val.Interface().(Marshaler).UnmarshalProtobuf3(raw)
}

// Decode into slice of things which can marshal themselves
func (o *Buffer) dec_slice_marshaler(p *Properties, base structPointer) error {
	raw, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	// build a reflect.Value of the slice
	ptr := unsafe.Pointer(uintptr(base) + uintptr(p.field))
	slice_type := reflect.SliceOf(p.stype)
	slice := reflect.NewAt(slice_type, ptr)

	// put a zero value at the end of the slice
	slice.Set(reflect.Append(slice, reflect.Zero(p.stype)))

	// and unmarshal into it
	val := slice.Index(slice.Len() - 1)
	return val.Interface().(Marshaler).UnmarshalProtobuf3(raw)
}

// custom decoder for protobuf3 standard Timestamp, decoding it into the standard go time.Time
func (o *Buffer) dec_time_Time(p *Properties, base structPointer) error {
	return o.decode_time_Time((*time.Time)(unsafe.Pointer(uintptr(base) + uintptr(p.field))))
}

// custom decoder for pointer to time.Time
func (o *Buffer) dec_ptr_time_Time(p *Properties, base structPointer) error {
	pptr := (**time.Time)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))
	ptr := *pptr
	if ptr == nil {
		ptr = new(time.Time)
		*pptr = ptr
	} // else overwwrite the existing time.Time like the protobuf standard says to do
	return o.decode_time_Time(ptr)
}

// inner code for decoding protobuf3 standard Timestamp to time.Time
func (o *Buffer) decode_time_Time(t *time.Time) error {
	// first decode the byte length and limit our decoding to that (since messages are encoded in WireBytes)
	buf, err := o.DecodeRawBytes()
	if err != nil {
		return err
	}

	// swizzle buf (saves gc pressure from a new Buffer)
	obuf, oi := o.buf, o.index
	o.buf, o.index = buf, 0

	var secs, nanos uint64
	for o.index < len(o.buf) {
		tag, err := o.DecodeVarint()
		if err != nil {
			o.buf, o.index = obuf, oi
			return err
		}
		switch tag {
		case 1<<3 | uint64(WireVarint): // seconds
			secs, err = o.DecodeVarint()
		case 2<<3 | uint64(WireVarint): // nanoseconds
			nanos, err = o.DecodeVarint()
		default:
			// do the protobuf thing and ignore unknown tags
			o.skip(nil, WireType(tag)&7)
		}
		if err != nil {
			o.buf, o.index = obuf, oi
			return err
		}
	}

	// save whatever we got (which might even be the zero value)
	*t = time.Unix(int64(secs), int64(nanos)).UTC() // time.Unix() returns local timezone, which we usually don't use

	o.buf, o.index = obuf, oi
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
	// the tag has been decoded, but not the byte length
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

// custom decoder for *time.Duration, ... protobuf Duration message
func (o *Buffer) dec_ptr_time_Duration(p *Properties, base structPointer) error {
	d, err := o.dec_Duration(p)
	if err != nil {
		return err
	}
	*(**time.Duration)(unsafe.Pointer(uintptr(base) + uintptr(p.field))) = &d
	return nil
}

// custom decode for []time.Duration
func (o *Buffer) dec_slice_time_Duration(p *Properties, base structPointer) error {
	v := (*[]time.Duration)(unsafe.Pointer(uintptr(base) + uintptr(p.field)))

	nn, err := o.DecodeVarint()
	if err != nil {
		return err
	}
	nb := int(nn) // number of bytes of encoded values

	fin := o.index + nb
	if fin < o.index {
		return errOverflow
	}
	y := *v
	for o.index < fin {
		u, err := p.valDec(o)
		if err != nil {
			return err
		}
		y = append(y, time.Duration(u))
	}
	*v = y
	return nil
}
