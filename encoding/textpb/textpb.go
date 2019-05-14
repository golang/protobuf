// Package textpb is deprecated.
package textpb

import "google.golang.org/protobuf/encoding/prototext"

var (
	Marshal   = prototext.Marshal
	Unmarshal = prototext.Unmarshal
)

type (
	MarshalOptions   = prototext.MarshalOptions
	UnmarshalOptions = prototext.UnmarshalOptions
)
