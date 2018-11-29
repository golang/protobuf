// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import "github.com/golang/protobuf/v2/reflect/protoreflect"

// Message is the top-level interface that all messages must implement.
type Message = protoreflect.ProtoMessage
