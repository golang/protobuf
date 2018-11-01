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

// TODO: The logic below this is a hack since v1 currently exposes no
// exported functionality for interacting with these data structures.
// Eventually make changes to v1 such that v2 can access the necessary
// fields without relying on unsafe.

var (
	extTypeA = reflect.TypeOf(map[int32]protoV1.Extension(nil))
	extTypeB = reflect.TypeOf(protoV1.XXX_InternalExtensions{})
)

type legacyExtensionIface interface {
	Len() int
	Get(pref.FieldNumber) legacyExtensionEntry
	Set(pref.FieldNumber, legacyExtensionEntry)
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
	m.p.mu.Lock()
	defer m.p.mu.Unlock()
	return m.p.m.Len()
}
func (m legacyExtensionSyncMap) Get(n pref.FieldNumber) legacyExtensionEntry {
	if m.p == nil {
		return legacyExtensionEntry{}
	}
	m.p.mu.Lock()
	defer m.p.mu.Unlock()
	return m.p.m.Get(n)
}
func (m *legacyExtensionSyncMap) Set(n pref.FieldNumber, x legacyExtensionEntry) {
	if m.p == nil {
		m.p = new(struct {
			mu sync.Mutex
			m  legacyExtensionMap
		})
	}
	m.p.mu.Lock()
	defer m.p.mu.Unlock()
	m.p.m.Set(n, x)
}
func (m legacyExtensionSyncMap) Range(f func(pref.FieldNumber, legacyExtensionEntry) bool) {
	if m.p == nil {
		return
	}
	m.p.mu.Lock()
	defer m.p.mu.Unlock()
	m.p.m.Range(f)
}

// legacyExtensionMap is identical to map[int32]protoV1.Extension.
// It implements legacyExtensionIface.
type legacyExtensionMap map[pref.FieldNumber]legacyExtensionEntry

func (m legacyExtensionMap) Len() int {
	return len(m)
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
