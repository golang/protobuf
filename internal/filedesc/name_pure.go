// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build purego appengine

package filedesc

import pref "google.golang.org/protobuf/reflect/protoreflect"

func getNameBuilder() *nameBuilder { return nil }
func putNameBuilder(*nameBuilder)  {}

type nameBuilder struct{}

// MakeFullName converts b to a protoreflect.FullName,
// where b must start with a leading dot.
func (*nameBuilder) MakeFullName(b []byte) pref.FullName {
	if len(b) == 0 || b[0] != '.' {
		panic("name reference must be fully qualified")
	}
	return pref.FullName(b[1:])
}

// AppendFullName is equivalent to protoreflect.FullName.Append.
func (*nameBuilder) AppendFullName(prefix pref.FullName, name []byte) pref.FullName {
	return prefix.Append(pref.Name(name))
}

// MakeString is equivalent to string(b), but optimized for large batches
// with a shared lifetime.
func (*nameBuilder) MakeString(b []byte) string {
	return string(b)
}
