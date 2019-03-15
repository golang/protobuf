// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build proto_reimpl

package proto

import "github.com/golang/protobuf/internal/proto"

func init() {
	setDefaultsAlt = proto.SetDefaults
	discardUnknownAlt = proto.DiscardUnknown
}
