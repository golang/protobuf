// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/runtime/protoimpl"
)

// SetDefaults sets unset protocol buffer fields to their default values.
// It only modifies fields that are both unset and have defined defaults.
// It recursively sets default values in any non-nil sub-messages.
// It does not descend into extension fields that are sub-messages.
func SetDefaults(m Message) {
	setDefaults(protoimpl.X.MessageOf(m))
}

func setDefaults(m pref.Message) {
	fieldDescs := m.Descriptor().Fields()
	knownFields := m.KnownFields()
	for i := 0; i < fieldDescs.Len(); i++ {
		fd := fieldDescs.Get(i)
		num := fd.Number()
		if !knownFields.Has(num) {
			if fd.HasDefault() {
				v := fd.Default()
				if fd.Kind() == pref.BytesKind {
					v = pref.ValueOf(append([]byte(nil), v.Bytes()...)) // copy the default bytes
				}
				knownFields.Set(num, v)
			}
			continue
		}
		switch {
		// Handle singular message.
		case fd.Cardinality() != pref.Repeated:
			if k := fd.Kind(); k == pref.MessageKind || k == pref.GroupKind {
				setDefaults(knownFields.Get(num).Message())
			}
		// Handle list of messages.
		case !fd.IsMap():
			if k := fd.Kind(); k == pref.MessageKind || k == pref.GroupKind {
				ls := knownFields.Get(num).List()
				for i := 0; i < ls.Len(); i++ {
					setDefaults(ls.Get(i).Message())
				}
			}
		// Handle map of messages.
		default:
			k := fd.Message().Fields().ByNumber(2).Kind()
			if k == pref.MessageKind || k == pref.GroupKind {
				ms := knownFields.Get(num).Map()
				ms.Range(func(_ pref.MapKey, v pref.Value) bool {
					setDefaults(v.Message())
					return true
				})
			}
		}
	}

	// NOTE: Historically, this function has never set the defaults for
	// extension fields, nor recursively visited sub-messages of such fields.
}
