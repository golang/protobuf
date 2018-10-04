// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build golden

package protogen

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	pluginpb "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func TestPluginParameters(t *testing.T) {
	var flags flag.FlagSet
	value := flags.Int("integer", 0, "")
	opts := &Options{
		ParamFunc: flags.Set,
	}
	const params = "integer=2"
	_, err := New(&pluginpb.CodeGeneratorRequest{
		Parameter: proto.String(params),
	}, opts)
	if err != nil {
		t.Errorf("New(generator parameters %q): %v", params, err)
	}
	if *value != 2 {
		t.Errorf("New(generator parameters %q): integer=%v, want 2", params, *value)
	}
}

func TestPluginParameterErrors(t *testing.T) {
	for _, parameter := range []string{
		"unknown=1",
		"boolean=error",
	} {
		var flags flag.FlagSet
		flags.Bool("boolean", false, "")
		opts := &Options{
			ParamFunc: flags.Set,
		}
		_, err := New(&pluginpb.CodeGeneratorRequest{
			Parameter: proto.String(parameter),
		}, opts)
		if err == nil {
			t.Errorf("New(generator parameters %q): want error, got nil", parameter)
		}
	}
}

func TestFiles(t *testing.T) {
	gen, err := New(makeRequest(t, "testdata/go_package/no_go_package_import.proto"), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		path         string
		wantGenerate bool
	}{
		{
			path:         "go_package/no_go_package_import.proto",
			wantGenerate: true,
		},
		{
			path:         "go_package/no_go_package.proto",
			wantGenerate: false,
		},
	} {
		f, ok := gen.FileByName(test.path)
		if !ok {
			t.Errorf("%q: not found by gen.FileByName", test.path)
			continue
		}
		if f.Generate != test.wantGenerate {
			t.Errorf("%q: Generate=%v, want %v", test.path, f.Generate, test.wantGenerate)
		}
	}
}

func TestPackageNamesAndPaths(t *testing.T) {
	const (
		filename         = "dir/filename.proto"
		protoPackageName = "proto.package"
	)
	for _, test := range []struct {
		desc               string
		parameter          string
		goPackageOption    string
		generate           bool
		wantPackageName    GoPackageName
		wantImportPath     GoImportPath
		wantFilenamePrefix string
	}{
		{
			desc:               "no parameters, no go_package option",
			generate:           true,
			wantPackageName:    "proto_package",
			wantImportPath:     "dir",
			wantFilenamePrefix: "dir/filename",
		},
		{
			desc:               "go_package option sets import path",
			goPackageOption:    "golang.org/x/foo",
			generate:           true,
			wantPackageName:    "foo",
			wantImportPath:     "golang.org/x/foo",
			wantFilenamePrefix: "golang.org/x/foo/filename",
		},
		{
			desc:               "go_package option sets import path and package",
			goPackageOption:    "golang.org/x/foo;bar",
			generate:           true,
			wantPackageName:    "bar",
			wantImportPath:     "golang.org/x/foo",
			wantFilenamePrefix: "golang.org/x/foo/filename",
		},
		{
			desc:               "go_package option sets package",
			goPackageOption:    "foo",
			generate:           true,
			wantPackageName:    "foo",
			wantImportPath:     "dir",
			wantFilenamePrefix: "dir/filename",
		},
		{
			desc:               "command line sets import path for a file",
			parameter:          "Mdir/filename.proto=golang.org/x/bar",
			goPackageOption:    "golang.org/x/foo",
			generate:           true,
			wantPackageName:    "foo",
			wantImportPath:     "golang.org/x/bar",
			wantFilenamePrefix: "golang.org/x/foo/filename",
		},
		{
			desc:               "import_path parameter sets import path of generated files",
			parameter:          "import_path=golang.org/x/bar",
			goPackageOption:    "golang.org/x/foo",
			generate:           true,
			wantPackageName:    "foo",
			wantImportPath:     "golang.org/x/bar",
			wantFilenamePrefix: "golang.org/x/foo/filename",
		},
		{
			desc:               "import_path parameter does not set import path of dependencies",
			parameter:          "import_path=golang.org/x/bar",
			goPackageOption:    "golang.org/x/foo",
			generate:           false,
			wantPackageName:    "foo",
			wantImportPath:     "golang.org/x/foo",
			wantFilenamePrefix: "golang.org/x/foo/filename",
		},
	} {
		context := fmt.Sprintf(`
TEST: %v
  --go_out=%v:.
  file %q: generate=%v
  option go_package = %q;

  `,
			test.desc, test.parameter, filename, test.generate, test.goPackageOption)

		req := &pluginpb.CodeGeneratorRequest{
			Parameter: proto.String(test.parameter),
			ProtoFile: []*descpb.FileDescriptorProto{
				{
					Name:    proto.String(filename),
					Package: proto.String(protoPackageName),
					Options: &descpb.FileOptions{
						GoPackage: proto.String(test.goPackageOption),
					},
				},
			},
		}
		if test.generate {
			req.FileToGenerate = []string{filename}
		}
		gen, err := New(req, nil)
		if err != nil {
			t.Errorf("%vNew(req) = %v", context, err)
			continue
		}
		gotFile, ok := gen.FileByName(filename)
		if !ok {
			t.Errorf("%v%v: missing file info", context, filename)
			continue
		}
		if got, want := gotFile.GoPackageName, test.wantPackageName; got != want {
			t.Errorf("%vGoPackageName=%v, want %v", context, got, want)
		}
		if got, want := gotFile.GoImportPath, test.wantImportPath; got != want {
			t.Errorf("%vGoImportPath=%v, want %v", context, got, want)
		}
		if got, want := gotFile.GeneratedFilenamePrefix, test.wantFilenamePrefix; got != want {
			t.Errorf("%vGeneratedFilenamePrefix=%v, want %v", context, got, want)
		}
	}
}

