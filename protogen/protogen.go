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
func Run(f func(*Plugin) error) {
	if err := run(f); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", filepath.Base(os.Args[0]), err)
		os.Exit(1)
	}
}

func run(f func(*Plugin) error) error {
	in, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		return err
	}
	gen, err := New(req)
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

	fileReg  *protoregistry.Files
	pathType pathType
	genFiles []*GeneratedFile
	err      error
}

// New returns a new Plugin.
func New(req *pluginpb.CodeGeneratorRequest) (*Plugin, error) {
	gen := &Plugin{
		Request:     req,
		filesByName: make(map[string]*File),
		fileReg:     protoregistry.NewFiles(),
	}

	// TODO: Figure out how to pass parameters to the generator.
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
		case "plugins":
			// TODO
		case "annotate_code":
			// TODO
		default:
			if param[0] != 'M' {
				return nil, fmt.Errorf("unknown parameter %q", param)
			}
			importPaths[param[1:]] = GoImportPath(value)
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
	Desc protoreflect.FileDescriptor

	GoPackageName GoPackageName // name of this file's Go package
	GoImportPath  GoImportPath  // import path of this file's Go package
	Messages      []*Message    // top-level message declarations
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
		f.Messages = append(f.Messages, newMessage(gen, f, nil, mdescs.Get(i), i))
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

	GoIdent  GoIdent    // name of the generated Go type
	Messages []*Message // nested message declarations
}

func newMessage(gen *Plugin, f *File, parent *Message, desc protoreflect.MessageDescriptor, index int) *Message {
	m := &Message{
		Desc:    desc,
		GoIdent: newGoIdent(f, desc),
	}
	for i, mdescs := 0, desc.Messages(); i < mdescs.Len(); i++ {
		m.Messages = append(m.Messages, newMessage(gen, f, m, mdescs.Get(i), i))
	}
	return m
}

// A GeneratedFile is a generated file.
type GeneratedFile struct {
	filename         string
	goImportPath     GoImportPath
	buf              bytes.Buffer
	packageNames     map[GoImportPath]GoPackageName
	usedPackageNames map[GoPackageName]bool
}

// NewGeneratedFile creates a new generated file with the given filename
// and import path.
func (gen *Plugin) NewGeneratedFile(filename string, goImportPath GoImportPath) *GeneratedFile {
	g := &GeneratedFile{
		filename:         filename,
		goImportPath:     goImportPath,
		packageNames:     make(map[GoImportPath]GoPackageName),
		usedPackageNames: make(map[GoPackageName]bool),
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
			if x.GoImportPath != g.goImportPath {
				fmt.Fprint(&g.buf, g.goPackageName(x.GoImportPath))
				fmt.Fprint(&g.buf, ".")
			}
			fmt.Fprint(&g.buf, x.GoName)
		default:
			fmt.Fprint(&g.buf, x)
		}
	}
	fmt.Fprintln(&g.buf)
}

func (g *GeneratedFile) goPackageName(importPath GoImportPath) GoPackageName {
	if name, ok := g.packageNames[importPath]; ok {
		return name
	}
	name := cleanPackageName(baseName(string(importPath)))
	for i, orig := 1, name; g.usedPackageNames[name]; i++ {
		name = orig + GoPackageName(strconv.Itoa(i))
	}
	g.packageNames[importPath] = name
	g.usedPackageNames[name] = true
	return name
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
	ast, err := parser.ParseFile(fset, "", original, parser.ParseComments)
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
		astutil.AddNamedImport(fset, ast, string(g.packageNames[GoImportPath(importPath)]), importPath)
	}

	var out bytes.Buffer
	if err = (&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(&out, fset, ast); err != nil {
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
