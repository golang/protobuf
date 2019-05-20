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

	prototext "google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// TestMessage runs the provided message through a series of tests
// exercising the protobuf reflection API.
func TestMessage(t testing.TB, message proto.Message) {
	md := message.ProtoReflect().Descriptor()

	m := message.ProtoReflect().New()
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		switch {
		case fd.IsList():
			testFieldList(t, m, fd)
		case fd.IsMap():
			testFieldMap(t, m, fd)
		case fd.Kind() == pref.FloatKind || fd.Kind() == pref.DoubleKind:
			testFieldFloat(t, m, fd)
		}
		testField(t, m, fd)
	}
	for i := 0; i < md.Oneofs().Len(); i++ {
		testOneof(t, m, md.Oneofs().Get(i))
	}

	// Test has/get/clear on a non-existent field.
	for num := pref.FieldNumber(1); ; num++ {
		if md.Fields().ByNumber(num) != nil {
			continue
		}
		if md.ExtensionRanges().Has(num) {
			continue
		}
		// Field num does not exist.
		if m.KnownFields().Has(num) {
			t.Errorf("non-existent field: Has(%v) = true, want false", num)
		}
		if v := m.KnownFields().Get(num); v.IsValid() {
			t.Errorf("non-existent field: Get(%v) = %v, want invalid", num, formatValue(v))
		}
		m.KnownFields().Clear(num) // noop
		break
	}

	// Test WhichOneof on a non-existent oneof.
	const invalidName = "invalid-name"
	if got, want := m.KnownFields().WhichOneof(invalidName), pref.FieldNumber(0); got != want {
		t.Errorf("non-existent oneof: WhichOneof(%q) = %v, want %v", invalidName, got, want)
	}

	// TODO: Extensions, unknown fields.

	// Test round-trip marshal/unmarshal.
	m1 := message.ProtoReflect().New().Interface()
	populateMessage(m1.ProtoReflect(), 1, nil)
	b, err := proto.Marshal(m1)
	if err != nil {
		t.Errorf("Marshal() = %v, want nil\n%v", err, marshalText(m1))
	}
	m2 := message.ProtoReflect().New().Interface()
	if err := proto.Unmarshal(b, m2); err != nil {
		t.Errorf("Unmarshal() = %v, want nil\n%v", err, marshalText(m1))
	}
	if !proto.Equal(m1, m2) {
		t.Errorf("round-trip marshal/unmarshal did not preserve message.\nOriginal:\n%v\nNew:\n%v", marshalText(m1), marshalText(m2))
	}
}

func marshalText(m proto.Message) string {
	b, _ := prototext.MarshalOptions{Indent: "  "}.Marshal(m)
	return string(b)
}

// testField exericises set/get/has/clear of a field.
func testField(t testing.TB, m pref.Message, fd pref.FieldDescriptor) {
	num := fd.Number()
	name := fd.FullName()
	known := m.KnownFields()

	// Set to a non-zero value, the zero value, different non-zero values.
	for _, n := range []seed{1, 0, minVal, maxVal} {
		v := newValue(m, fd, n, nil)
		known.Set(num, v)
		wantHas := true
		if n == 0 {
			if fd.Syntax() == pref.Proto3 && fd.Message() == nil {
				wantHas = false
			}
			if fd.Cardinality() == pref.Repeated {
				wantHas = false
			}
			if fd.ContainingOneof() != nil {
				wantHas = true
			}
		}
		if got, want := known.Has(num), wantHas; got != want {
			t.Errorf("after setting %q to %v:\nHas(%v) = %v, want %v", name, formatValue(v), num, got, want)
		}
		if got, want := known.Get(num), v; !valueEqual(got, want) {
			t.Errorf("after setting %q:\nGet(%v) = %v, want %v", name, num, formatValue(got), formatValue(want))
		}
	}

	known.Clear(num)
	if got, want := known.Has(num), false; got != want {
		t.Errorf("after clearing %q:\nHas(%v) = %v, want %v", name, num, got, want)
	}
	switch {
	case fd.IsList():
		if got := known.Get(num); got.List().Len() != 0 {
			t.Errorf("after clearing %q:\nGet(%v) = %v, want empty list", name, num, formatValue(got))
		}
	case fd.IsMap():
		if got := known.Get(num); got.Map().Len() != 0 {
			t.Errorf("after clearing %q:\nGet(%v) = %v, want empty list", name, num, formatValue(got))
		}
	default:
		if got, want := known.Get(num), fd.Default(); !valueEqual(got, want) {
			t.Errorf("after clearing %q:\nGet(%v) = %v, want default %v", name, num, formatValue(got), formatValue(want))
		}
	}
}

