// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This binary implements the conformance test subprocess protocol as documented
// in conformance.proto.
package main

import (
	"encoding/binary"
	"io"
	"log"
	"os"

	"github.com/golang/protobuf/v2/encoding/jsonpb"
	"github.com/golang/protobuf/v2/proto"

	pb "github.com/golang/protobuf/v2/internal/testprotos/conformance"
)

func main() {
	var sizeBuf [4]byte
	inbuf := make([]byte, 0, 4096)
	for {
		_, err := io.ReadFull(os.Stdin, sizeBuf[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("conformance: read request: %v", err)
		}
		size := binary.LittleEndian.Uint32(sizeBuf[:])
		if int(size) > cap(inbuf) {
			inbuf = make([]byte, size)
		}
		inbuf = inbuf[:size]
		if _, err := io.ReadFull(os.Stdin, inbuf); err != nil {
			log.Fatalf("conformance: read request: %v", err)
		}

		req := &pb.ConformanceRequest{}
		if err := proto.Unmarshal(inbuf, req); err != nil {
			log.Fatalf("conformance: parse request: %v", err)
		}
		res := handle(req)

		out, err := proto.Marshal(res)
		if err != nil {
			log.Fatalf("conformance: marshal response: %v", err)
		}
		binary.LittleEndian.PutUint32(sizeBuf[:], uint32(len(out)))
		if _, err := os.Stdout.Write(sizeBuf[:]); err != nil {
			log.Fatalf("conformance: write response: %v", err)
		}
		if _, err := os.Stdout.Write(out); err != nil {
			log.Fatalf("conformance: write response: %v", err)
		}
	}
}

func handle(req *pb.ConformanceRequest) *pb.ConformanceResponse {
	var err error
	var msg proto.Message = &pb.TestAllTypesProto2{}
	if req.GetMessageType() == "protobuf_test_messages.proto3.TestAllTypesProto3" {
		msg = &pb.TestAllTypesProto3{}
	}

	switch p := req.Payload.(type) {
	case *pb.ConformanceRequest_ProtobufPayload:
		err = proto.Unmarshal(p.ProtobufPayload, msg)
	case *pb.ConformanceRequest_JsonPayload:
		err = jsonpb.Unmarshal(msg, []byte(p.JsonPayload))
	default:
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_RuntimeError{
				RuntimeError: "unknown request payload type",
			},
		}
	}
	if err != nil {
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_ParseError{
				ParseError: err.Error(),
			},
		}
	}

	switch req.RequestedOutputFormat {
	case pb.WireFormat_PROTOBUF:
		p, err := proto.Marshal(msg)
		if err != nil {
			return &pb.ConformanceResponse{
				Result: &pb.ConformanceResponse_SerializeError{
					SerializeError: err.Error(),
				},
			}
		}
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_ProtobufPayload{
				ProtobufPayload: p,
			},
		}
	case pb.WireFormat_JSON:
		p, err := jsonpb.Marshal(msg)
		if err != nil {
			return &pb.ConformanceResponse{
				Result: &pb.ConformanceResponse_SerializeError{
					SerializeError: err.Error(),
				},
			}
		}
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_JsonPayload{
				JsonPayload: string(p),
			},
		}
	default:
		return &pb.ConformanceResponse{
			Result: &pb.ConformanceResponse_RuntimeError{
				RuntimeError: "unknown output format",
			},
		}
	}
}
