// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/runtime/protoimpl"
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
	fieldTypes := m.Type().Fields()
	knownFields := m.KnownFields()
	knownFields.Range(func(num pref.FieldNumber, val pref.Value) bool {
		fd := fieldTypes.ByNumber(num)
		if fd == nil {
			fd = knownFields.ExtensionTypes().ByNumber(num)
		}
		switch {
		// Handle singular message.
		case fd.Cardinality() != pref.Repeated:
			if k := fd.Kind(); k == pref.MessageKind || k == pref.GroupKind {
				discardUnknown(knownFields.Get(num).Message())
			}
		// Handle list of messages.
		case !fd.IsMap():
			if k := fd.Kind(); k == pref.MessageKind || k == pref.GroupKind {
				ls := knownFields.Get(num).List()
				for i := 0; i < ls.Len(); i++ {
					discardUnknown(ls.Get(i).Message())
				}
			}
		// Handle map of messages.
		default:
			k := fd.Message().Fields().ByNumber(2).Kind()
			if k == pref.MessageKind || k == pref.GroupKind {
				ms := knownFields.Get(num).Map()
				ms.Range(func(_ pref.MapKey, v pref.Value) bool {
					discardUnknown(v.Message())
					return true
				})
			}
		}
		return true
	})

	extRanges := m.Type().ExtensionRanges()
	unknownFields := m.UnknownFields()
	unknownFields.Range(func(num pref.FieldNumber, _ pref.RawFields) bool {
		// NOTE: Historically, this function did not discard unknown fields
		// that were within the extension field ranges.
		if !extRanges.Has(num) {
			unknownFields.Set(num, nil)
		}
		return true
	})
}
