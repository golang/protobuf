// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build purego appengine

package impl

import (
	"google.golang.org/protobuf/internal/encoding/wire"
)

func sizeEnum(p pointer, tagsize int, _ marshalOptions) (size int) {
	v := p.v.Elem().Int()
	return tagsize + wire.SizeVarint(uint64(v))
}

func appendEnum(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	v := p.v.Elem().Int()
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(v))
	return b, nil
}

var coderEnum = pointerCoderFuncs{sizeEnum, appendEnum}

func sizeEnumNoZero(p pointer, tagsize int, opts marshalOptions) (size int) {
	if p.v.Elem().Int() == 0 {
		return 0
	}
	return sizeEnum(p, tagsize, opts)
}

func appendEnumNoZero(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	if p.v.Elem().Int() == 0 {
		return b, nil
	}
	return appendEnum(b, p, wiretag, opts)
}

var coderEnumNoZero = pointerCoderFuncs{sizeEnumNoZero, appendEnumNoZero}

func sizeEnumPtr(p pointer, tagsize int, opts marshalOptions) (size int) {
	return sizeEnum(pointer{p.v.Elem()}, tagsize, opts)
}

func appendEnumPtr(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	return appendEnum(b, pointer{p.v.Elem()}, wiretag, opts)
}

var coderEnumPtr = pointerCoderFuncs{sizeEnumPtr, appendEnumPtr}

func sizeEnumSlice(p pointer, tagsize int, opts marshalOptions) (size int) {
	return sizeEnumSliceReflect(p.v.Elem(), tagsize, opts)
}

func appendEnumSlice(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	return appendEnumSliceReflect(b, p.v.Elem(), wiretag, opts)
}

var coderEnumSlice = pointerCoderFuncs{sizeEnumSlice, appendEnumSlice}

func sizeEnumPackedSlice(p pointer, tagsize int, _ marshalOptions) (size int) {
	s := p.v.Elem()
	slen := s.Len()
	if slen == 0 {
		return 0
	}
	n := 0
	for i := 0; i < slen; i++ {
		n += wire.SizeVarint(uint64(s.Index(i).Int()))
	}
	return tagsize + wire.SizeBytes(n)
}

func appendEnumPackedSlice(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
	s := p.v.Elem()
	slen := s.Len()
	if slen == 0 {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	n := 0
	for i := 0; i < slen; i++ {
		n += wire.SizeVarint(uint64(s.Index(i).Int()))
	}
	b = wire.AppendVarint(b, uint64(n))
	for i := 0; i < slen; i++ {
		b = wire.AppendVarint(b, uint64(s.Index(i).Int()))
	}
	return b, nil
}

var coderEnumPackedSlice = pointerCoderFuncs{sizeEnumPackedSlice, appendEnumPackedSlice}
