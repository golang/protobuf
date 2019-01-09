// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_gengo

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/protobuf/v2/protogen"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// TODO: Remove this flag.
// Remember to remove the copy in internal/protogen/goldentest.
var enableReflectFlag = os.Getenv("PROTOC_GEN_GO_ENABLE_REFLECT") != ""

func enableReflection(f *protogen.File) bool {
	return enableReflectFlag || isDescriptor(f)
}

// TODO: Remove special-casing for descriptor proto.
func isDescriptor(f *protogen.File) bool {
	return f.Desc.Path() == "google/protobuf/descriptor.proto" && f.Desc.Package() == "google.protobuf"
}

// minimumVersion is minimum version of the v2 proto package that is required.
// This is incremented every time the generated code relies on some property
// in the proto package that was introduced in a later version.
const minimumVersion = 0

const (
	protoimplPackage    = protogen.GoImportPath("github.com/golang/protobuf/v2/runtime/protoimpl")
	protoreflectPackage = protogen.GoImportPath("github.com/golang/protobuf/v2/reflect/protoreflect")
	prototypePackage    = protogen.GoImportPath("github.com/golang/protobuf/v2/reflect/prototype")
)

// TODO: Add support for proto options.

func genReflectInitFunction(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo) {
	if !enableReflection(f.File) {
		return
	}

	if len(f.allEnums)+len(f.allMessages)+len(f.allExtensions)+len(f.Services) == 0 {
		return
	}

	g.P("func init() {")

	// TODO: Fix up file imports to reference a protoreflect.FileDescriptor
	// in a remote dependency. Since we cannot yet rely on the existence of
	// a variable containing the file descriptor, we find a random message or
	// enum the package and see if we can ascend to the parent file descriptor.

	fileDescVar := fileDescVarName(f)
	enumTypesVar := enumTypesVarName(f)
	enumDescsVar := enumDescsVarName(f)
	messageTypesVar := messageTypesVarName(f)
	messageDescsVar := messageDescsVarName(f)

	// Populate all declarations for messages and enums.
	// These are not declared in the literals to avoid an initialization loop.
	if enums := f.Enums; len(enums) > 0 {
		i := f.allEnumsByPtr[enums[0]]
		g.P(fileDescVar, ".Enums = ", enumDescsVar, "[", i, ":", i+len(enums), "]")
	}
	if messages := f.Messages; len(messages) > 0 {
		i := f.allMessagesByPtr[messages[0]]
		g.P(fileDescVar, ".Messages = ", messageDescsVar, "[", i, ":", i+len(messages), "]")
	}
	for i, message := range f.allMessages {
		if enums := message.Enums; len(enums) > 0 {
			j := f.allEnumsByPtr[enums[0]]
			g.P(messageDescsVar, "[", i, "].Enums = ", enumDescsVar, "[", j, ":", j+len(enums), "]")
		}
		if messages := message.Messages; len(messages) > 0 {
			j := f.allMessagesByPtr[messages[0]]
			g.P(messageDescsVar, "[", i, "].Messages = ", messageDescsVar, "[", j, ":", j+len(messages), "]")
		}
	}

	// Populate all dependencies for messages and enums.
	//
	// Externally defined enums and messages may or may not support the
	// v2 protobuf reflection interfaces. The EnumTypeOf and MessageTypeOf
	// helper functions checks for compliance and derives a v2 type from the
	// legacy v1 enum or message if necessary.
	for i, message := range f.allMessages {
		for j, field := range message.Fields {
			fieldSel := fmt.Sprintf("[%d].Fields[%d]", i, j)
			if et := field.EnumType; et != nil {
				idx, ok := f.allEnumsByPtr[et]
				if ok {
					// Locally defined enums are found in the type array.
					g.P(messageDescsVar, fieldSel, ".EnumType = ", enumTypesVar, "[", idx, "]")
				} else {
					// Externally defined enums may need special handling.
					g.P(messageDescsVar, fieldSel, ".EnumType = ", protoimplPackage.Ident("X"), ".EnumTypeOf(", et.GoIdent, "(0))")
				}
			}
			if mt := field.MessageType; mt != nil {
				idx, ok := f.allMessagesByPtr[mt]
				if ok {
					if mt.Desc.IsMapEntry() {
						// Map entry types have no Go type generated for them.
						g.P(messageDescsVar, fieldSel, ".MessageType = ", messageDescsVar, "[", idx, "].Reference()")
					} else {
						// Locally defined messages are found in the type array.
						g.P(messageDescsVar, fieldSel, ".MessageType = ", messageTypesVar, "[", idx, "].Type")
					}
				} else {
					// Externally defined messages may need special handling.
					g.P(messageDescsVar, fieldSel, ".MessageType = ", protoimplPackage.Ident("X"), ".MessageTypeOf((*", mt.GoIdent, ")(nil))")
				}
			}
		}
	}
	// TODO: Fix up extension dependencies.
	// TODO: Fix up method dependencies.

	// Construct the file descriptor.
	g.P("var err error")
	g.P(f.GoDescriptorIdent, ", err = ", prototypePackage.Ident("NewFile"), "(&", fileDescVarName(f), ")")
	g.P("if err != nil { panic(err) }")

	// TODO: Add v2 registration and stop v1 registration in genInitFunction.

	// The descriptor proto needs to register the option types with the
	// prototype so that the package can properly handle those option types.
	if isDescriptor(f.File) {
		for _, m := range f.allMessages {
			name := m.GoIdent.GoName
			if strings.HasSuffix(name, "Options") {
				g.P(prototypePackage.Ident("X"), ".Register", name, "((*", name, ")(nil))")
			}
		}
	}

	g.P("}")
}

