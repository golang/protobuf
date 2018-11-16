// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_gengo

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/protogen"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// genOneofField generates the struct field for a oneof.
func genOneofField(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message, oneof *protogen.Oneof) {
	if g.PrintLeadingComments(oneof.Location) {
		g.P("//")
	}
	g.P("// Types that are valid to be assigned to ", oneofFieldName(oneof), ":")
	for _, field := range oneof.Fields {
		g.PrintLeadingComments(field.Location)
		g.P("//\t*", fieldOneofType(field))
	}
	g.Annotate(message.GoIdent.GoName+"."+oneofFieldName(oneof), oneof.Location)
	g.P(oneofFieldName(oneof), " ", oneofInterfaceName(oneof), " `protobuf_oneof:\"", oneof.Desc.Name(), "\"`")
}

// genOneofTypes generates the interface type used for a oneof field,
// and the wrapper types that satisfy that interface.
//
// It also generates the getter method for the parent oneof field
// (but not the member fields).
func genOneofTypes(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message, oneof *protogen.Oneof) {
	ifName := oneofInterfaceName(oneof)
	g.P("type ", ifName, " interface {")
	g.P(ifName, "()")
	g.P("}")
	g.P()
	for _, field := range oneof.Fields {
		name := fieldOneofType(field)
		g.Annotate(name.GoName, field.Location)
		g.Annotate(name.GoName+"."+field.GoName, field.Location)
		g.P("type ", name, " struct {")
		goType, _ := fieldGoType(g, field)
		tags := []string{
			fmt.Sprintf("protobuf:%q", fieldProtobufTag(field)),
		}
		g.P(field.GoName, " ", goType, " `", strings.Join(tags, " "), "`")
		g.P("}")
		g.P()
	}
	for _, field := range oneof.Fields {
		g.P("func (*", fieldOneofType(field), ") ", ifName, "() {}")
		g.P()
	}
	g.Annotate(message.GoIdent.GoName+".Get"+oneof.GoName, oneof.Location)
	g.P("func (m *", message.GoIdent.GoName, ") Get", oneof.GoName, "() ", ifName, " {")
	g.P("if m != nil {")
	g.P("return m.", oneofFieldName(oneof))
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.P()
}

// oneofFieldName returns the name of the struct field holding the oneof value.
//
// This function is trivial, but pulling out the name like this makes it easier
// to experiment with alternative oneof implementations.
func oneofFieldName(oneof *protogen.Oneof) string {
	return oneof.GoName
}

// oneofInterfaceName returns the name of the interface type implemented by
// the oneof field value types.
func oneofInterfaceName(oneof *protogen.Oneof) string {
	return fmt.Sprintf("is%s_%s", oneof.ParentMessage.GoIdent.GoName, oneof.GoName)
}

// genOneofFuncs generates the XXX_OneofFuncs method for a message.
func genOneofFuncs(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	protoMessage := g.QualifiedGoIdent(protoPackage.Ident("Message"))
	protoBuffer := g.QualifiedGoIdent(protoPackage.Ident("Buffer"))
	encFunc := "_" + message.GoIdent.GoName + "_OneofMarshaler"
	decFunc := "_" + message.GoIdent.GoName + "_OneofUnmarshaler"
	sizeFunc := "_" + message.GoIdent.GoName + "_OneofSizer"
	encSig := "(msg " + protoMessage + ", b *" + protoBuffer + ") error"
	decSig := "(msg " + protoMessage + ", tag, wire int, b *" + protoBuffer + ") (bool, error)"
	sizeSig := "(msg " + protoMessage + ") (n int)"

	// XXX_OneofFuncs
	g.P("// XXX_OneofFuncs is for the internal use of the proto package.")
	g.P("func (*", message.GoIdent.GoName, ") XXX_OneofFuncs() (func ", encSig, ", func ", decSig, ", func ", sizeSig, ", []interface{}) {")
	g.P("return ", encFunc, ", ", decFunc, ", ", sizeFunc, ", []interface{}{")
	for _, oneof := range message.Oneofs {
		for _, field := range oneof.Fields {
			g.P("(*", fieldOneofType(field), ")(nil),")
		}
	}
	g.P("}")
	g.P("}")
	g.P()

	// Marshaler
	g.P("func ", encFunc, encSig, " {")
	g.P("m := msg.(*", message.GoIdent, ")")
	for _, oneof := range message.Oneofs {
		g.P("// ", oneof.Desc.Name())
		g.P("switch x := m.", oneofFieldName(oneof), ".(type) {")
		for _, field := range oneof.Fields {
			genOneofFieldMarshal(g, field)
		}
		g.P("case nil:")
		g.P("default:")
		g.P("return ", fmtPackage.Ident("Errorf"), `("`, message.GoIdent.GoName, ".", oneofFieldName(oneof), ` has unexpected type %T", x)`)
		g.P("}")
	}
	g.P("return nil")
	g.P("}")
	g.P()

	// Unmarshaler
	g.P("func ", decFunc, decSig, " {")
	g.P("m := msg.(*", message.GoIdent, ")")
	g.P("switch tag {")
	for _, oneof := range message.Oneofs {
		for _, field := range oneof.Fields {
			genOneofFieldUnmarshal(g, field)
		}
	}
	g.P("default:")
	g.P("return false, nil")
	g.P("}")
	g.P("}")
	g.P()

	// Sizer
	g.P("func ", sizeFunc, sizeSig, " {")
	g.P("m := msg.(*", message.GoIdent, ")")
	for _, oneof := range message.Oneofs {
		g.P("// ", oneof.Desc.Name())
		g.P("switch x := m.", oneofFieldName(oneof), ".(type) {")
		for _, field := range oneof.Fields {
			genOneofFieldSizer(g, field)
		}
		g.P("case nil:")
		g.P("default:")
		g.P("panic(", fmtPackage.Ident("Sprintf"), `("proto: unexpected type %T in oneof", x))`)
		g.P("}")
	}
	g.P("return n")
	g.P("}")
	g.P()
}

