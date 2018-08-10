// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	pref "google.golang.org/proto/reflect/protoreflect"
)

// TODO: This is important to prevent users from creating invalid types,
// but is not functionality needed now.

func validateFile(t pref.FileDescriptor) error {
	return nil
}

func validateMessage(t pref.MessageDescriptor) error {
	return nil
}

func validateExtension(t pref.ExtensionDescriptor) error {
	return nil
}

func validateEnum(t pref.EnumDescriptor) error {
	return nil
}
