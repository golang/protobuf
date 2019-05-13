// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"fmt"

	"github.com/golang/protobuf/v2/internal/encoding/wire"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// Size returns the size in bytes of the wire-format encoding of m.
func Size(m Message) int {
	return MarshalOptions{}.Size(m)
}

// Size returns the size in bytes of the wire-format encoding of m.
func (o MarshalOptions) Size(m Message) int {
	if size, err := sizeMessageFast(m); err == nil {
		return size
	}
	return sizeMessage(m.ProtoReflect())
}

func sizeMessageFast(m Message) (int, error) {
	// TODO: Pass MarshalOptions to size to permit disabling fast path?
	methods := protoMethods(m)
	if methods == nil || methods.Size == nil {
		return 0, errInternalNoFast
	}
	return methods.Size(m), nil
}

func sizeMessage(m protoreflect.Message) (size int) {
	fieldDescs := m.Descriptor().Fields()
	knownFields := m.KnownFields()
	m.KnownFields().Range(func(num protoreflect.FieldNumber, value protoreflect.Value) bool {
		field := fieldDescs.ByNumber(num)
		if field == nil {
			field = knownFields.ExtensionTypes().ByNumber(num).Descriptor()
			if field == nil {
				panic(fmt.Errorf("no descriptor for field %d in %q", num, m.Descriptor().FullName()))
			}
		}
		size += sizeField(field, value)
		return true
	})
	m.UnknownFields().Range(func(_ protoreflect.FieldNumber, raw protoreflect.RawFields) bool {
		size += len(raw)
		return true
	})
	return size
}

func sizeField(fd protoreflect.FieldDescriptor, value protoreflect.Value) (size int) {
	num := fd.Number()
	switch {
	case fd.IsList():
		return sizeList(num, fd, value.List())
	case fd.IsMap():
		return sizeMap(num, fd, value.Map())
	default:
		return wire.SizeTag(num) + sizeSingular(num, fd.Kind(), value)
	}
}

func sizeList(num wire.Number, fd protoreflect.FieldDescriptor, list protoreflect.List) (size int) {
	if fd.IsPacked() {
		content := 0
		for i, llen := 0, list.Len(); i < llen; i++ {
			content += sizeSingular(num, fd.Kind(), list.Get(i))
		}
		return wire.SizeTag(num) + wire.SizeBytes(content)
	}

	for i, llen := 0, list.Len(); i < llen; i++ {
		size += wire.SizeTag(num) + sizeSingular(num, fd.Kind(), list.Get(i))
	}
	return size
}

func sizeMap(num wire.Number, fd protoreflect.FieldDescriptor, mapv protoreflect.Map) (size int) {
	mapv.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
		size += wire.SizeTag(num)
		size += wire.SizeBytes(sizeField(fd.MapKey(), key.Value()) + sizeField(fd.MapValue(), value))
		return true
	})
	return size
}
