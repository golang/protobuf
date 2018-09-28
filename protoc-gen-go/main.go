// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// protoc-gen-go is a plugin for the Google protocol buffer compiler to generate
// Go code.  Run it by building this program and putting it in your path with
// the name
// 	protoc-gen-go
// That word 'go' at the end becomes part of the option string set for the
// protocol compiler, so once the protocol compiler (protoc) is installed
// you can run
// 	protoc --go_out=output_directory input_directory/file.proto
// to generate Go bindings for the protocol defined by file.proto.
// With that input, the output will be written to
// 	output_directory/file.pb.go
//
// The generated code is documented in the package comment for
// the library.
//
// See the README and documentation for protocol buffers to learn more:
// 	https://developers.google.com/protocol-buffers/
package main

import (
	"flag"
	"fmt"
	"strings"

	gengogrpc "github.com/golang/protobuf/v2/cmd/protoc-gen-go-grpc/internal_gengogrpc"
	gengo "github.com/golang/protobuf/v2/cmd/protoc-gen-go/internal_gengo"
	"github.com/golang/protobuf/v2/protogen"
)

func main() {
	var (
		flags        flag.FlagSet
		plugins      = flags.String("plugins", "", "list of plugins to enable (supported values: grpc)")
		importPrefix = flags.String("import_prefix", "", "prefix to prepend to import paths")
	)
	importRewriteFunc := func(importPath protogen.GoImportPath) protogen.GoImportPath {
		switch importPath {
		case "context", "fmt", "math":
			return importPath
		}
		if *importPrefix != "" {
			return protogen.GoImportPath(*importPrefix) + importPath
		}
		return importPath
	}
	opts := &protogen.Options{
		ParamFunc:         flags.Set,
		ImportRewriteFunc: importRewriteFunc,
	}
	protogen.Run(opts, func(gen *protogen.Plugin) error {
		grpc := false
		for _, plugin := range strings.Split(*plugins, ",") {
			switch plugin {
			case "grpc":
				grpc = true
			case "":
			default:
				return fmt.Errorf("protoc-gen-go: unknown plugin %q", plugin)
			}
		}
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			filename := f.GeneratedFilenamePrefix + ".pb.go"
			g := gen.NewGeneratedFile(filename, f.GoImportPath)
			gengo.GenerateFile(gen, f, g)
			if grpc {
				gengogrpc.GenerateFileContent(gen, f, g)
			}
		}
		return nil
	})
}
