// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"github.com/golang/protobuf/internal/wire"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// DiscardUnknown recursively discards all unknown fields from this message
// and all embedded messages.
//
// When unmarshaling a message with unrecognized fields, the tags and values
// of such fields are preserved in the Message. This allows a later call to
// marshal to be able to produce a message that continues to have those
// unrecognized fields. To avoid this, DiscardUnknown is used to
// explicitly clear the unknown fields after unmarshaling.
//
// For proto2 messages, the unknown fields of message extensions are only
// discarded from messages that have been accessed via GetExtension.
func DiscardUnknown(m Message) {
	if m == nil {
		return
	}
	discardUnknown(protoimpl.X.MessageOf(m))
}

func discardUnknown(m pref.Message) {
	m.Range(func(fd pref.FieldDescriptor, val pref.Value) bool {
		switch {
		// Handle singular message.
		case fd.Cardinality() != pref.Repeated:
			if fd.Message() != nil {
				discardUnknown(m.Get(fd).Message())
			}
		// Handle list of messages.
		case fd.IsList():
			if fd.Message() != nil {
				ls := m.Get(fd).List()
				for i := 0; i < ls.Len(); i++ {
					discardUnknown(ls.Get(i).Message())
				}
			}
		// Handle map of messages.
		case fd.IsMap():
			if fd.MapValue().Message() != nil {
				ms := m.Get(fd).Map()
				ms.Range(func(_ pref.MapKey, v pref.Value) bool {
					discardUnknown(v.Message())
					return true
				})
			}
		}
		return true
	})

	// Discard unknown fields.
	var bo pref.RawFields
	extRanges := m.Descriptor().ExtensionRanges()
	for bi := m.GetUnknown(); len(bi) > 0; {
		// NOTE: Historically, this function did not discard unknown fields
		// that were within the extension field ranges.
		num, _, n := wire.ConsumeField(bi)
		if extRanges.Has(num) {
			bo = append(bo, bi[:n]...)
		}
		bi = bi[n:]
	}
	if bi := m.GetUnknown(); len(bi) != len(bo) {
		m.SetUnknown(bo)
	}
}
