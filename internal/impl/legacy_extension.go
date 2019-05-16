// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"sync"
	"sync/atomic"

	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
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

var extType = reflect.TypeOf(map[int32]ExtensionFieldV1{})

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
		return (*legacyExtensionMap)(v.(*map[int32]ExtensionFieldV1))
	}
}

type legacyExtensionFields struct {
	mi *MessageType
	x  *legacyExtensionMap
}

func (p legacyExtensionFields) Len() (n int) {
	p.x.Range(func(num pref.FieldNumber, _ ExtensionFieldV1) bool {
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
	t := extensionTypeFromDesc(x.Desc)
	d := t.Descriptor()
	if d.IsList() {
		return t.ValueOf(x.GetValue()).List().Len() > 0
	}
	return true
}

func (p legacyExtensionFields) Get(n pref.FieldNumber) pref.Value {
	x := p.x.Get(n)
	if x.Desc == nil {
		return pref.Value{}
	}
	t := extensionTypeFromDesc(x.Desc)
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
	if x.Desc == nil {
		panic("no extension descriptor registered")
	}
	t := extensionTypeFromDesc(x.Desc)
	x.SetEagerValue(t.InterfaceOf(v))
	p.x.Set(n, x)
}

func (p legacyExtensionFields) Clear(n pref.FieldNumber) {
	x := p.x.Get(n)
	if x.Desc == nil {
		return
	}
	t := extensionTypeFromDesc(x.Desc)
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
	p.x.Range(func(n pref.FieldNumber, x ExtensionFieldV1) bool {
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
	xt := extensionTypeFromDesc(x.Desc)
	return xt.New().Message()
}

func (p legacyExtensionFields) ExtensionTypes() pref.ExtensionFieldTypes {
	return legacyExtensionTypes(p)
}

type legacyExtensionTypes legacyExtensionFields

func (p legacyExtensionTypes) Len() (n int) {
	p.x.Range(func(_ pref.FieldNumber, x ExtensionFieldV1) bool {
		if x.Desc != nil {
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
	if x.Desc != nil {
		panic("extension descriptor already registered")
	}
	x.Desc = extensionDescFromType(t)
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
	if x.Desc != nil {
		return extensionTypeFromDesc(x.Desc)
	}
	return nil
}

func (p legacyExtensionTypes) ByName(s pref.FullName) (t pref.ExtensionType) {
	p.x.Range(func(_ pref.FieldNumber, x ExtensionFieldV1) bool {
		if x.Desc != nil && x.Desc.Name == string(s) {
			t = extensionTypeFromDesc(x.Desc)
			return false
		}
		return true
	})
	return t
}

func (p legacyExtensionTypes) Range(f func(pref.ExtensionType) bool) {
	p.x.Range(func(_ pref.FieldNumber, x ExtensionFieldV1) bool {
		if x.Desc != nil {
			if !f(extensionTypeFromDesc(x.Desc)) {
				return false
			}
		}
		return true
	})
}

func extensionDescFromType(typ pref.ExtensionType) *piface.ExtensionDescV1 {
	if xt, ok := typ.(interface {
		ProtoLegacyExtensionDesc() *piface.ExtensionDescV1
	}); ok {
		if desc := xt.ProtoLegacyExtensionDesc(); desc != nil {
			return desc
		}
	}
	return legacyWrapper.ExtensionDescFromType(typ)
}

func extensionTypeFromDesc(desc *piface.ExtensionDescV1) pref.ExtensionType {
	if desc.Type != nil {
		return desc.Type
	}
	return legacyWrapper.ExtensionTypeFromDesc(desc)
}

type ExtensionFieldV1 struct {
	// TODO: We should turn this into a type alias to an unnamed type,
	// which means that v1 can have the same struct, and we no longer have to
	// export this from the v2 API.

	// When an extension is stored in a message using SetExtension
	// only desc and value are set. When the message is marshaled
	// Raw will be set to the encoded form of the message.
	//
	// When a message is unmarshaled and contains extensions, each
	// extension will have only Raw set. When such an extension is
	// accessed using GetExtension (or GetExtensions) desc and value
	// will be set.
	Desc *piface.ExtensionDescV1 // TODO: switch to protoreflect.ExtensionType

	// value is either the value of GetValue,
	// or a *lazyExtensionValue that then returns the value of GetValue.
	//
	// TODO: unexport this.
	Value interface{}
}

// HasValue reports whether a value is set for the extension field.
// This may be called concurrently.
func (f ExtensionFieldV1) HasValue() bool {
	return f.Value != nil
}

// GetValue returns the concrete value for the extension field.
// Let the type of Desc.ExtensionType be the "API type" and
// the type of GetValue be the "storage type".
// The API type and storage type are the same except:
//	* for scalars (except []byte), where the API type uses *T,
//	while the storage type uses T.
//	* for repeated fields, where the API type uses []T,
//	while the storage type uses *[]T.
//
// The reason for the divergence is so that the storage type more naturally
// matches what is expected of when retrieving the values through the
// protobuf reflection APIs.
//
// GetValue is only populated if Desc is also populated.
// This may be called concurrently.
//
// TODO: switch interface{} to protoreflect.Value
func (f ExtensionFieldV1) GetValue() interface{} {
	if f, ok := f.Value.(*lazyExtensionValue); ok {
		return f.GetValue()
	}
	return f.Value
}

// SetEagerValue sets the current value of the extension.
// This must not be called concurrently.
func (f *ExtensionFieldV1) SetEagerValue(v interface{}) {
	f.Value = v
}

// SetLazyValue sets a value that is to be lazily evaluated upon first use.
// The returned value must not be nil.
// This must not be called concurrently.
func (f *ExtensionFieldV1) SetLazyValue(v func() interface{}) {
	f.Value = &lazyExtensionValue{value: v}
}

type lazyExtensionValue struct {
	once  uint32      // atomically set if value is valid
	mu    sync.Mutex  // protects value
	value interface{} // either the value itself or a func() interface{}
}

func (v *lazyExtensionValue) GetValue() interface{} {
	if atomic.LoadUint32(&v.once) == 0 {
		v.mu.Lock()
		if f, ok := v.value.(func() interface{}); ok {
			v.value = f()
		}
		atomic.StoreUint32(&v.once, 1)
		v.mu.Unlock()
	}
	return v.value
}

type legacyExtensionMap map[int32]ExtensionFieldV1

func (m legacyExtensionMap) Len() int {
	return len(m)
}
func (m legacyExtensionMap) Has(n pref.FieldNumber) bool {
	_, ok := m[int32(n)]
	return ok
}
func (m legacyExtensionMap) Get(n pref.FieldNumber) ExtensionFieldV1 {
	return m[int32(n)]
}
func (m *legacyExtensionMap) Set(n pref.FieldNumber, x ExtensionFieldV1) {
	if *m == nil {
		*m = make(map[int32]ExtensionFieldV1)
	}
	(*m)[int32(n)] = x
}
func (m *legacyExtensionMap) Clear(n pref.FieldNumber) {
	delete(*m, int32(n))
}
func (m legacyExtensionMap) Range(f func(pref.FieldNumber, ExtensionFieldV1) bool) {
	for n, x := range m {
		if !f(pref.FieldNumber(n), x) {
			return
		}
	}
}
