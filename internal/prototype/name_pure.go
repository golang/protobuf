// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build purego appengine

package prototype

import pref "github.com/golang/protobuf/v2/reflect/protoreflect"

func getNameBuilder() *nameBuilder { return nil }
func putNameBuilder(*nameBuilder)  {}

type nameBuilder struct{}

// Append is equivalent to protoreflect.FullName.Append.
func (*nameBuilder) Append(prefix pref.FullName, name pref.Name) pref.FullName {
	return prefix.Append(name)
}
