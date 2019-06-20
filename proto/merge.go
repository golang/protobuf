// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import "google.golang.org/protobuf/reflect/protoreflect"

// Merge merges src into dst, which must be messages with the same descriptor.
//
// Populated scalar fields in src are copied to dst, while populated
// singular messages in src are merged into dst by recursively calling Merge.
// The elements of every list field in src is appended to the corresponded
// list fields in dst. The entries of every map field in src is copied into
// the corresponding map field in dst, possibly replacing existing entries.
// The unknown fields of src are appended to the unknown fields of dst.
func Merge(dst, src Message) {
	mergeMessage(dst.ProtoReflect(), src.ProtoReflect())
}

func mergeMessage(dst, src protoreflect.Message) {
	if dst.Descriptor() != src.Descriptor() {
		panic("descriptor mismatch")
	}

	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList():
			mergeList(dst.Mutable(fd).List(), v.List(), fd)
		case fd.IsMap():
			mergeMap(dst.Mutable(fd).Map(), v.Map(), fd.MapValue())
		case fd.Message() != nil:
			mergeMessage(dst.Mutable(fd).Message(), v.Message())
		case fd.Kind() == protoreflect.BytesKind:
			dst.Set(fd, cloneBytes(v))
		default:
			dst.Set(fd, v)
		}
		return true
	})

	dst.SetUnknown(append(dst.GetUnknown(), src.GetUnknown()...))
}

func mergeList(dst, src protoreflect.List, fd protoreflect.FieldDescriptor) {
	for i := 0; i < src.Len(); i++ {
		switch v := src.Get(i); {
		case fd.Message() != nil:
			m := dst.NewMessage()
			mergeMessage(m, v.Message())
			dst.Append(protoreflect.ValueOf(m))
		case fd.Kind() == protoreflect.BytesKind:
			dst.Append(cloneBytes(v))
		default:
			dst.Append(v)
		}
	}
}

func mergeMap(dst, src protoreflect.Map, fd protoreflect.FieldDescriptor) {
	src.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		switch {
		case fd.Message() != nil:
			m := dst.NewMessage()
			mergeMessage(m, v.Message())
			dst.Set(k, protoreflect.ValueOf(m)) // may replace existing entry
		case fd.Kind() == protoreflect.BytesKind:
			dst.Set(k, cloneBytes(v))
		default:
			dst.Set(k, v)
		}
		return true
	})
}

func cloneBytes(v protoreflect.Value) protoreflect.Value {
	return protoreflect.ValueOf(append([]byte{}, v.Bytes()...))
}
