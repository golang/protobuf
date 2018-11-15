// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"sync"
	"unsafe"

	protoV1 "github.com/golang/protobuf/proto"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// TODO: The logic in the file is a hack and should be in the v1 repository.
// We need to break the dependency on proto v1 since it is v1 that will
// eventually need to depend on v2.

// TODO: The v1 API currently exposes no exported functionality for interacting
// with the extension data structures. We will need to make changes in v1 so
// that v2 can access these data structures without relying on unsafe.

var (
	extTypeA = reflect.TypeOf(map[int32]protoV1.Extension(nil))
	extTypeB = reflect.TypeOf(protoV1.XXX_InternalExtensions{})
)

type legacyExtensionIface interface {
	Len() int
	Has(pref.FieldNumber) bool
	Get(pref.FieldNumber) legacyExtensionEntry
	Set(pref.FieldNumber, legacyExtensionEntry)
	Clear(pref.FieldNumber)
	Range(f func(pref.FieldNumber, legacyExtensionEntry) bool)
}

func makeLegacyExtensionMapFunc(t reflect.Type) func(*messageDataType) legacyExtensionIface {
	fx1, _ := t.FieldByName("XXX_extensions")
	fx2, _ := t.FieldByName("XXX_InternalExtensions")
	switch {
	case fx1.Type == extTypeA:
		return func(p *messageDataType) legacyExtensionIface {
			rv := p.p.asType(t).Elem()
			return (*legacyExtensionMap)(unsafe.Pointer(rv.UnsafeAddr() + fx1.Offset))
		}
	case fx2.Type == extTypeB:
		return func(p *messageDataType) legacyExtensionIface {
			rv := p.p.asType(t).Elem()
			return (*legacyExtensionSyncMap)(unsafe.Pointer(rv.UnsafeAddr() + fx2.Offset))
		}
	default:
		return nil
	}
}

// TODO: We currently don't do locking with legacyExtensionSyncMap.p.mu.
// The locking behavior was already obscure "feature" beforehand,
// and it is not obvious how it translates to the v2 API.
// The v2 API presents a Range method, which calls a user provided function,
// which may in turn call other methods on the map. In such a use case,
// acquiring a lock within each method would result in a reentrant deadlock.

// legacyExtensionSyncMap is identical to protoV1.XXX_InternalExtensions.
// It implements legacyExtensionIface.
type legacyExtensionSyncMap struct {
	p *struct {
		mu sync.Mutex
		m  legacyExtensionMap
	}
}

func (m legacyExtensionSyncMap) Len() int {
	if m.p == nil {
		return 0
	}
	return m.p.m.Len()
}
func (m legacyExtensionSyncMap) Has(n pref.FieldNumber) bool {
	return m.p.m.Has(n)
}
func (m legacyExtensionSyncMap) Get(n pref.FieldNumber) legacyExtensionEntry {
	if m.p == nil {
		return legacyExtensionEntry{}
	}
	return m.p.m.Get(n)
}
func (m *legacyExtensionSyncMap) Set(n pref.FieldNumber, x legacyExtensionEntry) {
	if m.p == nil {
		m.p = new(struct {
			mu sync.Mutex
			m  legacyExtensionMap
		})
	}
	m.p.m.Set(n, x)
}
func (m legacyExtensionSyncMap) Clear(n pref.FieldNumber) {
	m.p.m.Clear(n)
}
func (m legacyExtensionSyncMap) Range(f func(pref.FieldNumber, legacyExtensionEntry) bool) {
	if m.p == nil {
		return
	}
	m.p.m.Range(f)
}

// legacyExtensionMap is identical to map[int32]protoV1.Extension.
// It implements legacyExtensionIface.
type legacyExtensionMap map[pref.FieldNumber]legacyExtensionEntry

func (m legacyExtensionMap) Len() int {
	return len(m)
}
func (m legacyExtensionMap) Has(n pref.FieldNumber) bool {
	_, ok := m[n]
	return ok
}
func (m legacyExtensionMap) Get(n pref.FieldNumber) legacyExtensionEntry {
	return m[n]
}
func (m *legacyExtensionMap) Set(n pref.FieldNumber, x legacyExtensionEntry) {
	if *m == nil {
		*m = make(map[pref.FieldNumber]legacyExtensionEntry)
	}
	(*m)[n] = x
}
func (m *legacyExtensionMap) Clear(n pref.FieldNumber) {
	delete(*m, n)
}
func (m legacyExtensionMap) Range(f func(pref.FieldNumber, legacyExtensionEntry) bool) {
	for n, x := range m {
		if !f(n, x) {
			return
		}
	}
}

// legacyExtensionEntry is identical to protoV1.Extension.
type legacyExtensionEntry struct {
	desc *protoV1.ExtensionDesc
	val  interface{}
	raw  []byte
}
