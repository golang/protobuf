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

# Public imports require at least Go 1.9.
supportTypeAliases=""
if go list -f '{{context.ReleaseTags}}' runtime | grep -q go1.9; then
  supportTypeAliases=1
fi

# Generate various test protos.
PROTO_DIRS=(
  cmd/protoc-gen-go/testdata
)
for dir in ${PROTO_DIRS[@]}; do
  for p in `find $dir -name "*.proto"`; do
    if [[ $p == */import_public/* && ! $supportTypeAliases ]]; then
      echo "# $p (skipped)"
      continue;
    fi
    echo "# $p"
    protoc -I$dir --go_out=paths=source_relative:$dir $p
  done
done
