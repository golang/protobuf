#!/bin/bash
# Copyright 2018 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

set -e

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
  cmd/protoc-gen-go-grpc/testdata
)
for dir in ${PROTO_DIRS[@]}; do
  for p in `find $dir -name "*.proto"`; do
    echo "# $p"
    protoc -I$dir \
      --go_out=paths=source_relative:$dir \
      --go-grpc_out=paths=source_relative:$dir \
      $p
  done
done
