// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
	"text/template"
)

type WireType string

const (
	WireVarint  WireType = "Varint"
	WireFixed32 WireType = "Fixed32"
	WireFixed64 WireType = "Fixed64"
	WireBytes   WireType = "Bytes"
	WireGroup   WireType = "Group"
)

func (w WireType) Expr() Expr {
	if w == WireGroup {
		return "wire.StartGroupType"
	}
	return "wire." + Expr(w) + "Type"
}

func (w WireType) Packable() bool {
	return w == WireVarint || w == WireFixed32 || w == WireFixed64
}

func (w WireType) ConstSize() bool {
	return w == WireFixed32 || w == WireFixed64
}

type GoType string

const (
	GoBool    = "bool"
	GoInt32   = "int32"
	GoUint32  = "uint32"
	GoInt64   = "int64"
	GoUint64  = "uint64"
	GoFloat32 = "float32"
	GoFloat64 = "float64"
	GoString  = "string"
	GoBytes   = "[]byte"
)

func (g GoType) Zero() Expr {
	switch g {
	case GoBool:
		return "false"
	case GoString:
		return `""`
	case GoBytes:
		return "nil"
	}
	return "0"
}

// Kind is the reflect.Kind of the type.
func (g GoType) Kind() Expr {
	if g == "" || g == GoBytes {
		return ""
	}
	return "reflect." + Expr(strings.ToUpper(string(g[:1]))+string(g[1:]))
}

// PointerMethod is the "internal/impl".pointer method used to access a pointer to this type.
func (g GoType) PointerMethod() Expr {
	if g == GoBytes {
		return "Bytes"
	}
	return Expr(strings.ToUpper(string(g[:1])) + string(g[1:]))
}

type ProtoKind struct {
	Name     string
	WireType WireType

	// Conversions to/from protoreflect.Value.
	ToValue   Expr
	FromValue Expr

	// Conversions to/from generated structures.
	GoType     GoType
	FromGoType Expr
	NoPointer  bool
}

func (k ProtoKind) Expr() Expr {
	return "protoreflect." + Expr(k.Name) + "Kind"
}

