// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb_test

import (
	"math"
	"strings"
	"testing"

	"github.com/golang/protobuf/v2/encoding/jsonpb"
	"github.com/golang/protobuf/v2/internal/encoding/pack"
	"github.com/golang/protobuf/v2/internal/scalar"
	"github.com/golang/protobuf/v2/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/golang/protobuf/v2/encoding/testprotos/pb2"
	"github.com/golang/protobuf/v2/encoding/testprotos/pb3"
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

func TestMarshal(t *testing.T) {
	tests := []struct {
		desc  string
		mo    jsonpb.MarshalOptions
		input proto.Message
		want  string
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
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			b, err := tt.mo.Marshal(tt.input)
			if err != nil {
				t.Errorf("Marshal() returned error: %v\n", err)
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
