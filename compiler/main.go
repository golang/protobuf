// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 Google Inc.  All rights reserved.
// http://code.google.com/p/goprotobuf/
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

/*
	A plugin for the Google protocol buffer compiler to generate Go code.

	This plugin takes no options and the protocol buffer file syntax does
	not yet define any options for Go, so program does no option evaluation.
	That may change.

	Not supported yet:
		Extensions
		Services
*/

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"unicode"

	"goprotobuf.googlecode.com/hg/proto"
	plugin "goprotobuf.googlecode.com/hg/compiler/plugin"
	descriptor "goprotobuf.googlecode.com/hg/compiler/descriptor"
)

func main() {
	// Begin by allocating a generator. The request and response structures are stored there
	// so we can do error handling easily - the response structure contains the field to
	// report failure.
	g := NewGenerator()

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		g.error(err, "reading input")
	}

	if err := proto.Unmarshal(data, g.request); err != nil {
		g.error(err, "parsing input proto")
	}

	if len(g.request.FileToGenerate) == 0 {
		g.fail("no files to generate")
	}

	// Create a wrapped version of the Descriptors and EnumDescriptors that
	// point to the file that defines them.
	g.WrapTypes()

	g.SetPackageNames()
	g.BuildTypeNameMap()

	g.GenerateAllFiles()

	// Send back the results.
	data, err = proto.Marshal(g.response)
	if err != nil {
		g.error(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		g.error(err, "failed to write output proto")
	}
}

// Each type we import as a protocol buffer (other than FileDescriptorProto) needs
// a pointer to the FileDescriptorProto that represents it.  These types achieve that
// wrapping by placing each Proto inside a struct with the pointer to its File. The
// structs have the same names as their contents, with "Proto" removed.
// FileDescriptor is used to store the things that it points to.

// The file and package name method are common to messages and enums.
type common struct {
	file *descriptor.FileDescriptorProto // File this object comes from.
}

// The package name we will produce in our output.
func (c *common) packageName() string { return uniquePackageOf(c.file) }

// A message (struct).
type Descriptor struct {
	common
	*descriptor.DescriptorProto
	parent   *Descriptor   // The containing message, if any.
	nested   []*Descriptor // Inner messages, if any.
	typename []string      // Cached typename vector.
}

// Return the elements of the dotted type name.  The package name is not part
// of this name.
func (d *Descriptor) typeName() []string {
	if d.typename != nil {
		return d.typename
	}
	n := 0
	for parent := d; parent != nil; parent = parent.parent {
		n++
	}
	s := make([]string, n, n)
	for parent := d; parent != nil; parent = parent.parent {
		n--
		s[n] = proto.GetString(parent.Name)
	}
	d.typename = s
	return s
}

// An enum. If it's at top level, its parent will be nil. Otherwise it will be
// the descriptor of the message in which it is defined.
type EnumDescriptor struct {
	common
	*descriptor.EnumDescriptorProto
	parent   *Descriptor // The containing message, if any.
	typename []string    // Cached typename vector.
}

// Return the elements of the dotted type name.
func (e *EnumDescriptor) typeName() (s []string) {
	if e.typename != nil {
		return e.typename
	}
	name := proto.GetString(e.Name)
	if e.parent == nil {
		s = make([]string, 1)
	} else {
		pname := e.parent.typeName()
		s = make([]string, len(pname)+1)
		copy(s, pname)
	}
	s[len(s)-1] = name
	e.typename = s
	return s
}

// Everything but the last element of the full type name, CamelCased.
// The values of type Foo.Bar are call Foo_value1... not Foo_Bar_value1... .
func (e *EnumDescriptor) prefix() string {
	typeName := e.typeName()
	ccPrefix := CamelCaseSlice(typeName[0:len(typeName)-1]) + "_"
	if e.parent == nil {
		// If the enum is not part of a message, the prefix is just the type name.
		ccPrefix = CamelCase(*e.Name) + "_"
	}
	return ccPrefix
}

