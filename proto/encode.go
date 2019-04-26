// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"sort"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/mapsort"
	"google.golang.org/protobuf/internal/pragma"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoiface"
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

	// UseCachedSize indicates that the result of a previous Size call
	// may be reused.
	//
	// Setting this option asserts that:
	//
	// 1. Size has previously been called on this message with identical
	// options (except for UseCachedSize itself).
	//
	// 2. The message and all its submessages have not changed in any
	// way since the Size call.
	//
	// If either of these invariants is broken, the results are undefined
	// but may include panics or invalid output.
	//
	// Implementations MAY take this option into account to provide
	// better performance, but there is no guarantee that they will do so.
	// There is absolutely no guarantee that Size followed by Marshal with
	// UseCachedSize set will perform equivalently to Marshal alone.
	UseCachedSize bool

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
	// Set AllowPartial in recursive calls to marshal to avoid duplicating
	// effort with the single initialization check below.
	allowPartial := o.AllowPartial
	o.AllowPartial = true
	out, err := o.marshalMessageFast(b, m)
	if err == errInternalNoFast {
		out, err = o.marshalMessage(b, m.ProtoReflect())
	}
	var nerr errors.NonFatal
	if !nerr.Merge(err) {
		return out, err
	}
	if !allowPartial {
		nerr.Merge(IsInitialized(m))
	}
	return out, nerr.E
}

func (o MarshalOptions) marshalMessageFast(b []byte, m Message) ([]byte, error) {
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
		o.UseCachedSize = true
	}
	return methods.MarshalAppend(b, m, protoiface.MarshalOptions(o))
}

func (o MarshalOptions) marshalMessage(b []byte, m protoreflect.Message) ([]byte, error) {
	// There are many choices for what order we visit fields in. The default one here
	// is chosen for reasonable efficiency and simplicity given the protoreflect API.
	// It is not deterministic, since Message.Range does not return fields in any
	// defined order.
	//
	// When using deterministic serialization, we sort the known fields by field number.
	var err error
	var nerr errors.NonFatal
	o.rangeFields(m, func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		b, err = o.marshalField(b, fd, v)
		if nerr.Merge(err) {
			err = nil
			return true
		}
		return false
	})
	if err != nil {
		return b, err
	}
	b = append(b, m.GetUnknown()...)
	return b, nerr.E
}

// rangeFields visits fields in field number order when deterministic
// serialization is enabled.
func (o MarshalOptions) rangeFields(m protoreflect.Message, f func(protoreflect.FieldDescriptor, protoreflect.Value) bool) {
	if !o.Deterministic {
		m.Range(f)
		return
	}
	fds := make([]protoreflect.FieldDescriptor, 0, m.Len())
	m.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		fds = append(fds, fd)
		return true
	})
	sort.Slice(fds, func(a, b int) bool {
		return fds[a].Number() < fds[b].Number()
	})
	for _, fd := range fds {
		if !f(fd, m.Get(fd)) {
			break
		}
	}
}

func (o MarshalOptions) marshalField(b []byte, fd protoreflect.FieldDescriptor, value protoreflect.Value) ([]byte, error) {
	switch {
	case fd.IsList():
		return o.marshalList(b, fd, value.List())
	case fd.IsMap():
		return o.marshalMap(b, fd, value.Map())
	default:
		b = wire.AppendTag(b, fd.Number(), wireTypes[fd.Kind()])
		return o.marshalSingular(b, fd, value)
	}
}

func (o MarshalOptions) marshalList(b []byte, fd protoreflect.FieldDescriptor, list protoreflect.List) ([]byte, error) {
	if fd.IsPacked() && list.Len() > 0 {
		b = wire.AppendTag(b, fd.Number(), wire.BytesType)
		b, pos := appendSpeculativeLength(b)
		var nerr errors.NonFatal
		for i, llen := 0, list.Len(); i < llen; i++ {
			var err error
			b, err = o.marshalSingular(b, fd, list.Get(i))
			if !nerr.Merge(err) {
				return b, err
			}
		}
		b = finishSpeculativeLength(b, pos)
		return b, nerr.E
	}

	kind := fd.Kind()
	var nerr errors.NonFatal
	for i, llen := 0, list.Len(); i < llen; i++ {
		var err error
		b = wire.AppendTag(b, fd.Number(), wireTypes[kind])
		b, err = o.marshalSingular(b, fd, list.Get(i))
		if !nerr.Merge(err) {
			return b, err
		}
	}
	return b, nerr.E
}

func (o MarshalOptions) marshalMap(b []byte, fd protoreflect.FieldDescriptor, mapv protoreflect.Map) ([]byte, error) {
	keyf := fd.MapKey()
	valf := fd.MapValue()
	var nerr errors.NonFatal
	var err error
	o.rangeMap(mapv, keyf.Kind(), func(key protoreflect.MapKey, value protoreflect.Value) bool {
		b = wire.AppendTag(b, fd.Number(), wire.BytesType)
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
