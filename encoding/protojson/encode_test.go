// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protojson_test

import (
	"bytes"
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/internal/encoding/pack"
	"google.golang.org/protobuf/internal/encoding/wire"
	pimpl "google.golang.org/protobuf/internal/impl"
	"google.golang.org/protobuf/internal/scalar"
	"google.golang.org/protobuf/proto"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoiface"

	"google.golang.org/protobuf/encoding/testprotos/pb2"
	"google.golang.org/protobuf/encoding/testprotos/pb3"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// splitLines is a cmpopts.Option for comparing strings with line breaks.
var splitLines = cmpopts.AcyclicTransformer("SplitLines", func(s string) []string {
	return strings.Split(s, "\n")
})

func pb2Enum(i int32) *pb2.Enum {
	p := new(pb2.Enum)
	*p = pb2.Enum(i)
	return p
}

func pb2Enums_NestedEnum(i int32) *pb2.Enums_NestedEnum {
	p := new(pb2.Enums_NestedEnum)
	*p = pb2.Enums_NestedEnum(i)
	return p
}

func setExtension(m proto.Message, xd *protoiface.ExtensionDescV1, val interface{}) {
	knownFields := m.ProtoReflect().KnownFields()
	extTypes := knownFields.ExtensionTypes()
	extTypes.Register(xd.Type)
	if val == nil {
		return
	}
	pval := xd.Type.ValueOf(val)
	knownFields.Set(wire.Number(xd.Field), pval)
}

// dhex decodes a hex-string and returns the bytes and panics if s is invalid.
func dhex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		desc    string
		mo      protojson.MarshalOptions
		input   proto.Message
		want    string
		wantErr bool // TODO: Verify error message substring.
	}{{
		desc:  "proto2 optional scalars not set",
		input: &pb2.Scalars{},
		want:  "{}",
	}, {
		desc:  "proto3 scalars not set",
		input: &pb3.Scalars{},
		want:  "{}",
	}, {
		desc: "proto2 optional scalars set to zero values",
		input: &pb2.Scalars{
			OptBool:     scalar.Bool(false),
			OptInt32:    scalar.Int32(0),
			OptInt64:    scalar.Int64(0),
			OptUint32:   scalar.Uint32(0),
			OptUint64:   scalar.Uint64(0),
			OptSint32:   scalar.Int32(0),
			OptSint64:   scalar.Int64(0),
			OptFixed32:  scalar.Uint32(0),
			OptFixed64:  scalar.Uint64(0),
			OptSfixed32: scalar.Int32(0),
			OptSfixed64: scalar.Int64(0),
			OptFloat:    scalar.Float32(0),
			OptDouble:   scalar.Float64(0),
			OptBytes:    []byte{},
			OptString:   scalar.String(""),
		},
		want: `{
  "optBool": false,
  "optInt32": 0,
  "optInt64": "0",
  "optUint32": 0,
  "optUint64": "0",
  "optSint32": 0,
  "optSint64": "0",
  "optFixed32": 0,
  "optFixed64": "0",
  "optSfixed32": 0,
  "optSfixed64": "0",
  "optFloat": 0,
  "optDouble": 0,
  "optBytes": "",
  "optString": ""
}`,
	}, {
		desc: "proto2 optional scalars set to some values",
		input: &pb2.Scalars{
			OptBool:     scalar.Bool(true),
			OptInt32:    scalar.Int32(0xff),
			OptInt64:    scalar.Int64(0xdeadbeef),
			OptUint32:   scalar.Uint32(47),
			OptUint64:   scalar.Uint64(0xdeadbeef),
			OptSint32:   scalar.Int32(-1001),
			OptSint64:   scalar.Int64(-0xffff),
			OptFixed64:  scalar.Uint64(64),
			OptSfixed32: scalar.Int32(-32),
			OptFloat:    scalar.Float32(1.02),
			OptDouble:   scalar.Float64(1.234),
			OptBytes:    []byte("谷歌"),
			OptString:   scalar.String("谷歌"),
		},
		want: `{
  "optBool": true,
  "optInt32": 255,
  "optInt64": "3735928559",
  "optUint32": 47,
  "optUint64": "3735928559",
  "optSint32": -1001,
  "optSint64": "-65535",
  "optFixed64": "64",
  "optSfixed32": -32,
  "optFloat": 1.02,
  "optDouble": 1.234,
  "optBytes": "6LC35q2M",
  "optString": "谷歌"
}`,
	}, {
		desc: "string",
		input: &pb3.Scalars{
			SString: "谷歌",
		},
		want: `{
  "sString": "谷歌"
}`,
	}, {
		desc: "string with invalid UTF8",
		input: &pb3.Scalars{
			SString: "abc\xff",
		},
		want:    "{\n  \"sString\": \"abc\xff\"\n}",
		wantErr: true,
	}, {
		desc: "float nan",
		input: &pb3.Scalars{
			SFloat: float32(math.NaN()),
		},
		want: `{
  "sFloat": "NaN"
}`,
	}, {
		desc: "float positive infinity",
		input: &pb3.Scalars{
			SFloat: float32(math.Inf(1)),
		},
		want: `{
  "sFloat": "Infinity"
}`,
	}, {
		desc: "float negative infinity",
		input: &pb3.Scalars{
			SFloat: float32(math.Inf(-1)),
		},
		want: `{
  "sFloat": "-Infinity"
}`,
	}, {
		desc: "double nan",
		input: &pb3.Scalars{
			SDouble: math.NaN(),
		},
		want: `{
  "sDouble": "NaN"
}`,
	}, {
		desc: "double positive infinity",
		input: &pb3.Scalars{
			SDouble: math.Inf(1),
		},
		want: `{
  "sDouble": "Infinity"
}`,
	}, {
		desc: "double negative infinity",
		input: &pb3.Scalars{
			SDouble: math.Inf(-1),
		},
		want: `{
  "sDouble": "-Infinity"
}`,
	}, {
		desc:  "proto2 enum not set",
		input: &pb2.Enums{},
		want:  "{}",
	}, {
		desc: "proto2 enum set to zero value",
		input: &pb2.Enums{
			OptEnum:       pb2Enum(0),
			OptNestedEnum: pb2Enums_NestedEnum(0),
		},
		want: `{
  "optEnum": 0,
  "optNestedEnum": 0
}`,
	}, {
		desc: "proto2 enum",
		input: &pb2.Enums{
			OptEnum:       pb2.Enum_ONE.Enum(),
			OptNestedEnum: pb2.Enums_UNO.Enum(),
		},
		want: `{
  "optEnum": "ONE",
  "optNestedEnum": "UNO"
}`,
	}, {
		desc: "proto2 enum set to numeric values",
		input: &pb2.Enums{
			OptEnum:       pb2Enum(2),
			OptNestedEnum: pb2Enums_NestedEnum(2),
		},
		want: `{
  "optEnum": "TWO",
  "optNestedEnum": "DOS"
}`,
	}, {
		desc: "proto2 enum set to unnamed numeric values",
		input: &pb2.Enums{
			OptEnum:       pb2Enum(101),
			OptNestedEnum: pb2Enums_NestedEnum(-101),
		},
		want: `{
  "optEnum": 101,
  "optNestedEnum": -101
}`,
	}, {
		desc:  "proto3 enum not set",
		input: &pb3.Enums{},
		want:  "{}",
	}, {
		desc: "proto3 enum set to zero value",
		input: &pb3.Enums{
			SEnum:       pb3.Enum_ZERO,
			SNestedEnum: pb3.Enums_CERO,
		},
		want: "{}",
	}, {
		desc: "proto3 enum",
		input: &pb3.Enums{
			SEnum:       pb3.Enum_ONE,
			SNestedEnum: pb3.Enums_UNO,
		},
		want: `{
  "sEnum": "ONE",
  "sNestedEnum": "UNO"
}`,
	}, {
		desc: "proto3 enum set to numeric values",
		input: &pb3.Enums{
			SEnum:       2,
			SNestedEnum: 2,
		},
		want: `{
  "sEnum": "TWO",
  "sNestedEnum": "DOS"
}`,
	}, {
		desc: "proto3 enum set to unnamed numeric values",
		input: &pb3.Enums{
			SEnum:       -47,
			SNestedEnum: 47,
		},
		want: `{
  "sEnum": -47,
  "sNestedEnum": 47
}`,
	}, {
		desc:  "proto2 nested message not set",
		input: &pb2.Nests{},
		want:  "{}",
	}, {
		desc: "proto2 nested message set to empty",
		input: &pb2.Nests{
			OptNested: &pb2.Nested{},
			Optgroup:  &pb2.Nests_OptGroup{},
		},
		want: `{
  "optNested": {},
  "optgroup": {}
}`,
	}, {
		desc: "proto2 nested messages",
		input: &pb2.Nests{
			OptNested: &pb2.Nested{
				OptString: scalar.String("nested message"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("another nested message"),
				},
			},
		},
		want: `{
  "optNested": {
    "optString": "nested message",
    "optNested": {
      "optString": "another nested message"
    }
  }
}`,
	}, {
		desc: "proto2 groups",
		input: &pb2.Nests{
			Optgroup: &pb2.Nests_OptGroup{
				OptString: scalar.String("inside a group"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("nested message inside a group"),
				},
				Optnestedgroup: &pb2.Nests_OptGroup_OptNestedGroup{
					OptFixed32: scalar.Uint32(47),
				},
			},
		},
		want: `{
  "optgroup": {
    "optString": "inside a group",
    "optNested": {
      "optString": "nested message inside a group"
    },
    "optnestedgroup": {
      "optFixed32": 47
    }
  }
}`,
	}, {
		desc:  "proto3 nested message not set",
		input: &pb3.Nests{},
		want:  "{}",
	}, {
		desc: "proto3 nested message set to empty",
		input: &pb3.Nests{
			SNested: &pb3.Nested{},
		},
		want: `{
  "sNested": {}
}`,
	}, {
		desc: "proto3 nested message",
		input: &pb3.Nests{
			SNested: &pb3.Nested{
				SString: "nested message",
				SNested: &pb3.Nested{
					SString: "another nested message",
				},
			},
		},
		want: `{
  "sNested": {
    "sString": "nested message",
    "sNested": {
      "sString": "another nested message"
    }
  }
}`,
	}, {
		desc:  "oneof not set",
		input: &pb3.Oneofs{},
		want:  "{}",
	}, {
		desc: "oneof set to empty string",
		input: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofString{},
		},
		want: `{
  "oneofString": ""
}`,
	}, {
		desc: "oneof set to string",
		input: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofString{
				OneofString: "hello",
			},
		},
		want: `{
  "oneofString": "hello"
}`,
	}, {
		desc: "oneof set to enum",
		input: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofEnum{
				OneofEnum: pb3.Enum_ZERO,
			},
		},
		want: `{
  "oneofEnum": "ZERO"
}`,
	}, {
		desc: "oneof set to empty message",
		input: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofNested{
				OneofNested: &pb3.Nested{},
			},
		},
		want: `{
  "oneofNested": {}
}`,
	}, {
		desc: "oneof set to message",
		input: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofNested{
				OneofNested: &pb3.Nested{
					SString: "nested message",
				},
			},
		},
		want: `{
  "oneofNested": {
    "sString": "nested message"
  }
}`,
	}, {
		desc:  "repeated fields not set",
		input: &pb2.Repeats{},
		want:  "{}",
	}, {
		desc: "repeated fields set to empty slices",
		input: &pb2.Repeats{
			RptBool:   []bool{},
			RptInt32:  []int32{},
			RptInt64:  []int64{},
			RptUint32: []uint32{},
			RptUint64: []uint64{},
			RptFloat:  []float32{},
			RptDouble: []float64{},
			RptBytes:  [][]byte{},
		},
		want: "{}",
	}, {
		desc: "repeated fields set to some values",
		input: &pb2.Repeats{
			RptBool:   []bool{true, false, true, true},
			RptInt32:  []int32{1, 6, 0, 0},
			RptInt64:  []int64{-64, 47},
			RptUint32: []uint32{0xff, 0xffff},
			RptUint64: []uint64{0xdeadbeef},
			RptFloat:  []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), 1.034},
			RptDouble: []float64{math.NaN(), math.Inf(1), math.Inf(-1), 1.23e-308},
			RptString: []string{"hello", "世界"},
			RptBytes: [][]byte{
				[]byte("hello"),
				[]byte("\xe4\xb8\x96\xe7\x95\x8c"),
			},
		},
		want: `{
  "rptBool": [
    true,
    false,
    true,
    true
  ],
  "rptInt32": [
    1,
    6,
    0,
    0
  ],
  "rptInt64": [
    "-64",
    "47"
  ],
  "rptUint32": [
    255,
    65535
  ],
  "rptUint64": [
    "3735928559"
  ],
  "rptFloat": [
    "NaN",
    "Infinity",
    "-Infinity",
    1.034
  ],
  "rptDouble": [
    "NaN",
    "Infinity",
    "-Infinity",
    1.23e-308
  ],
  "rptString": [
    "hello",
    "世界"
  ],
  "rptBytes": [
    "aGVsbG8=",
    "5LiW55WM"
  ]
}`,
	}, {
		desc: "repeated enums",
		input: &pb2.Enums{
			RptEnum:       []pb2.Enum{pb2.Enum_ONE, 2, pb2.Enum_TEN, 42},
			RptNestedEnum: []pb2.Enums_NestedEnum{2, 47, 10},
		},
		want: `{
  "rptEnum": [
    "ONE",
    "TWO",
    "TEN",
    42
  ],
  "rptNestedEnum": [
    "DOS",
    47,
    "DIEZ"
  ]
}`,
	}, {
		desc: "repeated messages set to empty",
		input: &pb2.Nests{
			RptNested: []*pb2.Nested{},
			Rptgroup:  []*pb2.Nests_RptGroup{},
		},
		want: "{}",
	}, {
		desc: "repeated messages",
		input: &pb2.Nests{
			RptNested: []*pb2.Nested{
				{
					OptString: scalar.String("repeat nested one"),
				},
				{
					OptString: scalar.String("repeat nested two"),
					OptNested: &pb2.Nested{
						OptString: scalar.String("inside repeat nested two"),
					},
				},
				{},
			},
		},
		want: `{
  "rptNested": [
    {
      "optString": "repeat nested one"
    },
    {
      "optString": "repeat nested two",
      "optNested": {
        "optString": "inside repeat nested two"
      }
    },
    {}
  ]
}`,
	}, {
		desc: "repeated messages contains nil value",
		input: &pb2.Nests{
			RptNested: []*pb2.Nested{nil, {}},
		},
		want: `{
  "rptNested": [
    {},
    {}
  ]
}`,
	}, {
		desc: "repeated groups",
		input: &pb2.Nests{
			Rptgroup: []*pb2.Nests_RptGroup{
				{
					RptString: []string{"hello", "world"},
				},
				{},
				nil,
			},
		},
		want: `{
  "rptgroup": [
    {
      "rptString": [
        "hello",
        "world"
      ]
    },
    {},
    {}
  ]
}`,
	}, {
		desc:  "map fields not set",
		input: &pb3.Maps{},
		want:  "{}",
	}, {
		desc: "map fields set to empty",
		input: &pb3.Maps{
			Int32ToStr:   map[int32]string{},
			BoolToUint32: map[bool]uint32{},
			Uint64ToEnum: map[uint64]pb3.Enum{},
			StrToNested:  map[string]*pb3.Nested{},
			StrToOneofs:  map[string]*pb3.Oneofs{},
		},
		want: "{}",
	}, {
		desc: "map fields 1",
		input: &pb3.Maps{
			BoolToUint32: map[bool]uint32{
				true:  42,
				false: 101,
			},
		},
		want: `{
  "boolToUint32": {
    "false": 101,
    "true": 42
  }
}`,
	}, {
		desc: "map fields 2",
		input: &pb3.Maps{
			Int32ToStr: map[int32]string{
				-101: "-101",
				0xff: "0xff",
				0:    "zero",
			},
		},
		want: `{
  "int32ToStr": {
    "-101": "-101",
    "0": "zero",
    "255": "0xff"
  }
}`,
	}, {
		desc: "map fields 3",
		input: &pb3.Maps{
			Uint64ToEnum: map[uint64]pb3.Enum{
				1:  pb3.Enum_ONE,
				2:  pb3.Enum_TWO,
				10: pb3.Enum_TEN,
				47: 47,
			},
		},
		want: `{
  "uint64ToEnum": {
    "1": "ONE",
    "2": "TWO",
    "10": "TEN",
    "47": 47
  }
}`,
	}, {
		desc: "map fields 4",
		input: &pb3.Maps{
			StrToNested: map[string]*pb3.Nested{
				"nested": &pb3.Nested{
					SString: "nested in a map",
				},
			},
		},
		want: `{
  "strToNested": {
    "nested": {
      "sString": "nested in a map"
    }
  }
}`,
	}, {
		desc: "map fields 5",
		input: &pb3.Maps{
			StrToOneofs: map[string]*pb3.Oneofs{
				"string": &pb3.Oneofs{
					Union: &pb3.Oneofs_OneofString{
						OneofString: "hello",
					},
				},
				"nested": &pb3.Oneofs{
					Union: &pb3.Oneofs_OneofNested{
						OneofNested: &pb3.Nested{
							SString: "nested oneof in map field value",
						},
					},
				},
			},
		},
		want: `{
  "strToOneofs": {
    "nested": {
      "oneofNested": {
        "sString": "nested oneof in map field value"
      }
    },
    "string": {
      "oneofString": "hello"
    }
  }
}`,
	}, {
		desc: "map field contains nil value",
		input: &pb3.Maps{
			StrToNested: map[string]*pb3.Nested{
				"nil": nil,
			},
		},
		want: `{
  "strToNested": {
    "nil": {}
  }
}`,
	}, {
		desc:    "required fields not set",
		input:   &pb2.Requireds{},
		want:    `{}`,
		wantErr: true,
	}, {
		desc: "required fields partially set",
		input: &pb2.Requireds{
			ReqBool:     scalar.Bool(false),
			ReqSfixed64: scalar.Int64(0),
			ReqDouble:   scalar.Float64(1.23),
			ReqString:   scalar.String("hello"),
			ReqEnum:     pb2.Enum_ONE.Enum(),
		},
		want: `{
  "reqBool": false,
  "reqSfixed64": "0",
  "reqDouble": 1.23,
  "reqString": "hello",
  "reqEnum": "ONE"
}`,
		wantErr: true,
	}, {
		desc: "required fields not set with AllowPartial",
		mo:   protojson.MarshalOptions{AllowPartial: true},
		input: &pb2.Requireds{
			ReqBool:     scalar.Bool(false),
			ReqSfixed64: scalar.Int64(0),
			ReqDouble:   scalar.Float64(1.23),
			ReqString:   scalar.String("hello"),
			ReqEnum:     pb2.Enum_ONE.Enum(),
		},
		want: `{
  "reqBool": false,
  "reqSfixed64": "0",
  "reqDouble": 1.23,
  "reqString": "hello",
  "reqEnum": "ONE"
}`,
	}, {
		desc: "required fields all set",
		input: &pb2.Requireds{
			ReqBool:     scalar.Bool(false),
			ReqSfixed64: scalar.Int64(0),
			ReqDouble:   scalar.Float64(1.23),
			ReqString:   scalar.String("hello"),
			ReqEnum:     pb2.Enum_ONE.Enum(),
			ReqNested:   &pb2.Nested{},
		},
		want: `{
  "reqBool": false,
  "reqSfixed64": "0",
  "reqDouble": 1.23,
  "reqString": "hello",
  "reqEnum": "ONE",
  "reqNested": {}
}`,
	}, {
		desc: "indirect required field",
		input: &pb2.IndirectRequired{
			OptNested: &pb2.NestedWithRequired{},
		},
		want: `{
  "optNested": {}
}`,
		wantErr: true,
	}, {
		desc: "indirect required field with AllowPartial",
		mo:   protojson.MarshalOptions{AllowPartial: true},
		input: &pb2.IndirectRequired{
			OptNested: &pb2.NestedWithRequired{},
		},
		want: `{
  "optNested": {}
}`,
	}, {
		desc: "indirect required field in empty repeated",
		input: &pb2.IndirectRequired{
			RptNested: []*pb2.NestedWithRequired{},
		},
		want: `{}`,
	}, {
		desc: "indirect required field in repeated",
		input: &pb2.IndirectRequired{
			RptNested: []*pb2.NestedWithRequired{
				&pb2.NestedWithRequired{},
			},
		},
		want: `{
  "rptNested": [
    {}
  ]
}`,
		wantErr: true,
	}, {
		desc: "indirect required field in repeated with AllowPartial",
		mo:   protojson.MarshalOptions{AllowPartial: true},
		input: &pb2.IndirectRequired{
			RptNested: []*pb2.NestedWithRequired{
				&pb2.NestedWithRequired{},
			},
		},
		want: `{
  "rptNested": [
    {}
  ]
}`,
	}, {
		desc: "indirect required field in empty map",
		input: &pb2.IndirectRequired{
			StrToNested: map[string]*pb2.NestedWithRequired{},
		},
		want: "{}",
	}, {
		desc: "indirect required field in map",
		input: &pb2.IndirectRequired{
			StrToNested: map[string]*pb2.NestedWithRequired{
				"fail": &pb2.NestedWithRequired{},
			},
		},
		want: `{
  "strToNested": {
    "fail": {}
  }
}`,
		wantErr: true,
	}, {
		desc: "indirect required field in map with AllowPartial",
		mo:   protojson.MarshalOptions{AllowPartial: true},
		input: &pb2.IndirectRequired{
			StrToNested: map[string]*pb2.NestedWithRequired{
				"fail": &pb2.NestedWithRequired{},
			},
		},
		want: `{
  "strToNested": {
    "fail": {}
  }
}`,
	}, {
		desc: "indirect required field in oneof",
		input: &pb2.IndirectRequired{
			Union: &pb2.IndirectRequired_OneofNested{
				OneofNested: &pb2.NestedWithRequired{},
			},
		},
		want: `{
  "oneofNested": {}
}`,
		wantErr: true,
	}, {
		desc: "indirect required field in oneof with AllowPartial",
		mo:   protojson.MarshalOptions{AllowPartial: true},
		input: &pb2.IndirectRequired{
			Union: &pb2.IndirectRequired_OneofNested{
				OneofNested: &pb2.NestedWithRequired{},
			},
		},
		want: `{
  "oneofNested": {}
}`,
	}, {
		desc: "unknown fields are ignored",
		input: &pb2.Scalars{
			OptString: scalar.String("no unknowns"),
			XXX_unrecognized: pack.Message{
				pack.Tag{101, pack.BytesType}, pack.String("hello world"),
			}.Marshal(),
		},
		want: `{
  "optString": "no unknowns"
}`,
	}, {
		desc: "json_name",
		input: &pb3.JSONNames{
			SString: "json_name",
		},
		want: `{
  "foo_bar": "json_name"
}`,
	}, {
		desc: "extensions of non-repeated fields",
		input: func() proto.Message {
			m := &pb2.Extensions{
				OptString: scalar.String("non-extension field"),
				OptBool:   scalar.Bool(true),
				OptInt32:  scalar.Int32(42),
			}
			setExtension(m, pb2.E_OptExtBool, true)
			setExtension(m, pb2.E_OptExtString, "extension field")
			setExtension(m, pb2.E_OptExtEnum, pb2.Enum_TEN)
			setExtension(m, pb2.E_OptExtNested, &pb2.Nested{
				OptString: scalar.String("nested in an extension"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("another nested in an extension"),
				},
			})
			return m
		}(),
		want: `{
  "optString": "non-extension field",
  "optBool": true,
  "optInt32": 42,
  "[pb2.opt_ext_bool]": true,
  "[pb2.opt_ext_enum]": "TEN",
  "[pb2.opt_ext_nested]": {
    "optString": "nested in an extension",
    "optNested": {
      "optString": "another nested in an extension"
    }
  },
  "[pb2.opt_ext_string]": "extension field"
}`,
	}, {
		desc: "extension message field set to nil",
		input: func() proto.Message {
			m := &pb2.Extensions{}
			setExtension(m, pb2.E_OptExtNested, nil)
			return m
		}(),
		want: "{}",
	}, {
		desc: "extensions of repeated fields",
		input: func() proto.Message {
			m := &pb2.Extensions{}
			setExtension(m, pb2.E_RptExtEnum, &[]pb2.Enum{pb2.Enum_TEN, 101, pb2.Enum_ONE})
			setExtension(m, pb2.E_RptExtFixed32, &[]uint32{42, 47})
			setExtension(m, pb2.E_RptExtNested, &[]*pb2.Nested{
				&pb2.Nested{OptString: scalar.String("one")},
				&pb2.Nested{OptString: scalar.String("two")},
				&pb2.Nested{OptString: scalar.String("three")},
			})
			return m
		}(),
		want: `{
  "[pb2.rpt_ext_enum]": [
    "TEN",
    101,
    "ONE"
  ],
  "[pb2.rpt_ext_fixed32]": [
    42,
    47
  ],
  "[pb2.rpt_ext_nested]": [
    {
      "optString": "one"
    },
    {
      "optString": "two"
    },
    {
      "optString": "three"
    }
  ]
}`,
	}, {
		desc: "extensions of non-repeated fields in another message",
		input: func() proto.Message {
			m := &pb2.Extensions{}
			setExtension(m, pb2.E_ExtensionsContainer_OptExtBool, true)
			setExtension(m, pb2.E_ExtensionsContainer_OptExtString, "extension field")
			setExtension(m, pb2.E_ExtensionsContainer_OptExtEnum, pb2.Enum_TEN)
			setExtension(m, pb2.E_ExtensionsContainer_OptExtNested, &pb2.Nested{
				OptString: scalar.String("nested in an extension"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("another nested in an extension"),
				},
			})
			return m
		}(),
		want: `{
  "[pb2.ExtensionsContainer.opt_ext_bool]": true,
  "[pb2.ExtensionsContainer.opt_ext_enum]": "TEN",
  "[pb2.ExtensionsContainer.opt_ext_nested]": {
    "optString": "nested in an extension",
    "optNested": {
      "optString": "another nested in an extension"
    }
  },
  "[pb2.ExtensionsContainer.opt_ext_string]": "extension field"
}`,
	}, {
		desc: "extensions of repeated fields in another message",
		input: func() proto.Message {
			m := &pb2.Extensions{
				OptString: scalar.String("non-extension field"),
				OptBool:   scalar.Bool(true),
				OptInt32:  scalar.Int32(42),
			}
			setExtension(m, pb2.E_ExtensionsContainer_RptExtEnum, &[]pb2.Enum{pb2.Enum_TEN, 101, pb2.Enum_ONE})
			setExtension(m, pb2.E_ExtensionsContainer_RptExtString, &[]string{"hello", "world"})
			setExtension(m, pb2.E_ExtensionsContainer_RptExtNested, &[]*pb2.Nested{
				&pb2.Nested{OptString: scalar.String("one")},
				&pb2.Nested{OptString: scalar.String("two")},
				&pb2.Nested{OptString: scalar.String("three")},
			})
			return m
		}(),
		want: `{
  "optString": "non-extension field",
  "optBool": true,
  "optInt32": 42,
  "[pb2.ExtensionsContainer.rpt_ext_enum]": [
    "TEN",
    101,
    "ONE"
  ],
  "[pb2.ExtensionsContainer.rpt_ext_nested]": [
    {
      "optString": "one"
    },
    {
      "optString": "two"
    },
    {
      "optString": "three"
    }
  ],
  "[pb2.ExtensionsContainer.rpt_ext_string]": [
    "hello",
    "world"
  ]
}`,
	}, {
		desc: "MessageSet",
		input: func() proto.Message {
			m := &pb2.MessageSet{}
			setExtension(m, pb2.E_MessageSetExtension_MessageSetExtension, &pb2.MessageSetExtension{
				OptString: scalar.String("a messageset extension"),
			})
			setExtension(m, pb2.E_MessageSetExtension_NotMessageSetExtension, &pb2.MessageSetExtension{
				OptString: scalar.String("not a messageset extension"),
			})
			setExtension(m, pb2.E_MessageSetExtension_ExtNested, &pb2.Nested{
				OptString: scalar.String("just a regular extension"),
			})
			return m
		}(),
		want: `{
  "[pb2.MessageSetExtension]": {
    "optString": "a messageset extension"
  },
  "[pb2.MessageSetExtension.ext_nested]": {
    "optString": "just a regular extension"
  },
  "[pb2.MessageSetExtension.not_message_set_extension]": {
    "optString": "not a messageset extension"
  }
}`,
	}, {
		desc: "not real MessageSet 1",
		input: func() proto.Message {
			m := &pb2.FakeMessageSet{}
			setExtension(m, pb2.E_FakeMessageSetExtension_MessageSetExtension, &pb2.FakeMessageSetExtension{
				OptString: scalar.String("not a messageset extension"),
			})
			return m
		}(),
		want: `{
  "[pb2.FakeMessageSetExtension.message_set_extension]": {
    "optString": "not a messageset extension"
  }
}`,
	}, {
		desc: "not real MessageSet 2",
		input: func() proto.Message {
			m := &pb2.MessageSet{}
			setExtension(m, pb2.E_MessageSetExtension, &pb2.FakeMessageSetExtension{
				OptString: scalar.String("another not a messageset extension"),
			})
			return m
		}(),
		want: `{
  "[pb2.message_set_extension]": {
    "optString": "another not a messageset extension"
  }
}`,
	}, {
		desc:  "BoolValue empty",
		input: &wrapperspb.BoolValue{},
		want:  `false`,
	}, {
		desc:  "BoolValue",
		input: &wrapperspb.BoolValue{Value: true},
		want:  `true`,
	}, {
		desc:  "Int32Value empty",
		input: &wrapperspb.Int32Value{},
		want:  `0`,
	}, {
		desc:  "Int32Value",
		input: &wrapperspb.Int32Value{Value: 42},
		want:  `42`,
	}, {
		desc:  "Int64Value",
		input: &wrapperspb.Int64Value{Value: 42},
		want:  `"42"`,
	}, {
		desc:  "UInt32Value",
		input: &wrapperspb.UInt32Value{Value: 42},
		want:  `42`,
	}, {
		desc:  "UInt64Value",
		input: &wrapperspb.UInt64Value{Value: 42},
		want:  `"42"`,
	}, {
		desc:  "FloatValue",
		input: &wrapperspb.FloatValue{Value: 1.02},
		want:  `1.02`,
	}, {
		desc:  "FloatValue Infinity",
		input: &wrapperspb.FloatValue{Value: float32(math.Inf(-1))},
		want:  `"-Infinity"`,
	}, {
		desc:  "DoubleValue",
		input: &wrapperspb.DoubleValue{Value: 1.02},
		want:  `1.02`,
	}, {
		desc:  "DoubleValue NaN",
		input: &wrapperspb.DoubleValue{Value: math.NaN()},
		want:  `"NaN"`,
	}, {
		desc:  "StringValue empty",
		input: &wrapperspb.StringValue{},
		want:  `""`,
	}, {
		desc:  "StringValue",
		input: &wrapperspb.StringValue{Value: "谷歌"},
		want:  `"谷歌"`,
	}, {
		desc:    "StringValue with invalid UTF8 error",
		input:   &wrapperspb.StringValue{Value: "abc\xff"},
		want:    "\"abc\xff\"",
		wantErr: true,
	}, {
		desc: "StringValue field with invalid UTF8 error",
		input: &pb2.KnownTypes{
			OptString: &wrapperspb.StringValue{Value: "abc\xff"},
		},
		want:    "{\n  \"optString\": \"abc\xff\"\n}",
		wantErr: true,
	}, {
		desc:  "BytesValue",
		input: &wrapperspb.BytesValue{Value: []byte("hello")},
		want:  `"aGVsbG8="`,
	}, {
		desc:  "Empty",
		input: &emptypb.Empty{},
		want:  `{}`,
	}, {
		desc:  "NullValue field",
		input: &pb2.KnownTypes{OptNull: new(structpb.NullValue)},
		want: `{
  "optNull": null
}`,
	}, {
		desc:    "Value empty",
		input:   &structpb.Value{},
		wantErr: true,
	}, {
		desc: "Value empty field",
		input: &pb2.KnownTypes{
			OptValue: &structpb.Value{},
		},
		wantErr: true,
	}, {
		desc:  "Value contains NullValue",
		input: &structpb.Value{Kind: &structpb.Value_NullValue{}},
		want:  `null`,
	}, {
		desc:  "Value contains BoolValue",
		input: &structpb.Value{Kind: &structpb.Value_BoolValue{}},
		want:  `false`,
	}, {
		desc:  "Value contains NumberValue",
		input: &structpb.Value{Kind: &structpb.Value_NumberValue{1.02}},
		want:  `1.02`,
	}, {
		desc:  "Value contains StringValue",
		input: &structpb.Value{Kind: &structpb.Value_StringValue{"hello"}},
		want:  `"hello"`,
	}, {
		desc:    "Value contains StringValue with invalid UTF8",
		input:   &structpb.Value{Kind: &structpb.Value_StringValue{"\xff"}},
		want:    "\"\xff\"",
		wantErr: true,
	}, {
		desc: "Value contains Struct",
		input: &structpb.Value{
			Kind: &structpb.Value_StructValue{
				&structpb.Struct{
					Fields: map[string]*structpb.Value{
						"null":   {Kind: &structpb.Value_NullValue{}},
						"number": {Kind: &structpb.Value_NumberValue{}},
						"string": {Kind: &structpb.Value_StringValue{}},
						"struct": {Kind: &structpb.Value_StructValue{}},
						"list":   {Kind: &structpb.Value_ListValue{}},
						"bool":   {Kind: &structpb.Value_BoolValue{}},
					},
				},
			},
		},
		want: `{
  "bool": false,
  "list": [],
  "null": null,
  "number": 0,
  "string": "",
  "struct": {}
}`,
	}, {
		desc: "Value contains ListValue",
		input: &structpb.Value{
			Kind: &structpb.Value_ListValue{
				&structpb.ListValue{
					Values: []*structpb.Value{
						{Kind: &structpb.Value_BoolValue{}},
						{Kind: &structpb.Value_NullValue{}},
						{Kind: &structpb.Value_NumberValue{}},
						{Kind: &structpb.Value_StringValue{}},
						{Kind: &structpb.Value_StructValue{}},
						{Kind: &structpb.Value_ListValue{}},
					},
				},
			},
		},
		want: `[
  false,
  null,
  0,
  "",
  {},
  []
]`,
	}, {
		desc:  "Struct with nil map",
		input: &structpb.Struct{},
		want:  `{}`,
	}, {
		desc: "Struct with empty map",
		input: &structpb.Struct{
			Fields: map[string]*structpb.Value{},
		},
		want: `{}`,
	}, {
		desc: "Struct",
		input: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"bool":   {Kind: &structpb.Value_BoolValue{true}},
				"null":   {Kind: &structpb.Value_NullValue{}},
				"number": {Kind: &structpb.Value_NumberValue{3.1415}},
				"string": {Kind: &structpb.Value_StringValue{"hello"}},
				"struct": {
					Kind: &structpb.Value_StructValue{
						&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"string": {Kind: &structpb.Value_StringValue{"world"}},
							},
						},
					},
				},
				"list": {
					Kind: &structpb.Value_ListValue{
						&structpb.ListValue{
							Values: []*structpb.Value{
								{Kind: &structpb.Value_BoolValue{}},
								{Kind: &structpb.Value_NullValue{}},
								{Kind: &structpb.Value_NumberValue{}},
							},
						},
					},
				},
			},
		},
		want: `{
  "bool": true,
  "list": [
    false,
    null,
    0
  ],
  "null": null,
  "number": 3.1415,
  "string": "hello",
  "struct": {
    "string": "world"
  }
}`,
	}, {
		desc: "Struct message with invalid UTF8 string",
		input: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"string": {Kind: &structpb.Value_StringValue{"\xff"}},
			},
		},
		want:    "{\n  \"string\": \"\xff\"\n}",
		wantErr: true,
	}, {
		desc:  "ListValue with nil values",
		input: &structpb.ListValue{},
		want:  `[]`,
	}, {
		desc: "ListValue with empty values",
		input: &structpb.ListValue{
			Values: []*structpb.Value{},
		},
		want: `[]`,
	}, {
		desc: "ListValue",
		input: &structpb.ListValue{
			Values: []*structpb.Value{
				{Kind: &structpb.Value_BoolValue{true}},
				{Kind: &structpb.Value_NullValue{}},
				{Kind: &structpb.Value_NumberValue{3.1415}},
				{Kind: &structpb.Value_StringValue{"hello"}},
				{
					Kind: &structpb.Value_ListValue{
						&structpb.ListValue{
							Values: []*structpb.Value{
								{Kind: &structpb.Value_BoolValue{}},
								{Kind: &structpb.Value_NullValue{}},
								{Kind: &structpb.Value_NumberValue{}},
							},
						},
					},
				},
				{
					Kind: &structpb.Value_StructValue{
						&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"string": {Kind: &structpb.Value_StringValue{"world"}},
							},
						},
					},
				},
			},
		},
		want: `[
  true,
  null,
  3.1415,
  "hello",
  [
    false,
    null,
    0
  ],
  {
    "string": "world"
  }
]`,
	}, {
		desc: "ListValue with invalid UTF8 string",
		input: &structpb.ListValue{
			Values: []*structpb.Value{
				{Kind: &structpb.Value_StringValue{"\xff"}},
			},
		},
		want:    "[\n  \"\xff\"\n]",
		wantErr: true,
	}, {
		desc:  "Duration empty",
		input: &durationpb.Duration{},
		want:  `"0s"`,
	}, {
		desc:  "Duration with secs",
		input: &durationpb.Duration{Seconds: 3},
		want:  `"3s"`,
	}, {
		desc:  "Duration with -secs",
		input: &durationpb.Duration{Seconds: -3},
		want:  `"-3s"`,
	}, {
		desc:  "Duration with nanos",
		input: &durationpb.Duration{Nanos: 1e6},
		want:  `"0.001s"`,
	}, {
		desc:  "Duration with -nanos",
		input: &durationpb.Duration{Nanos: -1e6},
		want:  `"-0.001s"`,
	}, {
		desc:  "Duration with large secs",
		input: &durationpb.Duration{Seconds: 1e10, Nanos: 1},
		want:  `"10000000000.000000001s"`,
	}, {
		desc:  "Duration with 6-digit nanos",
		input: &durationpb.Duration{Nanos: 1e4},
		want:  `"0.000010s"`,
	}, {
		desc:  "Duration with 3-digit nanos",
		input: &durationpb.Duration{Nanos: 1e6},
		want:  `"0.001s"`,
	}, {
		desc:  "Duration with -secs -nanos",
		input: &durationpb.Duration{Seconds: -123, Nanos: -450},
		want:  `"-123.000000450s"`,
	}, {
		desc:  "Duration max value",
		input: &durationpb.Duration{Seconds: 315576000000, Nanos: 999999999},
		want:  `"315576000000.999999999s"`,
	}, {
		desc:  "Duration min value",
		input: &durationpb.Duration{Seconds: -315576000000, Nanos: -999999999},
		want:  `"-315576000000.999999999s"`,
	}, {
		desc:    "Duration with +secs -nanos",
		input:   &durationpb.Duration{Seconds: 1, Nanos: -1},
		wantErr: true,
	}, {
		desc:    "Duration with -secs +nanos",
		input:   &durationpb.Duration{Seconds: -1, Nanos: 1},
		wantErr: true,
	}, {
		desc:    "Duration with +secs out of range",
		input:   &durationpb.Duration{Seconds: 315576000001},
		wantErr: true,
	}, {
		desc:    "Duration with -secs out of range",
		input:   &durationpb.Duration{Seconds: -315576000001},
		wantErr: true,
	}, {
		desc:    "Duration with +nanos out of range",
		input:   &durationpb.Duration{Seconds: 0, Nanos: 1e9},
		wantErr: true,
	}, {
		desc:    "Duration with -nanos out of range",
		input:   &durationpb.Duration{Seconds: 0, Nanos: -1e9},
		wantErr: true,
	}, {
		desc:  "Timestamp zero",
		input: &timestamppb.Timestamp{},
		want:  `"1970-01-01T00:00:00Z"`,
	}, {
		desc:  "Timestamp",
		input: &timestamppb.Timestamp{Seconds: 1553036601},
		want:  `"2019-03-19T23:03:21Z"`,
	}, {
		desc:  "Timestamp with nanos",
		input: &timestamppb.Timestamp{Seconds: 1553036601, Nanos: 1},
		want:  `"2019-03-19T23:03:21.000000001Z"`,
	}, {
		desc:  "Timestamp with 6-digit nanos",
		input: &timestamppb.Timestamp{Nanos: 1e3},
		want:  `"1970-01-01T00:00:00.000001Z"`,
	}, {
		desc:  "Timestamp with 3-digit nanos",
		input: &timestamppb.Timestamp{Nanos: 1e7},
		want:  `"1970-01-01T00:00:00.010Z"`,
	}, {
		desc:  "Timestamp max value",
		input: &timestamppb.Timestamp{Seconds: 253402300799, Nanos: 999999999},
		want:  `"9999-12-31T23:59:59.999999999Z"`,
	}, {
		desc:  "Timestamp min value",
		input: &timestamppb.Timestamp{Seconds: -62135596800},
		want:  `"0001-01-01T00:00:00Z"`,
	}, {
		desc:    "Timestamp with +secs out of range",
		input:   &timestamppb.Timestamp{Seconds: 253402300800},
		wantErr: true,
	}, {
		desc:    "Timestamp with -secs out of range",
		input:   &timestamppb.Timestamp{Seconds: -62135596801},
		wantErr: true,
	}, {
		desc:    "Timestamp with -nanos",
		input:   &timestamppb.Timestamp{Nanos: -1},
		wantErr: true,
	}, {
		desc:    "Timestamp with +nanos out of range",
		input:   &timestamppb.Timestamp{Nanos: 1e9},
		wantErr: true,
	}, {
		desc:  "FieldMask empty",
		input: &fieldmaskpb.FieldMask{},
		want:  `""`,
	}, {
		desc: "FieldMask",
		input: &fieldmaskpb.FieldMask{
			Paths: []string{
				"foo",
				"foo_bar",
				"foo.bar_qux",
				"_foo",
			},
		},
		want: `"foo,fooBar,foo.barQux,Foo"`,
	}, {
		desc: "FieldMask error 1",
		input: &fieldmaskpb.FieldMask{
			Paths: []string{"foo_"},
		},
		wantErr: true,
	}, {
		desc: "FieldMask error 2",
		input: &fieldmaskpb.FieldMask{
			Paths: []string{"foo__bar"},
		},
		wantErr: true,
	}, {
		desc:  "Any empty",
		input: &anypb.Any{},
		want:  `{}`,
	}, {
		desc: "Any with non-custom message",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&pb2.Nested{})),
		},
		input: func() proto.Message {
			m := &pb2.Nested{
				OptString: scalar.String("embedded inside Any"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("inception"),
				},
			}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "foo/pb2.Nested",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "foo/pb2.Nested",
  "optString": "embedded inside Any",
  "optNested": {
    "optString": "inception"
  }
}`,
	}, {
		desc: "Any with empty embedded message",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&pb2.Nested{})),
		},
		input: &anypb.Any{TypeUrl: "foo/pb2.Nested"},
		want: `{
  "@type": "foo/pb2.Nested"
}`,
	}, {
		desc:    "Any without registered type",
		mo:      protojson.MarshalOptions{Resolver: preg.NewTypes()},
		input:   &anypb.Any{TypeUrl: "foo/pb2.Nested"},
		wantErr: true,
	}, {
		desc: "Any with missing required error",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&pb2.PartialRequired{})),
		},
		input: func() proto.Message {
			m := &pb2.PartialRequired{
				OptString: scalar.String("embedded inside Any"),
			}
			b, err := proto.MarshalOptions{
				AllowPartial:  true,
				Deterministic: true,
			}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: string(m.ProtoReflect().Descriptor().FullName()),
				Value:   b,
			}
		}(),
		want: `{
  "@type": "pb2.PartialRequired",
  "optString": "embedded inside Any"
}`,
		wantErr: true,
	}, {
		desc: "Any with partial required and AllowPartial",
		mo: protojson.MarshalOptions{
			AllowPartial: true,
			Resolver:     preg.NewTypes(pimpl.Export{}.MessageTypeOf(&pb2.PartialRequired{})),
		},
		input: func() proto.Message {
			m := &pb2.PartialRequired{
				OptString: scalar.String("embedded inside Any"),
			}
			b, err := proto.MarshalOptions{
				AllowPartial:  true,
				Deterministic: true,
			}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: string(m.ProtoReflect().Descriptor().FullName()),
				Value:   b,
			}
		}(),
		want: `{
  "@type": "pb2.PartialRequired",
  "optString": "embedded inside Any"
}`,
	}, {
		desc: "Any with invalid UTF8",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&pb2.Nested{})),
		},
		input: func() proto.Message {
			m := &pb2.Nested{
				OptString: scalar.String("abc\xff"),
			}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "foo/pb2.Nested",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "foo/pb2.Nested",
  "optString": "` + "abc\xff" + `"
}`,
		wantErr: true,
	}, {
		desc: "Any with invalid value",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&pb2.Nested{})),
		},
		input: &anypb.Any{
			TypeUrl: "foo/pb2.Nested",
			Value:   dhex("80"),
		},
		wantErr: true,
	}, {
		desc: "Any with BoolValue",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&wrapperspb.BoolValue{})),
		},
		input: func() proto.Message {
			m := &wrapperspb.BoolValue{Value: true}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "type.googleapis.com/google.protobuf.BoolValue",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "type.googleapis.com/google.protobuf.BoolValue",
  "value": true
}`,
	}, {
		desc: "Any with Empty",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&emptypb.Empty{})),
		},
		input: func() proto.Message {
			m := &emptypb.Empty{}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "type.googleapis.com/google.protobuf.Empty",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "type.googleapis.com/google.protobuf.Empty",
  "value": {}
}`,
	}, {
		desc: "Any with StringValue containing invalid UTF8",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&wrapperspb.StringValue{})),
		},
		input: func() proto.Message {
			m := &wrapperspb.StringValue{Value: "abcd"}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "google.protobuf.StringValue",
				Value:   bytes.Replace(b, []byte("abcd"), []byte("abc\xff"), -1),
			}
		}(),
		want: `{
  "@type": "google.protobuf.StringValue",
  "value": "` + "abc\xff" + `"
}`,
		wantErr: true,
	}, {
		desc: "Any with Int64Value",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&wrapperspb.Int64Value{})),
		},
		input: func() proto.Message {
			m := &wrapperspb.Int64Value{Value: 42}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "google.protobuf.Int64Value",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "google.protobuf.Int64Value",
  "value": "42"
}`,
	}, {
		desc: "Any with Duration",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&durationpb.Duration{})),
		},
		input: func() proto.Message {
			m := &durationpb.Duration{}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "type.googleapis.com/google.protobuf.Duration",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "type.googleapis.com/google.protobuf.Duration",
  "value": "0s"
}`,
	}, {
		desc: "Any with empty Value",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&structpb.Value{})),
		},
		input: func() proto.Message {
			m := &structpb.Value{}
			b, err := proto.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "type.googleapis.com/google.protobuf.Value",
				Value:   b,
			}
		}(),
		wantErr: true,
	}, {
		desc: "Any with Value of StringValue",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&structpb.Value{})),
		},
		input: func() proto.Message {
			m := &structpb.Value{Kind: &structpb.Value_StringValue{"abcd"}}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "type.googleapis.com/google.protobuf.Value",
				Value:   bytes.Replace(b, []byte("abcd"), []byte("abc\xff"), -1),
			}
		}(),
		want: `{
  "@type": "type.googleapis.com/google.protobuf.Value",
  "value": "` + "abc\xff" + `"
}`,
		wantErr: true,
	}, {
		desc: "Any with Value of NullValue",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&structpb.Value{})),
		},
		input: func() proto.Message {
			m := &structpb.Value{Kind: &structpb.Value_NullValue{}}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "type.googleapis.com/google.protobuf.Value",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "type.googleapis.com/google.protobuf.Value",
  "value": null
}`,
	}, {
		desc: "Any with Struct",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(
				pimpl.Export{}.MessageTypeOf(&structpb.Struct{}),
				pimpl.Export{}.MessageTypeOf(&structpb.Value{}),
				pimpl.Export{}.MessageTypeOf(&wrapperspb.BoolValue{}),
				pimpl.Export{}.EnumTypeOf(structpb.NullValue_NULL_VALUE),
				pimpl.Export{}.MessageTypeOf(&wrapperspb.StringValue{}),
			),
		},
		input: func() proto.Message {
			m := &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"bool":   {Kind: &structpb.Value_BoolValue{true}},
					"null":   {Kind: &structpb.Value_NullValue{}},
					"string": {Kind: &structpb.Value_StringValue{"hello"}},
					"struct": {
						Kind: &structpb.Value_StructValue{
							&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"string": {Kind: &structpb.Value_StringValue{"world"}},
								},
							},
						},
					},
				},
			}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				TypeUrl: "google.protobuf.Struct",
				Value:   b,
			}
		}(),
		want: `{
  "@type": "google.protobuf.Struct",
  "value": {
    "bool": true,
    "null": null,
    "string": "hello",
    "struct": {
      "string": "world"
    }
  }
}`,
	}, {
		desc: "Any with missing type_url",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&wrapperspb.BoolValue{})),
		},
		input: func() proto.Message {
			m := &wrapperspb.BoolValue{Value: true}
			b, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &anypb.Any{
				Value: b,
			}
		}(),
		wantErr: true,
	}, {
		desc: "well known types as field values",
		mo: protojson.MarshalOptions{
			Resolver: preg.NewTypes(pimpl.Export{}.MessageTypeOf(&emptypb.Empty{})),
		},
		input: &pb2.KnownTypes{
			OptBool:      &wrapperspb.BoolValue{Value: false},
			OptInt32:     &wrapperspb.Int32Value{Value: 42},
			OptInt64:     &wrapperspb.Int64Value{Value: 42},
			OptUint32:    &wrapperspb.UInt32Value{Value: 42},
			OptUint64:    &wrapperspb.UInt64Value{Value: 42},
			OptFloat:     &wrapperspb.FloatValue{Value: 1.23},
			OptDouble:    &wrapperspb.DoubleValue{Value: 3.1415},
			OptString:    &wrapperspb.StringValue{Value: "hello"},
			OptBytes:     &wrapperspb.BytesValue{Value: []byte("hello")},
			OptDuration:  &durationpb.Duration{Seconds: 123},
			OptTimestamp: &timestamppb.Timestamp{Seconds: 1553036601},
			OptStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"string": {Kind: &structpb.Value_StringValue{"hello"}},
				},
			},
			OptList: &structpb.ListValue{
				Values: []*structpb.Value{
					{Kind: &structpb.Value_NullValue{}},
					{Kind: &structpb.Value_StringValue{}},
					{Kind: &structpb.Value_StructValue{}},
					{Kind: &structpb.Value_ListValue{}},
				},
			},
			OptValue: &structpb.Value{
				Kind: &structpb.Value_StringValue{"world"},
			},
			OptEmpty: &emptypb.Empty{},
			OptAny: &anypb.Any{
				TypeUrl: "google.protobuf.Empty",
			},
			OptFieldmask: &fieldmaskpb.FieldMask{
				Paths: []string{"foo_bar", "bar_foo"},
			},
		},
		want: `{
  "optBool": false,
  "optInt32": 42,
  "optInt64": "42",
  "optUint32": 42,
  "optUint64": "42",
  "optFloat": 1.23,
  "optDouble": 3.1415,
  "optString": "hello",
  "optBytes": "aGVsbG8=",
  "optDuration": "123s",
  "optTimestamp": "2019-03-19T23:03:21Z",
  "optStruct": {
    "string": "hello"
  },
  "optList": [
    null,
    "",
    {},
    []
  ],
  "optValue": "world",
  "optEmpty": {},
  "optAny": {
    "@type": "google.protobuf.Empty",
    "value": {}
  },
  "optFieldmask": "fooBar,barFoo"
}`,
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			// Use 2-space indentation on all MarshalOptions.
			tt.mo.Indent = "  "
			b, err := tt.mo.Marshal(tt.input)
			if err != nil && !tt.wantErr {
				t.Errorf("Marshal() returned error: %v\n", err)
			}
			if err == nil && tt.wantErr {
				t.Errorf("Marshal() got nil error, want error\n")
			}
			got := string(b)
			if got != tt.want {
				t.Errorf("Marshal()\n<got>\n%v\n<want>\n%v\n", got, tt.want)
				if diff := cmp.Diff(tt.want, got, splitLines); diff != "" {
					t.Errorf("Marshal() diff -want +got\n%v\n", diff)
				}
			}
		})
	}
}
