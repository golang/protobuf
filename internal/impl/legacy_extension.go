// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"

	papi "github.com/golang/protobuf/protoapi"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
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

var (
	extTypeA = reflect.TypeOf(map[int32]papi.ExtensionField(nil))
	extTypeB = reflect.TypeOf(papi.XXX_InternalExtensions{})
)

func makeLegacyExtensionMapFunc(t reflect.Type) func(*messageDataType) papi.ExtensionFields {
	fx1, _ := t.FieldByName("XXX_extensions")
	fx2, _ := t.FieldByName("XXX_InternalExtensions")
	switch {
	case fx1.Type == extTypeA:
		fieldOffset := offsetOf(fx1)
		return func(p *messageDataType) papi.ExtensionFields {
			v := p.p.Apply(fieldOffset).AsValueOf(fx1.Type).Interface()
			return papi.ExtensionFieldsOf(v)
		}
	case fx2.Type == extTypeB:
		fieldOffset := offsetOf(fx2)
		return func(p *messageDataType) papi.ExtensionFields {
			v := p.p.Apply(fieldOffset).AsValueOf(fx2.Type).Interface()
			return papi.ExtensionFieldsOf(v)
		}
	default:
		return nil
	}
}

type legacyExtensionFields struct {
	mi *MessageType
	x  papi.ExtensionFields
}

func (p legacyExtensionFields) Len() (n int) {
	p.x.Range(func(num pref.FieldNumber, _ papi.ExtensionField) bool {
		if p.Has(pref.FieldNumber(num)) {
			n++
		}
		return true
	})
	return n
}

func (p legacyExtensionFields) Has(n pref.FieldNumber) bool {
	x := p.x.Get(n)
	if x.Value == nil {
		return false
	}
	t := legacyWrapper.ExtensionTypeFromDesc(x.Desc)
	if t.Cardinality() == pref.Repeated {
		return t.ValueOf(x.Value).List().Len() > 0
	}
	return true
}

func (p legacyExtensionFields) Get(n pref.FieldNumber) pref.Value {
	x := p.x.Get(n)
	if x.Desc == nil {
		return pref.Value{}
	}
	t := legacyWrapper.ExtensionTypeFromDesc(x.Desc)
	if x.Value == nil {
		// NOTE: x.Value is never nil for Lists since they are always populated
		// during ExtensionFieldTypes.Register.
		if t.Kind() == pref.MessageKind || t.Kind() == pref.GroupKind {
			return pref.Value{}
		}
		return t.Default()
	}
	return t.ValueOf(x.Value)
}

func (p legacyExtensionFields) Set(n pref.FieldNumber, v pref.Value) {
	x := p.x.Get(n)
	if x.Desc == nil {
		panic("no extension descriptor registered")
	}
	t := legacyWrapper.ExtensionTypeFromDesc(x.Desc)
	x.Value = t.InterfaceOf(v)
	p.x.Set(n, x)
}

func (p legacyExtensionFields) Clear(n pref.FieldNumber) {
	x := p.x.Get(n)
	if x.Desc == nil {
		return
	}
	t := legacyWrapper.ExtensionTypeFromDesc(x.Desc)
	if t.Cardinality() == pref.Repeated {
		t.ValueOf(x.Value).List().Truncate(0)
		return
	}
	x.Value = nil
	p.x.Set(n, x)
}

func (p legacyExtensionFields) Range(f func(pref.FieldNumber, pref.Value) bool) {
	p.x.Range(func(n pref.FieldNumber, x papi.ExtensionField) bool {
		if p.Has(n) {
			return f(n, p.Get(n))
		}
		return true
	})
}

func (p legacyExtensionFields) NewMessage(n pref.FieldNumber) pref.Message {
	x := p.x.Get(n)
	if x.Desc == nil {
		panic("no extension descriptor registered")
	}
	xt := legacyWrapper.ExtensionTypeFromDesc(x.Desc)
	return xt.New().Message()
}

func (p legacyExtensionFields) ExtensionTypes() pref.ExtensionFieldTypes {
	return legacyExtensionTypes(p)
}

type legacyExtensionTypes legacyExtensionFields

func (p legacyExtensionTypes) Len() (n int) {
	p.x.Range(func(_ pref.FieldNumber, x papi.ExtensionField) bool {
		if x.Desc != nil {
			n++
		}
		return true
	})
	return n
}

func (p legacyExtensionTypes) Register(t pref.ExtensionType) {
	if p.mi.PBType.FullName() != t.ExtendedType().FullName() {
		panic("extended type mismatch")
	}
	if !p.mi.PBType.ExtensionRanges().Has(t.Number()) {
		panic("invalid extension field number")
	}
	x := p.x.Get(t.Number())
	if x.Desc != nil {
		panic("extension descriptor already registered")
	}
	x.Desc = legacyWrapper.ExtensionDescFromType(t)
	if t.Cardinality() == pref.Repeated {
		// If the field is repeated, initialize the entry with an empty list
		// so that future Get operations can return a mutable and concrete list.
		x.Value = t.InterfaceOf(t.New())
	}
	p.x.Set(t.Number(), x)
}

func (p legacyExtensionTypes) Remove(t pref.ExtensionType) {
	if !p.mi.PBType.ExtensionRanges().Has(t.Number()) {
		return
	}
	x := p.x.Get(t.Number())
	if t.Cardinality() == pref.Repeated {
		// Treat an empty repeated field as unpopulated.
		v := reflect.ValueOf(x.Value)
		if x.Value == nil || v.IsNil() || v.Elem().Len() == 0 {
			x.Value = nil
		}
	}
	if x.Value != nil {
		panic("value for extension descriptor still populated")
	}
	x.Desc = nil
	if len(x.Raw) == 0 {
		p.x.Clear(t.Number())
	} else {
		p.x.Set(t.Number(), x)
	}
}

func (p legacyExtensionTypes) ByNumber(n pref.FieldNumber) pref.ExtensionType {
	x := p.x.Get(n)
	if x.Desc != nil {
		return legacyWrapper.ExtensionTypeFromDesc(x.Desc)
	}
	return nil
}

func (p legacyExtensionTypes) ByName(s pref.FullName) (t pref.ExtensionType) {
	p.x.Range(func(_ pref.FieldNumber, x papi.ExtensionField) bool {
		if x.Desc != nil && x.Desc.Name == string(s) {
			t = legacyWrapper.ExtensionTypeFromDesc(x.Desc)
			return false
		}
		return true
	})
	return t
}

func (p legacyExtensionTypes) Range(f func(pref.ExtensionType) bool) {
	p.x.Range(func(_ pref.FieldNumber, x papi.ExtensionField) bool {
		if x.Desc != nil {
			if !f(legacyWrapper.ExtensionTypeFromDesc(x.Desc)) {
				return false
			}
		}
		return true
	})
}
