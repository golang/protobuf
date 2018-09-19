// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protogen provides support for writing protoc plugins.
//
// Plugins for protoc, the Protocol Buffers Compiler, are programs which read
// a CodeGeneratorRequest protocol buffer from standard input and write a
// CodeGeneratorResponse protocol buffer to standard output. This package
// provides support for writing plugins which generate Go code.
package protogen

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pluginpb "github.com/golang/protobuf/protoc-gen-go/plugin"
	"golang.org/x/tools/go/ast/astutil"
	"google.golang.org/proto/reflect/protoreflect"
	"google.golang.org/proto/reflect/protoregistry"
	"google.golang.org/proto/reflect/prototype"
)

// Run executes a function as a protoc plugin.
//
// It reads a CodeGeneratorRequest message from os.Stdin, invokes the plugin
// function, and writes a CodeGeneratorResponse message to os.Stdout.
//
// If a failure occurs while reading or writing, Run prints an error to
// os.Stderr and calls os.Exit(1).
//
// Passing a nil options is equivalent to passing a zero-valued one.
func Run(opts *Options, f func(*Plugin) error) {
	if err := run(opts, f); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", filepath.Base(os.Args[0]), err)
		os.Exit(1)
	}
}

func run(opts *Options, f func(*Plugin) error) error {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		return err
	}
	gen, err := New(req, opts)
	if err != nil {
		return err
	}
	if err := f(gen); err != nil {
		// Errors from the plugin function are reported by setting the
		// error field in the CodeGeneratorResponse.
		//
		// In contrast, errors that indicate a problem in protoc
		// itself (unparsable input, I/O errors, etc.) are reported
		// to stderr.
		gen.Error(err)
	}
	resp := gen.Response()
	out, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := os.Stdout.Write(out); err != nil {
		return err
	}
	return nil
}

// A Plugin is a protoc plugin invocation.
type Plugin struct {
	// Request is the CodeGeneratorRequest provided by protoc.
	Request *pluginpb.CodeGeneratorRequest

	// Files is the set of files to generate and everything they import.
	// Files appear in topological order, so each file appears before any
	// file that imports it.
	Files       []*File
	filesByName map[string]*File

	fileReg        *protoregistry.Files
	messagesByName map[protoreflect.FullName]*Message
	enumsByName    map[protoreflect.FullName]*Enum
	pathType       pathType
	genFiles       []*GeneratedFile
	err            error
}

// Options are optional parameters to New.
type Options struct {
	// If ParamFunc is non-nil, it will be called with each unknown
	// generator parameter.
	//
	// Plugins for protoc can accept parameters from the command line,
	// passed in the --<lang>_out protoc, separated from the output
	// directory with a colon; e.g.,
	//
	//   --go_out=<param1>=<value1>,<param2>=<value2>:<output_directory>
	//
	// Parameters passed in this fashion as a comma-separated list of
	// key=value pairs will be passed to the ParamFunc.
	//
	// The (flag.FlagSet).Set method matches this function signature,
	// so parameters can be converted into flags as in the following:
	//
	//   var flags flag.FlagSet
	//   value := flags.Bool("param", false, "")
	//   opts := &protogen.Options{
	//     ParamFunc: flags.Set,
	//   }
	//   protogen.Run(opts, func(p *protogen.Plugin) error {
	//     if *value { ... }
	//   })
	ParamFunc func(name, value string) error
}

