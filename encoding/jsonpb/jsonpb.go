// Package jsonpb is deprecated.
package jsonpb

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var Marshal = protojson.Marshal

func Unmarshal(m proto.Message, b []byte) error {
	return protojson.Unmarshal(b, m)
}

type (
	MarshalOptions   = protojson.MarshalOptions
	UnmarshalOptions = protojson.UnmarshalOptions
)
