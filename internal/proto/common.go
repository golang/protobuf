// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

// TODO: This file exists to provide the illusion to other source files that
// they live within the real proto package by providing functions and types
// that they would otherwise be able to call directly.

import (
	"github.com/golang/protobuf/v2/runtime/protoiface"
	_ "github.com/golang/protobuf/v2/runtime/protolegacy"
)

type (
	Message       = protoiface.MessageV1
	ExtensionDesc = protoiface.ExtensionDescV1
)
