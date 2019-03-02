// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

var (
	regenerate = flag.Bool("regenerate", false, "regenerate files")

	protobufVersion = "3.6.1"
	golangVersions  = []string{"1.9.7", "1.10.8", "1.11.5", "1.12"}
	golangLatest    = golangVersions[len(golangVersions)-1]

	// List of directories to avoid auto-deleting. After updating the versions,
	// it is worth placing the previous versions here for some time since
	// other developers may still have working branches using the old versions.
	keepDirs = []string{"go1.10.7", "go1.11.4"}

	// Variables initialized by mustInitDeps.
	goPath     string
	modulePath string
)

func Test(t *testing.T) {
	mustInitDeps(t)

	if *regenerate {
		t.Run("Generate", func(t *testing.T) {
			fmt.Print(mustRunCommand(t, ".", "go", "run", "./internal/cmd/generate-types", "-execute"))
			fmt.Print(mustRunCommand(t, ".", "go", "run", "./internal/cmd/generate-protos", "-execute"))
			files := strings.Split(strings.TrimSpace(mustRunCommand(t, ".", "git", "ls-files", "*.go")), "\n")
			mustRunCommand(t, ".", append([]string{"gofmt", "-w"}, files...)...)
		})
		t.SkipNow()
	}

	for _, v := range golangVersions {
		t.Run("Go"+v, func(t *testing.T) {
			runGo := func(label string, args ...string) {
				args[0] += v
				t.Run(label, func(t *testing.T) {
					t.Parallel()
					mustRunCommand(t, filepath.Join(goPath, "src", modulePath), args...)
				})
			}
			// TODO: "go build" does not descend into testdata,
			// which means that generated .pb.go files are not being built.
			runGo("Build", "go", "build", "./...")
			runGo("TestNormal", "go", "test", "-race", "./...")
			runGo("TestPureGo", "go", "test", "-race", "-tags", "purego", "./...")
			if v == golangLatest {
				runGo("TestLegacy", "go", "test", "-race", "-tags", "proto1_legacy", "./...")
			}
		})
	}

	t.Run("GeneratedGoFiles", func(t *testing.T) {
		diff := mustRunCommand(t, ".", "go", "run", "./internal/cmd/generate-types")
		if strings.TrimSpace(diff) != "" {
			t.Fatalf("stale generated files:\n%v", diff)
		}
		diff = mustRunCommand(t, ".", "go", "run", "./internal/cmd/generate-protos")
		if strings.TrimSpace(diff) != "" {
			t.Fatalf("stale generated files:\n%v", diff)
		}
	})
	t.Run("FormattedGoFiles", func(t *testing.T) {
		files := strings.Split(strings.TrimSpace(mustRunCommand(t, ".", "git", "ls-files", "*.go")), "\n")
		diff := mustRunCommand(t, ".", append([]string{"gofmt", "-d"}, files...)...)
		if strings.TrimSpace(diff) != "" {
			t.Fatalf("unformatted source files:\n%v", diff)
		}
	})
	t.Run("CommittedGitChanges", func(t *testing.T) {
		diff := mustRunCommand(t, ".", "git", "diff", "--no-prefix", "HEAD")
		if strings.TrimSpace(diff) != "" {
			t.Fatalf("uncommitted changes:\n%v", diff)
		}
	})
	t.Run("TrackedGitFiles", func(t *testing.T) {
		diff := mustRunCommand(t, ".", "git", "ls-files", "--others", "--exclude-standard")
		if strings.TrimSpace(diff) != "" {
			t.Fatalf("untracked files:\n%v", diff)
		}
	})
}

