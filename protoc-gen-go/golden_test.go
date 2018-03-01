package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
		args := []string{"-Itestdata", "--go_out=plugins=grpc:" + workdir}
		args = append(args, sources...)
		protoc(t, args)
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

		cmd := exec.Command("diff", "-u", goldenPath, genPath)
		out, _ := cmd.CombinedOutput()
		t.Errorf("golden file differs: %v\n%v", relPath, string(out))
		return nil
	})
}

var fdescRE = regexp.MustCompile(`(?ms)^var fileDescriptor.*}`)

func TestParameters(t *testing.T) {
	for _, test := range []struct {
		parameters  string
		wantFiles   map[string]bool
		wantImports map[string]bool
		goPackage   string
	}{
		{
			parameters: "",
			wantFiles: map[string]bool{
				"package/alpha/a.pb.go": true,
				"beta/b.pb.go":          true,
			},
			wantImports: map[string]bool{
				"github.com/golang/protobuf/proto": true,
				"beta": true,
			},
		},
		{
			parameters: "import_prefix=prefix",
			wantFiles: map[string]bool{
				"package/alpha/a.pb.go": true,
				"beta/b.pb.go":          true,
			},
			wantImports: map[string]bool{
				// This really doesn't seem like useful behavior.
				"prefixgithub.com/golang/protobuf/proto": true,
				"prefixbeta":                             true,
			},
		},
		{
			// import_path only affects the 'package' line.
			parameters: "import_path=import/path/of/pkg",
			wantFiles: map[string]bool{
				"package/alpha/a.pb.go": true,
				"beta/b.pb.go":          true,
			},
		},
		{
			parameters: "Mbeta/b.proto=package/gamma",
			wantFiles: map[string]bool{
				"package/alpha/a.pb.go": true,
				"beta/b.pb.go":          true,
			},
			wantImports: map[string]bool{
				"github.com/golang/protobuf/proto": true,
				// Rewritten by the M parameter.
				"package/gamma": true,
			},
		},
		{
			parameters: "import_prefix=prefix,Mbeta/b.proto=package/gamma",
			wantFiles: map[string]bool{
				"package/alpha/a.pb.go": true,
				"beta/b.pb.go":          true,
			},
			wantImports: map[string]bool{
				// import_prefix applies after M.
				"prefixpackage/gamma": true,
			},
		},
	} {
		name := test.parameters
		if name == "" {
			name = "defaults"
		}
		t.Run(name, func(t *testing.T) {
			workdir, err := ioutil.TempDir("", "proto-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(workdir)

			for _, dir := range []string{"alpha", "beta", "out"} {
				if err := os.MkdirAll(filepath.Join(workdir, dir), 0777); err != nil {
					t.Fatal(err)
				}
			}

			aProto := []byte(`
syntax = "proto3";
package alpha;
option go_package = "package/alpha";
import "beta/b.proto";
message M { beta.M field = 1; }
`)
			if err := ioutil.WriteFile(filepath.Join(workdir, "alpha", "a.proto"), aProto, 0666); err != nil {
				t.Fatal(err)
			}

			bProto := []byte(`
syntax = "proto3";
package beta;
// no go_package option
message M {}
`)
			if err := ioutil.WriteFile(filepath.Join(workdir, "beta", "b.proto"), bProto, 0666); err != nil {
				t.Fatal(err)
			}

			protoc(t, []string{
				"-I" + workdir,
				"--go_out=" + test.parameters + ":" + filepath.Join(workdir, "out"),
				filepath.Join(workdir, "alpha", "a.proto"),
			})
			protoc(t, []string{
				"-I" + workdir,
				"--go_out=" + test.parameters + ":" + filepath.Join(workdir, "out"),
				filepath.Join(workdir, "beta", "b.proto"),
			})

			var aGen string
			gotFiles := make(map[string]bool)
			outdir := filepath.Join(workdir, "out")
			filepath.Walk(outdir, func(p string, info os.FileInfo, _ error) error {
				if info.IsDir() {
					return nil
				}
				if filepath.Base(p) == "a.pb.go" {
					b, err := ioutil.ReadFile(p)
					if err != nil {
						t.Fatal(err)
					}
					aGen = string(b)
				}
				relPath, _ := filepath.Rel(outdir, p)
				gotFiles[relPath] = true
				return nil
			})
			for got := range gotFiles {
				if !test.wantFiles[got] {
					t.Errorf("unexpected output file: %v", got)
				}
			}
			for want := range test.wantFiles {
				if !gotFiles[want] {
					t.Errorf("missing output file:    %v", want)
				}
			}
			missingImport := false
			for want := range test.wantImports {
				// For each import, just check if there's a string which
				// matches it. We could parse the file and do a more
				// rigorous check, but that seems like overkill.
				if strings.Contains(aGen, strconv.Quote(want)) {
					continue
				}
				t.Errorf("output file a.pb.go does not contain expected import %q", want)
				missingImport = true
			}
			if missingImport {
				t.Error("got imports:")
				for _, line := range strings.Split(aGen, "\n") {
					if strings.HasPrefix(line, "import") {
						t.Errorf("  %v", line)
					}
				}
			}
		})
	}
}

func protoc(t *testing.T, args []string) {
	cmd := exec.Command("protoc", "--plugin=protoc-gen-go="+os.Args[0])
	cmd.Args = append(cmd.Args, args...)
	cmd.Env = append(os.Environ(), "RUN_AS_PROTOC_GEN_GO=1")
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
