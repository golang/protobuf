// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/fieldnum"
)

// To avoid a dependency from legacy to descriptor.proto, use a hand-written parser
// for the bits of the descriptor we need.
//
// TODO: Consider unifying this with the parser in fileinit.

type fileDescriptorProto struct {
	Syntax      string
	Package     string
	EnumType    []*enumDescriptorProto
	MessageType []*descriptorProto
}

func (fd fileDescriptorProto) GetSyntax() string  { return fd.Syntax }
func (fd fileDescriptorProto) GetPackage() string { return fd.Package }

func parseFileDescProto(b []byte) *fileDescriptorProto {
	fd := &fileDescriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		parseCheck(n)
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
				fd.EnumType = append(fd.EnumType, parseEnumDescProto(v))
			case fieldnum.FileDescriptorProto_MessageType:
				fd.MessageType = append(fd.MessageType, parseDescProto(v))
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			parseCheck(n)
			b = b[n:]
		}
	}
	return fd
}

type descriptorProto struct {
	Name       string
	NestedType []*descriptorProto
	EnumType   []*enumDescriptorProto
}

func (md descriptorProto) GetName() string { return md.Name }

func parseDescProto(b []byte) *descriptorProto {
	md := &descriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		parseCheck(n)
		b = b[n:]
		switch typ {
		case wire.BytesType:
			v, n := wire.ConsumeBytes(b)
			parseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.DescriptorProto_Name:
				md.Name = string(v)
			case fieldnum.DescriptorProto_NestedType:
				md.NestedType = append(md.NestedType, parseDescProto(v))
			case fieldnum.DescriptorProto_EnumType:
				md.EnumType = append(md.EnumType, parseEnumDescProto(v))
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			parseCheck(n)
			b = b[n:]
		}
	}
	return md
}

type enumDescriptorProto struct {
	Name  string
	Value []*enumValueDescriptorProto
}

func (ed enumDescriptorProto) GetName() string { return ed.Name }

func parseEnumDescProto(b []byte) *enumDescriptorProto {
	ed := &enumDescriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		parseCheck(n)
		b = b[n:]
		switch typ {
		case wire.BytesType:
			v, n := wire.ConsumeBytes(b)
			parseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.EnumDescriptorProto_Name:
				ed.Name = string(v)
			case fieldnum.EnumDescriptorProto_Value:
				ed.Value = append(ed.Value, parseEnumValueDescProto(v))
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			parseCheck(n)
			b = b[n:]
		}
	}
	return ed
}

type enumValueDescriptorProto struct {
	Name   string
	Number int32
}

func (ed enumValueDescriptorProto) GetName() string  { return ed.Name }
func (ed enumValueDescriptorProto) GetNumber() int32 { return ed.Number }

func parseEnumValueDescProto(b []byte) *enumValueDescriptorProto {
	vd := &enumValueDescriptorProto{}
	for len(b) > 0 {
		num, typ, n := wire.ConsumeTag(b)
		parseCheck(n)
		b = b[n:]
		switch typ {
		case wire.VarintType:
			v, n := wire.ConsumeVarint(b)
			parseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.EnumValueDescriptorProto_Number:
				vd.Number = int32(v)
			}
		case wire.BytesType:
			v, n := wire.ConsumeBytes(b)
			parseCheck(n)
			b = b[n:]
			switch num {
			case fieldnum.EnumDescriptorProto_Name:
				vd.Name = string(v)
			}
		default:
			n := wire.ConsumeFieldValue(num, typ, b)
			parseCheck(n)
			b = b[n:]
		}
	}
	return vd
}

func parseCheck(n int) {
	if n < 0 {
		panic(wire.ParseError(n))
	}
}