// New returns a new Plugin.
//
// Passing a nil Options is equivalent to passing a zero-valued one.
func New(req *pluginpb.CodeGeneratorRequest, opts *Options) (*Plugin, error) {
	if opts == nil {
		opts = &Options{}
	}
	gen := &Plugin{
		Request:        req,
		filesByName:    make(map[string]*File),
		fileReg:        protoregistry.NewFiles(),
		messagesByName: make(map[protoreflect.FullName]*Message),
		enumsByName:    make(map[protoreflect.FullName]*Enum),
	}

	packageNames := make(map[string]GoPackageName) // filename -> package name
	importPaths := make(map[string]GoImportPath)   // filename -> import path
	var packageImportPath GoImportPath
	for _, param := range strings.Split(req.GetParameter(), ",") {
		var value string
		if i := strings.Index(param, "="); i >= 0 {
			value = param[i+1:]
			param = param[0:i]
		}
		switch param {
		case "":
			// Ignore.
		case "import_prefix":
			// TODO
		case "import_path":
			packageImportPath = GoImportPath(value)
		case "paths":
			switch value {
			case "import":
				gen.pathType = pathTypeImport
			case "source_relative":
				gen.pathType = pathTypeSourceRelative
			default:
				return nil, fmt.Errorf(`unknown path type %q: want "import" or "source_relative"`, value)
			}
		case "annotate_code":
			// TODO
		default:
			if param[0] == 'M' {
				importPaths[param[1:]] = GoImportPath(value)
				continue
			}
			if opts.ParamFunc != nil {
				if err := opts.ParamFunc(param, value); err != nil {
					return nil, err
				}
			}
		}
	}

	// Figure out the import path and package name for each file.
	//
	// The rules here are complicated and have grown organically over time.
	// Interactions between different ways of specifying package information
	// may be surprising.
	//
	// The recommended approach is to include a go_package option in every
	// .proto source file specifying the full import path of the Go package
	// associated with this file.
	//
	//     option go_package = "github.com/golang/protobuf/ptypes/any";
	//
	// Build systems which want to exert full control over import paths may
	// specify M<filename>=<import_path> flags.
	//
	// Other approaches are not recommend.
	generatedFileNames := make(map[string]bool)
	for _, name := range gen.Request.FileToGenerate {
		generatedFileNames[name] = true
	}
	// We need to determine the import paths before the package names,
	// because the Go package name for a file is sometimes derived from
	// different file in the same package.
	packageNameForImportPath := make(map[GoImportPath]GoPackageName)
	for _, fdesc := range gen.Request.ProtoFile {
		filename := fdesc.GetName()
		packageName, importPath := goPackageOption(fdesc)
		switch {
		case importPaths[filename] != "":
			// Command line: M=foo.proto=quux/bar
			//
			// Explicit mapping of source file to import path.
		case generatedFileNames[filename] && packageImportPath != "":
			// Command line: import_path=quux/bar
			//
			// The import_path flag sets the import path for every file that
			// we generate code for.
			importPaths[filename] = packageImportPath
		case importPath != "":
			// Source file: option go_package = "quux/bar";
			//
			// The go_package option sets the import path. Most users should use this.
			importPaths[filename] = importPath
		default:
			// Source filename.
			//
			// Last resort when nothing else is available.
			importPaths[filename] = GoImportPath(path.Dir(filename))
		}
		if packageName != "" {
			packageNameForImportPath[importPaths[filename]] = packageName
		}
	}
	for _, fdesc := range gen.Request.ProtoFile {
		filename := fdesc.GetName()
		packageName, _ := goPackageOption(fdesc)
		defaultPackageName := packageNameForImportPath[importPaths[filename]]
		switch {
		case packageName != "":
			// Source file: option go_package = "quux/bar";
			packageNames[filename] = packageName
		case defaultPackageName != "":
			// A go_package option in another file in the same package.
			//
			// This is a poor choice in general, since every source file should
			// contain a go_package option. Supported mainly for historical
			// compatibility.
			packageNames[filename] = defaultPackageName
		case generatedFileNames[filename] && packageImportPath != "":
			// Command line: import_path=quux/bar
			packageNames[filename] = cleanPackageName(path.Base(string(packageImportPath)))
		case fdesc.GetPackage() != "":
			// Source file: package quux.bar;
			packageNames[filename] = cleanPackageName(fdesc.GetPackage())
		default:
			// Source filename.
			packageNames[filename] = cleanPackageName(baseName(filename))
		}
	}

	// Consistency check: Every file with the same Go import path should have
	// the same Go package name.
	packageFiles := make(map[GoImportPath][]string)
	for filename, importPath := range importPaths {
		packageFiles[importPath] = append(packageFiles[importPath], filename)
	}
	for importPath, filenames := range packageFiles {
		for i := 1; i < len(filenames); i++ {
			if a, b := packageNames[filenames[0]], packageNames[filenames[i]]; a != b {
				return nil, fmt.Errorf("Go package %v has inconsistent names %v (%v) and %v (%v)",
					importPath, a, filenames[0], b, filenames[i])
			}
		}
	}

	for _, fdesc := range gen.Request.ProtoFile {
		filename := fdesc.GetName()
		if gen.filesByName[filename] != nil {
			return nil, fmt.Errorf("duplicate file name: %q", filename)
		}
		f, err := newFile(gen, fdesc, packageNames[filename], importPaths[filename])
		if err != nil {
			return nil, err
		}
		gen.Files = append(gen.Files, f)
		gen.filesByName[filename] = f
	}
	for _, filename := range gen.Request.FileToGenerate {
		f, ok := gen.FileByName(filename)
		if !ok {
			return nil, fmt.Errorf("no descriptor for generated file: %v", filename)
		}
		f.Generate = true
	}
	return gen, nil
}

