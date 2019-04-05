// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"fmt"
	"sort"

	"github.com/golang/protobuf/v2/internal/encoding/wire"
	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/internal/mapsort"
	"github.com/golang/protobuf/v2/internal/pragma"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/runtime/protoiface"
)

// MarshalOptions configures the marshaler.
//
// Example usage:
//   b, err := MarshalOptions{Deterministic: true}.Marshal(m)
type MarshalOptions struct {
	// AllowPartial allows messages that have missing required fields to marshal
	// without returning an error. If AllowPartial is false (the default),
	// Marshal will return an error if there are any missing required fields.
	AllowPartial bool

	// Deterministic controls whether the same message will always be
	// serialized to the same bytes within the same binary.
	//
	// Setting this option guarantees that repeated serialization of
	// the same message will return the same bytes, and that different
	// processes of the same binary (which may be executing on different
	// machines) will serialize equal messages to the same bytes.
	//
	// Note that the deterministic serialization is NOT canonical across
	// languages. It is not guaranteed to remain stable over time. It is
	// unstable across different builds with schema changes due to unknown
	// fields. Users who need canonical serialization (e.g., persistent
	// storage in a canonical form, fingerprinting, etc.) must define
	// their own canonicalization specification and implement their own
	// serializer rather than relying on this API.
	//
	// If deterministic serialization is requested, map entries will be
	// sorted by keys in lexographical order. This is an implementation
	// detail and subject to change.
	Deterministic bool

	// Reflection forces use of the reflection-based encoder, even for
	// messages which implement fast-path serialization.
	Reflection bool

	pragma.NoUnkeyedLiterals
}

var _ = protoiface.MarshalOptions(MarshalOptions{})

// Marshal returns the wire-format encoding of m.
func Marshal(m Message) ([]byte, error) {
	return MarshalOptions{}.MarshalAppend(nil, m)
}

// Marshal returns the wire-format encoding of m.
func (o MarshalOptions) Marshal(m Message) ([]byte, error) {
	return o.MarshalAppend(nil, m)
}

// MarshalAppend appends the wire-format encoding of m to b,
// returning the result.
func (o MarshalOptions) MarshalAppend(b []byte, m Message) ([]byte, error) {
	b, err := o.marshalMessageFast(b, m)
	if err == errInternalNoFast {
		b, err = o.marshalMessage(b, m.ProtoReflect())
	}
	var nerr errors.NonFatal
	if !nerr.Merge(err) {
		return b, err
	}
	if !o.AllowPartial {
		nerr.Merge(IsInitialized(m))
	}
	return b, nerr.E
}

func (o MarshalOptions) marshalMessageFast(b []byte, m Message) ([]byte, error) {
	if o.Reflection {
		return nil, errInternalNoFast
	}
	methods := protoMethods(m)
	if methods == nil ||
		methods.MarshalAppend == nil ||
		(o.Deterministic && methods.Flags&protoiface.MethodFlagDeterministicMarshal == 0) {
		return nil, errInternalNoFast
	}
	if methods.Size != nil {
		sz := methods.Size(m)
		if cap(b) < len(b)+sz {
			x := make([]byte, len(b), len(b)+sz)
			copy(x, b)
			b = x
		}
	}
	return methods.MarshalAppend(b, m, protoiface.MarshalOptions(o))
}

func (o MarshalOptions) marshalMessage(b []byte, m protoreflect.Message) ([]byte, error) {
	// There are many choices for what order we visit fields in. The default one here
	// is chosen for reasonable efficiency and simplicity given the protoreflect API.
	// It is not deterministic, since KnownFields.Range does not return fields in any
	// defined order.
	//
	// When using deterministic serialization, we sort the known fields by field number.
	fields := m.Type().Fields()
	knownFields := m.KnownFields()
	var err error
	var nerr errors.NonFatal
	o.rangeKnown(knownFields, func(num protoreflect.FieldNumber, value protoreflect.Value) bool {
		field := fields.ByNumber(num)
		if field == nil {
			field = knownFields.ExtensionTypes().ByNumber(num)
			if field == nil {
				panic(fmt.Errorf("no descriptor for field %d in %q", num, m.Type().FullName()))
			}
		}
		b, err = o.marshalField(b, field, value)
		if nerr.Merge(err) {
			err = nil
			return true
		}
		return false
	})
	if err != nil {
		return b, err
	}
	m.UnknownFields().Range(func(_ protoreflect.FieldNumber, raw protoreflect.RawFields) bool {
		b = append(b, raw...)
		return true
	})
	return b, nerr.E
}

