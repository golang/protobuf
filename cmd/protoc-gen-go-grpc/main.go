// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The protoc-gen-go-grpc binary is a protoc plugin to generate Go gRPC
// service definitions.
package main

import (
	"github.com/golang/protobuf/v2/cmd/protoc-gen-go-grpc/internal_gengogrpc"
	"github.com/golang/protobuf/v2/protogen"
)

func main() {
	protogen.Run(nil, func(gen *protogen.Plugin) error {
		for _, f := range gen.Files {
			if f.Generate {
				internal_gengogrpc.GenerateFile(gen, f)
			}
		}
		return nil
	})
}