// testFieldMap tests set/get/has/clear of entries in a map field.
func testFieldMap(t testing.TB, m pref.Message, fd pref.FieldDescriptor) {
	num := fd.Number()
	name := fd.FullName()
	known := m.KnownFields()
	known.Clear(num) // start with an empty map
	mapv := known.Get(num).Map()

	// Add values.
	want := make(testMap)
	for i, n := range []seed{1, 0, minVal, maxVal} {
		if got, want := known.Has(num), i > 0; got != want {
			t.Errorf("after inserting %d elements to %q:\nHas(%v) = %v, want %v", i, name, num, got, want)
		}

		k := newMapKey(fd, n)
		v := newMapValue(fd, mapv, n, nil)
		mapv.Set(k, v)
		want.Set(k, v)
		if got, want := known.Get(num), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after inserting %d elements to %q:\nGet(%v) = %v, want %v", i, name, num, formatValue(got), formatValue(want))
		}
	}

	// Set values.
	want.Range(func(k pref.MapKey, v pref.Value) bool {
		nv := newMapValue(fd, mapv, 10, nil)
		mapv.Set(k, nv)
		want.Set(k, nv)
		if got, want := m.KnownFields().Get(num), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after setting element %v of %q:\nGet(%v) = %v, want %v", formatValue(k.Value()), name, num, formatValue(got), formatValue(want))
		}
		return true
	})

	// Clear values.
	want.Range(func(k pref.MapKey, v pref.Value) bool {
		mapv.Clear(k)
		want.Clear(k)
		if got, want := known.Has(num), want.Len() > 0; got != want {
			t.Errorf("after clearing elements of %q:\nHas(%v) = %v, want %v", name, num, got, want)
		}
		if got, want := m.KnownFields().Get(num), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after clearing elements of %q:\nGet(%v) = %v, want %v", name, num, formatValue(got), formatValue(want))
		}
		return true
	})

	// Non-existent map keys.
	missingKey := newMapKey(fd, 1)
	if got, want := mapv.Has(missingKey), false; got != want {
		t.Errorf("non-existent map key in %q: Has(%v) = %v, want %v", name, formatValue(missingKey.Value()), got, want)
	}
	if got, want := mapv.Get(missingKey).IsValid(), false; got != want {
		t.Errorf("non-existent map key in %q: Get(%v).IsValid() = %v, want %v", name, formatValue(missingKey.Value()), got, want)
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
	num := fd.Number()
	name := fd.FullName()
	known := m.KnownFields()
	known.Clear(num) // start with an empty list
	list := known.Get(num).List()

	// Append values.
	var want pref.List = &testList{}
	for i, n := range []seed{1, 0, minVal, maxVal} {
		if got, want := known.Has(num), i > 0; got != want {
			t.Errorf("after appending %d elements to %q:\nHas(%v) = %v, want %v", i, name, num, got, want)
		}
		v := newListElement(fd, list, n, nil)
		want.Append(v)
		list.Append(v)

		if got, want := m.KnownFields().Get(num), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after appending %d elements to %q:\nGet(%v) = %v, want %v", i+1, name, num, formatValue(got), formatValue(want))
		}
	}

	// Set values.
	for i := 0; i < want.Len(); i++ {
		v := newListElement(fd, list, seed(i+10), nil)
		want.Set(i, v)
		list.Set(i, v)
		if got, want := m.KnownFields().Get(num), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after setting element %d of %q:\nGet(%v) = %v, want %v", i, name, num, formatValue(got), formatValue(want))
		}
	}

	// Truncate.
	for want.Len() > 0 {
		n := want.Len() - 1
		want.Truncate(n)
		list.Truncate(n)
		if got, want := known.Has(num), want.Len() > 0; got != want {
			t.Errorf("after truncating %q to %d:\nHas(%v) = %v, want %v", name, n, num, got, want)
		}
		if got, want := m.KnownFields().Get(num), pref.ValueOf(want); !valueEqual(got, want) {
			t.Errorf("after truncating %q to %d:\nGet(%v) = %v, want %v", name, n, num, formatValue(got), formatValue(want))
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
	num := fd.Number()
	name := fd.FullName()
	known := m.KnownFields()
	for _, v := range []float64{math.Inf(-1), math.Inf(1), math.NaN(), math.Copysign(0, -1)} {
		var val pref.Value
		if fd.Kind() == pref.FloatKind {
			val = pref.ValueOf(float32(v))
		} else {
			val = pref.ValueOf(v)
		}
		known.Set(num, val)
		// Note that Has is true for -0.
		if got, want := known.Has(num), true; got != want {
			t.Errorf("after setting %v to %v: Get(%v) = %v, want %v", name, v, num, got, want)
		}
		if got, want := known.Get(num), val; !valueEqual(got, want) {
			t.Errorf("after setting %v: Get(%v) = %v, want %v", name, num, formatValue(got), formatValue(want))
		}
	}
}

// testOneof tests the behavior of fields in a oneof.
func testOneof(t testing.TB, m pref.Message, od pref.OneofDescriptor) {
	known := m.KnownFields()
	for i := 0; i < od.Fields().Len(); i++ {
		fda := od.Fields().Get(i)
		known.Set(fda.Number(), newValue(m, fda, 1, nil))
		if got, want := known.WhichOneof(od.Name()), fda.Number(); got != want {
			t.Errorf("after setting oneof field %q:\nWhichOneof(%q) = %v, want %v", fda.FullName(), fda.Name(), got, want)
		}
		for j := 0; j < od.Fields().Len(); j++ {
			fdb := od.Fields().Get(j)
			if got, want := known.Has(fdb.Number()), i == j; got != want {
				t.Errorf("after setting oneof field %q:\nGet(%q) = %v, want %v", fda.FullName(), fdb.FullName(), got, want)
			}
		}
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
	num := fd.Number()
	switch {
	case fd.IsList():
		list := m.New().KnownFields().Get(num).List()
		if n == 0 {
			return pref.ValueOf(list)
		}
		list.Append(newListElement(fd, list, 0, stack))
		list.Append(newListElement(fd, list, minVal, stack))
		list.Append(newListElement(fd, list, maxVal, stack))
		list.Append(newListElement(fd, list, n, stack))
		return pref.ValueOf(list)
	case fd.IsMap():
		mapv := m.New().KnownFields().Get(num).Map()
		if n == 0 {
			return pref.ValueOf(mapv)
		}
		mapv.Set(newMapKey(fd, 0), newMapValue(fd, mapv, 0, stack))
		mapv.Set(newMapKey(fd, minVal), newMapValue(fd, mapv, minVal, stack))
		mapv.Set(newMapKey(fd, maxVal), newMapValue(fd, mapv, maxVal, stack))
		mapv.Set(newMapKey(fd, n), newMapValue(fd, mapv, 10*n, stack))
		return pref.ValueOf(mapv)
	case fd.Message() != nil:
		return populateMessage(m.KnownFields().NewMessage(num), n, stack)
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
		// TODO use actual value
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
	known := m.KnownFields()
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)
		if fd.IsWeak() {
			continue
		}
		known.Set(fd.Number(), newValue(m, fd, 10*n+seed(i), stack))
	}
	return pref.ValueOf(m)
}