// genOneofFieldMarshal generates the marshal case for a oneof subfield.
func genOneofFieldMarshal(g *protogen.GeneratedFile, field *protogen.Field) {
	g.P("case *", fieldOneofType(field), ":")
	encodeTag := func(wireType string) {
		g.P("b.EncodeVarint(", field.Desc.Number(), "<<3|", protoPackage.Ident(wireType), ")")
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P("t := uint64(0)")
		g.P("if x.", field.GoName, " { t = 1 }")
		encodeTag("WireVarint")
		g.P("b.EncodeVarint(t)")
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		encodeTag("WireVarint")
		g.P("b.EncodeVarint(uint64(x.", field.GoName, "))")
	case protoreflect.Sint32Kind:
		encodeTag("WireVarint")
		g.P("b.EncodeZigzag32(uint64(x.", field.GoName, "))")
	case protoreflect.Sint64Kind:
		encodeTag("WireVarint")
		g.P("b.EncodeZigzag64(uint64(x.", field.GoName, "))")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		encodeTag("WireFixed32")
		g.P("b.EncodeFixed32(uint64(x.", field.GoName, "))")
	case protoreflect.FloatKind:
		encodeTag("WireFixed32")
		g.P("b.EncodeFixed32(uint64(", mathPackage.Ident("Float32bits"), "(x.", field.GoName, ")))")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		encodeTag("WireFixed64")
		g.P("b.EncodeFixed64(uint64(x.", field.GoName, "))")
	case protoreflect.DoubleKind:
		encodeTag("WireFixed64")
		g.P("b.EncodeFixed64(", mathPackage.Ident("Float64bits"), "(x.", field.GoName, "))")
	case protoreflect.StringKind:
		encodeTag("WireBytes")
		g.P("b.EncodeStringBytes(x.", field.GoName, ")")
	case protoreflect.BytesKind:
		encodeTag("WireBytes")
		g.P("b.EncodeRawBytes(x.", field.GoName, ")")
	case protoreflect.MessageKind:
		encodeTag("WireBytes")
		g.P("if err := b.EncodeMessage(x.", field.GoName, "); err != nil {")
		g.P("return err")
		g.P("}")
	case protoreflect.GroupKind:
		encodeTag("WireStartGroup")
		g.P("if err := b.Marshal(x.", field.GoName, "); err != nil {")
		g.P("return err")
		g.P("}")
		encodeTag("WireEndGroup")
	}
}