// The integer value of the named constant in this enumerated type.
func (e *EnumDescriptor) integerValueAsString(name string) string {
	for _, c := range e.Value {
		if proto.GetString(c.Name) == name {
			return fmt.Sprint(proto.GetInt32(c.Number))
		}
	}
	log.Exit("cannot find value for enum constant")
	return ""
}

// A file. Includes slices of all the messages and enums defined within it.
// Those slices are constructed by WrapTypes.
type FileDescriptor struct {
	*descriptor.FileDescriptorProto
	desc []*Descriptor     // All the messages defined in this file.
	enum []*EnumDescriptor // All the enums defined in this file.
}

// The package name we'll use in the generated code to refer to this file.
func (d *FileDescriptor) packageName() string { return uniquePackageOf(d.FileDescriptorProto) }

// The package named defined in the input for this file, possibly dotted.
func (d *FileDescriptor) originalPackageName() string {
	return proto.GetString(d.Package)
}

// Simplify some things by abstracting the abilities shared by enums and messages.
type Object interface {
	packageName() string // The name we use in our output (a_b_c), possibly renamed for uniqueness.
	typeName() []string
}

// Each package name we generate must be unique. The package we're generating
// gets its own name but every other package must have a unqiue name that does
// not conflict in the code we generate.  These names are chosen globally (although
// they don't have to be, it simplifies things to do them globally).
func uniquePackageOf(fd *descriptor.FileDescriptorProto) string {
	s, ok := uniquePackageName[fd]
	if !ok {
		log.Exit("internal error: no package name defined for", proto.GetString(fd.Name))
	}
	return s
}

// The type whose methods generate the output, stored in the associated response structure.
type Generator struct {
	bytes.Buffer

	request  *plugin.CodeGeneratorRequest  // The input.
	response *plugin.CodeGeneratorResponse // The output.

	packageName      string            // What we're calling ourselves.
	allFiles         []*FileDescriptor // All files in the tree
	genFiles         []*FileDescriptor // Those files we will generate output for.
	file             *FileDescriptor   // The file we are compiling now.
	typeNameToObject map[string]Object // Key is a fully-qualified name in input syntax.
	indent           string
}

// Create a new generator and allocate the request and response protobufs.
func NewGenerator() *Generator {
	g := new(Generator)
	g.request = plugin.NewCodeGeneratorRequest()
	g.response = plugin.NewCodeGeneratorResponse()
	return g
}

// Report problem, including an os.Error, and fail.
func (g *Generator) error(err os.Error, msgs ...string) {
	s := strings.Join(msgs, " ") + ":" + err.String()
	log.Stderr("protoc-gen-go: error: ", s)
	g.response.Error = proto.String(s)
	os.Exit(1)
}

// Report problem and fail.
func (g *Generator) fail(msgs ...string) {
	s := strings.Join(msgs, " ")
	log.Stderr("protoc-gen-go: error: ", s)
	g.response.Error = proto.String(s)
	os.Exit(1)
}

// If this file is in a different package, return the package name we're using for this file, plus ".".
// Otherwise return the empty string.
func (g *Generator) DefaultPackageName(obj Object) string {
	pkg := obj.packageName()
	if pkg == g.packageName {
		return ""
	}
	return pkg + "."
}

// For each input file, the unique package name to use, underscored.
var uniquePackageName = make(map[*descriptor.FileDescriptorProto]string)

// Set the package name for this run.  It must agree across all files being generated.
// Also define unique package names for all imported files.
func (g *Generator) SetPackageNames() {
	inUse := make(map[string]bool)
	pkg := proto.GetString(g.genFiles[0].Package)
	g.packageName = strings.Map(DotToUnderscore, pkg)
	inUse[pkg] = true
	for _, f := range g.genFiles {
		thisPkg := proto.GetString(f.Package)
		if thisPkg != pkg {
			g.fail("inconsistent package names:", thisPkg, pkg)
		}
	}
AllFiles:
	for _, f := range g.allFiles {
		for _, genf := range g.genFiles {
			if f == genf {
				// In this package already.
				uniquePackageName[f.FileDescriptorProto] = g.packageName
				continue AllFiles
			}
		}
		truePkg := proto.GetString(f.Package)
		pkg := truePkg
		for {
			_, present := inUse[pkg]
			if present {
				// It's a duplicate; must rename.
				pkg += "X"
				continue
			}
			break
		}
		// Install it.
		if pkg != truePkg {
			log.Stderr("renaming duplicate imported package named", truePkg, "to", pkg)
		}
		inUse[pkg] = true
		uniquePackageName[f.FileDescriptorProto] = strings.Map(DotToUnderscore, pkg)
	}
}

