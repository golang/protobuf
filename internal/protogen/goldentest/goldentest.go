// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goldentest compares the output of a protoc plugin to golden files.
package goldentest

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Plugin should be called at init time with a function that acts as a
// protoc plugin.
func Plugin(f func()) {
	// When the environment variable RUN_AS_PROTOC_PLUGIN is set, we skip
	// running tests and instead act as protoc-gen-go. This allows the
	// test binary to pass itself to protoc.
	if os.Getenv("RUN_AS_PROTOC_PLUGIN") != "" {
		f()
		os.Exit(0)
	}
}

// Run executes golden tests.
func Run(t *testing.T, regenerate bool) {
	workdir, err := ioutil.TempDir("", "proto-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workdir)

	// Find all the proto files we need to compile. We assume that each directory
	// contains the files for a single package.
	packages := map[string][]string{}
	err = filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".proto") {
			return nil
		}
		dir := filepath.Dir(path)
		packages[dir] = append(packages[dir], path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Compile each package, using this binary as protoc-gen-go.
	for _, sources := range packages {
		args := []string{"-Itestdata", "--go_out=paths=source_relative:" + workdir}
		args = append(args, sources...)
		Protoc(t, args)
	}

	// Compare each generated file to the golden version.
	filepath.Walk(workdir, func(genPath string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		// For each generated file, figure out the path to the corresponding
		// golden file in the testdata directory.
		relPath, err := filepath.Rel(workdir, genPath)
		if err != nil {
			t.Errorf("filepath.Rel(%q, %q): %v", workdir, genPath, err)
			return nil
		}
		if filepath.SplitList(relPath)[0] == ".." {
			t.Errorf("generated file %q is not relative to %q", genPath, workdir)
		}
		goldenPath := filepath.Join("testdata", relPath)

		got, err := ioutil.ReadFile(genPath)
		if err != nil {
			t.Error(err)
			return nil
		}
		if regenerate {
			// If --regenerate set, just rewrite the golden files.
			err := ioutil.WriteFile(goldenPath, got, 0666)
			if err != nil {
				t.Error(err)
			}
			return nil
		}

		want, err := ioutil.ReadFile(goldenPath)
		if err != nil {
			t.Error(err)
			return nil
		}

		want = fdescRE.ReplaceAll(want, nil)
		got = fdescRE.ReplaceAll(got, nil)
		if bytes.Equal(got, want) {
			return nil
		}

		cmd := exec.Command("diff", "-u", goldenPath, genPath)
		out, _ := cmd.CombinedOutput()
		t.Errorf("golden file differs: %v\n%v", relPath, string(out))
		return nil
	})
}

var fdescRE = regexp.MustCompile(`(?ms)^var fileDescriptor.*}`)

// Protoc runs protoc, using the function registered with Plugin as the protoc-gen-go plugin.
func Protoc(t *testing.T, args []string) {
	t.Helper()
	cmd := exec.Command("protoc", "--plugin=protoc-gen-go="+os.Args[0])
	cmd.Args = append(cmd.Args, args...)
	// We set the RUN_AS_PROTOC_PLUGIN environment variable to indicate that
	// the subprocess should act as a proto compiler rather than a test.
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
}
