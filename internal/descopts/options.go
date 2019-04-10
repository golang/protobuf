// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package descopts contains the nil pointers to concrete descriptor options.
//
// This package exists as a form of reverse dependency injection so that certain
// packages (e.g., internal/fileinit can avoid a direct dependency on the
// descriptor proto package).
package descopts

import pref "github.com/golang/protobuf/v2/reflect/protoreflect"

// These variables are set by the init function in descriptor.pb.go via logic
// in internal/fileinit. In other words, so long as the descriptor proto package
// is linked in, these variables will be populated.
//
// Each variable is populated with a nil pointer to the options struct.
var (
	File           pref.ProtoMessage
	Enum           pref.ProtoMessage
	EnumValue      pref.ProtoMessage
	Message        pref.ProtoMessage
	Field          pref.ProtoMessage
	Oneof          pref.ProtoMessage
	ExtensionRange pref.ProtoMessage
	Service        pref.ProtoMessage
	Method         pref.ProtoMessage
)
