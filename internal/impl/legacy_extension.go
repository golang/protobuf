// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"

	protoV1 "github.com/golang/protobuf/proto"
	ptag "github.com/golang/protobuf/v2/internal/encoding/tag"
	pvalue "github.com/golang/protobuf/v2/internal/value"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
)

func makeLegacyExtensionFieldsFunc(t reflect.Type) func(p *messageDataType) pref.KnownFields {
	f := makeLegacyExtensionMapFunc(t)
	if f == nil {
		return nil
	}
	return func(p *messageDataType) pref.KnownFields {
		return legacyExtensionFields{p.mi, f(p)}
	}
}

type legacyExtensionFields struct {
	mi *MessageType
	x  legacyExtensionIface
}

func (p legacyExtensionFields) Len() (n int) {
	p.x.Range(func(num pref.FieldNumber, _ legacyExtensionEntry) bool {
		if p.Has(num) {
			n++
		}
		return true
	})
	return n
}

func (p legacyExtensionFields) Has(n pref.FieldNumber) bool {
	x := p.x.Get(n)
	if x.val == nil {
		return false
	}
	t := legacyExtensionTypeOf(x.desc)
	if t.Cardinality() == pref.Repeated {
		return legacyExtensionValueOf(x.val, t).List().Len() > 0
	}
	return true
}

func (p legacyExtensionFields) Get(n pref.FieldNumber) pref.Value {
	x := p.x.Get(n)
	if x.desc == nil {
		return pref.Value{}
	}
	t := legacyExtensionTypeOf(x.desc)
	if x.val == nil {
		if t.Cardinality() == pref.Repeated {
			// TODO: What is the zero value for Lists?
			// TODO: This logic is racy.
			v := t.ValueOf(t.New())
			x.val = legacyExtensionInterfaceOf(v, t)
			p.x.Set(n, x)
			return v
		}
		if t.Kind() == pref.MessageKind || t.Kind() == pref.GroupKind {
			// TODO: What is the zero value for Messages?
			return pref.Value{}
		}
		return t.Default()
	}
	return legacyExtensionValueOf(x.val, t)
}

func (p legacyExtensionFields) Set(n pref.FieldNumber, v pref.Value) {
	x := p.x.Get(n)
	if x.desc == nil {
		panic("no extension descriptor registered")
	}
	t := legacyExtensionTypeOf(x.desc)
	x.val = legacyExtensionInterfaceOf(v, t)
	p.x.Set(n, x)
}

func (p legacyExtensionFields) Clear(n pref.FieldNumber) {
	x := p.x.Get(n)
	if x.desc == nil {
		return
	}
	x.val = nil
	p.x.Set(n, x)
}

func (p legacyExtensionFields) Mutable(n pref.FieldNumber) pref.Mutable {
	x := p.x.Get(n)
	if x.desc == nil {
		panic("no extension descriptor registered")
	}
	t := legacyExtensionTypeOf(x.desc)
	if x.val == nil {
		v := t.ValueOf(t.New())
		x.val = legacyExtensionInterfaceOf(v, t)
		p.x.Set(n, x)
	}
	return legacyExtensionValueOf(x.val, t).Interface().(pref.Mutable)
}

func (p legacyExtensionFields) Range(f func(pref.FieldNumber, pref.Value) bool) {
	p.x.Range(func(n pref.FieldNumber, x legacyExtensionEntry) bool {
		if p.Has(n) {
			return f(n, p.Get(n))
		}
		return true
	})
}

func (p legacyExtensionFields) ExtensionTypes() pref.ExtensionFieldTypes {
	return legacyExtensionTypes(p)
}

type legacyExtensionTypes legacyExtensionFields

func (p legacyExtensionTypes) Len() (n int) {
	p.x.Range(func(_ pref.FieldNumber, x legacyExtensionEntry) bool {
		if x.desc != nil {
			n++
		}
		return true
	})
	return n
}

