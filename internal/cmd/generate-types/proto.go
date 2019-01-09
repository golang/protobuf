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
	Name     string
	WireType WireType
	ToValue  Expr
}

func (k ProtoKind) Expr() Expr {
	return "protoreflect." + Expr(k.Name) + "Kind"
}

var ProtoKinds = []ProtoKind{
	{
		Name:     "Bool",
		WireType: WireVarint,
		ToValue:  "wire.DecodeBool(v)",
	},
	{
		Name:     "Enum",
		WireType: WireVarint,
		ToValue:  "protoreflect.EnumNumber(v)",
	},
	{
		Name:     "Int32",
		WireType: WireVarint,
		ToValue:  "int32(v)",
	},
	{
		Name:     "Sint32",
		WireType: WireVarint,
		ToValue:  "int32(wire.DecodeZigZag(v & math.MaxUint32))",
	},
	{
		Name:     "Uint32",
		WireType: WireVarint,
		ToValue:  "uint32(v)",
	},
	{
		Name:     "Int64",
		WireType: WireVarint,
		ToValue:  "int64(v)",
	},
	{
		Name:     "Sint64",
		WireType: WireVarint,
		ToValue:  "wire.DecodeZigZag(v)",
	},
	{
		Name:     "Uint64",
		WireType: WireVarint,
		ToValue:  "v",
	},
	{
		Name:     "Sfixed32",
		WireType: WireFixed32,
		ToValue:  "int32(v)",
	},
	{
		Name:     "Fixed32",
		WireType: WireFixed32,
		ToValue:  "uint32(v)",
	},
	{
		Name:     "Float",
		WireType: WireFixed32,
		ToValue:  "math.Float32frombits(uint32(v))",
	},
	{
		Name:     "Sfixed64",
		WireType: WireFixed64,
		ToValue:  "int64(v)",
	},
	{
		Name:     "Fixed64",
		WireType: WireFixed64,
		ToValue:  "v",
	},
	{
		Name:     "Double",
		WireType: WireFixed64,
		ToValue:  "math.Float64frombits(v)",
	},
	{
		Name:     "String",
		WireType: WireBytes,
		ToValue:  "string(v)",
	},
	{
		Name:     "Bytes",
		WireType: WireBytes,
		ToValue:  "append(([]byte)(nil), v...)",
	},
	{
		Name:     "Message",
		WireType: WireBytes,
		ToValue:  "v",
	},
	{
		Name:     "Group",
		WireType: WireGroup,
		ToValue:  "v",
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
		if err := o.unmarshalMessage(v, m); err != nil {
			return 0, err
		}
		list.Append(protoreflect.ValueOf(m))
		{{- else -}}
		list.Append(protoreflect.ValueOf({{.ToValue}}))
		{{- end}}
		return n, nil
	{{- end}}
	default:
		return 0, errUnknown
	}
}
`))
