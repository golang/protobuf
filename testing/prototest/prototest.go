// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prototest exercises protobuf reflection.
package prototest

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
)

// TODO: Test read-only properties of unpopulated composite values.
// TODO: Test invalid field descriptors or oneof descriptors.
// TODO: This should test the functionality that can be provided by fast-paths.

// MessageOptions configure message tests.
type MessageOptions struct {
	// ExtensionTypes is a list of types to test with.
	//
	// If nil, TestMessage will look for extension types in the global registry.
	ExtensionTypes []pref.ExtensionType
}

// TestMessage runs the provided m through a series of tests
// exercising the protobuf reflection API.
func TestMessage(t testing.TB, m proto.Message, opts MessageOptions) {
	md := m.ProtoReflect().Descriptor()
	m1 := m.ProtoReflect().New()
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		testField(t, m1, fd)
	}
	if opts.ExtensionTypes == nil {
		preg.GlobalTypes.RangeExtensionsByMessage(md.FullName(), func(e pref.ExtensionType) bool {
			opts.ExtensionTypes = append(opts.ExtensionTypes, e)
			return true
		})
	}
	for _, xt := range opts.ExtensionTypes {
		testField(t, m1, xt)
	}
	for i := 0; i < md.Oneofs().Len(); i++ {
		testOneof(t, m1, md.Oneofs().Get(i))
	}
	testUnknown(t, m1)

	// Test round-trip marshal/unmarshal.
	m2 := m.ProtoReflect().New().Interface()
	populateMessage(m2.ProtoReflect(), 1, nil)
	b, err := proto.Marshal(m2)
	if err != nil {
		t.Errorf("Marshal() = %v, want nil\n%v", err, marshalText(m2))
	}
	m3 := m.ProtoReflect().New().Interface()
	if err := proto.Unmarshal(b, m3); err != nil {
		t.Errorf("Unmarshal() = %v, want nil\n%v", err, marshalText(m2))
	}
	if !proto.Equal(m2, m3) {
		t.Errorf("round-trip marshal/unmarshal did not preserve message\nOriginal:\n%v\nNew:\n%v", marshalText(m2), marshalText(m3))
	}
}

func marshalText(m proto.Message) string {
	b, _ := prototext.MarshalOptions{Indent: "  "}.Marshal(m)
	return string(b)
}

// testField exercises set/get/has/clear of a field.
func testField(t testing.TB, m pref.Message, fd pref.FieldDescriptor) {
	name := fd.FullName()
	num := fd.Number()

	switch {
	case fd.IsList():
		testFieldList(t, m, fd)
	case fd.IsMap():
		testFieldMap(t, m, fd)
	case fd.Kind() == pref.FloatKind || fd.Kind() == pref.DoubleKind:
		testFieldFloat(t, m, fd)
	}

	// Set to a non-zero value, the zero value, different non-zero values.
	for _, n := range []seed{1, 0, minVal, maxVal} {
		v := newValue(m, fd, n, nil)
		m.Set(fd, v)
		wantHas := true
		if n == 0 {
			if fd.Syntax() == pref.Proto3 && fd.Message() == nil {
				wantHas = false
			}
			if fd.Cardinality() == pref.Repeated {
				wantHas = false
			}
			if fd.IsExtension() {
				wantHas = true
			}
			if fd.ContainingOneof() != nil {
				wantHas = true
			}
		}
		if got, want := m.Has(fd), wantHas; got != want {
			t.Errorf("after setting %q to %v:\nMessage.Has(%v) = %v, want %v", name, formatValue(v), num, got, want)
		}
		if got, want := m.Get(fd), v; !valueEqual(got, want) {
			t.Errorf("after setting %q:\nMessage.Get(%v) = %v, want %v", name, num, formatValue(got), formatValue(want))
		}
		found := false
		m.Range(func(d pref.FieldDescriptor, got pref.Value) bool {
			if fd != d {
				return true
			}
			found = true
			if want := v; !valueEqual(got, want) {
				t.Errorf("after setting %q:\nMessage.Range got value %v, want %v", name, formatValue(got), formatValue(want))
			}
			return true
		})
		if got, want := wantHas, found; got != want {
			t.Errorf("after setting %q:\nMessageRange saw field: %v, want %v", name, got, want)
		}
	}

	m.Clear(fd)
	if got, want := m.Has(fd), false; got != want {
		t.Errorf("after clearing %q:\nMessage.Has(%v) = %v, want %v", name, num, got, want)
	}
	switch {
	case fd.IsList():
		if got := m.Get(fd); got.List().Len() != 0 {
			t.Errorf("after clearing %q:\nMessage.Get(%v) = %v, want empty list", name, num, formatValue(got))
		}
	case fd.IsMap():
		if got := m.Get(fd); got.Map().Len() != 0 {
			t.Errorf("after clearing %q:\nMessage.Get(%v) = %v, want empty list", name, num, formatValue(got))
		}
	case fd.Message() == nil:
		if got, want := m.Get(fd), fd.Default(); !valueEqual(got, want) {
			t.Errorf("after clearing %q:\nMessage.Get(%v) = %v, want default %v", name, num, formatValue(got), formatValue(want))
		}
	}

	// Set to the wrong type.
	v := pref.ValueOf("")
	if fd.Kind() == pref.StringKind {
		v = pref.ValueOf(int32(0))
	}
	if !panics(func() {
		m.Set(fd, v)
	}) {
		t.Errorf("setting %v to %T succeeds, want panic", name, v.Interface())
	}
}

