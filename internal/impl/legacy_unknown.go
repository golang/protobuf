// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"container/list"
	"reflect"
	"sort"

	papi "github.com/golang/protobuf/protoapi"
	"github.com/golang/protobuf/v2/internal/encoding/wire"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

var bytesType = reflect.TypeOf([]byte(nil))

func makeLegacyUnknownFieldsFunc(t reflect.Type) func(p *messageDataType) pref.UnknownFields {
	fu, ok := t.FieldByName("XXX_unrecognized")
	if !ok || fu.Type != bytesType {
		return nil
	}
	fieldOffset := offsetOf(fu)
	unkFunc := func(p *messageDataType) pref.UnknownFields {
		if p.p.IsNil() {
			return emptyUnknownFields{}
		}
		rv := p.p.Apply(fieldOffset).AsValueOf(bytesType)
		return (*legacyUnknownBytes)(rv.Interface().(*[]byte))
	}
	extFunc := makeLegacyExtensionMapFunc(t)
	if extFunc != nil {
		return func(p *messageDataType) pref.UnknownFields {
			if p.p.IsNil() {
				return emptyUnknownFields{}
			}
			return &legacyUnknownBytesAndExtensionMap{
				unkFunc(p), extFunc(p), p.mi.PBType.ExtensionRanges(),
			}
		}
	}
	return unkFunc
}

// legacyUnknownBytesAndExtensionMap is a wrapper around both XXX_unrecognized
// and also the extension field map.
type legacyUnknownBytesAndExtensionMap struct {
	u pref.UnknownFields
	x papi.ExtensionFields
	r pref.FieldRanges
}

func (fs *legacyUnknownBytesAndExtensionMap) Len() int {
	n := fs.u.Len()
	fs.x.Range(func(_ pref.FieldNumber, x papi.ExtensionField) bool {
		if len(x.Raw) > 0 {
			n++
		}
		return true
	})
	return n
}

func (fs *legacyUnknownBytesAndExtensionMap) Get(num pref.FieldNumber) (raw pref.RawFields) {
	if fs.r.Has(num) {
		return fs.x.Get(num).Raw
	}
	return fs.u.Get(num)
}

func (fs *legacyUnknownBytesAndExtensionMap) Set(num pref.FieldNumber, raw pref.RawFields) {
	if fs.r.Has(num) {
		x := fs.x.Get(num)
		x.Raw = raw
		fs.x.Set(num, x)
		return
	}
	fs.u.Set(num, raw)
}

func (fs *legacyUnknownBytesAndExtensionMap) Range(f func(pref.FieldNumber, pref.RawFields) bool) {
	// Range over unknown fields not in the extension range.
	// Create a closure around f to capture whether iteration terminated early.
	var stop bool
	fs.u.Range(func(n pref.FieldNumber, b pref.RawFields) bool {
		stop = stop || !f(n, b)
		return !stop
	})
	if stop {
		return
	}

	// Range over unknown fields in the extension range in ascending order
	// to ensure protoreflect.UnknownFields.Range remains deterministic.
	type entry struct {
		num pref.FieldNumber
		raw pref.RawFields
	}
	var xs []entry
	fs.x.Range(func(n pref.FieldNumber, x papi.ExtensionField) bool {
		if len(x.Raw) > 0 {
			xs = append(xs, entry{n, x.Raw})
		}
		return true
	})
	sort.Slice(xs, func(i, j int) bool { return xs[i].num < xs[j].num })
	for _, x := range xs {
		if !f(x.num, x.raw) {
			return
		}
	}
}

func (fs *legacyUnknownBytesAndExtensionMap) IsSupported() bool {
	return true
}

// legacyUnknownBytes is a wrapper around XXX_unrecognized that implements
// the protoreflect.UnknownFields interface. This is challenging since we are
// limited to a []byte, so we do not have much flexibility in the choice
// of data structure that would have been ideal.
type legacyUnknownBytes []byte

func (fs *legacyUnknownBytes) Len() int {
	// Runtime complexity: O(n)
	b := *fs
	m := map[pref.FieldNumber]bool{}
	for len(b) > 0 {
		num, _, n := wire.ConsumeField(b)
		m[num] = true
		b = b[n:]
	}
	return len(m)
}

func (fs *legacyUnknownBytes) Get(num pref.FieldNumber) (raw pref.RawFields) {
	// Runtime complexity: O(n)
	b := *fs
	for len(b) > 0 {
		num2, _, n := wire.ConsumeField(b)
		if num == num2 {
			raw = append(raw, b[:n]...)
		}
		b = b[n:]
	}
	return raw
}

func (fs *legacyUnknownBytes) Set(num pref.FieldNumber, raw pref.RawFields) {
	num2, _, _ := wire.ConsumeTag(raw)
	if len(raw) > 0 && (!raw.IsValid() || num != num2) {
		panic("invalid raw fields")
	}

	// Remove all current fields of num.
	// Runtime complexity: O(n)
	b := *fs
	out := (*fs)[:0]
	for len(b) > 0 {
		num2, _, n := wire.ConsumeField(b)
		if num != num2 {
			out = append(out, b[:n]...)
		}
		b = b[n:]
	}
	*fs = out

	// Append new fields of num.
	*fs = append(*fs, raw...)
}

func (fs *legacyUnknownBytes) Range(f func(pref.FieldNumber, pref.RawFields) bool) {
	type entry struct {
		num pref.FieldNumber
		raw pref.RawFields
	}

	// Collect up a list of all the raw fields.
	// We preserve the order such that the latest encountered fields
	// are presented at the end.
	//
	// Runtime complexity: O(n)
	b := *fs
	l := list.New()
	m := map[pref.FieldNumber]*list.Element{}
	for len(b) > 0 {
		num, _, n := wire.ConsumeField(b)
		if e, ok := m[num]; ok {
			x := e.Value.(*entry)
			x.raw = append(x.raw, b[:n]...)
			l.MoveToBack(e)
		} else {
			x := &entry{num: num}
			x.raw = append(x.raw, b[:n]...)
			m[num] = l.PushBack(x)
		}
		b = b[n:]
	}

	// Iterate over all the raw fields.
	// This ranges over a snapshot of the current state such that mutations
	// while ranging are not observable.
	//
	// Runtime complexity: O(n)
	for e := l.Front(); e != nil; e = e.Next() {
		x := e.Value.(*entry)
		if !f(x.num, x.raw) {
			return
		}
	}
}

func (fs *legacyUnknownBytes) IsSupported() bool {
	return true
}
