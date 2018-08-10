#!/bin/bash
# Copyright 2018 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function print() { echo -e "\x1b[1m> $@\x1b[0m"; }

# Create a test directory.
# The Go and protobuf toolchains used for testing will be cached here.
TEST_DIR=/tmp/golang-protobuf-test
if [ ! -d $TEST_DIR ]; then
	print "mkdir $TEST_DIR"
	mkdir -p $TEST_DIR
fi
cd $TEST_DIR

# Download and build the protobuf toolchain.
# We avoid downloading the pre-compiled binaries since they do not contain
# the conformance test runner.
PROTOBUF_VERSION=3.6.1
PROTOBUF_DIR="protobuf-$PROTOBUF_VERSION"
if [ ! -d $PROTOBUF_DIR ]; then
	print "download and build $PROTOBUF_DIR"
	(curl -s -L https://github.com/google/protobuf/releases/download/v$PROTOBUF_VERSION/protobuf-all-$PROTOBUF_VERSION.tar.gz | tar -zxf -) || exit 1
	(cd $PROTOBUF_DIR && ./configure && make && cd conformance && make) || exit 1
fi
export PATH=$TEST_DIR/$PROTOBUF_DIR/src:$TEST_DIR/$PROTOBUF_DIR/conformance:$PATH

# Download each Go toolchain version.
GO_LATEST=1.11rc1
GO_VERSIONS=(1.10.3 $GO_LATEST)
for GO_VERSION in ${GO_VERSIONS[@]}; do
	GO_DIR=go$GO_VERSION
	if [ ! -d $GO_DIR ]; then
		print "download $GO_DIR"
		GOOS=$(uname | tr '[:upper:]' '[:lower:]')
		(mkdir $GO_DIR && curl -s -L https://dl.google.com/go/$GO_DIR.$GOOS-amd64.tar.gz | tar -zxf - -C $GO_DIR --strip-components 1) || exit 1
	fi
done

# Travis-CI sets GOROOT, which confuses later invocations of the Go toolchains.
# Explicitly clear GOROOT, so each toolchain uses their default GOROOT.
unset GOROOT

# Download dependencies using modules.
# For pre-module support, dump the dependencies in a vendor directory.
# TODO: use GOFLAGS="-mod=readonly" when https://golang.org/issue/26850 is fixed.
GO_LATEST_BIN=$TEST_DIR/go$GO_LATEST/bin/go
(cd $REPO_ROOT && $GO_LATEST_BIN mod tidy && $GO_LATEST_BIN mod vendor) || exit 1

# Setup GOPATH for pre-module support.
MODULE_PATH=$(cd $REPO_ROOT && $GO_LATEST_BIN list -m -f "{{.Path}}")
if [ ! -d gopath/src/$MODULE_PATH ]; then
	mkdir -p gopath/src/$(dirname $MODULE_PATH)
	(cd gopath/src/$(dirname $MODULE_PATH) && ln -s $REPO_ROOT $(basename $MODULE_PATH))
fi
export GOPATH=$TEST_DIR/gopath

# Run tests across every supported version of Go.
LABELS=()
PIDS=()
OUTS=()
function cleanup() { for OUT in ${OUTS[@]}; do rm $OUT; done; }
trap cleanup EXIT
for GO_VERSION in ${GO_VERSIONS[@]}; do
	# Run the go command in a background process.
	function go() {
		GO_BIN=go$GO_VERSION/bin/go
		LABELS+=("$(echo "go$GO_VERSION $@")")
		OUT=$(mktemp)
		(cd $GOPATH/src/$MODULE_PATH && $TEST_DIR/$GO_BIN $@ &> $OUT) &
		PIDS+=($!)
		OUTS+=($OUT)
	}

	go build ./...
	go test -race ./...
	go test -race -tags purego ./...
	go test -race -tags proto1_legacy ./...
done

# Wait for all processes to finish.
RET=0
for I in ${!PIDS[@]}; do
	print "${LABELS[$I]}"
	if ! wait ${PIDS[$I]}; then
		cat ${OUTS[$I]} # only output upon error
		RET=1
	fi
done

# Check for stale generated source files.
GEN_DIFF=$(cd $REPO_ROOT && ${GO_LATEST_BIN} run ./internal/cmd/generate-types 2>&1)
if [ ! -z "$GEN_DIFF" ]; then
	print "go run ./internal/cmd/generate-types"
	echo "$GEN_DIFF"
	RET=1
fi

# Check for unformatted Go source files.
FMT_DIFF=$(cd $REPO_ROOT && ${GO_LATEST_BIN}fmt -d . 2>&1)
if [ ! -z "$FMT_DIFF" ]; then
	print "gofmt -d ."
	echo "$FMT_DIFF"
	RET=1
fi

# Check for changed files.
GIT_DIFF=$(cd $REPO_ROOT && git diff --no-prefix HEAD 2>&1)
if [ ! -z "$GIT_DIFF" ]; then
	print "git diff HEAD"
	echo "$GIT_DIFF"
	RET=1
fi

# Check for untracked files.
GIT_UNTRACKED=$(cd $REPO_ROOT && git ls-files --others --exclude-standard 2>&1)
if [ ! -z "$GIT_UNTRACKED" ]; then
	print "git ls-files --others --exclude-standard"
	echo "$GIT_UNTRACKED"
	RET=1
fi

# Print termination status.
if [ $RET -eq 0 ]; then
	echo -e "\x1b[32;1mPASS\x1b[0m"
else
	echo -e "\x1b[31;1mFAIL\x1b[0m"
fi
exit $RET
