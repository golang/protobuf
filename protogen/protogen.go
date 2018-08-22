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
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pluginpb "github.com/golang/protobuf/protoc-gen-go/plugin"
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

	packageImportPath string // Go import path of the package we're generating code for.

	genFiles []*GeneratedFile
	err      error
}

// New returns a new Plugin.
func New(req *pluginpb.CodeGeneratorRequest) (*Plugin, error) {
	gen := &Plugin{
		Request:     req,
		filesByName: make(map[string]*File),
	}

	// TODO: Figure out how to pass parameters to the generator.
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
			gen.packageImportPath = value
		case "paths":
			// TODO
		case "plugins":
			// TODO
		case "annotate_code":
			// TODO
		default:
			if param[0] != 'M' {
				return nil, fmt.Errorf("unknown parameter %q", param)
			}
			// TODO
		}
	}

	for _, fdesc := range gen.Request.ProtoFile {
		f := newFile(gen, fdesc)
		name := f.Desc.GetName()
		if gen.filesByName[name] != nil {
			return nil, fmt.Errorf("duplicate file name: %q", name)
		}
		gen.Files = append(gen.Files, f)
		gen.filesByName[name] = f
	}
	for _, name := range gen.Request.FileToGenerate {
		f, ok := gen.FileByName(name)
		if !ok {
			return nil, fmt.Errorf("no descriptor for generated file: %v", name)
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
			Name:    proto.String(gf.path),
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
	Desc *descpb.FileDescriptorProto // TODO: protoreflect.FileDescriptor

	Messages []*Message // top-level message declartions
	Generate bool       // true if we should generate code for this file
}

func newFile(gen *Plugin, p *descpb.FileDescriptorProto) *File {
	f := &File{
		Desc: p,
	}
	for _, d := range p.MessageType {
		f.Messages = append(f.Messages, newMessage(gen, nil, d))
	}
	return f
}

// A Message describes a message.
type Message struct {
	Desc *descpb.DescriptorProto // TODO: protoreflect.MessageDescriptor

	GoIdent  GoIdent    // name of the generated Go type
	Messages []*Message // nested message declarations
}

func newMessage(gen *Plugin, parent *Message, p *descpb.DescriptorProto) *Message {
	m := &Message{
		Desc:    p,
		GoIdent: camelCase(p.GetName()),
	}
	if parent != nil {
		m.GoIdent = parent.GoIdent + "_" + m.GoIdent
	}
	for _, nested := range p.GetNestedType() {
		m.Messages = append(m.Messages, newMessage(gen, m, nested))
	}
	return m
}

// A GeneratedFile is a generated file.
type GeneratedFile struct {
	path string
	buf  bytes.Buffer
}

// NewGeneratedFile creates a new generated file with the given path.
func (gen *Plugin) NewGeneratedFile(path string) *GeneratedFile {
	g := &GeneratedFile{
		path: path,
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
		fmt.Fprint(&g.buf, x)
	}
	fmt.Fprintln(&g.buf)
}

// Write implements io.Writer.
func (g *GeneratedFile) Write(p []byte) (n int, err error) {
	return g.buf.Write(p)
}

// Content returns the contents of the generated file.
func (g *GeneratedFile) Content() ([]byte, error) {
	if !strings.HasSuffix(g.path, ".go") {
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
		return nil, fmt.Errorf("%v: unparsable Go source: %v\n%v", g.path, err, src.String())
	}
	var out bytes.Buffer
	if err = (&printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}).Fprint(&out, fset, ast); err != nil {
		return nil, fmt.Errorf("%v: can not reformat Go source: %v", g.path, err)
	}
	// TODO: Patch annotation locations.
	return out.Bytes(), nil

}
