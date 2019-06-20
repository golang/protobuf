#!/bin/bash
# Copyright 2019 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Download large benchmark datasets.

cd "$(git rev-parse --show-toplevel)"
mkdir -p .cache/benchdata
cd .cache/benchdata
curl -s https://storage.googleapis.com/protobuf_opensource_benchmark_data/datasets.tar.gz | tar zx