// testFieldMap tests set/get/has/clear of entries in a map field.
func testFieldMap(t testing.TB, m pref.Message, fd pref.FieldDescriptor) {
	name := fd.FullName()
	num := fd.Number()

	m.Clear(fd) // start with an empty map
	mapv := m.Mutable(fd).Map()

	// Add values.
	want := make(testMap)
	for i, n := range []seed{1, 0, minVal, maxVal} {
		if got, want := m.Has(fd), i > 0; got != want {
			t.Errorf("after inserting %d elements to %q:\nMessage.Has(%v) = %v, want %v", i, name, num, got, want)
		}

		k := newMapKey(fd, n)
		v := newMapValue(fd, mapv, n, nil)
		mapv.Set(k, v)
		want.Set(k, v)
		if got, want := m.Get(fd), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after inserting %d elements to %q:\nMessage.Get(%v) = %v, want %v", i, name, num, formatValue(got), formatValue(want))
		}
	}

	// Set values.
	want.Range(func(k pref.MapKey, v pref.Value) bool {
		nv := newMapValue(fd, mapv, 10, nil)
		mapv.Set(k, nv)
		want.Set(k, nv)
		if got, want := m.Get(fd), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after setting element %v of %q:\nMessage.Get(%v) = %v, want %v", formatValue(k.Value()), name, num, formatValue(got), formatValue(want))
		}
		return true
	})

	// Clear values.
	want.Range(func(k pref.MapKey, v pref.Value) bool {
		mapv.Clear(k)
		want.Clear(k)
		if got, want := m.Has(fd), want.Len() > 0; got != want {
			t.Errorf("after clearing elements of %q:\nMessage.Has(%v) = %v, want %v", name, num, got, want)
		}
		if got, want := m.Get(fd), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after clearing elements of %q:\nMessage.Get(%v) = %v, want %v", name, num, formatValue(got), formatValue(want))
		}
		return true
	})

	// Non-existent map keys.
	missingKey := newMapKey(fd, 1)
	if got, want := mapv.Has(missingKey), false; got != want {
		t.Errorf("non-existent map key in %q: Map.Has(%v) = %v, want %v", name, formatValue(missingKey.Value()), got, want)
	}
	if got, want := mapv.Get(missingKey).IsValid(), false; got != want {
		t.Errorf("non-existent map key in %q: Map.Get(%v).IsValid() = %v, want %v", name, formatValue(missingKey.Value()), got, want)
	}
	mapv.Clear(missingKey) // noop
}

type testMap map[interface{}]pref.Value

func (m testMap) Get(k pref.MapKey) pref.Value    { return m[k.Interface()] }
func (m testMap) Set(k pref.MapKey, v pref.Value) { m[k.Interface()] = v }
func (m testMap) Has(k pref.MapKey) bool          { return m.Get(k).IsValid() }
func (m testMap) Clear(k pref.MapKey)             { delete(m, k.Interface()) }
func (m testMap) Len() int                        { return len(m) }
func (m testMap) NewMessage() pref.Message        { panic("unimplemented") }
func (m testMap) Range(f func(pref.MapKey, pref.Value) bool) {
	for k, v := range m {
		if !f(pref.ValueOf(k).MapKey(), v) {
			return
		}
	}
}