func mustInitDeps(t *testing.T) {
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	// Determine the directory to place the test directory.
	repoRoot, err := os.Getwd()
	check(err)
	testDir := filepath.Join(repoRoot, ".cache")
	check(os.MkdirAll(testDir, 0775))

	// Travis-CI has a hard-coded timeout where it kills the test after
	// 10 minutes of a lack of activity on stdout.
	// We work around this restriction by periodically printing the timestamp.
	ticker := time.NewTicker(5 * time.Minute)
	done := make(chan struct{})
	go func() {
		now := time.Now()
		for {
			select {
			case t := <-ticker.C:
				fmt.Printf("\tt=%0.fmin\n", t.Sub(now).Minutes())
			case <-done:
				return
			}
		}
	}()
	defer close(done)
	defer ticker.Stop()

	// Delete the current directory if non-empty,
	// which only occurs if a dependency failed to initialize properly.
	var workingDir string
	defer func() {
		if workingDir != "" {
			os.RemoveAll(workingDir) // best-effort
		}
	}()

	// Delete other sub-directories that are no longer relevant.
	defer func() {
		subDirs := map[string]bool{"bin": true, "gocache": true, "gopath": true}
		subDirs["protobuf-"+protobufVersion] = true
		for _, v := range golangVersions {
			subDirs["go"+v] = true
		}
		for _, d := range keepDirs {
			subDirs[d] = true
		}

		fis, _ := ioutil.ReadDir(testDir)
		for _, fi := range fis {
			if !subDirs[fi.Name()] {
				fmt.Printf("delete %v\n", fi.Name())
				os.RemoveAll(filepath.Join(testDir, fi.Name())) // best-effort
			}
		}
	}()

	// The bin directory contains symlinks to each tool by version.
	// It is safe to delete this directory and run the test script from scratch.
	binPath := filepath.Join(testDir, "bin")
	check(os.RemoveAll(binPath))
	check(os.Mkdir(binPath, 0775))
	check(os.Setenv("PATH", binPath+":"+os.Getenv("PATH")))
	registerBinary := func(name, path string) {
		check(os.Symlink(path, filepath.Join(binPath, name)))
	}

	// Download and build the protobuf toolchain.
	// We avoid downloading the pre-compiled binaries since they do not contain
	// the conformance test runner.
	workingDir = filepath.Join(testDir, "protobuf-"+protobufVersion)
	if _, err := os.Stat(workingDir); err != nil {
		fmt.Printf("download %v\n", filepath.Base(workingDir))
		url := fmt.Sprintf("https://github.com/google/protobuf/releases/download/v%v/protobuf-all-%v.tar.gz", protobufVersion, protobufVersion)
		downloadArchive(check, workingDir, url, "protobuf-"+protobufVersion)

		fmt.Printf("build %v\n", filepath.Base(workingDir))
		mustRunCommand(t, workingDir, "./autogen.sh")
		mustRunCommand(t, workingDir, "./configure")
		mustRunCommand(t, workingDir, "make")
		mustRunCommand(t, filepath.Join(workingDir, "conformance"), "make")
	}
	patchProtos(check, workingDir)
	check(os.Setenv("PROTOBUF_ROOT", workingDir)) // for generate-protos
	registerBinary("conform-test-runner", filepath.Join(workingDir, "conformance", "conformance-test-runner"))
	registerBinary("protoc", filepath.Join(workingDir, "src", "protoc"))
	workingDir = ""

	// Download each Go toolchain version.
	for _, v := range golangVersions {
		workingDir = filepath.Join(testDir, "go"+v)
		if _, err := os.Stat(workingDir); err != nil {
			fmt.Printf("download %v\n", filepath.Base(workingDir))
			url := fmt.Sprintf("https://dl.google.com/go/go%v.%v-%v.tar.gz", v, runtime.GOOS, runtime.GOARCH)
			downloadArchive(check, workingDir, url, "go")
		}
		registerBinary("go"+v, filepath.Join(workingDir, "bin", "go"))
	}
	registerBinary("go", filepath.Join(testDir, "go"+golangLatest, "bin", "go"))
	registerBinary("gofmt", filepath.Join(testDir, "go"+golangLatest, "bin", "gofmt"))
	workingDir = ""

	// Travis-CI sets GOROOT, which confuses invocations of the Go toolchain.
	// Explicitly clear GOROOT, so each toolchain uses their default GOROOT.
	check(os.Unsetenv("GOROOT"))

	// Set a cache directory within the test directory.
	check(os.Setenv("GOCACHE", filepath.Join(testDir, "gocache")))

	// Setup GOPATH for pre-module support (i.e., go1.10 and earlier).
	goPath = filepath.Join(testDir, "gopath")
	modulePath = strings.TrimSpace(mustRunCommand(t, testDir, "go", "list", "-m", "-f", "{{.Path}}"))
	check(os.RemoveAll(filepath.Join(goPath, "src")))
	check(os.MkdirAll(filepath.Join(goPath, "src", filepath.Dir(modulePath)), 0775))
	check(os.Symlink(repoRoot, filepath.Join(goPath, "src", modulePath)))
	mustRunCommand(t, repoRoot, "go", "mod", "tidy")
	mustRunCommand(t, repoRoot, "go", "mod", "vendor")
	check(os.Setenv("GOPATH", goPath))
}