func (p legacyExtensionTypes) Register(t pref.ExtensionType) {
	if p.mi.Type.FullName() != t.ExtendedType().FullName() {
		panic("extended type mismatch")
	}
	if !p.mi.Type.ExtensionRanges().Has(t.Number()) {
		panic("invalid extension field number")
	}
	x := p.x.Get(t.Number())
	if x.desc != nil {
		panic("extension descriptor already registered")
	}
	x.desc = legacyExtensionDescOf(t, p.mi.goType)
	p.x.Set(t.Number(), x)
}

func (p legacyExtensionTypes) Remove(t pref.ExtensionType) {
	if !p.mi.Type.ExtensionRanges().Has(t.Number()) {
		return
	}
	x := p.x.Get(t.Number())
	if x.val != nil {
		panic("value for extension descriptor still populated")
	}
	x.desc = nil
	if len(x.raw) == 0 {
		p.x.Clear(t.Number())
	} else {
		p.x.Set(t.Number(), x)
	}
}

func (p legacyExtensionTypes) ByNumber(n pref.FieldNumber) pref.ExtensionType {
	x := p.x.Get(n)
	if x.desc != nil {
		return legacyExtensionTypeOf(x.desc)
	}
	return nil
}

func (p legacyExtensionTypes) ByName(s pref.FullName) (t pref.ExtensionType) {
	p.x.Range(func(_ pref.FieldNumber, x legacyExtensionEntry) bool {
		if x.desc != nil && x.desc.Name == string(s) {
			t = legacyExtensionTypeOf(x.desc)
			return false
		}
		return true
	})
	return t
}

func (p legacyExtensionTypes) Range(f func(pref.ExtensionType) bool) {
	p.x.Range(func(_ pref.FieldNumber, x legacyExtensionEntry) bool {
		if x.desc != nil {
			if !f(legacyExtensionTypeOf(x.desc)) {
				return false
			}
		}
		return true
	})
}

func legacyExtensionDescOf(t pref.ExtensionType, parent reflect.Type) *protoV1.ExtensionDesc {
	if t, ok := t.(*legacyExtensionType); ok {
		return t.desc
	}

	// Determine the v1 extension type, which is unfortunately not the same as
	// the v2 ExtensionType.GoType.
	extType := t.GoType()
	switch extType.Kind() {
	case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		extType = reflect.PtrTo(extType) // T -> *T for singular scalar fields
	case reflect.Ptr:
		if extType.Elem().Kind() == reflect.Slice {
			extType = extType.Elem() // *[]T -> []T for repeated fields
		}
	}

	// Reconstruct the legacy enum full name, which is an odd mixture of the
	// proto package name with the Go type name.
	var enumName string
	if t.Kind() == pref.EnumKind {
		enumName = t.GoType().Name()
		for d, ok := pref.Descriptor(t.EnumType()), true; ok; d, ok = d.Parent() {
			if fd, _ := d.(pref.FileDescriptor); fd != nil && fd.Package() != "" {
				enumName = string(fd.Package()) + "." + enumName
			}
		}
	}

	// Construct and return a v1 ExtensionDesc.
	return &protoV1.ExtensionDesc{
		ExtendedType:  reflect.Zero(parent).Interface().(protoV1.Message),
		ExtensionType: reflect.Zero(extType).Interface(),
		Field:         int32(t.Number()),
		Name:          string(t.FullName()),
		Tag:           ptag.Marshal(t, enumName),
	}
}

