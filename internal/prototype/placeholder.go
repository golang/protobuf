// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import "github.com/golang/protobuf/v2/reflect/protoreflect"

// PlaceholderFile returns a placeholder protoreflect.FileType where
// only the Path and Package accessors are valid.
func PlaceholderFile(path string, pkg protoreflect.FullName) protoreflect.FileDescriptor {
	// TODO: Is Package needed for placeholders?
	return placeholderFile{path, placeholderName(pkg)}
}

// PlaceholderMessage returns a placeholder protoreflect.MessageType
// where only the Name and FullName accessors are valid.
//
// A placeholder can be used within File literals when referencing a message
// that is declared within that file.
func PlaceholderMessage(name protoreflect.FullName) protoreflect.MessageDescriptor {
	return placeholderMessage{placeholderName(name)}
}

// PlaceholderEnum returns a placeholder protoreflect.EnumType
// where only the Name and FullName accessors are valid.
//
// A placeholder can be used within File literals when referencing an enum
// that is declared within that file.
func PlaceholderEnum(name protoreflect.FullName) protoreflect.EnumDescriptor {
	return placeholderEnum{placeholderName(name)}
}
