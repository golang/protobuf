// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

func makeLegacyExtensionFieldsFunc(t reflect.Type) func(p *messageDataType) pref.KnownFields {
	f := makeLegacyExtensionMapFunc(t)
	if f == nil {
		return nil
	}
	return func(p *messageDataType) pref.KnownFields {
		if p.p.IsNil() {
			return emptyExtensionFields{}
		}
		return legacyExtensionFields{p.mi, f(p)}
	}
}

var extType = reflect.TypeOf(map[int32]ExtensionField{})

func makeLegacyExtensionMapFunc(t reflect.Type) func(*messageDataType) *legacyExtensionMap {
	fx, _ := t.FieldByName("XXX_extensions")
	if fx.Type != extType {
		fx, _ = t.FieldByName("XXX_InternalExtensions")
	}
	if fx.Type != extType {
		return nil
	}

	fieldOffset := offsetOf(fx)
	return func(p *messageDataType) *legacyExtensionMap {
		v := p.p.Apply(fieldOffset).AsValueOf(fx.Type).Interface()
		return (*legacyExtensionMap)(v.(*map[int32]ExtensionField))
	}
}

type legacyExtensionFields struct {
	mi *MessageInfo
	x  *legacyExtensionMap
}

func (p legacyExtensionFields) Len() (n int) {
	p.x.Range(func(num pref.FieldNumber, _ ExtensionField) bool {
		if p.Has(pref.FieldNumber(num)) {
			n++
		}
		return true
	})
	return n
}

func (p legacyExtensionFields) Has(n pref.FieldNumber) bool {
	x := p.x.Get(n)
	if !x.HasValue() {
		return false
	}
	t := x.GetType()
	d := t.Descriptor()
	if d.IsList() {
		return t.ValueOf(x.GetValue()).List().Len() > 0
	}
	return true
}

func (p legacyExtensionFields) Get(n pref.FieldNumber) pref.Value {
	x := p.x.Get(n)
	if !x.HasType() {
		return pref.Value{}
	}
	t := x.GetType()
	d := t.Descriptor()
	if !x.HasValue() {
		// NOTE: x.Value is never nil for Lists since they are always populated
		// during ExtensionFieldTypes.Register.
		if d.Kind() == pref.MessageKind || d.Kind() == pref.GroupKind {
			return pref.Value{}
		}
		return d.Default()
	}
	return t.ValueOf(x.GetValue())
}

func (p legacyExtensionFields) Set(n pref.FieldNumber, v pref.Value) {
	x := p.x.Get(n)
	if !x.HasType() {
		panic("no extension descriptor registered")
	}
	t := x.GetType()
	x.SetEagerValue(t.InterfaceOf(v))
	p.x.Set(n, x)
}

func (p legacyExtensionFields) Clear(n pref.FieldNumber) {
	x := p.x.Get(n)
	if !x.HasType() {
		return
	}
	t := x.GetType()
	d := t.Descriptor()
	if d.IsList() {
		t.ValueOf(x.GetValue()).List().Truncate(0)
		return
	}
	x.SetEagerValue(nil)
	p.x.Set(n, x)
}

func (p legacyExtensionFields) WhichOneof(pref.Name) pref.FieldNumber {
	return 0
}

func (p legacyExtensionFields) Range(f func(pref.FieldNumber, pref.Value) bool) {
	p.x.Range(func(n pref.FieldNumber, x ExtensionField) bool {
		if p.Has(n) {
			return f(n, p.Get(n))
		}
		return true
	})
}

func (p legacyExtensionFields) NewMessage(n pref.FieldNumber) pref.Message {
	x := p.x.Get(n)
	if !x.HasType() {
		panic("no extension descriptor registered")
	}
	xt := x.GetType()
	return xt.New().Message()
}

func (p legacyExtensionFields) ExtensionTypes() pref.ExtensionFieldTypes {
	return legacyExtensionTypes(p)
}

type legacyExtensionTypes legacyExtensionFields

func (p legacyExtensionTypes) Len() (n int) {
	p.x.Range(func(_ pref.FieldNumber, x ExtensionField) bool {
		if x.HasType() {
			n++
		}
		return true
	})
	return n
}

func (p legacyExtensionTypes) Register(t pref.ExtensionType) {
	d := t.Descriptor()
	if p.mi.PBType.Descriptor().FullName() != d.ContainingMessage().FullName() {
		panic("extended type mismatch")
	}
	if !p.mi.PBType.Descriptor().ExtensionRanges().Has(d.Number()) {
		panic("invalid extension field number")
	}
	x := p.x.Get(d.Number())
	if x.HasType() {
		panic("extension descriptor already registered")
	}
	x.SetType(t)
	if d.IsList() {
		// If the field is repeated, initialize the entry with an empty list
		// so that future Get operations can return a mutable and concrete list.
		x.SetEagerValue(t.InterfaceOf(t.New()))
	}
	p.x.Set(d.Number(), x)
}

func (p legacyExtensionTypes) Remove(t pref.ExtensionType) {
	d := t.Descriptor()
	if !p.mi.PBType.Descriptor().ExtensionRanges().Has(d.Number()) {
		return
	}
	x := p.x.Get(d.Number())
	if d.IsList() {
		// Treat an empty repeated field as unpopulated.
		v := reflect.ValueOf(x.GetValue())
		if !x.HasValue() || v.IsNil() || v.Elem().Len() == 0 {
			x.SetEagerValue(nil)
		}
	}
	if x.GetValue() != nil {
		panic("value for extension descriptor still populated")
	}
	p.x.Clear(d.Number())
}

func (p legacyExtensionTypes) ByNumber(n pref.FieldNumber) pref.ExtensionType {
	x := p.x.Get(n)
	if x.HasType() {
		return x.GetType()
	}
	return nil
}

func (p legacyExtensionTypes) ByName(s pref.FullName) (t pref.ExtensionType) {
	p.x.Range(func(_ pref.FieldNumber, x ExtensionField) bool {
		if x.HasType() && x.GetType().FullName() == s {
			t = x.GetType()
			return false
		}
		return true
	})
	return t
}

func (p legacyExtensionTypes) Range(f func(pref.ExtensionType) bool) {
	p.x.Range(func(_ pref.FieldNumber, x ExtensionField) bool {
		if x.HasType() {
			if !f(x.GetType()) {
				return false
			}
		}
		return true
	})
}

type legacyExtensionMap map[int32]ExtensionField

func (m legacyExtensionMap) Len() int {
	return len(m)
}
func (m legacyExtensionMap) Has(n pref.FieldNumber) bool {
	_, ok := m[int32(n)]
	return ok
}
func (m legacyExtensionMap) Get(n pref.FieldNumber) ExtensionField {
	return m[int32(n)]
}
func (m *legacyExtensionMap) Set(n pref.FieldNumber, x ExtensionField) {
	if *m == nil {
		*m = make(map[int32]ExtensionField)
	}
	(*m)[int32(n)] = x
}
func (m *legacyExtensionMap) Clear(n pref.FieldNumber) {
	delete(*m, int32(n))
}
func (m legacyExtensionMap) Range(f func(pref.FieldNumber, ExtensionField) bool) {
	for n, x := range m {
		if !f(pref.FieldNumber(n), x) {
			return
		}
	}
}