func genReflectFileDescriptor(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo) {
	if !enableReflection(f.File) {
		return
	}

	// Emit a static check that enforces a minimum version of the proto package.
	g.P("const _ = ", protoimplPackage.Ident("EnforceVersion"), "(", protoimplPackage.Ident("Version"), " - ", minimumVersion, ")")

	g.P("var ", f.GoDescriptorIdent, " ", protoreflectPackage.Ident("FileDescriptor"))
	g.P()

	// Generate literal for file descriptor.
	fileDescVar := fileDescVarName(f)
	g.P("var ", fileDescVar, " = ", prototypePackage.Ident("File"), "{")
	g.P("Syntax: ", protoreflectPackage.Ident(f.Desc.Syntax().GoString()), ",")
	g.P("Path: ", strconv.Quote(f.Desc.Path()), ",")
	g.P("Package: ", strconv.Quote(string(f.Desc.Package())), ",")
	if imps := f.Desc.Imports(); imps.Len() > 0 {
		g.P("Imports: ", "[]", protoreflectPackage.Ident("FileImport"), "{")
		for i := 0; i < imps.Len(); i++ {
			imp := imps.Get(i)
			path := strconv.Quote(imp.Path())
			pkg := strconv.Quote(string(imp.Package()))
			var isPublic, isWeak string
			if imp.IsPublic {
				isPublic = ", IsPublic: true"
			}
			if imp.IsWeak {
				isWeak = ", IsWeak: true"
			}
			// NOTE: FileDescriptor may be updated later by init.
			g.P("{FileDescriptor: ", prototypePackage.Ident("PlaceholderFile"), "(", path, ", ", pkg, ")", isPublic, isWeak, "},")
		}
		g.P("},")
	}
	// NOTE: Messages, Enums, Extensions, and Services are populated by init.
	g.P("}")

	// Generate literals for enum descriptors.
	if len(f.allEnums) > 0 {
		enumTypesVar := enumTypesVarName(f)
		enumDescsVar := enumDescsVarName(f)
		g.P("var ", enumTypesVar, " = [", len(f.allEnums), "]", protoreflectPackage.Ident("EnumType"), "{")
		for i, enum := range f.allEnums {
			g.P(prototypePackage.Ident("GoEnum"), "(")
			g.P(enumDescsVar, "[", i, "].Reference(),")
			g.P("func(_ ", protoreflectPackage.Ident("EnumType"), ", n ", protoreflectPackage.Ident("EnumNumber"), ") ", protoreflectPackage.Ident("Enum"), " {")
			g.P("return ", enum.GoIdent, "(n)")
			g.P("},")
			g.P("),")
		}
		g.P("}")

		g.P("var ", enumDescsVar, " = [", len(f.allEnums), "]", prototypePackage.Ident("Enum"), "{")
		for _, enum := range f.allEnums {
			g.P("{")
			g.P("Name: ", strconv.Quote(string(enum.Desc.Name())), ",")
			g.P("Values: []", prototypePackage.Ident("EnumValue"), "{")
			for _, value := range enum.Values {
				g.P("{Name: ", strconv.Quote(string(value.Desc.Name())), ", Number: ", value.Desc.Number(), "},")
			}
			g.P("},")
			if resvNames := enum.Desc.ReservedNames(); resvNames.Len() > 0 {
				var ss []string
				for i := 0; i < resvNames.Len(); i++ {
					s := resvNames.Get(i)
					ss = append(ss, strconv.Quote(string(s)))
				}
				g.P("ReservedNames: []", protoreflectPackage.Ident("Name"), "{", strings.Join(ss, ","), "},")
			}
			if resvRanges := enum.Desc.ReservedRanges(); resvRanges.Len() > 0 {
				var ss []string
				for i := 0; i < resvRanges.Len(); i++ {
					r := resvRanges.Get(i)
					ss = append(ss, fmt.Sprintf("{%d,%d}", r[0], r[1]))
				}
				g.P("ReservedRanges: [][2]", protoreflectPackage.Ident("EnumNumber"), "{", strings.Join(ss, ","), "},")
			}
			g.P("},")
		}
		g.P("}")
	}

	// Generate literals for message descriptors.
	if len(f.allMessages) > 0 {
		messageTypesVar := messageTypesVarName(f)
		messageDescsVar := messageDescsVarName(f)
		g.P("var ", messageTypesVar, " = [", len(f.allMessages), "]", protoimplPackage.Ident("MessageType"), "{")
		for i, message := range f.allMessages {
			if message.Desc.IsMapEntry() {
				// Map entry types have no Go type generated for them.
				g.P("{ /* no message type for ", message.GoIdent, " */ },")
				continue
			}
			g.P("{Type: ", prototypePackage.Ident("GoMessage"), "(")
			g.P(messageDescsVar, "[", i, "].Reference(),")
			g.P("func(", protoreflectPackage.Ident("MessageType"), ") ", protoreflectPackage.Ident("Message"), " {")
			g.P("return ", shadowTypeName(message.GoIdent), "{new(", message.GoIdent, ")}")
			g.P("},")
			g.P(")},")
		}
		g.P("}")

		g.P("var ", messageDescsVar, " = [", len(f.allMessages), "]", prototypePackage.Ident("Message"), "{")
		for _, message := range f.allMessages {
			g.P("{")
			g.P("Name: ", strconv.Quote(string(message.Desc.Name())), ",")
			if fields := message.Desc.Fields(); fields.Len() > 0 {
				g.P("Fields: []", prototypePackage.Ident("Field"), "{")
				for i := 0; i < fields.Len(); i++ {
					field := fields.Get(i)
					g.P("{")
					g.P("Name: ", strconv.Quote(string(field.Name())), ",")
					g.P("Number: ", field.Number(), ",")
					g.P("Cardinality: ", protoreflectPackage.Ident(field.Cardinality().GoString()), ",")
					g.P("Kind: ", protoreflectPackage.Ident(field.Kind().GoString()), ",")
					if field.HasJSONName() {
						g.P("JSONName: ", strconv.Quote(field.JSONName()), ",")
					}
					if field.HasDefault() {
						v := field.Default().Interface()
						typeName := reflect.TypeOf(v).Name()
						valLit := fmt.Sprint(v)
						switch v.(type) {
						case protoreflect.EnumNumber:
							typeName = "string"
							valLit = strconv.Quote(string(field.DefaultEnumValue().Name()))
						case float32, float64:
							switch f := field.Default().Float(); {
							case math.IsInf(f, -1):
								valLit = g.QualifiedGoIdent(mathPackage.Ident("Inf")) + "(-1)"
							case math.IsInf(f, +1):
								valLit = g.QualifiedGoIdent(mathPackage.Ident("Inf")) + "(+1)"
							case math.IsNaN(f):
								valLit = g.QualifiedGoIdent(mathPackage.Ident("NaN")) + "()"
							}
						case string, []byte:
							valLit = fmt.Sprintf("%q", v)
						}
						g.P("Default: ", protoreflectPackage.Ident("ValueOf"), "(", typeName, "(", valLit, ")),")
					}
					if oneof := field.OneofType(); oneof != nil {
						g.P("OneofName: ", strconv.Quote(string(oneof.Name())), ",")
					}
					if field.IsPacked() {
						g.P("IsPacked: ", prototypePackage.Ident("True"), ",")
					} else {
						g.P("IsPacked: ", prototypePackage.Ident("False"), ",")
					}
					if field.IsWeak() {
						g.P("IsWeak: true,")
					}
					// NOTE: MessageType and EnumType are populated by init.
					g.P("},")
				}
				g.P("},")
			}
			if oneofs := message.Desc.Oneofs(); oneofs.Len() > 0 {
				g.P("Oneofs: []", prototypePackage.Ident("Oneof"), "{")
				for i := 0; i < oneofs.Len(); i++ {
					oneof := oneofs.Get(i)
					g.P("{Name: ", strconv.Quote(string(oneof.Name())), "},")
				}
				g.P("},")
			}
			if resvNames := message.Desc.ReservedNames(); resvNames.Len() > 0 {
				var ss []string
				for i := 0; i < resvNames.Len(); i++ {
					s := resvNames.Get(i)
					ss = append(ss, strconv.Quote(string(s)))
				}
				g.P("ReservedNames: []", protoreflectPackage.Ident("Name"), "{", strings.Join(ss, ","), "},")
			}
			if resvRanges := message.Desc.ReservedRanges(); resvRanges.Len() > 0 {
				var ss []string
				for i := 0; i < resvRanges.Len(); i++ {
					r := resvRanges.Get(i)
					ss = append(ss, fmt.Sprintf("{%d,%d}", r[0], r[1]))
				}
				g.P("ReservedRanges: [][2]", protoreflectPackage.Ident("FieldNumber"), "{", strings.Join(ss, ","), "},")
			}
			if extRanges := message.Desc.ExtensionRanges(); extRanges.Len() > 0 {
				var ss []string
				for i := 0; i < extRanges.Len(); i++ {
					r := extRanges.Get(i)
					ss = append(ss, fmt.Sprintf("{%d,%d}", r[0], r[1]))
				}
				g.P("ExtensionRanges: [][2]", protoreflectPackage.Ident("FieldNumber"), "{", strings.Join(ss, ","), "},")
			}
			if message.Desc.IsMapEntry() {
				g.P("IsMapEntry: true,")
			}
			// NOTE: Messages, Enums, and Extensions are populated by init.
			g.P("},")
		}
		g.P("}")
	}

	// TODO: Add support for extensions.
	// TODO: Add support for services.
}

