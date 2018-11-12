// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tag marshals and unmarshals the legacy struct tags as generated
// by historical versions of protoc-gen-go.
package tag

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	protoV1 "github.com/golang/protobuf/proto"
	descriptorV1 "github.com/golang/protobuf/protoc-gen-go/descriptor"
	ptext "github.com/golang/protobuf/v2/internal/encoding/text"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
)

var byteType = reflect.TypeOf(byte(0))

// Unmarshal decodes the tag into a prototype.Field.
//
// The goType is needed to determine the original protoreflect.Kind since the
// tag does not record sufficient information to determine that.
// The type is the underlying field type (e.g., a repeated field may be
// represented by []T, but the Go type passed in is just T).
// This does not populate the EnumType or MessageType (except for weak message).
//
// This function is a best effort attempt; parsing errors are ignored.
func Unmarshal(tag string, goType reflect.Type) ptype.Field {
	f := ptype.Field{Options: new(descriptorV1.FieldOptions)}
	for len(tag) > 0 {
		i := strings.IndexByte(tag, ',')
		if i < 0 {
			i = len(tag)
		}
		switch s := tag[:i]; {
		case strings.HasPrefix(s, "name="):
			f.Name = pref.Name(s[len("name="):])
		case strings.Trim(s, "0123456789") == "":
			n, _ := strconv.ParseUint(s, 10, 32)
			f.Number = pref.FieldNumber(n)
		case s == "opt":
			f.Cardinality = pref.Optional
		case s == "req":
			f.Cardinality = pref.Required
		case s == "rep":
			f.Cardinality = pref.Repeated
		case s == "varint":
			switch goType.Kind() {
			case reflect.Bool:
				f.Kind = pref.BoolKind
			case reflect.Int32:
				f.Kind = pref.Int32Kind
			case reflect.Int64:
				f.Kind = pref.Int64Kind
			case reflect.Uint32:
				f.Kind = pref.Uint32Kind
			case reflect.Uint64:
				f.Kind = pref.Uint64Kind
			}
		case s == "zigzag32":
			if goType.Kind() == reflect.Int32 {
				f.Kind = pref.Sint32Kind
			}
		case s == "zigzag64":
			if goType.Kind() == reflect.Int64 {
				f.Kind = pref.Sint64Kind
			}
		case s == "fixed32":
			switch goType.Kind() {
			case reflect.Int32:
				f.Kind = pref.Sfixed32Kind
			case reflect.Uint32:
				f.Kind = pref.Fixed32Kind
			case reflect.Float32:
				f.Kind = pref.FloatKind
			}
		case s == "fixed64":
			switch goType.Kind() {
			case reflect.Int64:
				f.Kind = pref.Sfixed64Kind
			case reflect.Uint64:
				f.Kind = pref.Fixed64Kind
			case reflect.Float64:
				f.Kind = pref.DoubleKind
			}
		case s == "bytes":
			switch {
			case goType.Kind() == reflect.String:
				f.Kind = pref.StringKind
			case goType.Kind() == reflect.Slice && goType.Elem() == byteType:
				f.Kind = pref.BytesKind
			default:
				f.Kind = pref.MessageKind
			}
		case s == "group":
			f.Kind = pref.GroupKind
		case strings.HasPrefix(s, "enum="):
			f.Kind = pref.EnumKind
		case strings.HasPrefix(s, "json="):
			f.JSONName = s[len("json="):]
		case s == "packed":
			f.Options.Packed = protoV1.Bool(true)
		case strings.HasPrefix(s, "weak="):
			f.Options.Weak = protoV1.Bool(true)
			f.MessageType = ptype.PlaceholderMessage(pref.FullName(s[len("weak="):]))
		case strings.HasPrefix(s, "def="):
			// The default tag is special in that everything afterwards is the
			// default regardless of the presence of commas.
			s, i = tag[len("def="):], len(tag)

			// Defaults are parsed last, so Kind is populated.
			switch f.Kind {
			case pref.BoolKind:
				switch s {
				case "1":
					f.Default = pref.ValueOf(true)
				case "0":
					f.Default = pref.ValueOf(false)
				}
			case pref.EnumKind:
				n, _ := strconv.ParseInt(s, 10, 32)
				f.Default = pref.ValueOf(pref.EnumNumber(n))
			case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
				n, _ := strconv.ParseInt(s, 10, 32)
				f.Default = pref.ValueOf(int32(n))
			case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
				n, _ := strconv.ParseInt(s, 10, 64)
				f.Default = pref.ValueOf(int64(n))
			case pref.Uint32Kind, pref.Fixed32Kind:
				n, _ := strconv.ParseUint(s, 10, 32)
				f.Default = pref.ValueOf(uint32(n))
			case pref.Uint64Kind, pref.Fixed64Kind:
				n, _ := strconv.ParseUint(s, 10, 64)
				f.Default = pref.ValueOf(uint64(n))
			case pref.FloatKind, pref.DoubleKind:
				n, _ := strconv.ParseFloat(s, 64)
				switch s {
				case "nan":
					n = math.NaN()
				case "inf":
					n = math.Inf(+1)
				case "-inf":
					n = math.Inf(-1)
				}
				if f.Kind == pref.FloatKind {
					f.Default = pref.ValueOf(float32(n))
				} else {
					f.Default = pref.ValueOf(float64(n))
				}
			case pref.StringKind:
				f.Default = pref.ValueOf(string(s))
			case pref.BytesKind:
				// The default value is in escaped form (C-style).
				// TODO: Export unmarshalString in the text package to avoid this hack.
				v, err := ptext.Unmarshal([]byte(`["` + s + `"]:0`))
				if err == nil && len(v.Message()) == 1 {
					s := v.Message()[0][0].String()
					f.Default = pref.ValueOf([]byte(s))
				}
			}
		}
		tag = strings.TrimPrefix(tag[i:], ",")
	}

	// The generator uses the group message name instead of the field name.
	// We obtain the real field name by lowercasing the group name.
	if f.Kind == pref.GroupKind {
		f.Name = pref.Name(strings.ToLower(string(f.Name)))
	}
	return f
}

