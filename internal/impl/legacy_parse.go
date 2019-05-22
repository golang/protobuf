// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/fieldnum"
)

// To avoid a dependency from legacy to descriptor.proto, use a hand-written parser
// for the bits of the descriptor we need.
//
// TODO: Consider unifying this with the parser in fileinit.

type legacyFileDescriptorProto struct {
	Syntax      string
	Package     string
	EnumType    []*legacyEnumDescriptorProto
	MessageType []*legacyDescriptorProto
}

func (fd legacyFileDescriptorProto) GetSyntax() string  { return fd.Syntax }
func (fd legacyFileDescriptorProto) GetPackage() string { return fd.Package }

func legacyParseFileDescProto(b []byte) *legacyFileDescriptorProto {
	fd := &legacyFileDescriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		legacyParseCheck(n)
		b = b[n:]
		switch typ {
		case wire.BytesType:
			v, n := wire.ConsumeBytes(b)
			b = b[n:]
			switch num {
			case fieldnum.FileDescriptorProto_Syntax:
				fd.Syntax = string(v)
			case fieldnum.FileDescriptorProto_Package:
				fd.Package = string(v)
			case fieldnum.FileDescriptorProto_EnumType:
				fd.EnumType = append(fd.EnumType, legacyParseEnumDescProto(v))
			case fieldnum.FileDescriptorProto_MessageType:
				fd.MessageType = append(fd.MessageType, parseDescProto(v))
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			legacyParseCheck(n)
			b = b[n:]
		}
	}
	return fd
}

type legacyDescriptorProto struct {
	Name       string
	NestedType []*legacyDescriptorProto
	EnumType   []*legacyEnumDescriptorProto
}

func (md legacyDescriptorProto) GetName() string { return md.Name }

func parseDescProto(b []byte) *legacyDescriptorProto {
	md := &legacyDescriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		legacyParseCheck(n)
		b = b[n:]
		switch typ {
		case wire.BytesType:
			v, n := wire.ConsumeBytes(b)
			legacyParseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.DescriptorProto_Name:
				md.Name = string(v)
			case fieldnum.DescriptorProto_NestedType:
				md.NestedType = append(md.NestedType, parseDescProto(v))
			case fieldnum.DescriptorProto_EnumType:
				md.EnumType = append(md.EnumType, legacyParseEnumDescProto(v))
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			legacyParseCheck(n)
			b = b[n:]
		}
	}
	return md
}

type legacyEnumDescriptorProto struct {
	Name  string
	Value []*legacyEnumValueDescriptorProto
}

func (ed legacyEnumDescriptorProto) GetName() string { return ed.Name }

func legacyParseEnumDescProto(b []byte) *legacyEnumDescriptorProto {
	ed := &legacyEnumDescriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		legacyParseCheck(n)
		b = b[n:]
		switch typ {
		case wire.BytesType:
			v, n := wire.ConsumeBytes(b)
			legacyParseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.EnumDescriptorProto_Name:
				ed.Name = string(v)
			case fieldnum.EnumDescriptorProto_Value:
				ed.Value = append(ed.Value, legacyParseEnumValueDescProto(v))
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			legacyParseCheck(n)
			b = b[n:]
		}
	}
	return ed
}

type legacyEnumValueDescriptorProto struct {
	Name   string
	Number int32
}

func (ed legacyEnumValueDescriptorProto) GetName() string  { return ed.Name }
func (ed legacyEnumValueDescriptorProto) GetNumber() int32 { return ed.Number }

func legacyParseEnumValueDescProto(b []byte) *legacyEnumValueDescriptorProto {
	vd := &legacyEnumValueDescriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		legacyParseCheck(n)
		b = b[n:]
		switch typ {
		case wire.VarintType:
			v, n := wire.ConsumeVarint(b)
			legacyParseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.EnumValueDescriptorProto_Number:
				vd.Number = int32(v)
			}
		case wire.BytesType:
			v, n := wire.ConsumeBytes(b)
			legacyParseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.EnumDescriptorProto_Name:
				vd.Name = string(v)
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			legacyParseCheck(n)
			b = b[n:]
		}
	}
	return vd
}

func legacyParseCheck(n int) {
	if n < 0 {
		panic(wire.ParseError(n))
	}
}
