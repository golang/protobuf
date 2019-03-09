// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_gengo

import (
	"fmt"
	"math"
	"os"
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
	reflectPackage      = protogen.GoImportPath("reflect")
	protoimplPackage    = protogen.GoImportPath("github.com/golang/protobuf/v2/runtime/protoimpl")
	protoreflectPackage = protogen.GoImportPath("github.com/golang/protobuf/v2/reflect/protoreflect")
	prototypePackage    = protogen.GoImportPath("github.com/golang/protobuf/v2/reflect/prototype")
)

// TODO: Add support for proto options.

func genReflectFileDescriptor(gen *protogen.Plugin, g *protogen.GeneratedFile, f *fileInfo) {
	if !enableReflection(f.File) {
		return
	}

	// Emit a static check that enforces a minimum version of the proto package.
	// TODO: This should appear higher up in the Go source file.
	g.P("const _ = ", protoimplPackage.Ident("EnforceVersion"), "(", protoimplPackage.Ident("Version"), " - ", minimumVersion, ")")

	g.P("var ", f.GoDescriptorIdent, " ", protoreflectPackage.Ident("FileDescriptor"))
	g.P()

	if len(f.allEnums) > 0 {
		g.P("var ", enumTypesVarName(f), " = make([]", protoreflectPackage.Ident("EnumType"), ",", len(f.allEnums), ")")
	}
	if len(f.allMessages) > 0 {
		g.P("var ", messageTypesVarName(f), " = make([]", protoimplPackage.Ident("MessageType"), ",", len(f.allMessages), ")")
	}

	// Generate a unique list of Go types for all declarations and dependencies,
	// and the associated index into the type list for all dependencies.
	var goTypes []string
	var depIdxs []string
	seen := map[protoreflect.FullName]int{}
	genDep := func(name protoreflect.FullName, depSource string) {
		if depSource != "" {
			line := fmt.Sprintf("%d, // %s -> %s", seen[name], depSource, name)
			depIdxs = append(depIdxs, line)
		}
	}
	genEnum := func(e *protogen.Enum, depSource string) {
		if e != nil {
			name := e.Desc.FullName()
			if _, ok := seen[name]; !ok {
				line := fmt.Sprintf("(%s)(0), // %d: %s", g.QualifiedGoIdent(e.GoIdent), len(goTypes), name)
				goTypes = append(goTypes, line)
				seen[name] = len(seen)
			}
			if depSource != "" {
				genDep(name, depSource)
			}
		}
	}
	genMessage := func(m *protogen.Message, depSource string) {
		if m != nil {
			name := m.Desc.FullName()
			if _, ok := seen[name]; !ok {
				line := fmt.Sprintf("(*%s)(nil), // %d: %s", g.QualifiedGoIdent(m.GoIdent), len(goTypes), name)
				if m.Desc.IsMapEntry() {
					// Map entry messages have no associated Go type.
					line = fmt.Sprintf("nil, // %d: %s", len(goTypes), name)
				}
				goTypes = append(goTypes, line)
				seen[name] = len(seen)
			}
			if depSource != "" {
				genDep(name, depSource)
			}
		}
	}

	// This ordering is significant. See protoimpl.FileBuilder.GoTypes.
	for _, enum := range f.allEnums {
		genEnum(enum, "")
	}
	for _, message := range f.allMessages {
		genMessage(message, "")
	}
	for _, extension := range f.allExtensions {
		source := string(extension.Desc.FullName())
		genMessage(extension.ExtendedType, source+":extendee")
	}
	for _, message := range f.allMessages {
		for _, field := range message.Fields {
			if field.Desc.IsWeak() {
				continue
			}
			source := string(field.Desc.FullName())
			genEnum(field.EnumType, source+":type_name")
			genMessage(field.MessageType, source+":type_name")
		}
	}
	for _, extension := range f.allExtensions {
		source := string(extension.Desc.FullName())
		genEnum(extension.EnumType, source+":type_name")
		genMessage(extension.MessageType, source+":type_name")
	}
	for _, service := range f.Services {
		for _, method := range service.Methods {
			source := string(method.Desc.FullName())
			genMessage(method.InputType, source+":input_type")
			genMessage(method.OutputType, source+":output_type")
		}
	}
	if len(depIdxs) > math.MaxInt32 {
		panic("too many dependencies") // sanity check
	}

	g.P("var ", goTypesVarName(f), " = []interface{}{")
	for _, s := range goTypes {
		g.P(s)
	}
	g.P("}")

	g.P("var ", depIdxsVarName(f), " = []int32{")
	for _, s := range depIdxs {
		g.P(s)
	}
	g.P("}")

	g.P("func init() { ", initFuncName(f.File), "() }")

	g.P("func ", initFuncName(f.File), "() {")
	g.P("if ", f.GoDescriptorIdent, " != nil {")
	g.P("return")
	g.P("}")

	// Ensure that initialization functions for different files in the same Go
	// package run in the correct order: Call the init funcs for every .proto file
	// imported by this one that is in the same Go package.
	for i, imps := 0, f.Desc.Imports(); i < imps.Len(); i++ {
		impFile, _ := gen.FileByName(imps.Get(i).Path())
		if impFile.GoImportPath != f.GoImportPath {
			continue
		}
		g.P(initFuncName(impFile), "()")
	}

	if len(f.allMessages) > 0 {
		g.P("messageTypes := make([]", protoreflectPackage.Ident("MessageType"), ",", len(f.allMessages), ")")
	}
	if len(f.allExtensions) > 0 {
		g.P("extensionTypes := make([]", protoreflectPackage.Ident("ExtensionType"), ",", len(f.allExtensions), ")")
	}

	g.P(f.GoDescriptorIdent, " = ", protoimplPackage.Ident("FileBuilder"), "{")
	g.P("RawDescriptor: ", f.descriptorRawVar, ",")
	g.P("GoTypes: ", goTypesVarName(f), ",")
	g.P("DependencyIndexes: ", depIdxsVarName(f), ",")
	if len(f.allEnums) > 0 {
		g.P("EnumOutputTypes: ", enumTypesVarName(f), ",")
	}
	if len(f.allMessages) > 0 {
		g.P("MessageOutputTypes: messageTypes,")
	}
	if len(f.allExtensions) > 0 {
		g.P("ExtensionOutputTypes: extensionTypes,")
	}
	g.P("}.Init()")

	// Copy the local list of message types into the global array.
	if len(f.allMessages) > 0 {
		g.P("messageGoTypes := ", goTypesVarName(f), "[", len(f.allEnums), ":][:", len(f.allMessages), "]")
		g.P("for i, mt := range messageTypes {")
		g.P(messageTypesVarName(f), "[i].GoType = ", reflectPackage.Ident("TypeOf"), "(messageGoTypes[i])")
		g.P(messageTypesVarName(f), "[i].PBType = mt")
		g.P("}")
	}

	// Copy the local list of extension types into each global variable.
	for i, extension := range f.allExtensions {
		g.P(extensionVar(f.File, extension), ".Type = extensionTypes[", i, "]")
	}

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

	g.P(goTypesVarName(f), " = nil") // allow GC to reclaim resource
	g.P(depIdxsVarName(f), " = nil") // allow GC to reclaim resource
	g.P("}")
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

	idx := f.allMessagesByPtr[message]
	typesVar := messageTypesVarName(f)
	g.P("func (m *", message.GoIdent, ") ProtoReflect() ", protoreflectPackage.Ident("Message"), " {")
	g.P("return ", typesVar, "[", idx, "].MessageOf(m)")
	g.P("}")
}

func goTypesVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_goTypes"
}
func depIdxsVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_depIdxs"
}
func enumTypesVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_enumTypes"
}
func messageTypesVarName(f *fileInfo) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_messageTypes"
}
func initFuncName(f *protogen.File) string {
	return "xxx_" + f.GoDescriptorIdent.GoName + "_init"
}
