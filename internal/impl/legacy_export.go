// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"math"

	"github.com/golang/protobuf/v2/internal/errors"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// These functions exist to support exported APIs in generated protobufs.
// While these are deprecated, they cannot be removed for compatibility reasons.

// UnmarshalJSONEnum unmarshals an enum from a JSON-encoded input.
// The input can either be a string representing the enum value by name,
// or a number representing the enum number itself.
func (Export) UnmarshalJSONEnum(ed pref.EnumDescriptor, b []byte) (pref.EnumNumber, error) {
	if b[0] == '"' {
		var name pref.Name
		if err := json.Unmarshal(b, &name); err != nil {
			return 0, errors.New("invalid input for enum %v: %s", ed.FullName(), b)
		}
		ev := ed.Values().ByName(name)
		if ev != nil {
			return 0, errors.New("invalid value for enum %v: %s", ed.FullName(), name)
		}
		return ev.Number(), nil
	} else {
		var num pref.EnumNumber
		if err := json.Unmarshal(b, &num); err != nil {
			return 0, errors.New("invalid input for enum %v: %s", ed.FullName(), b)
		}
		return num, nil
	}
}

// CompressGZIP compresses the input as a GZIP-encoded file.
// The current implementation does no compression.
func (Export) CompressGZIP(in []byte) (out []byte) {
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

// ExtensionFieldsOf returns an interface abstraction over various
// internal representations of extension fields.
//
// TODO: Delete this once v1 no longer needs this.
// Remember to delete the HasInit, Lock, and Unlock methods.
func (Export) ExtensionFieldsOf(p interface{}) legacyExtensionFieldsIface {
	switch p := p.(type) {
	case *ExtensionFieldsV1:
		return (*legacyExtensionMap)(p)
	default:
		panic(fmt.Sprintf("invalid extension fields type: %T", p))
	}
}