// testFieldList exercises set/get/append/truncate of values in a list.
func testFieldList(t testing.TB, m pref.Message, fd pref.FieldDescriptor) {
	name := fd.FullName()
	num := fd.Number()

	m.Clear(fd) // start with an empty list
	list := m.Mutable(fd).List()

	// Append values.
	var want pref.List = &testList{}
	for i, n := range []seed{1, 0, minVal, maxVal} {
		if got, want := m.Has(fd), i > 0 || fd.IsExtension(); got != want {
			t.Errorf("after appending %d elements to %q:\nMessage.Has(%v) = %v, want %v", i, name, num, got, want)
		}
		v := newListElement(fd, list, n, nil)
		want.Append(v)
		list.Append(v)

		if got, want := m.Get(fd), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after appending %d elements to %q:\nMessage.Get(%v) = %v, want %v", i+1, name, num, formatValue(got), formatValue(want))
		}
	}

	// Set values.
	for i := 0; i < want.Len(); i++ {
		v := newListElement(fd, list, seed(i+10), nil)
		want.Set(i, v)
		list.Set(i, v)
		if got, want := m.Get(fd), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after setting element %d of %q:\nMessage.Get(%v) = %v, want %v", i, name, num, formatValue(got), formatValue(want))
		}
	}

	// Truncate.
	for want.Len() > 0 {
		n := want.Len() - 1
		want.Truncate(n)
		list.Truncate(n)
		if got, want := m.Has(fd), want.Len() > 0 || fd.IsExtension(); got != want {
			t.Errorf("after truncating %q to %d:\nMessage.Has(%v) = %v, want %v", name, n, num, got, want)
		}
		if got, want := m.Get(fd), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after truncating %q to %d:\nMessage.Get(%v) = %v, want %v", name, n, num, formatValue(got), formatValue(want))
		}
	}
}

type testList struct {
	a []pref.Value
}

func (l *testList) Append(v pref.Value)      { l.a = append(l.a, v) }
func (l *testList) Get(n int) pref.Value     { return l.a[n] }
func (l *testList) Len() int                 { return len(l.a) }
func (l *testList) Set(n int, v pref.Value)  { l.a[n] = v }
func (l *testList) Truncate(n int)           { l.a = l.a[:n] }
func (l *testList) NewMessage() pref.Message { panic("unimplemented") }

// testFieldFloat exercises some interesting floating-point scalar field values.
func testFieldFloat(t testing.TB, m pref.Message, fd pref.FieldDescriptor) {
	name := fd.FullName()
	num := fd.Number()

	for _, v := range []float64{math.Inf(-1), math.Inf(1), math.NaN(), math.Copysign(0, -1)} {
		var val pref.Value
		if fd.Kind() == pref.FloatKind {
			val = pref.ValueOf(float32(v))
		} else {
			val = pref.ValueOf(v)
		}
		m.Set(fd, val)
		// Note that Has is true for -0.
		if got, want := m.Has(fd), true; got != want {
			t.Errorf("after setting %v to %v: Message.Has(%v) = %v, want %v", name, v, num, got, want)
		}
		if got, want := m.Get(fd), val; !valueEqual(got, want) {
			t.Errorf("after setting %v: Message.Get(%v) = %v, want %v", name, num, formatValue(got), formatValue(want))
		}
	}
}

// testOneof tests the behavior of fields in a oneof.
func testOneof(t testing.TB, m pref.Message, od pref.OneofDescriptor) {
	for i := 0; i < od.Fields().Len(); i++ {
		fda := od.Fields().Get(i)
		m.Set(fda, newValue(m, fda, 1, nil))
		if got, want := m.WhichOneof(od), fda; got != want {
			t.Errorf("after setting oneof field %q:\nWhichOneof(%q) = %v, want %v", fda.FullName(), fda.Name(), got, want)
		}
		for j := 0; j < od.Fields().Len(); j++ {
			fdb := od.Fields().Get(j)
			if got, want := m.Has(fdb), i == j; got != want {
				t.Errorf("after setting oneof field %q:\nGet(%q) = %v, want %v", fda.FullName(), fdb.FullName(), got, want)
			}
		}
	}
}