var ProtoKinds = []ProtoKind{
	{
		Name:       "Bool",
		WireType:   WireVarint,
		ToValue:    "wire.DecodeBool(v)",
		FromValue:  "wire.EncodeBool(v.Bool())",
		GoType:     GoBool,
		FromGoType: "wire.EncodeBool(v)",
	},
	{
		Name:      "Enum",
		WireType:  WireVarint,
		ToValue:   "protoreflect.EnumNumber(v)",
		FromValue: "uint64(v.Enum())",
	},
	{
		Name:       "Int32",
		WireType:   WireVarint,
		ToValue:    "int32(v)",
		FromValue:  "uint64(int32(v.Int()))",
		GoType:     GoInt32,
		FromGoType: "uint64(v)",
	},
	{
		Name:       "Sint32",
		WireType:   WireVarint,
		ToValue:    "int32(wire.DecodeZigZag(v & math.MaxUint32))",
		FromValue:  "wire.EncodeZigZag(int64(int32(v.Int())))",
		GoType:     GoInt32,
		FromGoType: "wire.EncodeZigZag(int64(v))",
	},
	{
		Name:       "Uint32",
		WireType:   WireVarint,
		ToValue:    "uint32(v)",
		FromValue:  "uint64(uint32(v.Uint()))",
		GoType:     GoUint32,
		FromGoType: "uint64(v)",
	},
	{
		Name:       "Int64",
		WireType:   WireVarint,
		ToValue:    "int64(v)",
		FromValue:  "uint64(v.Int())",
		GoType:     GoInt64,
		FromGoType: "uint64(v)",
	},
	{
		Name:       "Sint64",
		WireType:   WireVarint,
		ToValue:    "wire.DecodeZigZag(v)",
		FromValue:  "wire.EncodeZigZag(v.Int())",
		GoType:     GoInt64,
		FromGoType: "wire.EncodeZigZag(v)",
	},
	{
		Name:       "Uint64",
		WireType:   WireVarint,
		ToValue:    "v",
		FromValue:  "v.Uint()",
		GoType:     GoUint64,
		FromGoType: "v",
	},
	{
		Name:       "Sfixed32",
		WireType:   WireFixed32,
		ToValue:    "int32(v)",
		FromValue:  "uint32(v.Int())",
		GoType:     GoInt32,
		FromGoType: "uint32(v)",
	},
	{
		Name:       "Fixed32",
		WireType:   WireFixed32,
		ToValue:    "uint32(v)",
		FromValue:  "uint32(v.Uint())",
		GoType:     GoUint32,
		FromGoType: "v",
	},
	{
		Name:       "Float",
		WireType:   WireFixed32,
		ToValue:    "math.Float32frombits(uint32(v))",
		FromValue:  "math.Float32bits(float32(v.Float()))",
		GoType:     GoFloat32,
		FromGoType: "math.Float32bits(v)",
	},
	{
		Name:       "Sfixed64",
		WireType:   WireFixed64,
		ToValue:    "int64(v)",
		FromValue:  "uint64(v.Int())",
		GoType:     GoInt64,
		FromGoType: "uint64(v)",
	},
	{
		Name:       "Fixed64",
		WireType:   WireFixed64,
		ToValue:    "v",
		FromValue:  "v.Uint()",
		GoType:     GoUint64,
		FromGoType: "v",
	},
	{
		Name:       "Double",
		WireType:   WireFixed64,
		ToValue:    "math.Float64frombits(v)",
		FromValue:  "math.Float64bits(v.Float())",
		GoType:     GoFloat64,
		FromGoType: "math.Float64bits(v)",
	},
	{
		Name:       "String",
		WireType:   WireBytes,
		ToValue:    "string(v)",
		FromValue:  "[]byte(v.String())",
		GoType:     GoString,
		FromGoType: "[]byte(v)",
	},
	{
		Name:       "Bytes",
		WireType:   WireBytes,
		ToValue:    "append(([]byte)(nil), v...)",
		FromValue:  "v.Bytes()",
		GoType:     GoBytes,
		FromGoType: "v",
		NoPointer:  true,
	},
	{
		Name:      "Message",
		WireType:  WireBytes,
		ToValue:   "v",
		FromValue: "v",
	},
	{
		Name:      "Group",
		WireType:  WireGroup,
		ToValue:   "v",
		FromValue: "v",
	},
}

func generateProtoDecode() string {
	return mustExecute(protoDecodeTemplate, ProtoKinds)
}

var protoDecodeTemplate = template.Must(template.New("").Parse(`
// unmarshalScalar decodes a value of the given kind.
//
// Message values are decoded into a []byte which aliases the input data.
func (o UnmarshalOptions) unmarshalScalar(b []byte, wtyp wire.Type, fd protoreflect.FieldDescriptor) (val protoreflect.Value, n int, err error) {
	switch fd.Kind() {
	{{- range .}}
	case {{.Expr}}:
		if wtyp != {{.WireType.Expr}} {
			return val, 0, errUnknown
		}
		{{if (eq .WireType "Group") -}}
		v, n := wire.ConsumeGroup(fd.Number(), b)
		{{- else -}}
		v, n := wire.Consume{{.WireType}}(b)
		{{- end}}
		if n < 0 {
			return val, 0, wire.ParseError(n)
		}
		{{if (eq .Name "String") -}}
		if fd.Syntax() == protoreflect.Proto3 && !utf8.Valid(v) {
			var nerr errors.NonFatal
			nerr.AppendInvalidUTF8(string(fd.FullName()))
			return protoreflect.ValueOf(string(v)), n, nerr.E
		}
		{{end -}}
		return protoreflect.ValueOf({{.ToValue}}), n, nil
	{{- end}}
	default:
		return val, 0, errUnknown
	}
}

func (o UnmarshalOptions) unmarshalList(b []byte, wtyp wire.Type, list protoreflect.List, fd protoreflect.FieldDescriptor) (n int, err error) {
	var nerr errors.NonFatal
	switch fd.Kind() {
	{{- range .}}
	case {{.Expr}}:
		{{- if .WireType.Packable}}
		if wtyp == wire.BytesType {
			buf, n := wire.ConsumeBytes(b)
			if n < 0 {
				return 0, wire.ParseError(n)
			}
			for len(buf) > 0 {
				v, n := wire.Consume{{.WireType}}(buf)
				if n < 0 {
					return 0, wire.ParseError(n)
				}
				buf = buf[n:]
				list.Append(protoreflect.ValueOf({{.ToValue}}))
			}
			return n, nil
		}
		{{- end}}
		if wtyp != {{.WireType.Expr}} {
			return 0, errUnknown
		}
		{{if (eq .WireType "Group") -}}
		v, n := wire.ConsumeGroup(fd.Number(), b)
		{{- else -}}
		v, n := wire.Consume{{.WireType}}(b)
		{{- end}}
		if n < 0 {
			return 0, wire.ParseError(n)
		}
		{{if (eq .Name "String") -}}
		if fd.Syntax() == protoreflect.Proto3 && !utf8.Valid(v) {
			nerr.AppendInvalidUTF8(string(fd.FullName()))
		}
		{{end -}}
		{{if or (eq .Name "Message") (eq .Name "Group") -}}
		m := list.NewMessage()
		if err := o.unmarshalMessage(v, m); !nerr.Merge(err) {
			return 0, err
		}
		list.Append(protoreflect.ValueOf(m))
		{{- else -}}
		list.Append(protoreflect.ValueOf({{.ToValue}}))
		{{- end}}
		return n, nerr.E
	{{- end}}
	default:
		return 0, errUnknown
	}
}
`))

