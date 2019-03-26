// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !use_golang_protobuf_v1

package proto

import (
	"reflect"

	"github.com/golang/protobuf/internal/proto"
)

// Constants that identify the encoding of a value on the wire.
const (
	WireVarint     = 0
	WireFixed64    = 1
	WireBytes      = 2
	WireStartGroup = 3
	WireEndGroup   = 4
	WireFixed32    = 5
)

type (
	Properties       = proto.Properties
	StructProperties = proto.StructProperties
	OneofProperties  = proto.OneofProperties
)

func GetProperties(t reflect.Type) *StructProperties {
	return proto.GetProperties(t)
}