// testUnknown tests the behavior of unknown fields.
func testUnknown(t testing.TB, m pref.Message) {
	var b []byte
	b = wire.AppendTag(b, 1000, wire.VarintType)
	b = wire.AppendVarint(b, 1001)
	m.SetUnknown(pref.RawFields(b))
	if got, want := []byte(m.GetUnknown()), b; !bytes.Equal(got, want) {
		t.Errorf("after setting unknown fields:\nGetUnknown() = %v, want %v", got, want)
	}
}

func formatValue(v pref.Value) string {
	switch v := v.Interface().(type) {
	case pref.List:
		var buf bytes.Buffer
		buf.WriteString("list[")
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString(formatValue(v.Get(i)))
		}
		buf.WriteString("]")
		return buf.String()
	case pref.Map:
		var buf bytes.Buffer
		buf.WriteString("map[")
		var keys []pref.MapKey
		v.Range(func(k pref.MapKey, v pref.Value) bool {
			keys = append(keys, k)
			return true
		})
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})
		for i, k := range keys {
			if i > 0 {
				buf.WriteString(" ")
			}
			buf.WriteString(formatValue(k.Value()))
			buf.WriteString(":")
			buf.WriteString(formatValue(v.Get(k)))
		}
		buf.WriteString("]")
		return buf.String()
	case pref.Message:
		b, err := prototext.Marshal(v.Interface())
		if err != nil {
			return fmt.Sprintf("<%v>", err)
		}
		return fmt.Sprintf("%v{%v}", v.Descriptor().FullName(), string(b))
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprint(v)
	}
}

func valueEqual(a, b pref.Value) bool {
	ai, bi := a.Interface(), b.Interface()
	switch ai.(type) {
	case pref.Message:
		return proto.Equal(
			a.Message().Interface(),
			b.Message().Interface(),
		)
	case pref.List:
		lista, listb := a.List(), b.List()
		if lista.Len() != listb.Len() {
			return false
		}
		for i := 0; i < lista.Len(); i++ {
			if !valueEqual(lista.Get(i), listb.Get(i)) {
				return false
			}
		}
		return true
	case pref.Map:
		mapa, mapb := a.Map(), b.Map()
		if mapa.Len() != mapb.Len() {
			return false
		}
		equal := true
		mapa.Range(func(k pref.MapKey, v pref.Value) bool {
			if !valueEqual(v, mapb.Get(k)) {
				equal = false
				return false
			}
			return true
		})
		return equal
	case []byte:
		return bytes.Equal(a.Bytes(), b.Bytes())
	case float32:
		// NaNs are equal, but must be the same NaN.
		return math.Float32bits(ai.(float32)) == math.Float32bits(bi.(float32))
	case float64:
		// NaNs are equal, but must be the same NaN.
		return math.Float64bits(ai.(float64)) == math.Float64bits(bi.(float64))
	default:
		return ai == bi
	}
}

// A seed is used to vary the content of a value.
//
// A seed of 0 is the zero value. Messages do not have a zero-value; a 0-seeded messages
// is unpopulated.
//
// A seed of minVal or maxVal is the least or greatest value of the value type.
type seed int

const (
	minVal seed = -1
	maxVal seed = -2
)

// newValue returns a new value assignable to a field.
//
// The stack parameter is used to avoid infinite recursion when populating circular
// data structures.
func newValue(m pref.Message, fd pref.FieldDescriptor, n seed, stack []pref.MessageDescriptor) pref.Value {
	switch {
	case fd.IsList():
		list := m.New().Mutable(fd).List()
		if n == 0 {
			return pref.ValueOf(list)
		}
		list.Append(newListElement(fd, list, 0, stack))
		list.Append(newListElement(fd, list, minVal, stack))
		list.Append(newListElement(fd, list, maxVal, stack))
		list.Append(newListElement(fd, list, n, stack))
		return pref.ValueOf(list)
	case fd.IsMap():
		mapv := m.New().Mutable(fd).Map()
		if n == 0 {
			return pref.ValueOf(mapv)
		}
		mapv.Set(newMapKey(fd, 0), newMapValue(fd, mapv, 0, stack))
		mapv.Set(newMapKey(fd, minVal), newMapValue(fd, mapv, minVal, stack))
		mapv.Set(newMapKey(fd, maxVal), newMapValue(fd, mapv, maxVal, stack))
		mapv.Set(newMapKey(fd, n), newMapValue(fd, mapv, 10*n, stack))
		return pref.ValueOf(mapv)
	case fd.Message() != nil:
		return populateMessage(m.Mutable(fd).Message(), n, stack)
	default:
		return newScalarValue(fd, n)
	}
}

