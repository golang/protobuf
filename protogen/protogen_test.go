// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protogen

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	pluginpb "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func TestFiles(t *testing.T) {
	gen, err := New(makeRequest(t, "testdata/go_package/no_go_package_import.proto"))
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

func TestImports(t *testing.T) {
	gen, err := New(&pluginpb.CodeGeneratorRequest{})
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
	} {
		g.P("var _ = ", GoIdent{GoName: "X", GoImportPath: importPath}, " // ", importPath)
	}
	want := `package foo

import (
	bar "golang.org/x/bar"
	bar1 "golang.org/y/bar"
	baz "golang.org/x/baz"
)

var _ = X      // "golang.org/x/foo"
var _ = bar.X  // "golang.org/x/bar"
var _ = bar.X  // "golang.org/x/bar"
var _ = bar1.X // "golang.org/y/bar"
var _ = baz.X  // "golang.org/x/baz"
`
	got, err := g.Content()
	if err != nil {
		t.Fatalf("g.Content() = %v", err)
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
		Run(func(p *Plugin) error {
			g := p.NewGeneratedFile("request", "")
			return proto.MarshalText(g, p.Request)
		})
		os.Exit(0)
	}
}