func legacyExtensionTypeOf(d *protoV1.ExtensionDesc) pref.ExtensionType {
	// TODO: Add a field to protoV1.ExtensionDesc to contain a v2 descriptor.

	// Derive basic field information from the struct tag.
	t := reflect.TypeOf(d.ExtensionType)
	isOptional := t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct
	isRepeated := t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
	if isOptional || isRepeated {
		t = t.Elem()
	}
	f := ptag.Unmarshal(d.Tag, t)

	// Construct a v2 ExtensionType.
	conv := newConverter(t, f.Kind)
	xd, err := ptype.NewExtension(&ptype.StandaloneExtension{
		FullName:     pref.FullName(d.Name),
		Number:       pref.FieldNumber(d.Field),
		Cardinality:  f.Cardinality,
		Kind:         f.Kind,
		Default:      f.Default,
		Options:      f.Options,
		EnumType:     conv.EnumType,
		MessageType:  conv.MessageType,
		ExtendedType: legacyLoadMessageDesc(reflect.TypeOf(d.ExtendedType)),
	})
	if err != nil {
		panic(err)
	}
	xt := ptype.GoExtension(xd, conv.EnumType, conv.MessageType)

	// Return the extension type as is if the dependencies already support v2.
	xt2 := &legacyExtensionType{ExtensionType: xt, desc: d}
	if !conv.IsLegacy {
		return xt2
	}

	// If the dependency is a v1 enum or message, we need to create a custom
	// extension type where ExtensionType.GoType continues to use the legacy
	// v1 Go type, instead of the wrapped versions that satisfy the v2 API.
	if xd.Cardinality() != pref.Repeated {
		// Custom extension type for singular enums and messages.
		// The legacy wrappers use legacyEnumWrapper and legacyMessageWrapper
		// to implement the v2 interfaces for enums and messages.
		// Both of those type satisfy the value.Unwrapper interface.
		xt2.typ = t
		xt2.new = func() interface{} {
			return xt.New().(pvalue.Unwrapper).Unwrap()
		}
		xt2.valueOf = func(v interface{}) pref.Value {
			if reflect.TypeOf(v) != xt2.typ {
				panic(fmt.Sprintf("invalid type: got %T, want %v", v, xt2.typ))
			}
			if xd.Kind() == pref.EnumKind {
				return xt.ValueOf(legacyWrapEnum(reflect.ValueOf(v)))
			} else {
				return xt.ValueOf(legacyWrapMessage(reflect.ValueOf(v)))
			}
		}
		xt2.interfaceOf = func(v pref.Value) interface{} {
			return xt.InterfaceOf(v).(pvalue.Unwrapper).Unwrap()
		}
	} else {
		// Custom extension type for repeated enums and messages.
		xt2.typ = reflect.PtrTo(reflect.SliceOf(t))
		xt2.new = func() interface{} {
			return reflect.New(xt2.typ.Elem()).Interface()
		}
		xt2.valueOf = func(v interface{}) pref.Value {
			if reflect.TypeOf(v) != xt2.typ {
				panic(fmt.Sprintf("invalid type: got %T, want %v", v, xt2.typ))
			}
			return pref.ValueOf(pvalue.ListOf(v, conv))
		}
		xt2.interfaceOf = func(pv pref.Value) interface{} {
			v := pv.List().(pvalue.Unwrapper).Unwrap()
			if reflect.TypeOf(v) != xt2.typ {
				panic(fmt.Sprintf("invalid type: got %T, want %v", v, xt2.typ))
			}
			return v
		}
	}
	return xt2
}

type legacyExtensionType struct {
	pref.ExtensionType
	desc        *protoV1.ExtensionDesc
	typ         reflect.Type
	new         func() interface{}
	valueOf     func(interface{}) pref.Value
	interfaceOf func(pref.Value) interface{}
}

func (x *legacyExtensionType) GoType() reflect.Type {
	if x.typ != nil {
		return x.typ
	}
	return x.ExtensionType.GoType()
}
func (x *legacyExtensionType) New() interface{} {
	if x.new != nil {
		return x.new()
	}
	return x.ExtensionType.New()
}
func (x *legacyExtensionType) ValueOf(v interface{}) pref.Value {
	if x.valueOf != nil {
		return x.valueOf(v)
	}
	return x.ExtensionType.ValueOf(v)
}
func (x *legacyExtensionType) InterfaceOf(v pref.Value) interface{} {
	if x.interfaceOf != nil {
		return x.interfaceOf(v)
	}
	return x.ExtensionType.InterfaceOf(v)
}
