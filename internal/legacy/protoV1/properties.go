// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoV1

// Constants that identify the encoding of a value on the wire.
const (
	WireVarint     = 0
	WireFixed64    = 1
	WireBytes      = 2
	WireStartGroup = 3
	WireEndGroup   = 4
	WireFixed32    = 5
)

type (
	oneofFuncsIface interface {
		XXX_OneofFuncs() (func(Message, *Buffer) error, func(Message, int, int, *Buffer) (bool, error), func(Message) int, []interface{})
	}
	oneofWrappersIface interface {
		XXX_OneofWrappers() []interface{}
	}
)
