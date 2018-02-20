package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Set --regenerate to regenerate the golden files.
var regenerate = flag.Bool("regenerate", false, "regenerate golden files")

// When the environment variable RUN_AS_PROTOC_GEN_GO is set, we skip running
// tests and instead act as protoc-gen-go. This allows the test binary to
// pass itself to protoc.
func init() {
	if os.Getenv("RUN_AS_PROTOC_GEN_GO") != "" {
		main()
		os.Exit(0)
	}
}

func TestGolden(t *testing.T) {
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
	//
	// We set the RUN_AS_PROTOC_GEN_GO environment variable to indicate that
	// the subprocess should act as a proto compiler rather than a test.
	for _, sources := range packages {
		cmd := exec.Command(
			"protoc",
			"--plugin=protoc-gen-go="+os.Args[0],
			"-Itestdata",
			"--go_out="+workdir,
		)
		cmd.Args = append(cmd.Args, sources...)
		cmd.Env = append(os.Environ(), "RUN_AS_PROTOC_GEN_GO=1")
		t.Log(strings.Join(cmd.Args, " "))
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			t.Log(string(out))
		}
		if err != nil {
			t.Fatalf("failed to compile: %v", sources)
		}
	}

	// Compare each generated file to the golden version.
	relRoot := filepath.Join(workdir, "github.com/golang/protobuf/protoc-gen-go/testdata")
	filepath.Walk(workdir, func(genPath string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		// For each generated file, figure out the path to the corresponding
		// golden file in the testdata directory.
		relPath, err := filepath.Rel(relRoot, genPath)
		if err != nil {
			t.Errorf("filepath.Rel(%q, %q): %v", relRoot, genPath, err)
			return nil
		}
		if filepath.SplitList(relPath)[0] == ".." {
			t.Errorf("generated file %q is not relative to %q", genPath, relRoot)
		}
		goldenPath := filepath.Join("testdata", relPath)

		got, err := ioutil.ReadFile(genPath)
		if err != nil {
			t.Error(err)
			return nil
		}
		if *regenerate {
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

		//

		cmd := exec.Command("diff", "-u", goldenPath, genPath)
		out, _ := cmd.CombinedOutput()
		t.Errorf("golden file differs: %v\n%v", relPath, string(out))
		return nil
	})
}

var fdescRE = regexp.MustCompile(`(?ms)^var fileDescriptor.*}`)