func genReflectEnum(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, enum *protogen.Enum) {
	if !enableReflection(f.File) {
		return
	}

	idx := f.allEnumsByPtr[enum]
	typesVar := enumTypesVarName(f)
	g.P("func (e ", enum.GoIdent, ") Type() ", protoreflectPackage.Ident("EnumType"), " {")
	g.P("return ", typesVar, "[", idx, "]")
	g.P("}")
	g.P("func (e ", enum.GoIdent, ") Number() ", protoreflectPackage.Ident("EnumNumber"), " {")
	g.P("return ", protoreflectPackage.Ident("EnumNumber"), "(e)")
	g.P("}")
}

func genReflectMessage(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo, message *protogen.Message) {
	if !enableReflection(f.File) {
		return
	}

	shadowType := shadowTypeName(message.GoIdent)
	g.P("type ", shadowType, " struct{m *", message.GoIdent, "}")
	g.P()

	idx := f.allMessagesByPtr[message]
	typesVar := messageTypesVarName(f)
	g.P("func (m *", message.GoIdent, ") ProtoReflect() ", protoreflectPackage.Ident("Message"), " {")
	g.P("return ", shadowType, "{m}")
	g.P("}")
	g.P("func (m ", shadowType, ") Type() ", protoreflectPackage.Ident("MessageType"), " {")
	g.P("return ", typesVar, "[", idx, "].Type")
	g.P("}")
	g.P("func (m ", shadowType, ") KnownFields() ", protoreflectPackage.Ident("KnownFields"), " {")
	g.P("return ", typesVar, "[", idx, "].KnownFieldsOf(m.m)")
	g.P("}")
	g.P("func (m ", shadowType, ") UnknownFields() ", protoreflectPackage.Ident("UnknownFields"), " {")
	g.P("return ", typesVar, "[", idx, "].UnknownFieldsOf(m.m)")
	g.P("}")
	g.P("func (m ", shadowType, ") Interface() ", protoreflectPackage.Ident("ProtoMessage"), " {")
	g.P("return m.m")
	g.P("}")
	g.P()
}

func fileDescVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_FileDesc"
}
func enumTypesVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_EnumTypes"
}
func enumDescsVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_EnumDescs"
}
func messageTypesVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_MessageTypes"
}
func messageDescsVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_MessageDescs"
}
func extensionDescsVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_ExtensionDescs"
}
func shadowTypeName(ident protogen.GoIdent) string {
	return "xxx_" + ident.GoName
}