func TestPackageNameInference(t *testing.T) {
	gen, err := New(&pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descpb.FileDescriptorProto{
			{
				Name:    proto.String("dir/file1.proto"),
				Package: proto.String("proto.package"),
			},
			{
				Name:    proto.String("dir/file2.proto"),
				Package: proto.String("proto.package"),
				Options: &descpb.FileOptions{
					GoPackage: proto.String("foo"),
				},
			},
		},
		FileToGenerate: []string{"dir/file1.proto", "dir/file2.proto"},
	}, nil)
	if err != nil {
		t.Fatalf("New(req) = %v", err)
	}
	if f1, ok := gen.FileByName("dir/file1.proto"); !ok {
		t.Errorf("missing file info for dir/file1.proto")
	} else if f1.GoPackageName != "foo" {
		t.Errorf("dir/file1.proto: GoPackageName=%v, want foo; package name should be derived from dir/file2.proto", f1.GoPackageName)
	}
}

func TestInconsistentPackageNames(t *testing.T) {
	_, err := New(&pluginpb.CodeGeneratorRequest{
		ProtoFile: []*descpb.FileDescriptorProto{
			{
				Name:    proto.String("dir/file1.proto"),
				Package: proto.String("proto.package"),
				Options: &descpb.FileOptions{
					GoPackage: proto.String("golang.org/x/foo"),
				},
			},
			{
				Name:    proto.String("dir/file2.proto"),
				Package: proto.String("proto.package"),
				Options: &descpb.FileOptions{
					GoPackage: proto.String("golang.org/x/foo;bar"),
				},
			},
		},
		FileToGenerate: []string{"dir/file1.proto", "dir/file2.proto"},
	}, nil)
	if err == nil {
		t.Fatalf("inconsistent package names for the same import path: New(req) = nil, want error")
	}
}

func TestImports(t *testing.T) {
	gen, err := New(&pluginpb.CodeGeneratorRequest{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := gen.NewGeneratedFile("foo.go", "golang.org/x/foo")
	g.P("package foo")
	g.P()
	for _, importPath := range []GoImportPath{
		"golang.org/x/foo",
		// Multiple references to the same package.
		"golang.org/x/bar",
		"golang.org/x/bar",
		// Reference to a different package with the same basename.
		"golang.org/y/bar",
		"golang.org/x/baz",
		// Reference to a package conflicting with a predeclared identifier.
		"golang.org/z/string",
	} {
		g.P("var _ = ", GoIdent{GoName: "X", GoImportPath: importPath}, " // ", importPath)
	}
	want := `package foo

import (
	bar "golang.org/x/bar"
	baz "golang.org/x/baz"
	bar1 "golang.org/y/bar"
	string1 "golang.org/z/string"
)

var _ = X         // "golang.org/x/foo"
var _ = bar.X     // "golang.org/x/bar"
var _ = bar.X     // "golang.org/x/bar"
var _ = bar1.X    // "golang.org/y/bar"
var _ = baz.X     // "golang.org/x/baz"
var _ = string1.X // "golang.org/z/string"
`
	got, err := g.content()
	if err != nil {
		t.Fatalf("g.content() = %v", err)
	}
	if want != string(got) {
		t.Fatalf(`want:
==========
%v
==========

got:
==========
%v
==========`,
			want, string(got))
	}
}

func TestImportRewrites(t *testing.T) {
	gen, err := New(&pluginpb.CodeGeneratorRequest{}, &Options{
		ImportRewriteFunc: func(i GoImportPath) GoImportPath {
			return "prefix/" + i
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	g := gen.NewGeneratedFile("foo.go", "golang.org/x/foo")
	g.P("package foo")
	g.P("var _ = ", GoIdent{GoName: "X", GoImportPath: "golang.org/x/bar"})
	want := `package foo

import bar "prefix/golang.org/x/bar"

var _ = bar.X
`
	got, err := g.content()
	if err != nil {
		t.Fatalf("g.content() = %v", err)
	}
	if want != string(got) {
		t.Fatalf(`want:
==========
%v
==========

got:
==========
%v
==========`,
			want, string(got))
	}
}

// makeRequest returns a CodeGeneratorRequest for the given protoc inputs.
//
// It does this by running protoc with the current binary as the protoc-gen-go
// plugin. This "plugin" produces a single file, named 'request', which contains
// the code generator request.
func makeRequest(t *testing.T, args ...string) *pluginpb.CodeGeneratorRequest {
	workdir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workdir)

	cmd := exec.Command("protoc", "--plugin=protoc-gen-go="+os.Args[0])
	cmd.Args = append(cmd.Args, "--go_out="+workdir, "-Itestdata")
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = append(os.Environ(), "RUN_AS_PROTOC_PLUGIN=1")
	out, err := cmd.CombinedOutput()
	if len(out) > 0 || err != nil {
		t.Log("RUNNING: ", strings.Join(cmd.Args, " "))
	}
	if len(out) > 0 {
		t.Log(string(out))
	}
	if err != nil {
		t.Fatalf("protoc: %v", err)
	}

	b, err := ioutil.ReadFile(filepath.Join(workdir, "request"))
	if err != nil {
		t.Fatal(err)
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.UnmarshalText(string(b), req); err != nil {
		t.Fatal(err)
	}
	return req
}

func init() {
	if os.Getenv("RUN_AS_PROTOC_PLUGIN") != "" {
		Run(nil, func(p *Plugin) error {
			g := p.NewGeneratedFile("request", "")
			return proto.MarshalText(g, p.Request)
		})
		os.Exit(0)
	}
}
