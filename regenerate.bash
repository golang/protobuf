#!/bin/bash
# Copyright 2018 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -e

go generate internal/cmd/generate-types/main.go

# Install the working tree's protoc-gen-gen in a tempdir.
tmpdir=$(mktemp -d -t protobuf-regen.XXXXXX)
trap 'rm -rf $tmpdir' EXIT
mkdir -p $tmpdir/bin
PATH=$tmpdir/bin:$PATH
GOBIN=$tmpdir/bin go install ./cmd/protoc-gen-go
GOBIN=$tmpdir/bin go install ./cmd/protoc-gen-go-grpc

# Generate various test protos.
PROTO_DIRS=(
  cmd/protoc-gen-go/testdata
  internal/testprotos/test
)
GRPC_PROTO_DIRS=(
  cmd/protoc-gen-go-grpc/testdata
)
for dir in ${PROTO_DIRS[@]}; do
  for p in `find $dir -name "*.proto"`; do
    echo "# $p"
    PROTOC_GEN_GO_ENABLE_REFLECT=1 protoc -I$dir \
      --go_out=paths=source_relative:$dir \
      $p
  done
done
for dir in ${GRPC_PROTO_DIRS[@]}; do
  for p in `find $dir -name "*.proto"`; do
    echo "# $p"
    PROTOC_GEN_GO_ENABLE_REFLECT=1 protoc -I$dir \
      --go-grpc_out=paths=source_relative:$dir \
      $p
  done
done

# Generate descriptor and plugin.
# TODO: Make this more automated.

echo "# types/descriptor/descriptor.proto"
mkdir -p $tmpdir/src/google/protobuf
cp ./types/descriptor/descriptor.proto $tmpdir/src/google/protobuf/descriptor.proto
PROTOC_GEN_GO_ENABLE_REFLECT=1 protoc -I$tmpdir/src \
  --go_out=paths=source_relative:$tmpdir/src \
  $tmpdir/src/google/protobuf/descriptor.proto
cp $tmpdir/src/google/protobuf/descriptor.pb.go ./types/descriptor/descriptor.pb.go

echo "# types/plugin/plugin.proto"
mkdir -p $tmpdir/src/google/protobuf/compiler
cp ./types/plugin/plugin.proto $tmpdir/src/google/protobuf/compiler/plugin.proto
PROTOC_GEN_GO_ENABLE_REFLECT=1 protoc -I$tmpdir/src \
  --go_out=paths=source_relative:$tmpdir/src \
  $tmpdir/src/google/protobuf/compiler/plugin.proto
cp $tmpdir/src/google/protobuf/compiler/plugin.pb.go ./types/plugin/plugin.pb.go

echo "# encoding/textpb/testprotos/pb?/test.proto"
PROTOC_GEN_GO_ENABLE_REFLECT=1 protoc --go_out=paths=source_relative:. \
  encoding/textpb/testprotos/pb?/test.proto

echo "# reflect/protoregistry/testprotos/test.proto"
PROTOC_GEN_GO_ENABLE_REFLECT=1 protoc --go_out=paths=source_relative:. \
  reflect/protoregistry/testprotos/test.proto
