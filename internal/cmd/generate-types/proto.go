// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "text/template"

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

type ProtoKind struct {
	Name      string
	WireType  WireType
	ToValue   Expr
	FromValue Expr
}

func (k ProtoKind) Expr() Expr {
	return "protoreflect." + Expr(k.Name) + "Kind"
}

var ProtoKinds = []ProtoKind{
	{
		Name:      "Bool",
		WireType:  WireVarint,
		ToValue:   "wire.DecodeBool(v)",
		FromValue: "wire.EncodeBool(v.Bool())",
	},
	{
		Name:      "Enum",
		WireType:  WireVarint,
		ToValue:   "protoreflect.EnumNumber(v)",
		FromValue: "uint64(v.Enum())",
	},
	{
		Name:      "Int32",
		WireType:  WireVarint,
		ToValue:   "int32(v)",
		FromValue: "uint64(int32(v.Int()))",
	},
	{
		Name:      "Sint32",
		WireType:  WireVarint,
		ToValue:   "int32(wire.DecodeZigZag(v & math.MaxUint32))",
		FromValue: "wire.EncodeZigZag(int64(int32(v.Int())))",
	},
	{
		Name:      "Uint32",
		WireType:  WireVarint,
		ToValue:   "uint32(v)",
		FromValue: "uint64(uint32(v.Uint()))",
	},
	{
		Name:      "Int64",
		WireType:  WireVarint,
		ToValue:   "int64(v)",
		FromValue: "uint64(v.Int())",
	},
	{
		Name:      "Sint64",
		WireType:  WireVarint,
		ToValue:   "wire.DecodeZigZag(v)",
		FromValue: "wire.EncodeZigZag(v.Int())",
	},
	{
		Name:      "Uint64",
		WireType:  WireVarint,
		ToValue:   "v",
		FromValue: "v.Uint()",
	},
	{
		Name:      "Sfixed32",
		WireType:  WireFixed32,
		ToValue:   "int32(v)",
		FromValue: "uint32(v.Int())",
	},
	{
		Name:      "Fixed32",
		WireType:  WireFixed32,
		ToValue:   "uint32(v)",
		FromValue: "uint32(v.Uint())",
	},
	{
		Name:      "Float",
		WireType:  WireFixed32,
		ToValue:   "math.Float32frombits(uint32(v))",
		FromValue: "math.Float32bits(float32(v.Float()))",
	},
	{
		Name:      "Sfixed64",
		WireType:  WireFixed64,
		ToValue:   "int64(v)",
		FromValue: "uint64(v.Int())",
	},
	{
		Name:      "Fixed64",
		WireType:  WireFixed64,
		ToValue:   "v",
		FromValue: "v.Uint()",
	},
	{
		Name:      "Double",
		WireType:  WireFixed64,
		ToValue:   "math.Float64frombits(v)",
		FromValue: "math.Float64bits(v.Float())",
	},
	{
		Name:      "String",
		WireType:  WireBytes,
		ToValue:   "string(v)",
		FromValue: "[]byte(v.String())",
	},
	{
		Name:      "Bytes",
		WireType:  WireBytes,
		ToValue:   "append(([]byte)(nil), v...)",
		FromValue: "v.Bytes()",
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
func (o UnmarshalOptions) unmarshalScalar(b []byte, wtyp wire.Type, num wire.Number, kind protoreflect.Kind) (val protoreflect.Value, n int, err error) {
	switch kind {
	{{- range .}}
	case {{.Expr}}:
		if wtyp != {{.WireType.Expr}} {
			return val, 0, errUnknown
		}
		{{if (eq .WireType "Group") -}}
		v, n := wire.ConsumeGroup(num, b)
		{{- else -}}
		v, n := wire.Consume{{.WireType}}(b)
		{{- end}}
		if n < 0 {
			return val, 0, wire.ParseError(n)
		}
		return protoreflect.ValueOf({{.ToValue}}), n, nil
	{{- end}}
	default:
		return val, 0, errUnknown
	}
}

func (o UnmarshalOptions) unmarshalList(b []byte, wtyp wire.Type, num wire.Number, list protoreflect.List, kind protoreflect.Kind) (n int, err error) {
	var nerr errors.NonFatal
	switch kind {
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
		v, n := wire.ConsumeGroup(num, b)
		{{- else -}}
		v, n := wire.Consume{{.WireType}}(b)
		{{- end}}
		if n < 0 {
			return 0, wire.ParseError(n)
		}
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

func (o MarshalOptions) marshalSingular(b []byte, num wire.Number, kind protoreflect.Kind, v protoreflect.Value) ([]byte, error) {
	var nerr errors.NonFatal
	switch kind {
	{{- range .}}
	case {{.Expr}}:
		{{if (eq .Name "Message") -}}
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
		b = wire.AppendVarint(b, wire.EncodeTag(num, wire.EndGroupType))
		{{- else -}}
		b = wire.Append{{.WireType}}(b, {{.FromValue}})
		{{- end}}
	{{- end}}
	default:
		return b, errors.New("invalid kind %v", kind)
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