func downloadArchive(check func(error), dstPath, srcURL, skipPrefix string) {
	check(os.RemoveAll(dstPath))

	resp, err := http.Get(srcURL)
	check(err)
	defer resp.Body.Close()

	zr, err := gzip.NewReader(resp.Body)
	check(err)

	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return
		}
		check(err)

		// Skip directories or files outside the prefix directory.
		if len(skipPrefix) > 0 {
			if !strings.HasPrefix(h.Name, skipPrefix) {
				continue
			}
			if len(h.Name) > len(skipPrefix) && h.Name[len(skipPrefix)] != '/' {
				continue
			}
		}

		path := strings.TrimPrefix(strings.TrimPrefix(h.Name, skipPrefix), "/")
		path = filepath.Join(dstPath, filepath.FromSlash(path))
		mode := os.FileMode(h.Mode & 0777)
		switch h.Typeflag {
		case tar.TypeReg:
			b, err := ioutil.ReadAll(tr)
			check(err)
			check(ioutil.WriteFile(path, b, mode))
		case tar.TypeDir:
			check(os.Mkdir(path, mode))
		}
	}
}

// patchProtos patches proto files with v2 locations of Go packages.
// TODO: Commit these changes upstream.
func patchProtos(check func(error), repoRoot string) {
	javaPackageRx := regexp.MustCompile(`^option\s+java_package\s*=\s*".*"\s*;\s*$`)
	goPackageRx := regexp.MustCompile(`^option\s+go_package\s*=\s*".*"\s*;\s*$`)
	files := map[string]string{
		"src/google/protobuf/any.proto":             "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/api.proto":             "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/duration.proto":        "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/empty.proto":           "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/field_mask.proto":      "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/source_context.proto":  "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/struct.proto":          "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/timestamp.proto":       "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/type.proto":            "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/wrappers.proto":        "github.com/golang/protobuf/v2/types/known;known_proto",
		"src/google/protobuf/descriptor.proto":      "github.com/golang/protobuf/v2/types/descriptor;descriptor_proto",
		"src/google/protobuf/compiler/plugin.proto": "github.com/golang/protobuf/v2/types/plugin;plugin_proto",
		"conformance/conformance.proto":             "github.com/golang/protobuf/v2/internal/testprotos/conformance;conformance_proto",
	}
	for pbpath, gopath := range files {
		b, err := ioutil.ReadFile(filepath.Join(repoRoot, pbpath))
		check(err)
		ss := strings.Split(string(b), "\n")

		// Locate java_package and (possible) go_package options.
		javaPackageIdx, goPackageIdx := -1, -1
		for i, s := range ss {
			if javaPackageIdx < 0 && javaPackageRx.MatchString(s) {
				javaPackageIdx = i
			}
			if goPackageIdx < 0 && goPackageRx.MatchString(s) {
				goPackageIdx = i
			}
		}

		// Ensure the proto file has the correct go_package option.
		opt := `option go_package = "` + gopath + `";`
		if goPackageIdx >= 0 {
			if ss[goPackageIdx] == opt {
				continue // no changes needed
			}
			ss[goPackageIdx] = opt
		} else {
			// Insert go_package option before java_package option.
			ss = append(ss[:javaPackageIdx], append([]string{opt}, ss[javaPackageIdx:]...)...)
		}

		fmt.Println("patch " + pbpath)
		b = []byte(strings.Join(ss, "\n"))
		check(ioutil.WriteFile(filepath.Join(repoRoot, pbpath), b, 0664))
	}
}

func mustRunCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	stdout := new(bytes.Buffer)
	combined := new(bytes.Buffer)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PWD="+dir)
	cmd.Stdout = io.MultiWriter(stdout, combined)
	cmd.Stderr = combined
	if err := cmd.Run(); err != nil {
		t.Fatalf("executing (%v): %v\n%s", strings.Join(args, " "), err, combined.String())
	}
	return stdout.String()
}
