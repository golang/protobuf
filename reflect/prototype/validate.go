// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// TODO: This is important to prevent users from creating invalid types,
// but is not functionality needed now.
//
// Things to verify:
//	* Weak fields are only used if flags.Proto1Legacy is set
//	* Weak fields can only reference singular messages
//	(check if this the case for oneof fields)
//	* FieldDescriptor.MessageType cannot reference a remote type when the
//	remote name is a type within the local file.
//	* Default enum identifiers resolve to a declared number.
//	* Default values are only allowed in proto2.
//	* Default strings are valid UTF-8? Note that protoc does not check this.
//	* Field extensions are only valid in proto2, except when extending the
//	descriptor options.
//	* Remote enum and message types are actually found in imported files.
//	* Placeholder messages and types may only be for weak fields.
//	* Placeholder full names must be valid.
//	* The name of each descriptor must be valid.
//	* Options are of the correct Go type (e.g. *descriptorpb.MessageOptions).
// 	* len(ExtensionRangeOptions) <= len(ExtensionRanges)

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