func generateProtoEncode() string {
	return mustExecute(protoEncodeTemplate, ProtoKinds)
}

var protoEncodeTemplate = template.Must(template.New("").Parse(`
var wireTypes = map[protoreflect.Kind]wire.Type{
{{- range .}}
	{{.Expr}}: {{.WireType.Expr}},
{{- end}}
}

func (o MarshalOptions) marshalSingular(b []byte, fd protoreflect.FieldDescriptor, v protoreflect.Value) ([]byte, error) {
	var nerr errors.NonFatal
	switch fd.Kind() {
	{{- range .}}
	case {{.Expr}}:
		{{- if (eq .Name "String") }}
		if fd.Syntax() == protoreflect.Proto3 && !utf8.ValidString(v.String()) {
			nerr.AppendInvalidUTF8(string(fd.FullName()))
		}
		{{end -}}
		{{- if (eq .Name "Message") -}}
		var pos int
		var err error
		b, pos = appendSpeculativeLength(b)
		b, err = o.marshalMessage(b, v.Message())
		if !nerr.Merge(err) {
			return b, err
		}
		b = finishSpeculativeLength(b, pos)
		{{- else if (eq .Name "Group") -}}
		var err error
		b, err = o.marshalMessage(b, v.Message())
		if !nerr.Merge(err) {
			return b, err
		}
		b = wire.AppendVarint(b, wire.EncodeTag(fd.Number(), wire.EndGroupType))
		{{- else -}}
		b = wire.Append{{.WireType}}(b, {{.FromValue}})
		{{- end}}
	{{- end}}
	default:
		return b, errors.New("invalid kind %v", fd.Kind())
	}
	return b, nerr.E
}
`))

func generateProtoSize() string {
	return mustExecute(protoSizeTemplate, ProtoKinds)
}

var protoSizeTemplate = template.Must(template.New("").Parse(`
func sizeSingular(num wire.Number, kind protoreflect.Kind, v protoreflect.Value) int {
	switch kind {
	{{- range .}}
	case {{.Expr}}:
		{{if (eq .Name "Message") -}}
		return wire.SizeBytes(sizeMessage(v.Message()))
		{{- else if or (eq .WireType "Fixed32") (eq .WireType "Fixed64") -}}
		return wire.Size{{.WireType}}()
		{{- else if (eq .WireType "Bytes") -}}
		return wire.Size{{.WireType}}(len({{.FromValue}}))
		{{- else if (eq .WireType "Group") -}}
		return wire.Size{{.WireType}}(num, sizeMessage(v.Message()))
		{{- else -}}
		return wire.Size{{.WireType}}({{.FromValue}})
		{{- end}}
	{{- end}}
	default:
		return 0
	}
}
`))
