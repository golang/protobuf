// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run . -execute

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gengo "github.com/golang/protobuf/v2/cmd/protoc-gen-go/internal_gengo"
	"github.com/golang/protobuf/v2/protogen"
)

func init() {
	// When the environment variable RUN_AS_PROTOC_PLUGIN is set,
	// we skip running main and instead act as a protoc plugin.
	// This allows the binary to pass itself to protoc.
	if os.Getenv("RUN_AS_PROTOC_PLUGIN") == "1" {
		protogen.Run(nil, func(gen *protogen.Plugin) error {
			for _, file := range gen.Files {
				if file.Generate {
					gengo.GenerateFile(gen, file)
				}
			}
			return nil
		})
		os.Exit(0)
	}
}

var (
	run        bool
	protoRoot  string
	repoRoot   string
	modulePath string
)

func main() {
	flag.BoolVar(&run, "execute", false, "Write generated files to destination.")
	flag.StringVar(&protoRoot, "protoroot", os.Getenv("PROTOBUF_ROOT"), "The root of the protobuf source tree.")
	flag.Parse()
	if protoRoot == "" {
		panic("protobuf source root is not set")
	}

	// Determine repository root path.
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	check(err)
	repoRoot = strings.TrimSpace(string(out))

	// Determine the module path.
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}")
	cmd.Dir = repoRoot
	out, err = cmd.CombinedOutput()
	check(err)
	modulePath = strings.TrimSpace(string(out))

	generateLocalProtos()
}

func generateLocalProtos() {
	tmpDir, err := ioutil.TempDir(repoRoot, "tmp")
	check(err)
	defer os.RemoveAll(tmpDir)

	// Generate all local proto files.
	dirs := []struct {
		path     string
		relative bool
	}{
		{path: "jsonpb", relative: true},
		{path: "proto", relative: true},
		{path: "protoc-gen-go"},
		{path: "ptypes"},
	}
	for _, d := range dirs {
		srcDir := filepath.Join(repoRoot, filepath.FromSlash(d.path))
		filepath.Walk(srcDir, func(srcPath string, _ os.FileInfo, err error) error {
			if !strings.HasSuffix(srcPath, ".proto") {
				return nil
			}

			var impPath, relPath string
			if d.relative {
				impPath = srcDir

				relPath, err = filepath.Rel(srcDir, srcPath)
				check(err)
			} else {
				impPath = tmpDir

				relPath, err = filepath.Rel(repoRoot, srcPath)
				check(err)
				relPath = filepath.Join(filepath.FromSlash(modulePath), relPath)

				dstDir := filepath.Join(tmpDir, filepath.Dir(relPath))
				check(os.MkdirAll(dstDir, 0775))
				check(os.Link(srcPath, filepath.Join(tmpDir, relPath)))
			}

			protoc("-I"+filepath.Join(protoRoot, "src"), "-I"+impPath, "--go_out="+tmpDir, relPath)
			return nil
		})
	}

	syncOutput(repoRoot, filepath.Join(tmpDir, filepath.FromSlash(modulePath)))
}

func protoc(args ...string) {
	cmd := exec.Command("protoc", "--plugin=protoc-gen-go="+os.Args[0])
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = append(os.Environ(), "RUN_AS_PROTOC_PLUGIN=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(args)
		fmt.Printf("executing: %v\n%s\n", strings.Join(cmd.Args, " "), out)
	}
	check(err)
}

func syncOutput(dstDir, srcDir string) {
	filepath.Walk(srcDir, func(srcPath string, _ os.FileInfo, _ error) error {
		if !strings.HasSuffix(srcPath, ".go") {
			return nil
		}
		relPath, err := filepath.Rel(srcDir, srcPath)
		check(err)
		dstPath := filepath.Join(dstDir, relPath)

		if run {
			fmt.Println("#", relPath)
			b, err := ioutil.ReadFile(srcPath)
			check(err)
			check(os.MkdirAll(filepath.Dir(dstPath), 0775))
			check(ioutil.WriteFile(dstPath, b, 0664))
		} else {
			cmd := exec.Command("diff", dstPath, srcPath, "-N", "-u")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
		return nil
	})
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