// Walk the incoming data, wrapping DescriptorProtos and EnumDescriptorProtos
// into file-referenced objects within the Generator.  Also create the list of files
// to generate
func (g *Generator) WrapTypes() {
	g.allFiles = make([]*FileDescriptor, len(g.request.ProtoFile))
	for i, f := range g.request.ProtoFile {
		pkg := proto.GetString(f.Package)
		if pkg == "" {
			g.fail(proto.GetString(f.Name), "is missing a package declaration")
		}
		// We must wrap the descriptors before we wrap the enums
		descs := WrapDescriptors(f)
		g.BuildNestedDescriptors(descs)
		enums := WrapEnumDescriptors(f, descs)
		g.allFiles[i] = &FileDescriptor{
			FileDescriptorProto: f,
			desc:                descs,
			enum:                enums,
		}
	}

	g.genFiles = make([]*FileDescriptor, len(g.request.FileToGenerate))
FindFiles:
	for i, fileName := range g.request.FileToGenerate {
		// Search the list.  This algorithm is n^2 but n is tiny.
		for _, file := range g.allFiles {
			if fileName == proto.GetString(file.Name) {
				g.genFiles[i] = file
				continue FindFiles
			}
		}
		g.fail("could not find file named", fileName)
	}
	g.response.File = make([]*plugin.CodeGeneratorResponse_File, len(g.genFiles))
}

// Scan the descriptors in this file.  For each one, build the slice of nested descriptors
func (g *Generator) BuildNestedDescriptors(descs []*Descriptor) {
	for _, desc := range descs {
		if len(desc.NestedType) != 0 {
			desc.nested = make([]*Descriptor, len(desc.NestedType))
			n := 0
			for _, nest := range descs {
				if nest.parent == desc {
					desc.nested[n] = nest
					n++
				}
			}
			if n != len(desc.NestedType) {
				g.fail("internal error: nesting failure for", proto.GetString(desc.Name))
			}
		}
	}
}

// Construct the Descriptor and add it to the slice
func AddDescriptor(sl []*Descriptor, desc *descriptor.DescriptorProto, parent *Descriptor, file *descriptor.FileDescriptorProto) []*Descriptor {
	if len(sl) == cap(sl) {
		nsl := make([]*Descriptor, len(sl), 2*len(sl))
		copy(nsl, sl)
		sl = nsl
	}
	sl = sl[0 : len(sl)+1]
	sl[len(sl)-1] = &Descriptor{common{file: file}, desc, parent, nil, nil}
	return sl
}

// Return a slice of all the Descriptors defined within this file
func WrapDescriptors(file *descriptor.FileDescriptorProto) []*Descriptor {
	sl := make([]*Descriptor, 0, len(file.MessageType)+10)
	for _, desc := range file.MessageType {
		sl = WrapThisDescriptor(sl, desc, nil, file)
	}
	return sl
}

// Wrap this Descriptor, recursively
func WrapThisDescriptor(sl []*Descriptor, desc *descriptor.DescriptorProto, parent *Descriptor, file *descriptor.FileDescriptorProto) []*Descriptor {
	sl = AddDescriptor(sl, desc, parent, file)
	me := sl[len(sl)-1]
	for _, nested := range desc.NestedType {
		sl = WrapThisDescriptor(sl, nested, me, file)
	}
	return sl
}

