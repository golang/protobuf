// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

// Type represents a type expressible in the JSON format.
type Type uint

const (
	_ Type = (1 << iota) / 2
	EOF
	Null
	Bool
	Number
	String
	StartObject
	EndObject
	Name
	StartArray
	EndArray

	// comma is only for parsing in between values and should not be exposed.
	comma
)

func (t Type) String() string {
	switch t {
	case EOF:
		return "eof"
	case Null:
		return "null"
	case Bool:
		return "bool"
	case Number:
		return "number"
	case String:
		return "string"
	case StartObject:
		return "{"
	case EndObject:
		return "}"
	case Name:
		return "name"
	case StartArray:
		return "["
	case EndArray:
		return "]"
	}
	return "<invalid>"
}
