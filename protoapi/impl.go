// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoapi

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math"
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

// CompressGZIP compresses the input as a GZIP-encoded file.
// The current implementation does no compression.
func CompressGZIP(in []byte) (out []byte) {
	// RFC 1952, section 2.3.1.
	var gzipHeader = [10]byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff}

	// RFC 1951, section 3.2.4.
	var blockHeader [5]byte
	const maxBlockSize = math.MaxUint16
	numBlocks := 1 + len(in)/maxBlockSize

	// RFC 1952, section 2.3.1.
	var gzipFooter [8]byte
	binary.LittleEndian.PutUint32(gzipFooter[0:4], crc32.ChecksumIEEE(in))
	binary.LittleEndian.PutUint32(gzipFooter[4:8], uint32(len(in)))

	// Encode the input without compression using raw DEFLATE blocks.
	out = make([]byte, 0, len(gzipHeader)+len(blockHeader)*numBlocks+len(in)+len(gzipFooter))
	out = append(out, gzipHeader[:]...)
	for blockHeader[0] == 0 {
		blockSize := maxBlockSize
		if blockSize > len(in) {
			blockHeader[0] = 0x01 // final bit per RFC 1951, section 3.2.3.
			blockSize = len(in)
		}
		binary.LittleEndian.PutUint16(blockHeader[1:3], uint16(blockSize)^0x0000)
		binary.LittleEndian.PutUint16(blockHeader[3:5], uint16(blockSize)^0xffff)
		out = append(out, blockHeader[:]...)
		out = append(out, in[:blockSize]...)
		in = in[blockSize:]
	}
	out = append(out, gzipFooter[:]...)
	return out
}

// TODO: Remove this when v2 textpb is available.
var CompactTextString func(Message) string
