// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !purego,!appengine

package prototype

import (
	"strings"
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

// Append is equivalent to protoreflect.FullName.Append, but is optimized for
// large batches of operations where each name has a shared lifetime.
func (b *nameBuilder) Append(prefix pref.FullName, name pref.Name) pref.FullName {
	const batchSize = 1 << 12
	n := len(prefix) + len(".") + len(name)
	if b.sb.Cap()-b.sb.Len() < n {
		b.sb.Reset()
		b.sb.Grow(batchSize)
	}
	if !strings.HasSuffix(b.sb.String(), string(prefix)) {
		b.sb.WriteString(string(prefix))
	}
	b.sb.WriteByte('.')
	b.sb.WriteString(string(name))
	s := b.sb.String()
	return pref.FullName(strings.TrimPrefix(s[len(s)-n:], "."))
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
