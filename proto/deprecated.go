// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Deprecated: Do not use.
var ErrInternalBadWireType = errors.New("proto: internal error: bad wiretype for oneof")

// Deprecated: Do not use.
type Stats struct{ Emalloc, Dmalloc, Encode, Decode, Chit, Cmiss, Size uint64 }

// Deprecated: Do not use.
func GetStats() Stats { return Stats{} }

// Deprecated: Do not use.
func MarshalMessageSet(interface{}) ([]byte, error) {
	return nil, errors.New("proto: not implemented")
}

// Deprecated: Do not use.
func UnmarshalMessageSet([]byte, interface{}) error {
	return errors.New("proto: not implemented")
}

// Deprecated: Do not use.
func MarshalMessageSetJSON(interface{}) ([]byte, error) {
	return nil, errors.New("proto: not implemented")
}

// Deprecated: Do not use.
func UnmarshalMessageSetJSON([]byte, interface{}) error {
	return errors.New("proto: not implemented")
}

// Deprecated: Do not use.
func RegisterMessageSetType(Message, int32, string) {}

// Deprecated: Do not use.
func EnumName(m map[int32]string, v int32) string {
	if s, ok := m[v]; ok {
		return s
	}
	return strconv.Itoa(int(v))
}

// Deprecated: Do not use.
func UnmarshalJSONEnum(m map[string]int32, b []byte, enumName string) (int32, error) {
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return 0, fmt.Errorf("proto: invalid input for enum %v: %s", enumName, b)
		}
		v, ok := m[s]
		if !ok {
			return 0, fmt.Errorf("proto: invalid value for enum %v: %s", enumName, b)
		}
		return v, nil
	} else {
		var v int32
		if err := json.Unmarshal(b, &v); err != nil {
			return 0, fmt.Errorf("proto: invalid input for enum %v: %s", enumName, b)
		}
		return v, nil
	}
}