// Error records an error in code generation. The generator will report the
// error back to protoc and will not produce output.
func (gen *Plugin) Error(err error) {
	if gen.err == nil {
		gen.err = err
	}
}

// Response returns the generator output.
func (gen *Plugin) Response() *pluginpb.CodeGeneratorResponse {
	resp := &pluginpb.CodeGeneratorResponse{}
	if gen.err != nil {
		resp.Error = proto.String(gen.err.Error())
		return resp
	}
	for _, gf := range gen.genFiles {
		content, err := gf.Content()
		if err != nil {
			return &pluginpb.CodeGeneratorResponse{
				Error: proto.String(err.Error()),
			}
		}
		resp.File = append(resp.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String(gf.filename),
			Content: proto.String(string(content)),
		})
	}
	return resp
}

// FileByName returns the file with the given name.
func (gen *Plugin) FileByName(name string) (f *File, ok bool) {
	f, ok = gen.filesByName[name]
	return f, ok
}

// A File describes a .proto source file.
type File struct {
	Desc  protoreflect.FileDescriptor
	Proto *descpb.FileDescriptorProto

	GoPackageName GoPackageName // name of this file's Go package
	GoImportPath  GoImportPath  // import path of this file's Go package
	Messages      []*Message    // top-level message declarations
	Enums         []*Enum       // top-level enum declarations
	Extensions    []*Extension  // top-level extension declarations
	Generate      bool          // true if we should generate code for this file

	// GeneratedFilenamePrefix is used to construct filenames for generated
	// files associated with this source file.
	//
	// For example, the source file "dir/foo.proto" might have a filename prefix
	// of "dir/foo". Appending ".pb.go" produces an output file of "dir/foo.pb.go".
	GeneratedFilenamePrefix string
}