// genOneofFieldUnmarshal generates the unmarshal case for a oneof subfield.
func genOneofFieldUnmarshal(g *protogen.GeneratedFile, field *protogen.Field) {
	oneof := field.OneofType
	g.P("case ", field.Desc.Number(), ": // ", oneof.Desc.Name(), ".", field.Desc.Name())
	checkTag := func(wireType string) {
		g.P("if wire != ", protoPackage.Ident(wireType), " {")
		g.P("return true, ", protoPackage.Ident("ErrInternalBadWireType"))
		g.P("}")
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		checkTag("WireVarint")
		g.P("x, err := b.DecodeVarint()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{x != 0}")
	case protoreflect.EnumKind:
		checkTag("WireVarint")
		g.P("x, err := b.DecodeVarint()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{", field.EnumType.GoIdent, "(x)}")
	case protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		checkTag("WireVarint")
		g.P("x, err := b.DecodeVarint()")
		x := "x"
		if goType, _ := fieldGoType(g, field); goType != "uint64" {
			x = goType + "(x)"
		}
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{", x, "}")
	case protoreflect.Sint32Kind:
		checkTag("WireVarint")
		g.P("x, err := b.DecodeZigzag32()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{int32(x)}")
	case protoreflect.Sint64Kind:
		checkTag("WireVarint")
		g.P("x, err := b.DecodeZigzag64()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{int64(x)}")
	case protoreflect.Sfixed32Kind:
		checkTag("WireFixed32")
		g.P("x, err := b.DecodeFixed32()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{int32(x)}")
	case protoreflect.Fixed32Kind:
		checkTag("WireFixed32")
		g.P("x, err := b.DecodeFixed32()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{uint32(x)}")
	case protoreflect.FloatKind:
		checkTag("WireFixed32")
		g.P("x, err := b.DecodeFixed32()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{", mathPackage.Ident("Float32frombits"), "(uint32(x))}")
	case protoreflect.Sfixed64Kind:
		checkTag("WireFixed64")
		g.P("x, err := b.DecodeFixed64()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{int64(x)}")
	case protoreflect.Fixed64Kind:
		checkTag("WireFixed64")
		g.P("x, err := b.DecodeFixed64()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{x}")
	case protoreflect.DoubleKind:
		checkTag("WireFixed64")
		g.P("x, err := b.DecodeFixed64()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{", mathPackage.Ident("Float64frombits"), "(x)}")
	case protoreflect.StringKind:
		checkTag("WireBytes")
		g.P("x, err := b.DecodeStringBytes()")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{x}")
	case protoreflect.BytesKind:
		checkTag("WireBytes")
		g.P("x, err := b.DecodeRawBytes(true)")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{x}")
	case protoreflect.MessageKind:
		checkTag("WireBytes")
		g.P("msg := new(", field.MessageType.GoIdent, ")")
		g.P("err := b.DecodeMessage(msg)")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{msg}")
	case protoreflect.GroupKind:
		checkTag("WireStartGroup")
		g.P("msg := new(", field.MessageType.GoIdent, ")")
		g.P("err := b.DecodeGroup(msg)")
		g.P("m.", oneofFieldName(oneof), " = &", fieldOneofType(field), "{msg}")
	}
	g.P("return true, err")
}

// genOneofFieldSizer  generates the sizer case for a oneof subfield.
func genOneofFieldSizer(g *protogen.GeneratedFile, field *protogen.Field) {
	sizeProto := protoPackage.Ident("Size")
	sizeVarint := protoPackage.Ident("SizeVarint")
	g.P("case *", fieldOneofType(field), ":")
	if field.Desc.Kind() == protoreflect.MessageKind {
		g.P("s := ", sizeProto, "(x.", field.GoName, ")")
	}
	// Tag and wire varint is known statically.
	tagAndWireSize := proto.SizeVarint(uint64(field.Desc.Number()) << 3) // wire doesn't affect varint size
	g.P("n += ", tagAndWireSize, " // tag and wire")
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P("n += 1")
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		g.P("n += ", sizeVarint, "(uint64(x.", field.GoName, "))")
	case protoreflect.Sint32Kind:
		g.P("n += ", sizeVarint, "(uint64((uint32(x.", field.GoName, ") << 1) ^ uint32((int32(x.", field.GoName, ") >> 31))))")
	case protoreflect.Sint64Kind:
		g.P("n += ", sizeVarint, "(uint64(uint64(x.", field.GoName, "<<1) ^ uint64((int64(x.", field.GoName, ") >> 63))))")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind:
		g.P("n += 4")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		g.P("n += 8")
	case protoreflect.StringKind, protoreflect.BytesKind:
		g.P("n += ", sizeVarint, "(uint64(len(x.", field.GoName, ")))")
		g.P("n += len(x.", field.GoName, ")")
	case protoreflect.MessageKind:
		g.P("n += ", sizeVarint, "(uint64(s))")
		g.P("n += s")
	case protoreflect.GroupKind:
		g.P("n += ", sizeProto, "(x.", field.GoName, ")")
		g.P("n += ", tagAndWireSize, " // tag and wire")
	}
}

// fieldOneofType returns the wrapper type used to represent a field in a oneof.
func fieldOneofType(field *protogen.Field) protogen.GoIdent {
	ident := protogen.GoIdent{
		GoImportPath: field.ParentMessage.GoIdent.GoImportPath,
		GoName:       field.ParentMessage.GoIdent.GoName + "_" + field.GoName,
	}
	// Check for collisions with nested messages or enums.
	//
	// This conflict resolution is incomplete: Among other things, it
	// does not consider collisions with other oneof field types.
	//
	// TODO: Consider dropping this entirely. Detecting conflicts and
	// producing an error is almost certainly better than permuting
	// field and type names in mostly unpredictable ways.
Loop:
	for {
		for _, message := range field.ParentMessage.Messages {
			if message.GoIdent == ident {
				ident.GoName += "_"
				continue Loop
			}
		}
		for _, enum := range field.ParentMessage.Enums {
			if enum.GoIdent == ident {
				ident.GoName += "_"
				continue Loop
			}
		}
		return ident
	}
}