// Construct the EnumDescriptor and add it to the slice
func AddEnumDescriptor(sl []*EnumDescriptor, desc *descriptor.EnumDescriptorProto, parent *Descriptor, file *descriptor.FileDescriptorProto) []*EnumDescriptor {
	if len(sl) == cap(sl) {
		nsl := make([]*EnumDescriptor, len(sl), 2*len(sl))
		copy(nsl, sl)
		sl = nsl
	}
	sl = sl[0 : len(sl)+1]
	sl[len(sl)-1] = &EnumDescriptor{common{file: file}, desc, parent, nil}
	return sl
}

// Return a slice of all the EnumDescriptors defined within this file
func WrapEnumDescriptors(file *descriptor.FileDescriptorProto, descs []*Descriptor) []*EnumDescriptor {
	sl := make([]*EnumDescriptor, 0, len(file.EnumType)+10)
	for _, enum := range file.EnumType {
		sl = AddEnumDescriptor(sl, enum, nil, file)
	}
	for _, nested := range descs {
		sl = WrapEnumDescriptorsInMessage(sl, nested, file)
	}
	return sl
}

// Wrap this EnumDescriptor, recursively
func WrapEnumDescriptorsInMessage(sl []*EnumDescriptor, desc *Descriptor, file *descriptor.FileDescriptorProto) []*EnumDescriptor {
	for _, enum := range desc.EnumType {
		sl = AddEnumDescriptor(sl, enum, desc, file)
	}
	for _, nested := range desc.nested {
		sl = WrapEnumDescriptorsInMessage(sl, nested, file)
	}
	return sl
}

// Build the map from fully qualified type names to objects.  The key for the map
// comes from the input data, which puts a period at the beginning.
func (g *Generator) BuildTypeNameMap() {
	g.typeNameToObject = make(map[string]Object)
	for _, f := range g.allFiles {
		dottedPkg := "." + f.originalPackageName() + "."
		for _, enum := range f.enum {
			name := dottedPkg + DottedSlice(enum.typeName())
			g.typeNameToObject[name] = enum
		}
		for _, desc := range f.desc {
			name := dottedPkg + DottedSlice(desc.typeName())
			g.typeNameToObject[name] = desc
		}
	}
}

// Given a fully-qualified input type name, return the descriptor for the message or enum with that type.
func (g *Generator) objectNamed(typeName string) Object {
	f, ok := g.typeNameToObject[typeName]
	if !ok {
		g.fail("can't find object with type", typeName)
	}
	return f
}

// Print the arguments, handling indirections because they may be *string, etc.
func (g *Generator) p(str ...interface{}) {
	g.WriteString(g.indent)
	for _, v := range str {
		switch s := v.(type) {
		case string:
			g.WriteString(s)
		case *string:
			g.WriteString(*s)
		case *int32:
			g.WriteString(fmt.Sprintf("%d", *s))
		default:
			g.fail(fmt.Sprintf("unknown type in printer: %T", v))
		}
	}
	g.WriteByte('\n')
}

// Indent the output one tab stop.
func (g *Generator) in() { g.indent += "\t" }

// Unindent the output one tab stop.
func (g *Generator) out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[1:]
	}
}

// Generate the output for all the files we're generating output for.
func (g *Generator) GenerateAllFiles() {
	for i, file := range g.genFiles {
		g.Reset()
		g.Generate(file)
		g.response.File[i] = plugin.NewCodeGeneratorResponse_File()
		g.response.File[i].Name = proto.String(GoName(*file.Name))
		g.response.File[i].Content = proto.String(g.String())
	}
}

// Return the FileDescriptor for this FileDescriptorProto
func (g *Generator) FileOf(fd *descriptor.FileDescriptorProto) *FileDescriptor {
	for _, file := range g.allFiles {
		if file.FileDescriptorProto == fd {
			return file
		}
	}
	g.fail("could not find file in table:", proto.GetString(fd.Name))
	return nil
}

// Fill the response protocol buffer with the generated output for all the files we're
// supposed to generate.
func (g *Generator) Generate(file *FileDescriptor) {
	g.file = g.FileOf(file.FileDescriptorProto)
	g.GenerateHeader()
	g.GenerateImports()
	for _, enum := range g.file.enum {
		g.GenerateEnum(enum)
	}
	for _, desc := range g.file.desc {
		g.GenerateMessage(desc)
	}
	g.GenerateInitFunction()
}

