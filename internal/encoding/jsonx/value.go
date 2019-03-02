// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package json implements the JSON format.
// This package has no semantic understanding for protocol buffers and is only
// a parser and composer for the format.
//
// This follows RFC 7159, with some notable implementation specifics:
//	* numbers that are out of range result in a decoding error
//	* duplicate keys in objects are not rejected
//
// Reasons why the standard encoding/json package is not suitable:
//	* information about duplicate keys is lost
//	* invalid UTF-8 is silently coerced into utf8.RuneError
package json

import (
	"fmt"
	"strings"
)

// Type represents a type expressible in the JSON format.
type Type uint8

const (
	_ Type = iota
	// Null is the null literal (i.e., "null").
	Null
	// Bool is a boolean (i.e., "true" or "false").
	Bool
	// Number is a floating-point number (e.g., "1.234" or "1e100").
	Number
	// String is an escaped string (e.g., `"the quick brown fox"`).
	String
	// Array is an ordered list of values (e.g., `[0, "one", true]`).
	Array
	// Object is an ordered map of values (e.g., `{"key": null}`).
	Object
)

func (t Type) String() string {
	switch t {
	case Null:
		return "null"
	case Bool:
		return "bool"
	case Number:
		return "number"
	case String:
		return "string"
	case Array:
		return "array"
	case Object:
		return "object"
	default:
		return "<invalid>"
	}
}

// Value contains a value of a given Type.
type Value struct {
	typ Type
	raw []byte     // raw bytes of the serialized data
	str string     // only for String
	num float64    // only for Bool or Number
	arr []Value    // only for Array
	obj [][2]Value // only for Object
}

// ValueOf returns a Value for a given Go value:
//	nil                =>  Null
//	bool               =>  Bool
//	int32, int64       =>  Number
//	uint32, uint64     =>  Number
//	float32, float64   =>  Number
//	string, []byte     =>  String
//	[]Value            =>  Array
//	[][2]Value         =>  Object
//
// ValueOf panics if the Go type is not one of the above.
func ValueOf(v interface{}) Value {
	switch v := v.(type) {
	case nil:
		return Value{typ: Null}
	case bool:
		if v {
			return Value{typ: Bool, num: 1}
		} else {
			return Value{typ: Bool, num: 0}
		}
	case int32:
		return Value{typ: Number, num: float64(v)}
	case int64:
		return Value{typ: Number, num: float64(v)} // possible loss of precision
	case uint32:
		return Value{typ: Number, num: float64(v)}
	case uint64:
		return Value{typ: Number, num: float64(v)} // possible loss of precision
	case float32:
		return Value{typ: Number, num: float64(v)}
	case float64:
		return Value{typ: Number, num: float64(v)}
	case string:
		return Value{typ: String, str: string(v)}
	case []byte:
		return Value{typ: String, str: string(v)}
	case []Value:
		return Value{typ: Array, arr: v}
	case [][2]Value:
		return Value{typ: Object, obj: v}
	default:
		panic(fmt.Sprintf("invalid type %T", v))
	}
}
func rawValueOf(v interface{}, raw []byte) Value {
	v2 := ValueOf(v)
	v2.raw = raw
	return v2
}

// Type is the type of the value.
func (v Value) Type() Type {
	return v.typ
}

// Bool returns v as a bool and panics if it is not a Bool.
func (v Value) Bool() bool {
	if v.typ != Bool {
		panic("value is not a boolean")
	}
	return v.num != 0
}

// Number returns v as a float64 and panics if it is not a Number.
func (v Value) Number() float64 {
	if v.typ != Number {
		panic("value is not a number")
	}
	return v.num
}

// String returns v as a string if the Type is String.
// Otherwise, this returns a formatted string of v for debugging purposes.
//
// Since JSON strings must be UTF-8, the marshaler and unmarshaler will verify
// for UTF-8 correctness.
func (v Value) String() string {
	if v.typ != String {
		return v.stringValue()
	}
	return v.str
}
func (v Value) stringValue() string {
	switch v.typ {
	case Null, Bool, Number:
		return string(v.Raw())
	case Array:
		var ss []string
		for _, v := range v.Array() {
			ss = append(ss, v.String())
		}
		return "[" + strings.Join(ss, ",") + "]"
	case Object:
		var ss []string
		for _, v := range v.Object() {
			ss = append(ss, v[0].String()+":"+v[1].String())
		}
		return "{" + strings.Join(ss, ",") + "}"
	default:
		return "<invalid>"
	}
}

// Array returns the elements of v and panics if the Type is not Array.
// Mutations on the return value may not be observable from the Raw method.
func (v Value) Array() []Value {
	if v.typ != Array {
		panic("value is not an array")
	}
	return v.arr
}

// Object returns the items of v and panics if the Type is not Object.
// The [2]Value represents a key (of type String) and value pair.
//
// Mutations on the return value may not be observable from the Raw method.
func (v Value) Object() [][2]Value {
	if v.typ != Object {
		panic("value is not an object")
	}
	return v.obj
}

// Raw returns the raw representation of the value.
// The returned value may alias the input given to Unmarshal.
func (v Value) Raw() []byte {
	if len(v.raw) > 0 {
		return v.raw
	}
	p := encoder{}
	if err := p.marshalValue(v); !p.nerr.Merge(err) {
		return []byte("<invalid>")
	}
	return p.out
}
