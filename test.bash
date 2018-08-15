#!/bin/bash
# Copyright 2018 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function print() { echo -e "\x1b[1m> $@\x1b[0m"; }

# The test directory contains the Go and protobuf toolchains used for testing.
# The bin directory contains symlinks to each tool by version.
# It is safe to delete this directory and run the test script from scratch.
TEST_DIR=/tmp/golang-protobuf-test
mkdir -p $TEST_DIR/bin
function register_binary() {
	rm -f $TEST_DIR/bin/$1 # best-effort delete
	ln -s $TEST_DIR/$2 $TEST_DIR/bin/$1
}
export PATH=$TEST_DIR/bin:$PATH
cd $TEST_DIR # install toolchains relative to test directory

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
register_binary conformance-test-runner $PROTOBUF_DIR/conformance/conformance-test-runner
register_binary protoc $PROTOBUF_DIR/src/protoc

# Download each Go toolchain version.
GO_LATEST=go1.11rc1
GO_VERSIONS=(go1.10.3 $GO_LATEST)
for GO_VERSION in ${GO_VERSIONS[@]}; do
	if [ ! -d $GO_VERSION ]; then
		print "download $GO_VERSION"
		GOOS=$(uname | tr '[:upper:]' '[:lower:]')
		(mkdir $GO_VERSION && curl -s -L https://dl.google.com/go/$GO_VERSION.$GOOS-amd64.tar.gz | tar -zxf - -C $GO_VERSION --strip-components 1) || exit 1
	fi
	register_binary $GO_VERSION $GO_VERSION/bin/go
done
register_binary go $GO_LATEST/bin/go
register_binary gofmt $GO_LATEST/bin/gofmt

# Travis-CI sets GOROOT, which confuses later invocations of the Go toolchains.
# Explicitly clear GOROOT, so each toolchain uses their default GOROOT.
unset GOROOT

# Setup GOPATH for pre-module support.
MODULE_PATH=$(cd $REPO_ROOT && go list -m -f "{{.Path}}")
rm -f gopath/src/$MODULE_PATH # best-effort delete
mkdir -p gopath/src/$(dirname $MODULE_PATH)
(cd gopath/src/$(dirname $MODULE_PATH) && ln -s $REPO_ROOT $(basename $MODULE_PATH))
export GOPATH=$TEST_DIR/gopath

# Download dependencies using modules.
# For pre-module support, dump the dependencies in a vendor directory.
(cd $REPO_ROOT && go mod tidy && go mod vendor) || exit 1

# Run tests across every supported version of Go.
LABELS=()
PIDS=()
OUTS=()
function cleanup() { for OUT in ${OUTS[@]}; do rm $OUT; done; }
trap cleanup EXIT
for GO_VERSION in ${GO_VERSIONS[@]}; do
	# Run the go command in a background process.
	function go() {
		# Use a per-version Go cache to work around bugs in Go build caching.
		# See https://golang.org/issue/26883
		GO_CACHE="$TEST_DIR/cache.$GO_VERSION"
		LABELS+=("$(echo "$GO_VERSION $@")")
		OUT=$(mktemp)
		(cd $GOPATH/src/$MODULE_PATH && GOCACHE=$GO_CACHE $GO_VERSION "$@" &> $OUT) &
		PIDS+=($!)
		OUTS+=($OUT)
	}

	go build ./...
	go test -race ./...
	go test -race -tags purego ./...
	go test -race -tags proto1_legacy ./...

	unset go # to avoid confusing later invocations of "go"
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

# Run commands that produce output when there is a failure.
function check() {
	OUT=$(cd $REPO_ROOT && "$@" 2>&1)
	if [ ! -z "$OUT" ]; then
		print "$@"
		echo "$OUT"
		RET=1
	fi
}

# Check for stale or unformatted source files.
check go run ./internal/cmd/generate-types
check gofmt -d .

# Check for changed or untracked files.
check git diff --no-prefix HEAD
check git ls-files --others --exclude-standard

# Print termination status.
if [ $RET -eq 0 ]; then
	echo -e "\x1b[32;1mPASS\x1b[0m"
else
	echo -e "\x1b[31;1mFAIL\x1b[0m"
fi
exit $RET