// Generate the header, including package definition and imports
func (g *Generator) GenerateHeader() {
	g.p("// Code generated by protoc-gen-go from ", Quote(*g.file.Name))
	g.p("// DO NOT EDIT!")
	g.p()
	g.p("package ", g.file.packageName())
	g.p()
}

// Generate the header, including package definition and imports
func (g *Generator) GenerateImports() {
	if len(g.file.enum) > 0 {
		g.p(`import "goprotobuf.googlecode.com/hg/proto"`)
	}
	for _, s := range g.file.Dependency {
		// Need to find the descriptor for this file
		for _, fd := range g.allFiles {
			if proto.GetString(fd.Name) == s {
				filename := GoName(s)
				if strings.HasSuffix(filename, ".go") {
					filename = filename[0:len(filename)-3]
				}
				g.p("import ", fd.packageName(), " ", Quote(filename))
				break
			}
		}
	}
	g.p()
}

// Generate the enum definitions for this EnumDescriptor.
func (g *Generator) GenerateEnum(enum *EnumDescriptor) {
	// The full type name
	typeName := enum.typeName()
	// The full type name, CamelCased.
	ccTypeName := CamelCaseSlice(typeName)
	ccPrefix := enum.prefix()
	g.p("type ", ccTypeName, " int32")
	g.p("const (")
	g.in()
	for _, e := range enum.Value {
		g.p(ccPrefix+*e.Name, " = ", e.Number)
	}
	g.out()
	g.p(")")
	g.p("var ", ccTypeName, "_name = map[int32] string {")
	g.in()
	generated := make(map[int32] bool)	// avoid duplicate values
	for _, e := range enum.Value {
		duplicate := ""
		if _, present := generated[*e.Number]; present {
			duplicate = "// Duplicate value: "
		}
		g.p(duplicate, e.Number, ": ", Quote(*e.Name), ",")
		generated[*e.Number] = true
	}
	g.out()
	g.p("}")
	g.p("var ", ccTypeName, "_value = map[string] int32 {")
	g.in()
	for _, e := range enum.Value {
		g.p(Quote(*e.Name), ": ", e.Number, ",")
	}
	g.out()
	g.p("}")
	g.p("func New", ccTypeName, "(x int32) *", ccTypeName, " {")
	g.in()
	g.p("e := ", ccTypeName, "(x)")
	g.p("return &e")
	g.out()
	g.p("}")
	g.p()
}

// The tag is a string like "PB(varint,2,opt,name=fieldname,def=7)" that
// identifies details of the field for the protocol buffer marshaling and unmarshaling
// code.  The fields are:
//	wire encoding
//	protocol tag number
//	opt,req,rep for optional, required, or repeated
//	name= the original declared name
//	enum= the name of the enum type if it is an enum-typed field.
//	def= string representation of the default value, if any.
// The default value must be in a representation that can be used at run-time
// to generate the default value. Thus bools become 0 and 1, for instance.
func (g *Generator) GoTag(field *descriptor.FieldDescriptorProto, wiretype string) string {
	optrepreq := ""
	switch {
	case IsOptional(field):
		optrepreq = "opt"
	case IsRequired(field):
		optrepreq = "req"
	case IsRepeated(field):
		optrepreq = "rep"
	}
	defaultValue := proto.GetString(field.DefaultValue)
	if defaultValue != "" {
		switch *field.Type {
		case descriptor.FieldDescriptorProto_TYPE_BOOL:
			if defaultValue == "true" {
				defaultValue = "1"
			} else {
				defaultValue = "0"
			}
		case descriptor.FieldDescriptorProto_TYPE_STRING,
			descriptor.FieldDescriptorProto_TYPE_BYTES:
			// Protect frogs.
			defaultValue = Quote(defaultValue)
			// Don't need the quotes
			defaultValue = defaultValue[1 : len(defaultValue)-1]
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			// For enums we need to provide the integer constant.
			obj := g.objectNamed(proto.GetString(field.TypeName))
			enum, ok := obj.(*EnumDescriptor)
			if !ok {
				g.fail("enum type inconsistent for", CamelCaseSlice(obj.typeName()))
			}
			defaultValue = enum.integerValueAsString(defaultValue)
		}
		defaultValue = ",def=" + defaultValue
	}
	enum := ""
	if *field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM {
		obj := g.objectNamed(proto.GetString(field.TypeName))
		enum = ",enum=" + obj.packageName() + "." + CamelCaseSlice(obj.typeName())
	}
	name := proto.GetString(field.Name)
	if name == CamelCase(name) {
		name = ""
	} else {
		name = ",name=" + name
	}
	return Quote(fmt.Sprintf("PB(%s,%d,%s%s%s%s)",
		wiretype,
		proto.GetInt32(field.Number),
		optrepreq,
		name,
		enum,
		defaultValue))
}

