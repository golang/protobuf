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

func sizeField(field protoreflect.FieldDescriptor, value protoreflect.Value) (size int) {
	num := field.Number()
	kind := field.Kind()
	switch {
	case field.Cardinality() != protoreflect.Repeated:
		return wire.SizeTag(num) + sizeSingular(num, kind, value)
	case field.IsMap():
		return sizeMap(num, kind, field.Message(), value.Map())
	case field.IsPacked():
		return sizePacked(num, kind, value.List())
	default:
		return sizeList(num, kind, value.List())
	}
}

func sizeMap(num wire.Number, kind protoreflect.Kind, mdesc protoreflect.MessageDescriptor, mapv protoreflect.Map) (size int) {
	keyf := mdesc.Fields().ByNumber(1)
	valf := mdesc.Fields().ByNumber(2)
	mapv.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
		size += wire.SizeTag(num)
		size += wire.SizeBytes(sizeField(keyf, key.Value()) + sizeField(valf, value))
		return true
	})
	return size
}

func sizePacked(num wire.Number, kind protoreflect.Kind, list protoreflect.List) (size int) {
	content := 0
	for i, llen := 0, list.Len(); i < llen; i++ {
		content += sizeSingular(num, kind, list.Get(i))
	}
	return wire.SizeTag(num) + wire.SizeBytes(content)
}

func sizeList(num wire.Number, kind protoreflect.Kind, list protoreflect.List) (size int) {
	for i, llen := 0, list.Len(); i < llen; i++ {
		size += wire.SizeTag(num) + sizeSingular(num, kind, list.Get(i))
	}
	return size
}