// Marshal encodes the protoreflect.FieldDescriptor as a tag.
//
// The enumName must be provided if the kind is an enum.
// Historically, the formulation of the enum "name" was the proto package
// dot-concatenated with the generated Go identifier for the enum type.
// Depending on the context on how Marshal is called, there are different ways
// through which that information is determined. As such it is the caller's
// responsibility to provide a function to obtain that information.
func Marshal(fd pref.FieldDescriptor, enumName string) string {
	var tag []string
	switch fd.Kind() {
	case pref.BoolKind, pref.EnumKind, pref.Int32Kind, pref.Uint32Kind, pref.Int64Kind, pref.Uint64Kind:
		tag = append(tag, "varint")
	case pref.Sint32Kind:
		tag = append(tag, "zigzag32")
	case pref.Sint64Kind:
		tag = append(tag, "zigzag64")
	case pref.Sfixed32Kind, pref.Fixed32Kind, pref.FloatKind:
		tag = append(tag, "fixed32")
	case pref.Sfixed64Kind, pref.Fixed64Kind, pref.DoubleKind:
		tag = append(tag, "fixed64")
	case pref.StringKind, pref.BytesKind, pref.MessageKind:
		tag = append(tag, "bytes")
	case pref.GroupKind:
		tag = append(tag, "group")
	}
	tag = append(tag, strconv.Itoa(int(fd.Number())))
	switch fd.Cardinality() {
	case pref.Optional:
		tag = append(tag, "opt")
	case pref.Required:
		tag = append(tag, "req")
	case pref.Repeated:
		tag = append(tag, "rep")
	}
	if fd.IsPacked() {
		tag = append(tag, "packed")
	}
	// TODO: Weak fields?
	name := string(fd.Name())
	if fd.Kind() == pref.GroupKind {
		// The name of the FieldDescriptor for a group field is
		// lowercased. To find the original capitalization, we
		// look in the field's MessageType.
		name = string(fd.MessageType().Name())
	}
	tag = append(tag, "name="+name)
	if jsonName := fd.JSONName(); jsonName != "" && jsonName != name {
		tag = append(tag, "json="+jsonName)
	}
	// The previous implementation does not tag extension fields as proto3,
	// even when the field is defined in a proto3 file. Match that behavior
	// for consistency.
	if fd.Syntax() == pref.Proto3 && fd.ExtendedType() == nil {
		tag = append(tag, "proto3")
	}
	if fd.Kind() == pref.EnumKind && enumName != "" {
		tag = append(tag, "enum="+enumName)
	}
	if fd.OneofType() != nil {
		tag = append(tag, "oneof")
	}
	// This must appear last in the tag, since commas in strings aren't escaped.
	if fd.HasDefault() {
		var def string
		switch fd.Kind() {
		case pref.BoolKind:
			if fd.Default().Bool() {
				def = "1"
			} else {
				def = "0"
			}
		case pref.BytesKind:
			// Preserve protoc-gen-go's historical output of escaped bytes.
			// This behavior is buggy, but fixing it makes it impossible to
			// distinguish between the escaped and unescaped forms.
			//
			// To match the exact output of protoc, this is identical to the
			// CEscape function in strutil.cc of the protoc source code.
			var b []byte
			for _, c := range fd.Default().Bytes() {
				switch c {
				case '\n':
					b = append(b, `\n`...)
				case '\r':
					b = append(b, `\r`...)
				case '\t':
					b = append(b, `\t`...)
				case '"':
					b = append(b, `\"`...)
				case '\'':
					b = append(b, `\'`...)
				case '\\':
					b = append(b, `\\`...)
				default:
					if c >= 0x20 && c <= 0x7e { // printable ASCII
						b = append(b, c)
					} else {
						b = append(b, fmt.Sprintf(`\%03o`, c)...)
					}
				}
			}
			def = string(b)
		case pref.FloatKind, pref.DoubleKind:
			f := fd.Default().Float()
			switch {
			case math.IsInf(f, -1):
				def = "-inf"
			case math.IsInf(f, 1):
				def = "inf"
			case math.IsNaN(f):
				def = "nan"
			default:
				def = fmt.Sprint(fd.Default().Interface())
			}
		default:
			def = fmt.Sprint(fd.Default().Interface())
		}
		tag = append(tag, "def="+def)
	}
	return strings.Join(tag, ",")
}
