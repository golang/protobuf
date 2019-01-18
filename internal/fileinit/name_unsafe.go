// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !purego,!appengine

package fileinit

import (
	"sync"
	"unsafe"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

var nameBuilderPool = sync.Pool{
	New: func() interface{} { return new(nameBuilder) },
}

func getNameBuilder() *nameBuilder {
	return nameBuilderPool.Get().(*nameBuilder)
}
func putNameBuilder(b *nameBuilder) {
	nameBuilderPool.Put(b)
}

type nameBuilder struct {
	sb stringBuilder
}

// AppendFullName is equivalent to protoreflect.FullName.Append,
// but optimized for large batches where each name has a shared lifetime.
func (nb *nameBuilder) AppendFullName(prefix pref.FullName, name []byte) fullName {
	n := len(prefix) + len(".") + len(name)
	if len(prefix) == 0 {
		n -= len(".")
	}
	nb.grow(n)
	nb.sb.WriteString(string(prefix))
	nb.sb.WriteByte('.')
	nb.sb.Write(name)
	return fullName{
		shortLen: len(name),
		fullName: pref.FullName(nb.last(n)),
	}
}

// MakeString is equivalent to string(b), but optimized for large batches
// with a shared lifetime.
func (nb *nameBuilder) MakeString(b []byte) string {
	nb.grow(len(b))
	nb.sb.Write(b)
	return nb.last(len(b))
}

// MakeJSONName creates a JSON name from the protobuf short name.
func (nb *nameBuilder) MakeJSONName(s pref.Name) string {
	nb.grow(len(s))
	var n int
	var wasUnderscore bool
	for i := 0; i < len(s); i++ { // proto identifiers are always ASCII
		c := s[i]
		if c != '_' {
			isLower := 'a' <= c && c <= 'z'
			if wasUnderscore && isLower {
				c -= 'a' - 'A'
			}
			nb.sb.WriteByte(c)
			n++
		}
		wasUnderscore = c == '_'
	}
	return nb.last(n)
}

func (nb *nameBuilder) last(n int) string {
	s := nb.sb.String()
	return s[len(s)-n:]
}

func (nb *nameBuilder) grow(n int) {
	const batchSize = 1 << 16
	if nb.sb.Cap()-nb.sb.Len() < n {
		nb.sb.Reset()
		nb.sb.Grow(batchSize)
	}
}

// stringsBuilder is a simplified copy of the strings.Builder from Go1.12:
//	* removed the shallow copy check
//	* removed methods that we do not use (e.g. WriteRune)
//
// A forked version is used:
//	* to enable Go1.9 support, but strings.Builder was added in Go1.10
//	* for the Cap method, which was missing until Go1.12
//
// TODO: Remove this when Go1.12 is the minimally supported toolchain version.
type stringBuilder struct {
	buf []byte
}

func (b *stringBuilder) String() string {
	return *(*string)(unsafe.Pointer(&b.buf))
}
func (b *stringBuilder) Len() int {
	return len(b.buf)
}
func (b *stringBuilder) Cap() int {
	return cap(b.buf)
}
func (b *stringBuilder) Reset() {
	b.buf = nil
}
func (b *stringBuilder) grow(n int) {
	buf := make([]byte, len(b.buf), 2*cap(b.buf)+n)
	copy(buf, b.buf)
	b.buf = buf
}
func (b *stringBuilder) Grow(n int) {
	if n < 0 {
		panic("stringBuilder.Grow: negative count")
	}
	if cap(b.buf)-len(b.buf) < n {
		b.grow(n)
	}
}
func (b *stringBuilder) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}
func (b *stringBuilder) WriteByte(c byte) error {
	b.buf = append(b.buf, c)
	return nil
}
func (b *stringBuilder) WriteString(s string) (int, error) {
	b.buf = append(b.buf, s...)
	return len(s), nil
}
