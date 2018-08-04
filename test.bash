#!/bin/bash
# Copyright 2018 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create a test directory.
# The Go and protobuf toolchains used for testing will be cached here.
TEST_DIR=/tmp/golang-protobuf-test
if [ ! -d $TEST_DIR ]; then
	echo "mkdir $TEST_DIR"
	mkdir -p $TEST_DIR
fi
cd $TEST_DIR

# Download and build the protobuf toolchain.
# We avoid downloading the pre-compiled binaries since they do not contain
# the conformance test runner.
PROTOBUF_VERSION=3.6.1
PROTOBUF_DIR="protobuf-$PROTOBUF_VERSION"
if [ ! -d $PROTOBUF_DIR ]; then
	echo "download and build $PROTOBUF_DIR"
	(curl -s -L https://github.com/google/protobuf/releases/download/v$PROTOBUF_VERSION/protobuf-all-$PROTOBUF_VERSION.tar.gz | tar -zxf -) || exit 1
	(cd $PROTOBUF_DIR && ./configure && make && cd conformance && make) || exit 1
fi

# Download each Go toolchain version.
GO_VERSIONS=(1.9.7 1.10.3 1.11beta3)
for GO_VERSION in ${GO_VERSIONS[@]}; do
	GO_DIR=go$GO_VERSION
	if [ ! -d $GO_DIR ]; then
		echo "download $GO_DIR"
		GOOS=$(uname | tr '[:upper:]' '[:lower:]')
		(mkdir $GO_DIR && curl -s -L https://dl.google.com/go/$GO_DIR.$GOOS-amd64.tar.gz | tar -zxf - -C $GO_DIR --strip-components 1) || exit 1
	fi
done

# Setup GOPATH for pre-module support.
MODULE_PATH=$(grep '^module ' $REPO_ROOT/go.mod | cut -d " " -f 2)
if [ ! -d gopath/src/$MODULE_PATH ]; then
	mkdir -p gopath/src/$(dirname $MODULE_PATH)
	(cd gopath/src/$(dirname $MODULE_PATH) && ln -s $REPO_ROOT $(basename $MODULE_PATH))
fi
export GOPATH=$TEST_DIR/gopath

# Run tests across every supported version of Go.
FAIL=0
for GO_VERSION in ${GO_VERSIONS[@]}; do
	export GOROOT=$TEST_DIR/go$GO_VERSION
	GO_BIN=go$GO_VERSION/bin/go
	function go_build() {
		echo "$GO_BIN build $@"
		(cd $GOPATH/src/$MODULE_PATH && $TEST_DIR/$GO_BIN build $@) || FAIL=1
	}
	function go_test() {
		echo "$GO_BIN test $@"
		(cd $GOPATH/src/$MODULE_PATH && $TEST_DIR/$GO_BIN test $@) || FAIL=1
	}

	go_build ./...
	go_test -race ./...
	go_test -race -tags proto1_legacy ./...
done
exit $FAIL