func newFile(gen *Plugin, p *descpb.FileDescriptorProto, packageName GoPackageName, importPath GoImportPath) (*File, error) {
	desc, err := prototype.NewFileFromDescriptorProto(p, gen.fileReg)
	if err != nil {
		return nil, fmt.Errorf("invalid FileDescriptorProto %q: %v", p.GetName(), err)
	}
	if err := gen.fileReg.Register(desc); err != nil {
		return nil, fmt.Errorf("cannot register descriptor %q: %v", p.GetName(), err)
	}
	f := &File{
		Desc:          desc,
		Proto:         p,
		GoPackageName: packageName,
		GoImportPath:  importPath,
	}

	// Determine the prefix for generated Go files.
	prefix := p.GetName()
	if ext := path.Ext(prefix); ext == ".proto" || ext == ".protodevel" {
		prefix = prefix[:len(prefix)-len(ext)]
	}
	if gen.pathType == pathTypeImport {
		// If paths=import (the default) and the file contains a go_package option
		// with a full import path, the output filename is derived from the Go import
		// path.
		//
		// Pass the paths=source_relative flag to always derive the output filename
		// from the input filename instead.
		if _, importPath := goPackageOption(p); importPath != "" {
			prefix = path.Join(string(importPath), path.Base(prefix))
		}
	}
	f.GeneratedFilenamePrefix = prefix

	for i, mdescs := 0, desc.Messages(); i < mdescs.Len(); i++ {
		f.Messages = append(f.Messages, newMessage(gen, f, nil, mdescs.Get(i)))
	}
	for i, edescs := 0, desc.Enums(); i < edescs.Len(); i++ {
		f.Enums = append(f.Enums, newEnum(gen, f, nil, edescs.Get(i)))
	}
	for i, extdescs := 0, desc.Extensions(); i < extdescs.Len(); i++ {
		f.Extensions = append(f.Extensions, newField(gen, f, nil, extdescs.Get(i)))
	}
	for _, message := range f.Messages {
		if err := message.init(gen); err != nil {
			return nil, err
		}
	}
	for _, extension := range f.Extensions {
		if err := extension.init(gen); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// goPackageOption interprets a file's go_package option.
// If there is no go_package, it returns ("", "").
// If there's a simple name, it returns (pkg, "").
// If the option implies an import path, it returns (pkg, impPath).
func goPackageOption(d *descpb.FileDescriptorProto) (pkg GoPackageName, impPath GoImportPath) {
	opt := d.GetOptions().GetGoPackage()
	if opt == "" {
		return "", ""
	}
	// A semicolon-delimited suffix delimits the import path and package name.
	if i := strings.Index(opt, ";"); i >= 0 {
		return cleanPackageName(opt[i+1:]), GoImportPath(opt[:i])
	}
	// The presence of a slash implies there's an import path.
	if i := strings.LastIndex(opt, "/"); i >= 0 {
		return cleanPackageName(opt[i+1:]), GoImportPath(opt)
	}
	return cleanPackageName(opt), ""
}

// A Message describes a message.
type Message struct {
	Desc protoreflect.MessageDescriptor

	GoIdent    GoIdent      // name of the generated Go type
	Fields     []*Field     // message field declarations
	Oneofs     []*Oneof     // oneof declarations
	Messages   []*Message   // nested message declarations
	Enums      []*Enum      // nested enum declarations
	Extensions []*Extension // nested extension declarations
	Path       []int32      // location path of this message
}

func newMessage(gen *Plugin, f *File, parent *Message, desc protoreflect.MessageDescriptor) *Message {
	var path []int32
	if parent != nil {
		path = pathAppend(parent.Path, messageMessageField, int32(desc.Index()))
	} else {
		path = []int32{fileMessageField, int32(desc.Index())}
	}
	message := &Message{
		Desc:    desc,
		GoIdent: newGoIdent(f, desc),
		Path:    path,
	}
	gen.messagesByName[desc.FullName()] = message
	for i, mdescs := 0, desc.Messages(); i < mdescs.Len(); i++ {
		message.Messages = append(message.Messages, newMessage(gen, f, message, mdescs.Get(i)))
	}
	for i, edescs := 0, desc.Enums(); i < edescs.Len(); i++ {
		message.Enums = append(message.Enums, newEnum(gen, f, message, edescs.Get(i)))
	}
	for i, odescs := 0, desc.Oneofs(); i < odescs.Len(); i++ {
		message.Oneofs = append(message.Oneofs, newOneof(gen, f, message, odescs.Get(i)))
	}
	for i, fdescs := 0, desc.Fields(); i < fdescs.Len(); i++ {
		message.Fields = append(message.Fields, newField(gen, f, message, fdescs.Get(i)))
	}
	for i, extdescs := 0, desc.Extensions(); i < extdescs.Len(); i++ {
		message.Extensions = append(message.Extensions, newField(gen, f, message, extdescs.Get(i)))
	}

	// Field name conflict resolution.
	//
	// We assume well-known method names that may be attached to a generated
	// message type, as well as a 'Get*' method for each field. For each
	// field in turn, we add _s to its name until there are no conflicts.
	//
	// Any change to the following set of method names is a potential
	// incompatible API change because it may change generated field names.
	//
	// TODO: If we ever support a 'go_name' option to set the Go name of a
	// field, we should consider dropping this entirely. The conflict
	// resolution algorithm is subtle and surprising (changing the order
	// in which fields appear in the .proto source file can change the
	// names of fields in generated code), and does not adapt well to
	// adding new per-field methods such as setters.
	usedNames := map[string]bool{
		"Reset":               true,
		"String":              true,
		"ProtoMessage":        true,
		"Marshal":             true,
		"Unmarshal":           true,
		"ExtensionRangeArray": true,
		"ExtensionMap":        true,
		"Descriptor":          true,
	}
	makeNameUnique := func(name string) string {
		for usedNames[name] || usedNames["Get"+name] {
			name += "_"
		}
		usedNames[name] = true
		usedNames["Get"+name] = true
		return name
	}
	seenOneofs := make(map[int]bool)
	for _, field := range message.Fields {
		field.GoName = makeNameUnique(field.GoName)
		if field.OneofType != nil {
			if !seenOneofs[field.OneofType.Desc.Index()] {
				// If this is a field in a oneof that we haven't seen before,
				// make the name for that oneof unique as well.
				field.OneofType.GoName = makeNameUnique(field.OneofType.GoName)
				seenOneofs[field.OneofType.Desc.Index()] = true
			}
		}
	}

	return message
}

func (message *Message) init(gen *Plugin) error {
	for _, child := range message.Messages {
		if err := child.init(gen); err != nil {
			return err
		}
	}
	for _, field := range message.Fields {
		if err := field.init(gen); err != nil {
			return err
		}
	}
	for _, oneof := range message.Oneofs {
		oneof.init(gen, message)
	}
	for _, extension := range message.Extensions {
		if err := extension.init(gen); err != nil {
			return err
		}
	}
	return nil
}

// A Field describes a message field.
type Field struct {
	Desc protoreflect.FieldDescriptor

	// GoName is the base name of this field's Go field and methods.
	// For code generated by protoc-gen-go, this means a field named
	// '{{GoName}}' and a getter method named 'Get{{GoName}}'.
	GoName string

	ParentMessage *Message // message in which this field is defined; nil if top-level extension
	ExtendedType  *Message // extended message for extension fields; nil otherwise
	MessageType   *Message // type for message or group fields; nil otherwise
	EnumType      *Enum    // type for enum fields; nil otherwise
	OneofType     *Oneof   // containing oneof; nil if not part of a oneof
	Path          []int32  // location path of this field
}

func newField(gen *Plugin, f *File, message *Message, desc protoreflect.FieldDescriptor) *Field {
	var path []int32
	switch {
	case desc.ExtendedType() != nil && message == nil:
		path = []int32{fileExtensionField, int32(desc.Index())}
	case desc.ExtendedType() != nil && message != nil:
		path = pathAppend(message.Path, messageExtensionField, int32(desc.Index()))
	default:
		path = pathAppend(message.Path, messageFieldField, int32(desc.Index()))
	}
	field := &Field{
		Desc:          desc,
		GoName:        camelCase(string(desc.Name())),
		ParentMessage: message,
		Path:          path,
	}
	if desc.OneofType() != nil {
		field.OneofType = message.Oneofs[desc.OneofType().Index()]
	}
	return field
}

// Extension is an alias of Field for documentation.
type Extension = Field

func (field *Field) init(gen *Plugin) error {
	desc := field.Desc
	switch desc.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		mname := desc.MessageType().FullName()
		message, ok := gen.messagesByName[mname]
		if !ok {
			return fmt.Errorf("field %v: no descriptor for type %v", desc.FullName(), mname)
		}
		field.MessageType = message
	case protoreflect.EnumKind:
		ename := field.Desc.EnumType().FullName()
		enum, ok := gen.enumsByName[ename]
		if !ok {
			return fmt.Errorf("field %v: no descriptor for enum %v", desc.FullName(), ename)
		}
		field.EnumType = enum
	}
	if desc.ExtendedType() != nil {
		mname := desc.ExtendedType().FullName()
		message, ok := gen.messagesByName[mname]
		if !ok {
			return fmt.Errorf("field %v: no descriptor for type %v", desc.FullName(), mname)
		}
		field.ExtendedType = message
	}
	return nil
}

// A Oneof describes a oneof field.
type Oneof struct {
	Desc protoreflect.OneofDescriptor

	GoName        string   // Go field name of this oneof
	ParentMessage *Message // message in which this oneof occurs
	Fields        []*Field // fields that are part of this oneof
	Path          []int32  // location path of this oneof
}

func newOneof(gen *Plugin, f *File, message *Message, desc protoreflect.OneofDescriptor) *Oneof {
	return &Oneof{
		Desc:          desc,
		ParentMessage: message,
		GoName:        camelCase(string(desc.Name())),
		Path:          pathAppend(message.Path, messageOneofField, int32(desc.Index())),
	}
}

func (oneof *Oneof) init(gen *Plugin, parent *Message) {
	for i, fdescs := 0, oneof.Desc.Fields(); i < fdescs.Len(); i++ {
		oneof.Fields = append(oneof.Fields, parent.Fields[fdescs.Get(i).Index()])
	}
}

// An Enum describes an enum.
type Enum struct {
	Desc protoreflect.EnumDescriptor

	GoIdent GoIdent      // name of the generated Go type
	Values  []*EnumValue // enum values
	Path    []int32      // location path of this enum
}

func newEnum(gen *Plugin, f *File, parent *Message, desc protoreflect.EnumDescriptor) *Enum {
	var path []int32
	if parent != nil {
		path = pathAppend(parent.Path, messageEnumField, int32(desc.Index()))
	} else {
		path = []int32{fileEnumField, int32(desc.Index())}
	}
	enum := &Enum{
		Desc:    desc,
		GoIdent: newGoIdent(f, desc),
		Path:    path,
	}
	gen.enumsByName[desc.FullName()] = enum
	for i, evdescs := 0, enum.Desc.Values(); i < evdescs.Len(); i++ {
		enum.Values = append(enum.Values, newEnumValue(gen, f, parent, enum, evdescs.Get(i)))
	}
	return enum
}

// An EnumValue describes an enum value.
type EnumValue struct {
	Desc protoreflect.EnumValueDescriptor

	GoIdent GoIdent // name of the generated Go type
	Path    []int32 // location path of this enum value
}

func newEnumValue(gen *Plugin, f *File, message *Message, enum *Enum, desc protoreflect.EnumValueDescriptor) *EnumValue {
	// A top-level enum value's name is: EnumName_ValueName
	// An enum value contained in a message is: MessageName_ValueName
	//
	// Enum value names are not camelcased.
	parentIdent := enum.GoIdent
	if message != nil {
		parentIdent = message.GoIdent
	}
	name := parentIdent.GoName + "_" + string(desc.Name())
	return &EnumValue{
		Desc: desc,
		GoIdent: GoIdent{
			GoName:       name,
			GoImportPath: f.GoImportPath,
		},
		Path: pathAppend(enum.Path, enumValueField, int32(desc.Index())),
	}
}

// A GeneratedFile is a generated file.
type GeneratedFile struct {
	filename         string
	goImportPath     GoImportPath
	buf              bytes.Buffer
	packageNames     map[GoImportPath]GoPackageName
	usedPackageNames map[GoPackageName]bool
	manualImports    map[GoImportPath]bool
}

// NewGeneratedFile creates a new generated file with the given filename
// and import path.
func (gen *Plugin) NewGeneratedFile(filename string, goImportPath GoImportPath) *GeneratedFile {
	g := &GeneratedFile{
		filename:         filename,
		goImportPath:     goImportPath,
		packageNames:     make(map[GoImportPath]GoPackageName),
		usedPackageNames: make(map[GoPackageName]bool),
		manualImports:    make(map[GoImportPath]bool),
	}
	gen.genFiles = append(gen.genFiles, g)
	return g
}

// P prints a line to the generated output. It converts each parameter to a
// string following the same rules as fmt.Print. It never inserts spaces
// between parameters.
//
// TODO: .meta file annotations.
func (g *GeneratedFile) P(v ...interface{}) {
	for _, x := range v {
		switch x := x.(type) {
		case GoIdent:
			fmt.Fprint(&g.buf, g.QualifiedGoIdent(x))
		default:
			fmt.Fprint(&g.buf, x)
		}
	}
	fmt.Fprintln(&g.buf)
}

// QualifiedGoIdent returns the string to use for a Go identifier.
//
// If the identifier is from a different Go package than the generated file,
// the returned name will be qualified (package.name) and an import statement
// for the identifier's package will be included in the file.
func (g *GeneratedFile) QualifiedGoIdent(ident GoIdent) string {
	if ident.GoImportPath == g.goImportPath {
		return ident.GoName
	}
	if packageName, ok := g.packageNames[ident.GoImportPath]; ok {
		return string(packageName) + "." + ident.GoName
	}
	packageName := cleanPackageName(baseName(string(ident.GoImportPath)))
	for i, orig := 1, packageName; g.usedPackageNames[packageName]; i++ {
		packageName = orig + GoPackageName(strconv.Itoa(i))
	}
	g.packageNames[ident.GoImportPath] = packageName
	g.usedPackageNames[packageName] = true
	return string(packageName) + "." + ident.GoName
}

// Import ensures a package is imported by the generated file.
//
// Packages referenced by QualifiedGoIdent are automatically imported.
// Explicitly importing a package with Import is generally only necessary
// when the import will be blank (import _ "package").
func (g *GeneratedFile) Import(importPath GoImportPath) {
	g.manualImports[importPath] = true
}

// Write implements io.Writer.
func (g *GeneratedFile) Write(p []byte) (n int, err error) {
	return g.buf.Write(p)
}

// Content returns the contents of the generated file.
func (g *GeneratedFile) Content() ([]byte, error) {
	if !strings.HasSuffix(g.filename, ".go") {
		return g.buf.Bytes(), nil
	}

	// Reformat generated code.
	original := g.buf.Bytes()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", original, parser.ParseComments)
	if err != nil {
		// Print out the bad code with line numbers.
		// This should never happen in practice, but it can while changing generated code
		// so consider this a debugging aid.
		var src bytes.Buffer
		s := bufio.NewScanner(bytes.NewReader(original))
		for line := 1; s.Scan(); line++ {
			fmt.Fprintf(&src, "%5d\t%s\n", line, s.Bytes())
		}
		return nil, fmt.Errorf("%v: unparsable Go source: %v\n%v", g.filename, err, src.String())
	}

	// Add imports.
	var importPaths []string
	for importPath := range g.packageNames {
		importPaths = append(importPaths, string(importPath))
	}
	sort.Strings(importPaths)
	for _, importPath := range importPaths {
		astutil.AddNamedImport(fset, file, string(g.packageNames[GoImportPath(importPath)]), importPath)
	}
	for importPath := range g.manualImports {
		if _, ok := g.packageNames[importPath]; ok {
			continue
		}
		astutil.AddNamedImport(fset, file, "_", string(importPath))
	}
	ast.SortImports(fset, file)

	var out bytes.Buffer
	if err = (&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(&out, fset, file); err != nil {
		return nil, fmt.Errorf("%v: can not reformat Go source: %v", g.filename, err)
	}
	// TODO: Annotations.
	return out.Bytes(), nil

}

type pathType int

const (
	pathTypeImport pathType = iota
	pathTypeSourceRelative
)

// The SourceCodeInfo message describes the location of elements of a parsed
// .proto file by way of a "path", which is a sequence of integers that
// describe the route from a FileDescriptorProto to the relevant submessage.
// The path alternates between a field number of a repeated field, and an index
// into that repeated field. The constants below define the field numbers that
// are used.
//
// See descriptor.proto for more information about this.
const (
	// field numbers in FileDescriptorProto
	filePackageField   = 2 // package
	fileMessageField   = 4 // message_type
	fileEnumField      = 5 // enum_type
	fileExtensionField = 7 // extension
	// field numbers in DescriptorProto
	messageFieldField     = 2 // field
	messageMessageField   = 3 // nested_type
	messageEnumField      = 4 // enum_type
	messageExtensionField = 6 // extension
	messageOneofField     = 8 // oneof_decl
	// field numbers in EnumDescriptorProto
	enumValueField = 2 // value
)

// pathAppend appends elements to a location path.
// It does not alias the original path.
func pathAppend(path []int32, a ...int32) []int32 {
	var n []int32
	n = append(n, path...)
	n = append(n, a...)
	return n
}
