// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Message is the top-level interface that all messages must implement.
type Message = protoreflect.ProtoMessage

// errInternalNoFast indicates that fast-path operations are not available for a message.
var errInternalNoFast = errors.New("BUG: internal error (errInternalNoFast)")
