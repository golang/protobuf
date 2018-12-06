// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpb_test

import (
	"math"
	"strings"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/encoding/textpb"
	"github.com/golang/protobuf/v2/internal/detrand"
	"github.com/golang/protobuf/v2/internal/impl"
	"github.com/golang/protobuf/v2/internal/scalar"
	"github.com/golang/protobuf/v2/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	// The legacy package must be imported prior to use of any legacy messages.
	// TODO: Remove this when protoV1 registers these hooks for you.
	_ "github.com/golang/protobuf/v2/internal/legacy"

	"github.com/golang/protobuf/v2/encoding/textpb/testprotos/pb2"
	"github.com/golang/protobuf/v2/encoding/textpb/testprotos/pb3"
)

func init() {
	// Disable detrand to enable direct comparisons on outputs.
	detrand.Disable()
}

func M(m interface{}) proto.Message {
	return impl.Export{}.MessageOf(m).Interface()
}

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
		desc    string
		input   protoV1.Message
		want    string
		wantErr bool
	}{{
		desc:  "proto2 optional scalar fields not set",
		input: &pb2.Scalars{},
		want:  "\n",
	}, {
		desc:  "proto3 scalar fields not set",
		input: &pb3.Scalars{},
		want:  "\n",
	}, {
		desc: "proto2 optional scalar fields set to zero values",
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
		want: `opt_bool: false
opt_int32: 0
opt_int64: 0
opt_uint32: 0
opt_uint64: 0
opt_sint32: 0
opt_sint64: 0
opt_fixed32: 0
opt_fixed64: 0
opt_sfixed32: 0
opt_sfixed64: 0
opt_float: 0
opt_double: 0
opt_bytes: ""
opt_string: ""
`,
	}, {
		desc: "proto3 scalar fields set to zero values",
		input: &pb3.Scalars{
			SBool:     false,
			SInt32:    0,
			SInt64:    0,
			SUint32:   0,
			SUint64:   0,
			SSint32:   0,
			SSint64:   0,
			SFixed32:  0,
			SFixed64:  0,
			SSfixed32: 0,
			SSfixed64: 0,
			SFloat:    0,
			SDouble:   0,
			SBytes:    []byte{},
			SString:   "",
		},
		want: "\n",
	}, {
		desc: "proto2 optional scalar fields set to some values",
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
			// TODO: Update encoder to output same decimals.
			OptFloat:  scalar.Float32(1.02),
			OptDouble: scalar.Float64(1.23e100),
			// TODO: Update encoder to not output UTF8 for bytes.
			OptBytes:  []byte("\xe8\xb0\xb7\xe6\xad\x8c"),
			OptString: scalar.String("谷歌"),
		},
		want: `opt_bool: true
opt_int32: 255
opt_int64: 3735928559
opt_uint32: 47
opt_uint64: 3735928559
opt_sint32: -1001
opt_sint64: -65535
opt_fixed64: 64
opt_sfixed32: -32
opt_float: 1.0199999809265137
opt_double: 1.23e+100
opt_bytes: "谷歌"
opt_string: "谷歌"
`,
	}, {
		desc: "float32 nan",
		input: &pb3.Scalars{
			SFloat: float32(math.NaN()),
		},
		want: "s_float: nan\n",
	}, {
		desc: "float32 positive infinity",
		input: &pb3.Scalars{
			SFloat: float32(math.Inf(1)),
		},
		want: "s_float: inf\n",
	}, {
		desc: "float32 negative infinity",
		input: &pb3.Scalars{
			SFloat: float32(math.Inf(-1)),
		},
		want: "s_float: -inf\n",
	}, {
		desc: "float64 nan",
		input: &pb3.Scalars{
			SDouble: math.NaN(),
		},
		want: "s_double: nan\n",
	}, {
		desc: "float64 positive infinity",
		input: &pb3.Scalars{
			SDouble: math.Inf(1),
		},
		want: "s_double: inf\n",
	}, {
		desc: "float64 negative infinity",
		input: &pb3.Scalars{
			SDouble: math.Inf(-1),
		},
		want: "s_double: -inf\n",
	}, {
		desc: "proto2 bytes set to empty string",
		input: &pb2.Scalars{
			OptBytes: []byte(""),
		},
		want: "opt_bytes: \"\"\n",
	}, {
		desc: "proto3 bytes set to empty string",
		input: &pb3.Scalars{
			SBytes: []byte(""),
		},
		want: "\n",
	}, {
		desc:  "proto2 enum not set",
		input: &pb2.Enums{},
		want:  "\n",
	}, {
		desc: "proto2 enum set to zero value",
		input: &pb2.Enums{
			OptEnum:       pb2.Enum_UNKNOWN.Enum(),
			OptNestedEnum: pb2Enums_NestedEnum(0),
		},
		want: `opt_enum: UNKNOWN
opt_nested_enum: 0
`,
	}, {
		desc: "proto2 enum",
		input: &pb2.Enums{
			OptEnum:       pb2.Enum_FIRST.Enum(),
			OptNestedEnum: pb2.Enums_UNO.Enum(),
		},
		want: `opt_enum: FIRST
opt_nested_enum: UNO
`,
	}, {
		desc: "proto2 enum set to numeric values",
		input: &pb2.Enums{
			OptEnum:       pb2Enum(1),
			OptNestedEnum: pb2Enums_NestedEnum(2),
		},
		want: `opt_enum: FIRST
opt_nested_enum: DOS
`,
	}, {
		desc: "proto2 enum set to unnamed numeric values",
		input: &pb2.Enums{
			OptEnum:       pb2Enum(101),
			OptNestedEnum: pb2Enums_NestedEnum(-101),
		},
		want: `opt_enum: 101
opt_nested_enum: -101
`,
	}, {
		desc:  "proto3 enum not set",
		input: &pb3.Enums{},
		want:  "\n",
	}, {
		desc: "proto3 enum set to zero value",
		input: &pb3.Enums{
			SEnum:       pb3.Enum_ZERO,
			SNestedEnum: pb3.Enums_CERO,
		},
		want: "\n",
	}, {
		desc: "proto3 enum",
		input: &pb3.Enums{
			SEnum:       pb3.Enum_ONE,
			SNestedEnum: pb3.Enums_DIEZ,
		},
		want: `s_enum: ONE
s_nested_enum: DIEZ
`,
	}, {
		desc: "proto3 enum set to numeric values",
		input: &pb3.Enums{
			SEnum:       2,
			SNestedEnum: 1,
		},
		want: `s_enum: TWO
s_nested_enum: UNO
`,
	}, {
		desc: "proto3 enum set to unnamed numeric values",
		input: &pb3.Enums{
			SEnum:       -47,
			SNestedEnum: 47,
		},
		want: `s_enum: -47
s_nested_enum: 47
`,
	}, {
		desc:  "proto2 nested message not set",
		input: &pb2.Nests{},
		want:  "\n",
	}, {
		desc: "proto2 nested message set to empty",
		input: &pb2.Nests{
			OptNested: &pb2.Nested{},
			Optgroup:  &pb2.Nests_OptGroup{},
		},
		want: `opt_nested: {}
optgroup: {}
`,
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
		want: `opt_nested: {
  opt_string: "nested message"
  opt_nested: {
    opt_string: "another nested message"
  }
}
`,
	}, {
		desc: "proto2 group fields",
		input: &pb2.Nests{
			Optgroup: &pb2.Nests_OptGroup{
				OptBool:   scalar.Bool(true),
				OptString: scalar.String("inside a group"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("nested message inside a group"),
				},
				Optnestedgroup: &pb2.Nests_OptGroup_OptNestedGroup{
					OptEnum: pb2.Enum_TENTH.Enum(),
				},
			},
		},
		want: `optgroup: {
  opt_bool: true
  opt_string: "inside a group"
  opt_nested: {
    opt_string: "nested message inside a group"
  }
  optnestedgroup: {
    opt_enum: TENTH
  }
}
`,
	}, {
		desc:  "proto3 nested message not set",
		input: &pb3.Nests{},
		want:  "\n",
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
		want: `s_nested: {
  s_string: "nested message"
  s_nested: {
    s_string: "another nested message"
  }
}
`,
	}, {
		desc:  "oneof fields",
		input: &pb2.Oneofs{},
		want:  "\n",
	}, {
		desc: "oneof field set to empty string",
		input: &pb2.Oneofs{
			Union: &pb2.Oneofs_Str{},
		},
		want: "str: \"\"\n",
	}, {
		desc: "oneof field set to string",
		input: &pb2.Oneofs{
			Union: &pb2.Oneofs_Str{
				Str: "hello",
			},
		},
		want: "str: \"hello\"\n",
	}, {
		desc: "oneof field set to empty message",
		input: &pb2.Oneofs{
			Union: &pb2.Oneofs_Msg{
				Msg: &pb2.Nested{},
			},
		},
		want: "msg: {}\n",
	}, {
		desc: "oneof field set to message",
		input: &pb2.Oneofs{
			Union: &pb2.Oneofs_Msg{
				Msg: &pb2.Nested{
					OptString: scalar.String("nested message"),
				},
			},
		},
		want: `msg: {
  opt_string: "nested message"
}
`,
	}, {
		desc:  "repeated not set",
		input: &pb2.Repeats{},
		want:  "\n",
	}, {
		desc: "repeated set to empty slices",
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
		want: "\n",
	}, {
		desc: "repeated set to some values",
		input: &pb2.Repeats{
			RptBool:   []bool{true, false, true, true},
			RptInt32:  []int32{1, 6, 0, 0},
			RptInt64:  []int64{-64, 47},
			RptUint32: []uint32{0xff, 0xffff},
			RptUint64: []uint64{0xdeadbeef},
			// TODO: add float32 examples.
			RptDouble: []float64{math.NaN(), math.Inf(1), math.Inf(-1), 1.23e-308},
			RptString: []string{"hello", "世界"},
			RptBytes: [][]byte{
				[]byte("hello"),
				[]byte("\xe4\xb8\x96\xe7\x95\x8c"),
			},
		},
		want: `rpt_bool: true
rpt_bool: false
rpt_bool: true
rpt_bool: true
rpt_int32: 1
rpt_int32: 6
rpt_int32: 0
rpt_int32: 0
rpt_int64: -64
rpt_int64: 47
rpt_uint32: 255
rpt_uint32: 65535
rpt_uint64: 3735928559
rpt_double: nan
rpt_double: inf
rpt_double: -inf
rpt_double: 1.23e-308
rpt_string: "hello"
rpt_string: "世界"
rpt_bytes: "hello"
rpt_bytes: "世界"
`,
	}, {
		desc: "repeated enum",
		input: &pb2.Enums{
			RptEnum:       []pb2.Enum{pb2.Enum_FIRST, 2, pb2.Enum_TENTH, 42},
			RptNestedEnum: []pb2.Enums_NestedEnum{2, 47, 10},
		},
		want: `rpt_enum: FIRST
rpt_enum: SECOND
rpt_enum: TENTH
rpt_enum: 42
rpt_nested_enum: DOS
rpt_nested_enum: 47
rpt_nested_enum: DIEZ
`,
	}, {
		desc: "repeated nested message set to empty",
		input: &pb2.Nests{
			RptNested: []*pb2.Nested{},
			Rptgroup:  []*pb2.Nests_RptGroup{},
		},
		want: "\n",
	}, {
		desc: "repeated nested messages",
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
		want: `rpt_nested: {
  opt_string: "repeat nested one"
}
rpt_nested: {
  opt_string: "repeat nested two"
  opt_nested: {
    opt_string: "inside repeat nested two"
  }
}
rpt_nested: {}
`,
	}, {
		desc: "repeated group fields",
		input: &pb2.Nests{
			Rptgroup: []*pb2.Nests_RptGroup{
				{
					RptBool: []bool{true, false},
				},
				{},
			},
		},
		want: `rptgroup: {
  rpt_bool: true
  rpt_bool: false
}
rptgroup: {}
`,
	}, {
		desc:  "map fields empty",
		input: &pb2.Maps{},
		want:  "\n",
	}, {
		desc: "map fields set to empty maps",
		input: &pb2.Maps{
			Int32ToStr:     map[int32]string{},
			Sfixed64ToBool: map[int64]bool{},
			BoolToUint32:   map[bool]uint32{},
			Uint64ToEnum:   map[uint64]pb2.Enum{},
			StrToNested:    map[string]*pb2.Nested{},
			StrToOneofs:    map[string]*pb2.Oneofs{},
		},
		want: "\n",
	}, {
		desc: "map fields 1",
		input: &pb2.Maps{
			Int32ToStr: map[int32]string{
				-101: "-101",
				0xff: "0xff",
				0:    "zero",
			},
			Sfixed64ToBool: map[int64]bool{
				0xcafe: true,
				0:      false,
			},
			BoolToUint32: map[bool]uint32{
				true:  42,
				false: 101,
			},
		},
		want: `int32_to_str: {
  key: -101
  value: "-101"
}
int32_to_str: {
  key: 0
  value: "zero"
}
int32_to_str: {
  key: 255
  value: "0xff"
}
sfixed64_to_bool: {
  key: 0
  value: false
}
sfixed64_to_bool: {
  key: 51966
  value: true
}
bool_to_uint32: {
  key: false
  value: 101
}
bool_to_uint32: {
  key: true
  value: 42
}
`,
	}, {
		desc: "map fields 2",
		input: &pb2.Maps{
			Uint64ToEnum: map[uint64]pb2.Enum{
				1:  pb2.Enum_FIRST,
				2:  pb2.Enum_SECOND,
				10: pb2.Enum_TENTH,
			},
		},
		want: `uint64_to_enum: {
  key: 1
  value: FIRST
}
uint64_to_enum: {
  key: 2
  value: SECOND
}
uint64_to_enum: {
  key: 10
  value: TENTH
}
`,
	}, {
		desc: "map fields 3",
		input: &pb2.Maps{
			StrToNested: map[string]*pb2.Nested{
				"nested_one": &pb2.Nested{
					OptString: scalar.String("nested in a map"),
				},
			},
		},
		want: `str_to_nested: {
  key: "nested_one"
  value: {
    opt_string: "nested in a map"
  }
}
`,
	}, {
		desc: "map fields 4",
		input: &pb2.Maps{
			StrToOneofs: map[string]*pb2.Oneofs{
				"string": &pb2.Oneofs{
					Union: &pb2.Oneofs_Str{
						Str: "hello",
					},
				},
				"nested": &pb2.Oneofs{
					Union: &pb2.Oneofs_Msg{
						Msg: &pb2.Nested{
							OptString: scalar.String("nested oneof in map field value"),
						},
					},
				},
			},
		},
		want: `str_to_oneofs: {
  key: "nested"
  value: {
    msg: {
      opt_string: "nested oneof in map field value"
    }
  }
}
str_to_oneofs: {
  key: "string"
  value: {
    str: "hello"
  }
}
`,
	}, {
		desc:    "proto2 required fields not set",
		input:   &pb2.Requireds{},
		want:    "\n",
		wantErr: true,
	}, {
		desc: "proto2 required fields partially set",
		input: &pb2.Requireds{
			ReqBool:     scalar.Bool(false),
			ReqFixed32:  scalar.Uint32(47),
			ReqSfixed64: scalar.Int64(0xbeefcafe),
			ReqDouble:   scalar.Float64(math.NaN()),
			ReqString:   scalar.String("hello"),
			ReqEnum:     pb2.Enum_FIRST.Enum(),
		},
		want: `req_bool: false
req_fixed32: 47
req_sfixed64: 3203386110
req_double: nan
req_string: "hello"
req_enum: FIRST
`,
		wantErr: true,
	}, {
		desc: "proto2 required fields all set",
		input: &pb2.Requireds{
			ReqBool:     scalar.Bool(false),
			ReqFixed32:  scalar.Uint32(0),
			ReqFixed64:  scalar.Uint64(0),
			ReqSfixed32: scalar.Int32(0),
			ReqSfixed64: scalar.Int64(0),
			ReqFloat:    scalar.Float32(0),
			ReqDouble:   scalar.Float64(0),
			ReqString:   scalar.String(""),
			ReqEnum:     pb2.Enum_UNKNOWN.Enum(),
			ReqBytes:    []byte{},
			ReqNested:   &pb2.Nested{},
		},
		want: `req_bool: false
req_fixed32: 0
req_fixed64: 0
req_sfixed32: 0
req_sfixed64: 0
req_float: 0
req_double: 0
req_string: ""
req_bytes: ""
req_enum: UNKNOWN
req_nested: {}
`,
	}, {
		desc: "indirect required field",
		input: &pb2.IndirectRequired{
			OptNested: &pb2.NestedWithRequired{},
		},
		want:    "opt_nested: {}\n",
		wantErr: true,
	}, {
		desc: "indirect required field in empty repeated",
		input: &pb2.IndirectRequired{
			RptNested: []*pb2.NestedWithRequired{},
		},
		want: "\n",
	}, {
		desc: "indirect required field in repeated",
		input: &pb2.IndirectRequired{
			RptNested: []*pb2.NestedWithRequired{
				&pb2.NestedWithRequired{},
			},
		},
		want:    "rpt_nested: {}\n",
		wantErr: true,
	}, {
		desc: "indirect required field in empty map",
		input: &pb2.IndirectRequired{
			StrToNested: map[string]*pb2.NestedWithRequired{},
		},
		want: "\n",
	}, {
		desc: "indirect required field in map",
		input: &pb2.IndirectRequired{
			StrToNested: map[string]*pb2.NestedWithRequired{
				"fail": &pb2.NestedWithRequired{},
			},
		},
		want: `str_to_nested: {
  key: "fail"
  value: {}
}
`,
		wantErr: true,
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			b, err := textpb.Marshal(M(tt.input))
			if err != nil && !tt.wantErr {
				t.Errorf("Marshal() returned error: %v\n\n", err)
			}
			if err == nil && tt.wantErr {
				t.Error("Marshal() got nil error, want error\n\n")
			}
			got := string(b)
			if tt.want != "" && got != tt.want {
				t.Errorf("Marshal()\n<got>\n%v\n<want>\n%v\n", got, tt.want)
				if diff := cmp.Diff(tt.want, got, splitLines); diff != "" {
					t.Errorf("Marshal() diff -want +got\n%v\n", diff)
				}
			}
		})
	}
}
