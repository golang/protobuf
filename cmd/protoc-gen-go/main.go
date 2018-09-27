// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The protoc-gen-go binary is a protoc plugin to generate a Go protocol
// buffer package.
package main

import (
	"errors"
	"flag"

	gengo "github.com/golang/protobuf/v2/cmd/protoc-gen-go/internal_gengo"
	"github.com/golang/protobuf/v2/protogen"
)

func main() {
	var flags flag.FlagSet
	plugins := flags.String("plugins", "", "deprecated option")
	opts := &protogen.Options{
		ParamFunc: flags.Set,
	}
	protogen.Run(opts, func(gen *protogen.Plugin) error {
		if *plugins != "" {
			return errors.New("protoc-gen-go: plugins are not supported; use 'protoc --go-grpc_out=...' to generate gRPC")
		}
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			filename := f.GeneratedFilenamePrefix + ".pb.go"
			g := gen.NewGeneratedFile(filename, f.GoImportPath)
			gengo.GenerateFile(gen, f, g)
		}
		return nil
	})
}
