# Copyright 2010 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

all:	install

install:
	go install ./proto ./jsonpb ./ptypes ./protoc-gen-go

test:
	go test ./... ./protoc-gen-go/testdata
	go test -tags purego ./... ./protoc-gen-go/testdata
	go build ./protoc-gen-go/testdata/grpc/grpc.pb.go
	make -C conformance test

clean:
	go clean ./...

nuke:
	go clean -i ./...

regenerate:
	./regenerate.sh
