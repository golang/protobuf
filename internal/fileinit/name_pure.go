// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build purego appengine

package fileinit

import pref "github.com/golang/protobuf/v2/reflect/protoreflect"

func getNameBuilder() *nameBuilder { return nil }
func putNameBuilder(*nameBuilder)  {}

type nameBuilder struct{}

// AppendFullName is equivalent to protoreflect.FullName.Append.
func (*nameBuilder) AppendFullName(prefix pref.FullName, name []byte) fullName {
	return fullName{
		shortLen: len(name),
		fullName: prefix.Append(pref.Name(name)),
	}
}

// MakeString is equivalent to string(b), but optimized for large batches
// with a shared lifetime.
func (*nameBuilder) MakeString(b []byte) string {
	return string(b)
}

// MakeJSONName creates a JSON name from the protobuf short name.
func (*nameBuilder) MakeJSONName(s pref.Name) string {
	var b []byte
	var wasUnderscore bool
	for i := 0; i < len(s); i++ { // proto identifiers are always ASCII
		c := s[i]
		if c != '_' {
			isLower := 'a' <= c && c <= 'z'
			if wasUnderscore && isLower {
				c -= 'a' - 'A'
			}
			b = append(b, c)
		}
		wasUnderscore = c == '_'
	}
	return string(b)
}
