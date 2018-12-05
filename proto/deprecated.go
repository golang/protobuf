// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"errors"

	"github.com/golang/protobuf/protoapi"
)

// Deprecated: do not use.
type Stats struct{ Emalloc, Dmalloc, Encode, Decode, Chit, Cmiss, Size uint64 }

// Deprecated: do not use.
func GetStats() Stats { return Stats{} }

// Deprecated: do not use.
func MarshalMessageSet(interface{}) ([]byte, error) {
	return nil, errors.New("proto: not implemented")
}

// Deprecated: do not use.
func UnmarshalMessageSet([]byte, interface{}) error {
	return errors.New("proto: not implemented")
}

// Deprecated: do not use.
func MarshalMessageSetJSON(interface{}) ([]byte, error) {
	return nil, errors.New("proto: not implemented")
}

// Deprecated: do not use.
func UnmarshalMessageSetJSON([]byte, interface{}) error {
	return errors.New("proto: not implemented")
}

// Deprecated: do not use.
func RegisterMessageSetType(Message, int32, string) {}

// Deprecated: do not use.
func EnumName(m map[int32]string, v int32) string {
	return protoapi.EnumName(m, v)
}

// Deprecated: do not use.
func UnmarshalJSONEnum(m map[string]int32, data []byte, enumName string) (int32, error) {
	return protoapi.UnmarshalJSONEnum(m, data, enumName)
}
