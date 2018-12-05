// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoapi

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// TODO: Are these the final signatures that we want?

// EnumName is a helper function to simplify printing protocol buffer enums
// by name.  Given an enum map and a value, it returns a useful string.
func EnumName(m map[int32]string, v int32) string {
	s, ok := m[v]
	if ok {
		return s
	}
	return strconv.Itoa(int(v))
}

// UnmarshalJSONEnum is a helper function to simplify recovering enum int values
// from their JSON-encoded representation. Given a map from the enum's symbolic
// names to its int values, and a byte buffer containing the JSON-encoded
// value, it returns an int32 that can be cast to the enum type by the caller.
//
// The function can deal with both JSON representations, numeric and symbolic.
func UnmarshalJSONEnum(m map[string]int32, data []byte, enumName string) (int32, error) {
	if data[0] == '"' {
		// New style: enums are strings.
		var repr string
		if err := json.Unmarshal(data, &repr); err != nil {
			return -1, err
		}
		val, ok := m[repr]
		if !ok {
			return 0, fmt.Errorf("unrecognized enum %s value %q", enumName, repr)
		}
		return val, nil
	}
	// Old style: enums are ints.
	var val int32
	if err := json.Unmarshal(data, &val); err != nil {
		return 0, fmt.Errorf("cannot unmarshal %#q into enum %s", data, enumName)
	}
	return val, nil
}

// TODO: Remove this when v2 textpb is available.
var CompactTextString func(Message) string
