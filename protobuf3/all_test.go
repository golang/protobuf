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

package protobuf3_test

import (
	"bytes"
	"fmt"
	"testing"

	. "github.com/mistsys/protobuf3/protobuf3"
)

var globalO *Buffer

func old() *Buffer {
	if globalO == nil {
		globalO = NewBuffer(nil)
	}
	globalO.Reset()
	return globalO
}

func equalbytes(b1, b2 []byte, t *testing.T) {
	if len(b1) != len(b2) {
		t.Errorf("wrong lengths: 2*%d != %d", len(b1), len(b2))
		return
	}
	for i := 0; i < len(b1); i++ {
		if b1[i] != b2[i] {
			t.Errorf("bad byte[%d]:%x %x: %s %s", i, b1[i], b2[i], b1, b2)
		}
	}
}

func fail(msg string, b *bytes.Buffer, s string, t *testing.T) {
	data := b.Bytes()
	ld := len(data)
	ls := len(s) / 2

	fmt.Printf("fail %s ld=%d ls=%d\n", msg, ld, ls)

	// find the interesting spot - n
	n := ls
	if ld < ls {
		n = ld
	}
	j := 0
	for i := 0; i < n; i++ {
		bs := hex(s[j])*16 + hex(s[j+1])
		j += 2
		if data[i] == bs {
			continue
		}
		n = i
		break
	}
	l := n - 10
	if l < 0 {
		l = 0
	}
	h := n + 10

	// find the interesting spot - n
	fmt.Printf("is[%d]:", l)
	for i := l; i < h; i++ {
		if i >= ld {
			fmt.Printf(" --")
			continue
		}
		fmt.Printf(" %.2x", data[i])
	}
	fmt.Printf("\n")

	fmt.Printf("sb[%d]:", l)
	for i := l; i < h; i++ {
		if i >= ls {
			fmt.Printf(" --")
			continue
		}
		bs := hex(s[j])*16 + hex(s[j+1])
		j += 2
		fmt.Printf(" %.2x", bs)
	}
	fmt.Printf("\n")

	t.Fail()
}

func hex(c uint8) uint8 {
	if '0' <= c && c <= '9' {
		return c - '0'
	}
	if 'a' <= c && c <= 'f' {
		return 10 + c - 'a'
	}
	if 'A' <= c && c <= 'F' {
		return 10 + c - 'A'
	}
	return 0
}

func equal(b []byte, s string, t *testing.T) bool {
	if 2*len(b) != len(s) {
		//		fail(fmt.Sprintf("wrong lengths: 2*%d != %d", len(b), len(s)), b, s, t)
		fmt.Printf("wrong lengths: 2*%d != %d\n", len(b), len(s))
		return false
	}
	for i, j := 0, 0; i < len(b); i, j = i+1, j+2 {
		x := hex(s[j])*16 + hex(s[j+1])
		if b[i] != x {
			//			fail(fmt.Sprintf("bad byte[%d]:%x %x", i, b[i], x), b, s, t)
			fmt.Printf("bad byte[%d]:%x %x", i, b[i], x)
			return false
		}
	}
	return true
}

// Simple tests for numeric encode/decode primitives (varint, etc.)
func TestNumericPrimitives(t *testing.T) {
	for i := uint64(0); i < 1e6; i += 111 {
		o := old()
		o.EncodeVarint(i)
		x, e := o.DecodeVarint()
		if e != nil {
			t.Fatal("DecodeVarint")
		}
		if x != i {
			t.Fatal("varint decode fail:", i, x)
		}

		o = old()
		o.EncodeFixed32(i)
		x, e = o.DecodeFixed32()
		if e != nil {
			t.Fatal("decFixed32")
		}
		if x != i {
			t.Fatal("fixed32 decode fail:", i, x)
		}

		o = old()
		o.EncodeFixed64(i * 1234567)
		x, e = o.DecodeFixed64()
		if e != nil {
			t.Error("decFixed64")
			break
		}
		if x != i*1234567 {
			t.Error("fixed64 decode fail:", i*1234567, x)
			break
		}

		o = old()
		i32 := int32(i - 12345)
		o.EncodeZigzag32(uint64(i32))
		x, e = o.DecodeZigzag32()
		if e != nil {
			t.Fatal("DecodeZigzag32")
		}
		if x != uint64(uint32(i32)) {
			t.Fatal("zigzag32 decode fail:", i32, x)
		}

		o = old()
		i64 := int64(i - 12345)
		o.EncodeZigzag64(uint64(i64))
		x, e = o.DecodeZigzag64()
		if e != nil {
			t.Fatal("DecodeZigzag64")
		}
		if x != uint64(i64) {
			t.Fatal("zigzag64 decode fail:", i64, x)
		}
	}
}

// Simple tests for bytes
func TestBytesPrimitives(t *testing.T) {
	o := old()
	bytes := []byte{'n', 'o', 'w', ' ', 'i', 's', ' ', 't', 'h', 'e', ' ', 't', 'i', 'm', 'e'}
	o.EncodeRawBytes(bytes)
	decb, e := o.DecodeRawBytes(false)
	if e != nil {
		t.Error("DecodeRawBytes")
	}
	equalbytes(bytes, decb, t)
}

// Simple tests for strings
func TestStringPrimitives(t *testing.T) {
	o := old()
	s := "now is the time"
	o.EncodeStringBytes(s)
	decs, e := o.DecodeRawBytes(true)
	if e != nil {
		t.Error("dec_string")
	}
	if s != string(decs) {
		t.Error("string encode/decode fail:", s, decs)
	}
}
