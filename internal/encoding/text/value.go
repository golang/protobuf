// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package text implements the text format for protocol buffers.
// This package has no semantic understanding for protocol buffers and is only
// a parser and composer for the format.
//
// There is no formal specification for the protobuf text format, as such the
// C++ implementation (see google::protobuf::TextFormat) is the reference
// implementation of the text format.
//
// This package is neither a superset nor a subset of the C++ implementation.
// This implementation permits a more liberal grammar in some cases to be
// backwards compatible with the historical Go implementation.
// Future parsings unique to Go should not be added.
// Some grammars allowed by the C++ implementation are deliberately
// not implemented here because they are considered a bug by the protobuf team
// and should not be replicated.
//
// The Go implementation should implement a sufficient amount of the C++
// grammar such that the default text serialization by C++ can be parsed by Go.
// However, just because the C++ parser accepts some input does not mean that
// the Go implementation should as well.
//
// The text format is almost a superset of JSON except:
//	* message keys are not quoted strings, but identifiers
//	* the top-level value must be a message without the delimiters
package text

import (
	"fmt"
	"math"
	"strings"

	"github.com/golang/protobuf/v2/internal/flags"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// Type represents a type expressible in the text format.
type Type uint8

const (
	_ Type = iota

	// Bool is a boolean (e.g., "true" or "false").
	Bool
	// Int is a signed integer (e.g., "-1423").
	Int
	// Uint is an unsigned integer (e.g., "0xdeadbeef").
	Uint
	// Float32 is a 32-bit floating-point number (e.g., "1.234" or "1e38").
	// This allows encoding to differentiate the bitsize used for formatting.
	Float32
	// Float64 is a 64-bit floating-point number.
	Float64
	// String is a quoted string (e.g., `"the quick brown fox"`).
	String
	// Name is a protocol buffer identifier (e.g., `field_name`).
	Name
	// List is an ordered list of values (e.g., `[0, "one", true]`).
	List
	// Message is an ordered map of values (e.g., `{"key": null}`).
	Message
)

func (t Type) String() string {
	switch t {
	case Bool:
		return "bool"
	case Int:
		return "int"
	case Uint:
		return "uint"
	case Float32:
		return "float32"
	case Float64:
		return "float64"
	case String:
		return "string"
	case Name:
		return "name"
	case List:
		return "list"
	case Message:
		return "message"
	default:
		return "<invalid>"
	}
}

// Value contains a value of a given Type.
type Value struct {
	typ Type
	raw []byte     // raw bytes of the serialized data
	str string     // only for String or Name
	num uint64     // only for Bool, Int, Uint, Float32, or Float64
	arr []Value    // only for List
	obj [][2]Value // only for Message
}

// ValueOf returns a Value for a given Go value:
//	bool               =>  Bool
//	int32, int64       =>  Int
//	uint32, uint64     =>  Uint
//	float32            =>  Float32
//	float64            =>  Float64
//	string, []byte     =>  String
//	protoreflect.Name  =>  Name
//	[]Value            =>  List
//	[][2]Value         =>  Message
//
// ValueOf panics if the Go type is not one of the above.
func ValueOf(v interface{}) Value {
	switch v := v.(type) {
	case bool:
		if v {
			return Value{typ: Bool, num: 1}
		} else {
			return Value{typ: Bool, num: 0}
		}
	case int32:
		return Value{typ: Int, num: uint64(v)}
	case int64:
		return Value{typ: Int, num: uint64(v)}
	case uint32:
		return Value{typ: Uint, num: uint64(v)}
	case uint64:
		return Value{typ: Uint, num: uint64(v)}
	case float32:
		// Store as float64 bits.
		return Value{typ: Float32, num: math.Float64bits(float64(v))}
	case float64:
		return Value{typ: Float64, num: math.Float64bits(float64(v))}
	case string:
		return Value{typ: String, str: string(v)}
	case []byte:
		return Value{typ: String, str: string(v)}
	case protoreflect.Name:
		return Value{typ: Name, str: string(v)}
	case []Value:
		return Value{typ: List, arr: v}
	case [][2]Value:
		return Value{typ: Message, obj: v}
	default:
		panic(fmt.Sprintf("invalid type %T", v))
	}
}
func rawValueOf(v interface{}, raw []byte) Value {
	v2 := ValueOf(v)
	v2.raw = raw
	return v2
}

// Type is the type of the value. When parsing, this is a best-effort guess
// at the resulting type. However, there are ambiguities as to the exact type
// of the value (e.g., "false" is either a bool or a name).
// Thus, some of the types are convertible with each other.
// The Bool, Int, Uint, Float32, Float64, and Name methods return a boolean to
// report whether the conversion was successful.
func (v Value) Type() Type {
	return v.typ
}

// Bool returns v as a bool and reports whether the conversion succeeded.
func (v Value) Bool() (x bool, ok bool) {
	switch v.typ {
	case Bool:
		return v.num > 0, true
	case Uint, Int:
		// C++ allows a 1-bit unsigned integer (e.g., "0", "1", or "0x1").
		if len(v.raw) > 0 && v.raw[0] != '-' && v.num < 2 {
			return v.num > 0, true
		}
	}
	return false, false
}

// Int returns v as an int64 of the specified precision and reports whether
// the conversion succeeded.
func (v Value) Int(b64 bool) (x int64, ok bool) {
	switch v.typ {
	case Int:
		n := int64(v.num)
		if b64 || (math.MinInt32 <= n && n <= math.MaxInt32) {
			return int64(n), true
		}
	case Uint:
		n := uint64(v.num)
		if (!b64 && n <= math.MaxInt32) || (b64 && n <= math.MaxInt64) {
			return int64(n), true
		}
		// C++ accepts large positive hex numbers as negative values.
		// This feature is here for proto1 backwards compatibility purposes.
		if flags.Proto1Legacy && len(v.raw) > 1 && v.raw[0] == '0' && v.raw[1] == 'x' {
			if !b64 {
				return int64(int32(n)), n <= math.MaxUint32
			}
			// if !b64 && n <= math.MaxUint32 {
			// 	return int64(int32(n)), true
			// }
			return int64(n), true
		}
	}
	return 0, false
}

// Uint returns v as an uint64 of the specified precision and reports whether
// the conversion succeeded.
func (v Value) Uint(b64 bool) (x uint64, ok bool) {
	switch v.typ {
	case Int:
		n := int64(v.num)
		if len(v.raw) > 0 && v.raw[0] != '-' && (b64 || n <= math.MaxUint32) {
			return uint64(n), true
		}
	case Uint:
		n := uint64(v.num)
		if b64 || n <= math.MaxUint32 {
			return uint64(n), true
		}
	}
	return 0, false
}

// Float32 returns v as a float32 of the specified precision and reports whether
// the conversion succeeded.
func (v Value) Float32() (x float32, ok bool) {
	switch v.typ {
	case Int:
		return float32(int64(v.num)), true // possibly lossy, but allowed
	case Uint:
		return float32(uint64(v.num)), true // possibly lossy, but allowed
	case Float32, Float64:
		n := math.Float64frombits(v.num)
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return float32(n), true
		}
		if math.Abs(n) <= math.MaxFloat32 {
			return float32(n), true
		}
	}
	return 0, false
}