// rangeKnown visits known fields in field number order when deterministic
// serialization is enabled.
func (o MarshalOptions) rangeKnown(knownFields protoreflect.KnownFields, f func(protoreflect.FieldNumber, protoreflect.Value) bool) {
	if !o.Deterministic {
		knownFields.Range(f)
		return
	}
	nums := make([]protoreflect.FieldNumber, 0, knownFields.Len())
	knownFields.Range(func(num protoreflect.FieldNumber, _ protoreflect.Value) bool {
		nums = append(nums, num)
		return true
	})
	sort.Slice(nums, func(a, b int) bool {
		return nums[a] < nums[b]
	})
	for _, num := range nums {
		if !f(num, knownFields.Get(num)) {
			break
		}
	}
}

func (o MarshalOptions) marshalField(b []byte, field protoreflect.FieldDescriptor, value protoreflect.Value) ([]byte, error) {
	num := field.Number()
	kind := field.Kind()
	switch {
	case field.Cardinality() != protoreflect.Repeated:
		b = wire.AppendTag(b, num, wireTypes[kind])
		return o.marshalSingular(b, num, kind, value)
	case field.IsMap():
		return o.marshalMap(b, num, kind, field.MessageType(), value.Map())
	case field.IsPacked():
		return o.marshalPacked(b, num, kind, value.List())
	default:
		return o.marshalList(b, num, kind, value.List())
	}
}

func (o MarshalOptions) marshalMap(b []byte, num wire.Number, kind protoreflect.Kind, mdesc protoreflect.MessageDescriptor, mapv protoreflect.Map) ([]byte, error) {
	keyf := mdesc.Fields().ByNumber(1)
	valf := mdesc.Fields().ByNumber(2)
	var nerr errors.NonFatal
	var err error
	o.rangeMap(mapv, keyf.Kind(), func(key protoreflect.MapKey, value protoreflect.Value) bool {
		b = wire.AppendTag(b, num, wire.BytesType)
		var pos int
		b, pos = appendSpeculativeLength(b)

		b, err = o.marshalField(b, keyf, key.Value())
		if !nerr.Merge(err) {
			return false
		}
		b, err = o.marshalField(b, valf, value)
		if !nerr.Merge(err) {
			return false
		}
		err = nil

		b = finishSpeculativeLength(b, pos)
		return true
	})
	if err != nil {
		return b, err
	}
	return b, nerr.E
}

func (o MarshalOptions) rangeMap(mapv protoreflect.Map, kind protoreflect.Kind, f func(protoreflect.MapKey, protoreflect.Value) bool) {
	if !o.Deterministic {
		mapv.Range(f)
		return
	}
	mapsort.Range(mapv, kind, f)
}

func (o MarshalOptions) marshalPacked(b []byte, num wire.Number, kind protoreflect.Kind, list protoreflect.List) ([]byte, error) {
	b = wire.AppendTag(b, num, wire.BytesType)
	b, pos := appendSpeculativeLength(b)
	var nerr errors.NonFatal
	for i, llen := 0, list.Len(); i < llen; i++ {
		var err error
		b, err = o.marshalSingular(b, num, kind, list.Get(i))
		if !nerr.Merge(err) {
			return b, err
		}
	}
	b = finishSpeculativeLength(b, pos)
	return b, nerr.E
}

func (o MarshalOptions) marshalList(b []byte, num wire.Number, kind protoreflect.Kind, list protoreflect.List) ([]byte, error) {
	var nerr errors.NonFatal
	for i, llen := 0, list.Len(); i < llen; i++ {
		var err error
		b = wire.AppendTag(b, num, wireTypes[kind])
		b, err = o.marshalSingular(b, num, kind, list.Get(i))
		if !nerr.Merge(err) {
			return b, err
		}
	}
	return b, nerr.E
}

// When encoding length-prefixed fields, we speculatively set aside some number of bytes
// for the length, encode the data, and then encode the length (shifting the data if necessary
// to make room).
const speculativeLength = 1

func appendSpeculativeLength(b []byte) ([]byte, int) {
	pos := len(b)
	b = append(b, "\x00\x00\x00\x00"[:speculativeLength]...)
	return b, pos
}

func finishSpeculativeLength(b []byte, pos int) []byte {
	mlen := len(b) - pos - speculativeLength
	msiz := wire.SizeVarint(uint64(mlen))
	if msiz != speculativeLength {
		for i := 0; i < msiz-speculativeLength; i++ {
			b = append(b, 0)
		}
		copy(b[pos+msiz:], b[pos+speculativeLength:])
		b = b[:pos+msiz+mlen]
	}
	wire.AppendVarint(b[:pos], uint64(mlen))
	return b
}