func NeedsStar(typ descriptor.FieldDescriptorProto_Type) bool {
	switch typ {
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		return false
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return false
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return false
	}
	return true
}

// The type name appropriate for an item. If it's in the current file,
// drop the package name and underscore the rest.
// Otherwise it's from another package; use the underscored package name
// followed by the field name.  The result has an initial capital.
func (g *Generator) TypeName(obj Object) string {
	return g.DefaultPackageName(obj) + CamelCaseSlice(obj.typeName())
}

// Like TypeName, but always includes the package name even if it's our own package.
func (g *Generator) TypeNameWithPackage(obj Object) string {
	return obj.packageName() + CamelCaseSlice(obj.typeName())
}

// Returns a string representing the type name, and the wire type
func (g *Generator) GoType(message *Descriptor, field *descriptor.FieldDescriptorProto) (typ string, wire string) {
	// TODO: Options.
	switch *field.Type {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		typ, wire = "float64", "fixed64"
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		typ, wire = "float32", "fixed32"
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		typ, wire = "int64", "varint"
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		typ, wire = "uint64", "varint"
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		typ, wire = "int32", "varint"
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		typ, wire = "uint32", "varint"
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		typ, wire = "uint64", "fixed64"
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		typ, wire = "uint32", "fixed32"
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		typ, wire = "bool", "varint"
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		typ, wire = "string", "bytes"
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		desc := g.objectNamed(proto.GetString(field.TypeName))
		typ, wire = "*"+g.TypeName(desc), "group"
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		desc := g.objectNamed(proto.GetString(field.TypeName))
		typ, wire = "*"+g.TypeName(desc), "bytes"
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		typ, wire = "[]byte", "bytes"
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		desc := g.objectNamed(proto.GetString(field.TypeName))
		typ, wire = g.TypeName(desc), "varint"
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		typ, wire = "int32", "fixed32"
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		typ, wire = "int64", "fixed64"
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		typ, wire = "int32", "zigzag32"
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		typ, wire = "int64", "zigzag64"
	default:
		g.fail("unknown type for", proto.GetString(field.Name))
	}
	if IsRepeated(field) {
		typ = "[]" + typ
	} else if NeedsStar(*field.Type) {
		typ = "*" + typ
	}
	return
}

