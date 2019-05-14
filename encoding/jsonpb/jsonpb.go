// Package jsonpb is deprecated.
package jsonpb

import "google.golang.org/protobuf/encoding/protojson"

var (
	Marshal   = protojson.Marshal
	Unmarshal = protojson.Unmarshal
)

type (
	MarshalOptions   = protojson.MarshalOptions
	UnmarshalOptions = protojson.UnmarshalOptions
)