// Float64 returns v as a float64 of the specified precision and reports whether
// the conversion succeeded.
func (v Value) Float64() (x float64, ok bool) {
	switch v.typ {
	case Int:
		return float64(int64(v.num)), true // possibly lossy, but allowed
	case Uint:
		return float64(uint64(v.num)), true // possibly lossy, but allowed
	case Float32:
		f, ok := v.Float32()
		return float64(f), ok
	case Float64:
		n := math.Float64frombits(v.num)
		return n, true
	}
	return 0, false
}

// String returns v as a string if the Type is String.
// Otherwise, this returns a formatted string of v for debugging purposes.
//
// Since String is used to represent both text and binary, it is not validated
// to contain valid UTF-8. When using this value with the string type in proto,
// it is the user's responsibility perform additional UTF-8 validation.
func (v Value) String() string {
	if v.typ != String {
		return v.stringValue()
	}
	return v.str
}
func (v Value) stringValue() string {
	switch v.typ {
	case Bool, Int, Uint, Float32, Float64, Name:
		return string(v.Raw())
	case List:
		var ss []string
		for _, v := range v.List() {
			ss = append(ss, v.String())
		}
		return "[" + strings.Join(ss, ",") + "]"
	case Message:
		var ss []string
		for _, v := range v.Message() {
			k := v[0].String()
			if v[0].Type() == String {
				k = "[" + k + "]"
			}
			ss = append(ss, k+":"+v[1].String())
		}
		return "{" + strings.Join(ss, ",") + "}"
	default:
		return "<invalid>"
	}
}

// Name returns the field name or enum value name and reports whether the value
// can be treated as an identifier.
func (v Value) Name() (protoreflect.Name, bool) {
	switch v.typ {
	case Bool, Float32, Float64:
		// Ambiguity arises in unmarshalValue since "nan" may interpreted as
		// either a Name type (for enum values) or a Float32/Float64 type.
		// Similarly, "true" may be interpreted as either a Name or Bool type.
		n := protoreflect.Name(v.raw)
		if n.IsValid() {
			return n, true
		}
	case Name:
		return protoreflect.Name(v.str), true
	}
	return "", false
}

// List returns the elements of v and panics if the Type is not List.
// Mutations on the return value may not be observable from the Raw method.
func (v Value) List() []Value {
	if v.typ != List {
		panic("value is not a list")
	}
	return v.arr
}

// Message returns the items of v and panics if the Type is not Message.
// The [2]Value represents a key and value pair, where the key is either
// a Name (representing a field name), a String (representing extension field
// names or the Any type URL), or an Uint for unknown fields.
//
// Mutations on the return value may not be observable from the Raw method.
func (v Value) Message() [][2]Value {
	if v.typ != Message {
		panic("value is not a message")
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