// Generate the type and default constant definitions for this Descriptor.
func (g *Generator) GenerateMessage(message *Descriptor) {
	// The full type name
	typeName := message.typeName()
	// The full type name, CamelCased.
	ccTypeName := CamelCaseSlice(typeName)

	g.p("type ", ccTypeName, " struct {")
	g.in()
	for _, field := range message.Field {
		fieldname := CamelCase(*field.Name)
		typename, wiretype := g.GoType(message, field)
		tag := g.GoTag(field, wiretype)
		g.p(fieldname, "\t", typename, "\t", tag)
	}
	g.p("XXX_unrecognized\t[]byte")
	g.out()
	g.p("}")

	// Reset and New functions
	g.p("func (this *", ccTypeName, ") Reset() {")
	g.in()
	g.p("*this = ", ccTypeName, "{}")
	g.out()
	g.p("}")
	g.p("func New", ccTypeName, "() *", ccTypeName, " {")
	g.in()
	g.p("return new(", ccTypeName, ")")
	g.out()
	g.p("}")

	// Default constants
	for _, field := range message.Field {
		def := proto.GetString(field.DefaultValue)
		if def == "" {
			continue
		}
		fieldname := "Default_" + ccTypeName + "_" + CamelCase(*field.Name)
		typename, _ := g.GoType(message, field)
		if typename[0] == '*' {
			typename = typename[1:]
		}
		kind := "const "
		switch {
		case typename == "bool":
		case typename == "string":
			def = Quote(def)
		case typename == "[]byte":
			def = "[]byte(" + Quote(def) + ")"
			kind = "var "
		case 	*field.Type == descriptor.FieldDescriptorProto_TYPE_ENUM:
			// Must be an enum.  Need to construct the prefixed name.
			obj := g.objectNamed(proto.GetString(field.TypeName))
			enum, ok := obj.(*EnumDescriptor)
			if !ok {
				log.Stderr("don't know how to generate constant for", fieldname)
				continue
			}
			def = enum.prefix() + def
		}
		g.p(kind, fieldname, " ", typename, " = ", def)
	}
	g.p()
}

func (g *Generator) GenerateInitFunction() {
	g.p("func init() {")
	g.in()
	for _, enum := range g.file.enum {
		g.GenerateEnumRegistration(enum)
	}
	g.out()
	g.p("}")
}

func (g *Generator) GenerateEnumRegistration(enum *EnumDescriptor) {
	pkg := g.packageName + "." // We always print the full package name here.
	// The full type name
	typeName := enum.typeName()
	// The full type name, CamelCased.
	ccTypeName := CamelCaseSlice(typeName)
	g.p("proto.RegisterEnum(", Quote(pkg+ccTypeName), ", ", ccTypeName+"_name, ", ccTypeName+"_value)")
}

// And now lots of helper functions.

// Return change foo_bar_Baz to FooBar_Baz.
func CamelCase(name string) string {
	elems := strings.Split(name, "_", 0)
	for i, e := range elems {
		if e == "" {
			elems[i] = "_"
			continue
		}
		runes := []int(e)
		if unicode.IsLower(runes[0]) {
			runes[0] = unicode.ToUpper(runes[0])
			elems[i] = string(runes)
		} else {
			if i > 0 {
				elems[i] = "_" + e
			}
		}
	}
	s := strings.Join(elems, "")
	// Name must not begin with an underscore.
	if len(s) > 0 && s[0] == '_' {
		s = "X" + s[1:]
	}
	return s
}

// Like CamelCase, but the argument is a slice of strings to
// be concatenated with "_"
func CamelCaseSlice(elem []string) string { return CamelCase(strings.Join(elem, "_")) }

// Turn a sliced name into a dotted name
func DottedSlice(elem []string) string { return strings.Join(elem, ".") }

// Return a Go-source quoted string representation of s.
func Quote(s string) string { return fmt.Sprintf("%q", s) }

// Given a .proto file name, return the output name for the generated Go program.
func GoName(name string) string {
	if strings.HasSuffix(name, ".proto") {
		name = name[0 : len(name)-6]
	}
	return name + ".pb.go"
}

// Is this field optional?
func IsOptional(field *descriptor.FieldDescriptorProto) bool {
	return field.Label != nil && *field.Label == descriptor.FieldDescriptorProto_LABEL_OPTIONAL
}

// Is this field required?
func IsRequired(field *descriptor.FieldDescriptorProto) bool {
	return field.Label != nil && *field.Label == descriptor.FieldDescriptorProto_LABEL_REQUIRED
}

// Is this field repeated?
func IsRepeated(field *descriptor.FieldDescriptorProto) bool {
	return field.Label != nil && *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED
}

// Mapping function used to generate Go names from package names, which can be dotted.
func DotToUnderscore(rune int) int {
	if rune == '.' {
		return '_'
	}
	return rune
}
