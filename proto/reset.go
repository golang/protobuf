// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style.
// license that can be found in the LICENSE file.

package proto

import "google.golang.org/protobuf/reflect/protoreflect"

// Reset clears every field in the message.
func Reset(m Message) {
	// TODO: Document memory aliasing guarantees.
	// TODO: Add fast-path for reset?
	resetMessage(m.ProtoReflect())
}

func resetMessage(m protoreflect.Message) {
	// Clear all known fields.
	fds := m.Descriptor().Fields()
	for i := 0; i < fds.Len(); i++ {
		m.Clear(fds.Get(i))
	}

	// Clear extension fields.
	m.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		m.Clear(fd)
		return true
	})

	// Clear unknown fields.
	m.SetUnknown(nil)
}