func newListElement(fd pref.FieldDescriptor, list pref.List, n seed, stack []pref.MessageDescriptor) pref.Value {
	if fd.Message() == nil {
		return newScalarValue(fd, n)
	}
	return populateMessage(list.NewMessage(), n, stack)
}

func newMapKey(fd pref.FieldDescriptor, n seed) pref.MapKey {
	kd := fd.MapKey()
	return newScalarValue(kd, n).MapKey()
}

func newMapValue(fd pref.FieldDescriptor, mapv pref.Map, n seed, stack []pref.MessageDescriptor) pref.Value {
	vd := fd.MapValue()
	if vd.Message() == nil {
		return newScalarValue(vd, n)
	}
	return populateMessage(mapv.NewMessage(), n, stack)
}

func newScalarValue(fd pref.FieldDescriptor, n seed) pref.Value {
	switch fd.Kind() {
	case pref.BoolKind:
		return pref.ValueOf(n != 0)
	case pref.EnumKind:
		// TODO: use actual value
		return pref.ValueOf(pref.EnumNumber(n))
	case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
		switch n {
		case minVal:
			return pref.ValueOf(int32(math.MinInt32))
		case maxVal:
			return pref.ValueOf(int32(math.MaxInt32))
		default:
			return pref.ValueOf(int32(n))
		}
	case pref.Uint32Kind, pref.Fixed32Kind:
		switch n {
		case minVal:
			// Only use 0 for the zero value.
			return pref.ValueOf(uint32(1))
		case maxVal:
			return pref.ValueOf(uint32(math.MaxInt32))
		default:
			return pref.ValueOf(uint32(n))
		}
	case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
		switch n {
		case minVal:
			return pref.ValueOf(int64(math.MinInt64))
		case maxVal:
			return pref.ValueOf(int64(math.MaxInt64))
		default:
			return pref.ValueOf(int64(n))
		}
	case pref.Uint64Kind, pref.Fixed64Kind:
		switch n {
		case minVal:
			// Only use 0 for the zero value.
			return pref.ValueOf(uint64(1))
		case maxVal:
			return pref.ValueOf(uint64(math.MaxInt64))
		default:
			return pref.ValueOf(uint64(n))
		}
	case pref.FloatKind:
		switch n {
		case minVal:
			return pref.ValueOf(float32(math.SmallestNonzeroFloat32))
		case maxVal:
			return pref.ValueOf(float32(math.MaxFloat32))
		default:
			return pref.ValueOf(1.5 * float32(n))
		}
	case pref.DoubleKind:
		switch n {
		case minVal:
			return pref.ValueOf(float64(math.SmallestNonzeroFloat64))
		case maxVal:
			return pref.ValueOf(float64(math.MaxFloat64))
		default:
			return pref.ValueOf(1.5 * float64(n))
		}
	case pref.StringKind:
		if n == 0 {
			return pref.ValueOf("")
		}
		return pref.ValueOf(fmt.Sprintf("%d", n))
	case pref.BytesKind:
		if n == 0 {
			return pref.ValueOf([]byte(nil))
		}
		return pref.ValueOf([]byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)})
	}
	panic("unhandled kind")
}

func populateMessage(m pref.Message, n seed, stack []pref.MessageDescriptor) pref.Value {
	if n == 0 {
		return pref.ValueOf(m)
	}
	md := m.Descriptor()
	for _, x := range stack {
		if md == x {
			return pref.ValueOf(m)
		}
	}
	stack = append(stack, md)
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		if fd.IsWeak() {
			continue
		}
		m.Set(fd, newValue(m, fd, 10*n+seed(i), stack))
	}
	return pref.ValueOf(m)
}

func panics(f func()) (didPanic bool) {
	defer func() {
		if err := recover(); err != nil {
			didPanic = true
		}
	}()
	f()
	return false
}
