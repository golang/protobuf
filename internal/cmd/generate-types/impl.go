// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"text/template"
)

func generateImplEncode() string {
	return mustExecute(implEncodeTemplate, ProtoKinds)
}

var implEncodeTemplate = template.Must(template.New("").Parse(`
{{- /*
  IsZero is an expression testing if 'v' is the zero value.
*/ -}}
{{- define "IsZero" -}}
{{if eq .WireType "Bytes" -}}
len(v) == 0
{{- else if or (eq .Name "Double") (eq .Name "Float") -}}
v == 0 && !math.Signbit(float64(v))
{{- else -}}
v == {{.GoType.Zero}}
{{- end -}}
{{- end -}}

{{- /*
  Size is an expression computing the size of 'v'.
*/ -}}
{{- define "Size" -}}
{{- if .WireType.ConstSize -}}
wire.Size{{.WireType}}()
{{- else if eq .WireType "Bytes" -}}
wire.SizeBytes(len({{.FromGoType}}))
{{- else -}}
wire.Size{{.WireType}}({{.FromGoType}})
{{- end -}}
{{- end -}}

{{- /*
  Append is a set of statements appending 'v' to 'b'.
*/ -}}
{{- define "Append" -}}
{{- if eq .Name "String" -}}
b = wire.AppendString(b, {{.FromGoType}})
{{- else -}}
b = wire.Append{{.WireType}}(b, {{.FromGoType}})
{{- end -}}
{{- end -}}

{{- range .}}
{{- if .FromGoType }}
// size{{.Name}} returns the size of wire encoding a {{.GoType}} pointer as a {{.Name}}.
func size{{.Name}}(p pointer, tagsize int, _ marshalOptions) (size int) {
	{{if not .WireType.ConstSize -}}
	v := *p.{{.GoType.PointerMethod}}()
	{{- end}}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}} wire encodes a {{.GoType}} pointer as a {{.Name}}.
func append{{.Name}}(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.{{.GoType.PointerMethod}}()
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

var coder{{.Name}} = pointerCoderFuncs{
	size:    size{{.Name}},
	marshal: append{{.Name}},
}

// size{{.Name}} returns the size of wire encoding a {{.GoType}} pointer as a {{.Name}}.
// The zero value is not encoded.
func size{{.Name}}NoZero(p pointer, tagsize int, _ marshalOptions) (size int) {
	v := *p.{{.GoType.PointerMethod}}()
	if {{template "IsZero" .}} {
		return 0
	}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}} wire encodes a {{.GoType}} pointer as a {{.Name}}.
// The zero value is not encoded.
func append{{.Name}}NoZero(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.{{.GoType.PointerMethod}}()
	if {{template "IsZero" .}} {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

var coder{{.Name}}NoZero = pointerCoderFuncs{
	size:    size{{.Name}}NoZero,
	marshal: append{{.Name}}NoZero,
}

{{- if not .NoPointer}}
// size{{.Name}}Ptr returns the size of wire encoding a *{{.GoType}} pointer as a {{.Name}}.
// It panics if the pointer is nil.
func size{{.Name}}Ptr(p pointer, tagsize int, _ marshalOptions) (size int) {
	{{if not .WireType.ConstSize -}}
	v := **p.{{.GoType.PointerMethod}}Ptr()
	{{end -}}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}} wire encodes a *{{.GoType}} pointer as a {{.Name}}.
// It panics if the pointer is nil.
func append{{.Name}}Ptr(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := **p.{{.GoType.PointerMethod}}Ptr()
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

var coder{{.Name}}Ptr = pointerCoderFuncs{
	size:    size{{.Name}}Ptr,
	marshal: append{{.Name}}Ptr,
}
{{end}}

// size{{.Name}}Slice returns the size of wire encoding a []{{.GoType}} pointer as a repeated {{.Name}}.
func size{{.Name}}Slice(p pointer, tagsize int, _ marshalOptions) (size int) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	{{if .WireType.ConstSize -}}
	size = len(s) * (tagsize + {{template "Size" .}})
	{{- else -}}
	for _, v := range s {
		size += tagsize + {{template "Size" .}}
	}
	{{- end}}
	return size
}

// append{{.Name}}Slice encodes a []{{.GoType}} pointer as a repeated {{.Name}}.
func append{{.Name}}Slice(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		{{template "Append" .}}
	}
	return b, nil
}

var coder{{.Name}}Slice = pointerCoderFuncs{
	size:    size{{.Name}}Slice,
	marshal: append{{.Name}}Slice,
}

{{if or (eq .WireType "Varint") (eq .WireType "Fixed32") (eq .WireType "Fixed64")}}
// size{{.Name}}PackedSlice returns the size of wire encoding a []{{.GoType}} pointer as a packed repeated {{.Name}}.
func size{{.Name}}PackedSlice(p pointer, tagsize int, _ marshalOptions) (size int) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	if len(s) == 0 {
		return 0
	}
	{{if .WireType.ConstSize -}}
	n := len(s) * {{template "Size" .}}
	{{- else -}}
	n := 0
	for _, v := range s {
		n += {{template "Size" .}}
	}
	{{- end}}
	return tagsize + wire.SizeBytes(n)
}

// append{{.Name}}PackedSlice encodes a []{{.GoType}} pointer as a packed repeated {{.Name}}.
func append{{.Name}}PackedSlice(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *p.{{.GoType.PointerMethod}}Slice()
	if len(s) == 0 {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	{{if .WireType.ConstSize -}}
	n := len(s) * {{template "Size" .}}
	{{- else -}}
	n := 0
	for _, v := range s {
		n += {{template "Size" .}}
	}
	{{- end}}
	b = wire.AppendVarint(b, uint64(n))
	for _, v := range s {
		{{template "Append" .}}
	}
	return b, nil
}

var coder{{.Name}}PackedSlice = pointerCoderFuncs{
	size:    size{{.Name}}PackedSlice,
	marshal: append{{.Name}}PackedSlice,
}
{{end}}

// size{{.Name}}Iface returns the size of wire encoding a {{.GoType}} value as a {{.Name}}.
func size{{.Name}}Iface(ival interface{}, tagsize int, _ marshalOptions) int {
	{{- if not .WireType.ConstSize}}
	v := ival.({{.GoType}})
	{{end -}}
	return tagsize + {{template "Size" .}}
}

// append{{.Name}}Iface encodes a {{.GoType}} value as a {{.Name}}.
func append{{.Name}}Iface(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := ival.({{.GoType}})
	b = wire.AppendVarint(b, wiretag)
	{{template "Append" .}}
	return b, nil
}

var coder{{.Name}}Iface = ifaceCoderFuncs{
	size:    size{{.Name}}Iface,
	marshal: append{{.Name}}Iface,
}

// size{{.Name}}SliceIface returns the size of wire encoding a []{{.GoType}} value as a repeated {{.Name}}.
func size{{.Name}}SliceIface(ival interface{}, tagsize int, _ marshalOptions) (size int) {
	s := *ival.(*[]{{.GoType}})
	{{if .WireType.ConstSize -}}
	size = len(s) * (tagsize + {{template "Size" .}})
	{{- else -}}
	for _, v := range s {
		size += tagsize + {{template "Size" .}}
	}
	{{- end}}
	return size
}

// append{{.Name}}SliceIface encodes a []{{.GoType}} value as a repeated {{.Name}}.
func append{{.Name}}SliceIface(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *ival.(*[]{{.GoType}})
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		{{template "Append" .}}
	}
	return b, nil
}

var coder{{.Name}}SliceIface = ifaceCoderFuncs{
	size:    size{{.Name}}SliceIface,
	marshal: append{{.Name}}SliceIface,
}

{{end -}}
{{end -}}

var wireTypes = map[protoreflect.Kind]wire.Type{
{{range . -}}
	protoreflect.{{.Name}}Kind: {{.WireType.Expr}},
{{end}}
}
`))
