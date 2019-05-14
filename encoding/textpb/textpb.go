// Package textpb is deprecated.
package textpb

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var Marshal = prototext.Marshal

func Unmarshal(m proto.Message, b []byte) error {
	return prototext.Unmarshal(b, m)
}

type (
	MarshalOptions   = prototext.MarshalOptions
	UnmarshalOptions = prototext.UnmarshalOptions
)
