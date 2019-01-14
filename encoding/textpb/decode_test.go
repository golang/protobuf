// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpb_test

import (
	"math"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoapi"
	"github.com/golang/protobuf/v2/encoding/textpb"
	"github.com/golang/protobuf/v2/internal/legacy"
	"github.com/golang/protobuf/v2/internal/scalar"
	"github.com/golang/protobuf/v2/proto"
	preg "github.com/golang/protobuf/v2/reflect/protoregistry"

	// The legacy package must be imported prior to use of any legacy messages.
	// TODO: Remove this when protoV1 registers these hooks for you.
	_ "github.com/golang/protobuf/v2/internal/legacy"

	"github.com/golang/protobuf/v2/encoding/textpb/testprotos/pb2"
	"github.com/golang/protobuf/v2/encoding/textpb/testprotos/pb3"
)

func init() {
	registerExtension(pb2.E_OptExtBool)
	registerExtension(pb2.E_OptExtString)
	registerExtension(pb2.E_OptExtEnum)
	registerExtension(pb2.E_OptExtNested)
	registerExtension(pb2.E_RptExtFixed32)
	registerExtension(pb2.E_RptExtEnum)
	registerExtension(pb2.E_RptExtNested)
	registerExtension(pb2.E_ExtensionsContainer_OptExtBool)
	registerExtension(pb2.E_ExtensionsContainer_OptExtString)
	registerExtension(pb2.E_ExtensionsContainer_OptExtEnum)
	registerExtension(pb2.E_ExtensionsContainer_OptExtNested)
	registerExtension(pb2.E_ExtensionsContainer_RptExtString)
	registerExtension(pb2.E_ExtensionsContainer_RptExtEnum)
	registerExtension(pb2.E_ExtensionsContainer_RptExtNested)
}

func registerExtension(xd *protoapi.ExtensionDesc) {
	xt := legacy.Export{}.ExtensionTypeFromDesc(xd)
	preg.GlobalTypes.Register(xt)
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		desc         string
		inputMessage proto.Message
		inputText    string
		wantMessage  proto.Message
		wantErr      bool
	}{{
		desc:         "proto2 empty message",
		inputMessage: &pb2.Scalars{},
		wantMessage:  &pb2.Scalars{},
	}, {
		desc:         "proto2 optional scalar fields set to zero values",
		inputMessage: &pb2.Scalars{},
		inputText: `opt_bool: false
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
		wantMessage: &pb2.Scalars{
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
	}, {
		desc:         "proto3 scalar fields set to zero values",
		inputMessage: &pb3.Scalars{},
		inputText: `s_bool: false
s_int32: 0
s_int64: 0
s_uint32: 0
s_uint64: 0
s_sint32: 0
s_sint64: 0
s_fixed32: 0
s_fixed64: 0
s_sfixed32: 0
s_sfixed64: 0
s_float: 0
s_double: 0
s_bytes: ""
s_string: ""
`,
		wantMessage: &pb3.Scalars{},
	}, {
		desc:         "proto2 optional scalar fields",
		inputMessage: &pb2.Scalars{},
		inputText: `opt_bool: true
opt_int32: 255
opt_int64: 3735928559
opt_uint32: 0xff
opt_uint64: 0xdeadbeef
opt_sint32: -1001
opt_sint64: -0xffff
opt_fixed64: 64
opt_sfixed32: -32
opt_float: 1.234
opt_double: 1.23e+100
opt_bytes: "\xe8\xb0\xb7\xe6\xad\x8c"
opt_string: "谷歌"
`,
		wantMessage: &pb2.Scalars{
			OptBool:     scalar.Bool(true),
			OptInt32:    scalar.Int32(0xff),
			OptInt64:    scalar.Int64(0xdeadbeef),
			OptUint32:   scalar.Uint32(0xff),
			OptUint64:   scalar.Uint64(0xdeadbeef),
			OptSint32:   scalar.Int32(-1001),
			OptSint64:   scalar.Int64(-0xffff),
			OptFixed64:  scalar.Uint64(64),
			OptSfixed32: scalar.Int32(-32),
			OptFloat:    scalar.Float32(1.234),
			OptDouble:   scalar.Float64(1.23e100),
			OptBytes:    []byte("\xe8\xb0\xb7\xe6\xad\x8c"),
			OptString:   scalar.String("谷歌"),
		},
	}, {
		desc:         "proto3 scalar fields",
		inputMessage: &pb3.Scalars{},
		inputText: `s_bool: true
s_int32: 255
s_int64: 3735928559
s_uint32: 0xff
s_uint64: 0xdeadbeef
s_sint32: -1001
s_sint64: -0xffff
s_fixed64: 64
s_sfixed32: -32
s_float: 1.234
s_double: 1.23e+100
s_bytes: "\xe8\xb0\xb7\xe6\xad\x8c"
s_string: "谷歌"
`,
		wantMessage: &pb3.Scalars{
			SBool:     true,
			SInt32:    0xff,
			SInt64:    0xdeadbeef,
			SUint32:   0xff,
			SUint64:   0xdeadbeef,
			SSint32:   -1001,
			SSint64:   -0xffff,
			SFixed64:  64,
			SSfixed32: -32,
			SFloat:    1.234,
			SDouble:   1.23e100,
			SBytes:    []byte("\xe8\xb0\xb7\xe6\xad\x8c"),
			SString:   "谷歌",
		},
	}, {
		desc:         "proto2 message contains unknown field",
		inputMessage: &pb2.Scalars{},
		inputText:    "unknown_field: 123",
		wantErr:      true,
	}, {
		desc:         "proto3 message contains unknown field",
		inputMessage: &pb3.Scalars{},
		inputText:    "unknown_field: 456",
		wantErr:      true,
	}, {
		desc:         "proto2 numeric key field",
		inputMessage: &pb2.Scalars{},
		inputText:    "1: true",
		wantErr:      true,
	}, {
		desc:         "proto3 numeric key field",
		inputMessage: &pb3.Scalars{},
		inputText:    "1: true",
		wantErr:      true,
	}, {
		desc:         "invalid bool value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_bool: 123",
		wantErr:      true,
	}, {
		desc:         "invalid int32 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_int32: not_a_num",
		wantErr:      true,
	}, {
		desc:         "invalid int64 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_int64: 'not a num either'",
		wantErr:      true,
	}, {
		desc:         "invalid uint32 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_fixed32: -42",
		wantErr:      true,
	}, {
		desc:         "invalid uint64 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_uint64: -47",
		wantErr:      true,
	}, {
		desc:         "invalid sint32 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_sint32: '42'",
		wantErr:      true,
	}, {
		desc:         "invalid sint64 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_sint64: '-47'",
		wantErr:      true,
	}, {
		desc:         "invalid fixed32 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_fixed32: -42",
		wantErr:      true,
	}, {
		desc:         "invalid fixed64 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_fixed64: -42",
		wantErr:      true,
	}, {
		desc:         "invalid sfixed32 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_sfixed32: 'not valid'",
		wantErr:      true,
	}, {
		desc:         "invalid sfixed64 value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_sfixed64: bad",
		wantErr:      true,
	}, {
		desc:         "float32 positive infinity",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_float: inf",
		wantMessage: &pb3.Scalars{
			SFloat: float32(math.Inf(1)),
		},
	}, {
		desc:         "float32 negative infinity",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_float: -inf",
		wantMessage: &pb3.Scalars{
			SFloat: float32(math.Inf(-1)),
		},
	}, {
		desc:         "float64 positive infinity",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_double: inf",
		wantMessage: &pb3.Scalars{
			SDouble: math.Inf(1),
		},
	}, {
		desc:         "float64 negative infinity",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_double: -inf",
		wantMessage: &pb3.Scalars{
			SDouble: math.Inf(-1),
		},
	}, {
		desc:         "invalid string value",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_string: invalid_string",
		wantErr:      true,
	}, {
		desc:         "proto2 bytes set to empty string",
		inputMessage: &pb2.Scalars{},
		inputText:    "opt_bytes: ''",
		wantMessage: &pb2.Scalars{
			OptBytes: []byte(""),
		},
	}, {
		desc:         "proto3 bytes set to empty string",
		inputMessage: &pb3.Scalars{},
		inputText:    "s_bytes: ''",
		wantMessage:  &pb3.Scalars{},
	}, {
		desc:         "proto2 duplicate singular field",
		inputMessage: &pb2.Scalars{},
		inputText: `
opt_bool: true
opt_bool: false
`,
		wantErr: true,
	}, {
		desc:         "proto2 invalid singular field",
		inputMessage: &pb2.Scalars{},
		inputText: `
opt_bool: [true, false]
`,
		wantErr: true,
	}, {
		desc:         "proto2 more duplicate singular field",
		inputMessage: &pb2.Scalars{},
		inputText: `
opt_bool: true
opt_string: "hello"
opt_bool: false
`,
		wantErr: true,
	}, {
		desc:         "proto3 duplicate singular field",
		inputMessage: &pb3.Scalars{},
		inputText: `
s_bool: false
s_bool: true
`,
		wantErr: true,
	}, {
		desc:         "proto3 more duplicate singular field",
		inputMessage: &pb3.Scalars{},
		inputText: `
s_bool: false
s_string: ""
s_bool: true
`,
		wantErr: true,
	}, {
		desc:         "proto2 enum",
		inputMessage: &pb2.Enums{},
		inputText: `
opt_enum: FIRST
opt_nested_enum: UNO
`,
		wantMessage: &pb2.Enums{
			OptEnum:       pb2.Enum_FIRST.Enum(),
			OptNestedEnum: pb2.Enums_UNO.Enum(),
		},
	}, {
		desc:         "proto2 enum set to numeric values",
		inputMessage: &pb2.Enums{},
		inputText: `
opt_enum: 1
opt_nested_enum: 2
`,
		wantMessage: &pb2.Enums{
			OptEnum:       pb2.Enum_FIRST.Enum(),
			OptNestedEnum: pb2.Enums_DOS.Enum(),
		},
	}, {
		desc:         "proto2 enum set to unnamed numeric values",
		inputMessage: &pb2.Enums{},
		inputText: `
opt_enum: 101
opt_nested_enum: -101
`,
		wantMessage: &pb2.Enums{
			OptEnum:       pb2Enum(101),
			OptNestedEnum: pb2Enums_NestedEnum(-101),
		},
	}, {
		desc:         "proto2 enum set to invalid named",
		inputMessage: &pb2.Enums{},
		inputText: `
opt_enum: UNNAMED 
opt_nested_enum: UNNAMED_TOO
`,
		wantErr: true,
	}, {
		desc:         "proto3 enum name value",
		inputMessage: &pb3.Enums{},
		inputText: `
s_enum: ONE
s_nested_enum: DIEZ
`,
		wantMessage: &pb3.Enums{
			SEnum:       pb3.Enum_ONE,
			SNestedEnum: pb3.Enums_DIEZ,
		},
	}, {
		desc:         "proto3 enum numeric value",
		inputMessage: &pb3.Enums{},
		inputText: `
s_enum: 2
s_nested_enum: 1
`,
		wantMessage: &pb3.Enums{
			SEnum:       pb3.Enum_TWO,
			SNestedEnum: pb3.Enums_UNO,
		},
	}, {
		desc:         "proto3 enum unnamed numeric value",
		inputMessage: &pb3.Enums{},
		inputText: `
s_enum: 0x7fffffff
s_nested_enum: -0x80000000
`,
		wantMessage: &pb3.Enums{
			SEnum:       0x7fffffff,
			SNestedEnum: -0x80000000,
		},
	}, {
		desc:         "proto2 nested empty messages",
		inputMessage: &pb2.Nests{},
		inputText: `
opt_nested: {}
OptGroup: {}
`,
		wantMessage: &pb2.Nests{
			OptNested: &pb2.Nested{},
			Optgroup:  &pb2.Nests_OptGroup{},
		},
	}, {
		desc:         "proto2 nested messages",
		inputMessage: &pb2.Nests{},
		inputText: `
opt_nested: {
  opt_string: "nested message"
  opt_nested: {
    opt_string: "another nested message"
  }
}
`,
		wantMessage: &pb2.Nests{
			OptNested: &pb2.Nested{
				OptString: scalar.String("nested message"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("another nested message"),
				},
			},
		},
	}, {
		desc:         "proto3 nested empty message",
		inputMessage: &pb3.Nests{},
		inputText:    "s_nested: {}",
		wantMessage: &pb3.Nests{
			SNested: &pb3.Nested{},
		},
	}, {
		desc:         "proto3 nested message",
		inputMessage: &pb3.Nests{},
		inputText: `
s_nested: {
  s_string: "nested message"
  s_nested: {
    s_string: "another nested message"
  }
}
`,
		wantMessage: &pb3.Nests{
			SNested: &pb3.Nested{
				SString: "nested message",
				SNested: &pb3.Nested{
					SString: "another nested message",
				},
			},
		},
	}, {
		desc:         "oneof field set to empty string",
		inputMessage: &pb2.Oneofs{},
		inputText:    "str: ''",
		wantMessage: &pb2.Oneofs{
			Union: &pb2.Oneofs_Str{},
		},
	}, {
		desc:         "oneof field set to string",
		inputMessage: &pb2.Oneofs{},
		inputText:    "str: 'hello'",
		wantMessage: &pb2.Oneofs{
			Union: &pb2.Oneofs_Str{
				Str: "hello",
			},
		},
	}, {
		desc:         "oneof field set to empty message",
		inputMessage: &pb2.Oneofs{},
		inputText:    "msg: {}",
		wantMessage: &pb2.Oneofs{
			Union: &pb2.Oneofs_Msg{
				Msg: &pb2.Nested{},
			},
		},
	}, {
		desc:         "oneof field set to message",
		inputMessage: &pb2.Oneofs{},
		inputText: `
msg: {
  opt_string: "nested message"
}
`,
		wantMessage: &pb2.Oneofs{
			Union: &pb2.Oneofs_Msg{
				Msg: &pb2.Nested{
					OptString: scalar.String("nested message"),
				},
			},
		},
	}, {
		desc:         "repeated scalar using same field name",
		inputMessage: &pb2.Repeats{},
		inputText: `
rpt_string: "a"
rpt_string: "b"
rpt_int32: 0xff
rpt_float: 1.23
rpt_bytes: "bytes"
`,
		wantMessage: &pb2.Repeats{
			RptString: []string{"a", "b"},
			RptInt32:  []int32{0xff},
			RptFloat:  []float32{1.23},
			RptBytes:  [][]byte{[]byte("bytes")},
		},
	}, {
		desc:         "repeated using mix of [] and repeated field name",
		inputMessage: &pb2.Repeats{},
		inputText: `
rpt_string: "a"
rpt_bool: true
rpt_string: ["x", "y"]
rpt_bool: [ false, true ]
rpt_string: "b"
`,
		wantMessage: &pb2.Repeats{
			RptString: []string{"a", "x", "y", "b"},
			RptBool:   []bool{true, false, true},
		},
	}, {
		desc:         "repeated enums",
		inputMessage: &pb2.Enums{},
		inputText: `
rpt_enum: TENTH
rpt_enum: 1
rpt_nested_enum: [DOS, 2]
rpt_enum: 42
rpt_nested_enum: -47
`,
		wantMessage: &pb2.Enums{
			RptEnum:       []pb2.Enum{pb2.Enum_TENTH, pb2.Enum_FIRST, 42},
			RptNestedEnum: []pb2.Enums_NestedEnum{pb2.Enums_DOS, pb2.Enums_DOS, -47},
		},
	}, {
		desc:         "repeated nested messages",
		inputMessage: &pb2.Nests{},
		inputText: `
rpt_nested: {
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
		wantMessage: &pb2.Nests{
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
	}, {
		desc:         "repeated group fields",
		inputMessage: &pb2.Nests{},
		inputText: `
RptGroup: {
  rpt_bool: true
  rpt_bool: false
}
RptGroup: {}
`,
		wantMessage: &pb2.Nests{
			Rptgroup: []*pb2.Nests_RptGroup{
				{
					RptBool: []bool{true, false},
				},
				{},
			},
		},
	}, {
		desc:         "map fields 1",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: -101
  value: "-101"
}
int32_to_str: {
  key: 0
  value: "zero"
}
sfixed64_to_bool: {
  key: 0
  value: false
}
int32_to_str: {
  key: 255
  value: "0xff"
}
bool_to_uint32: {
  key: false
  value: 101
}
sfixed64_to_bool: {
  key: 51966
  value: true
}
bool_to_uint32: {
  key: true
  value: 42
}
`,
		wantMessage: &pb2.Maps{
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
	}, {
		desc:         "map fields 2",
		inputMessage: &pb2.Maps{},
		inputText: `
uint64_to_enum: {
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
		wantMessage: &pb2.Maps{
			Uint64ToEnum: map[uint64]pb2.Enum{
				1:  pb2.Enum_FIRST,
				2:  pb2.Enum_SECOND,
				10: pb2.Enum_TENTH,
			},
		},
	}, {
		desc:         "map fields 3",
		inputMessage: &pb2.Maps{},
		inputText: `
str_to_nested: {
  key: "nested_one"
  value: {
    opt_string: "nested in a map"
  }
}
`,
		wantMessage: &pb2.Maps{
			StrToNested: map[string]*pb2.Nested{
				"nested_one": &pb2.Nested{
					OptString: scalar.String("nested in a map"),
				},
			},
		},
	}, {
		desc:         "map fields 4",
		inputMessage: &pb2.Maps{},
		inputText: `
str_to_oneofs: {
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
		wantMessage: &pb2.Maps{
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
	}, {
		desc:         "map contains duplicate keys",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: 0
  value: "cero"
}
int32_to_str: {
  key: 0
  value: "zero"
}
`,
		wantMessage: &pb2.Maps{
			Int32ToStr: map[int32]string{
				0: "zero",
			},
		},
	}, {
		desc:         "map contains duplicate key fields",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: 0
  key: 1
  value: "cero"
}
`,
		wantErr: true,
	}, {
		desc:         "map contains duplicate value fields",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: 1
  value: "cero"
  value: "uno"
}
`,
		wantErr: true,
	}, {
		desc:         "map contains missing key",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  value: "zero"
}
`,
		wantMessage: &pb2.Maps{
			Int32ToStr: map[int32]string{
				0: "zero",
			},
		},
	}, {
		desc:         "map contains missing value",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: 100
}
`,
		wantMessage: &pb2.Maps{
			Int32ToStr: map[int32]string{
				100: "",
			},
		},
	}, {
		desc:         "map contains missing key and value",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {}
`,
		wantMessage: &pb2.Maps{
			Int32ToStr: map[int32]string{
				0: "",
			},
		},
	}, {
		desc:         "map contains unknown field",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: 0
  value: "cero"
  unknown: "bad"
}
`,
		wantErr: true,
	}, {
		desc:         "map contains extension-like key field",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  [key]: 10
  value: "ten"
}
`,
		wantErr: true,
	}, {
		desc:         "map contains invalid key",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: "invalid"
  value: "cero"
}
`,
		wantErr: true,
	}, {
		desc:         "map contains invalid value",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: 100
  value: 101
}
`,
		wantErr: true,
	}, {
		desc:         "map using mix of [] and repeated",
		inputMessage: &pb2.Maps{},
		inputText: `
int32_to_str: {
  key: 1
  value: "one"
}
int32_to_str: [
  {
    key: 2
    value: "not this"
  },
  {
  },
  {
    key: 3
    value: "three"
  }
]
int32_to_str: {
  key: 2
  value: "two"
}
`,
		wantMessage: &pb2.Maps{
			Int32ToStr: map[int32]string{
				0: "",
				1: "one",
				2: "two",
				3: "three",
			},
		},
	}, {
		desc:         "proto2 required fields not set",
		inputMessage: &pb2.Requireds{},
		wantErr:      true,
	}, {
		desc:         "proto2 required field set but not optional",
		inputMessage: &pb2.PartialRequired{},
		inputText:    "req_string: 'this is required'",
		wantMessage: &pb2.PartialRequired{
			ReqString: scalar.String("this is required"),
		},
	}, {
		desc:         "proto2 required fields partially set",
		inputMessage: &pb2.Requireds{},
		inputText: `
req_bool: false
req_fixed32: 47
req_sfixed64: 3203386110
req_string: "hello"
req_enum: FIRST
`,
		wantMessage: &pb2.Requireds{
			ReqBool:     scalar.Bool(false),
			ReqFixed32:  scalar.Uint32(47),
			ReqSfixed64: scalar.Int64(0xbeefcafe),
			ReqString:   scalar.String("hello"),
			ReqEnum:     pb2.Enum_FIRST.Enum(),
		},
		wantErr: true,
	}, {
		desc:         "proto2 required fields all set",
		inputMessage: &pb2.Requireds{},
		inputText: `
req_bool: false
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
		wantMessage: &pb2.Requireds{
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
	}, {
		desc:         "indirect required field",
		inputMessage: &pb2.IndirectRequired{},
		inputText:    "opt_nested: {}",
		wantMessage: &pb2.IndirectRequired{
			OptNested: &pb2.NestedWithRequired{},
		},
		wantErr: true,
	}, {
		desc:         "indirect required field in repeated",
		inputMessage: &pb2.IndirectRequired{},
		inputText: `
rpt_nested: {
  req_string: "one"
}
rpt_nested: {}
rpt_nested: {
  req_string: "three"
}
`,
		wantMessage: &pb2.IndirectRequired{
			RptNested: []*pb2.NestedWithRequired{
				{
					ReqString: scalar.String("one"),
				},
				{},
				{
					ReqString: scalar.String("three"),
				},
			},
		},
		wantErr: true,
	}, {
		desc:         "indirect required field in map",
		inputMessage: &pb2.IndirectRequired{},
		inputText: `
str_to_nested: {
  key: "missing"
}
str_to_nested: {
  key: "contains"
  value: {
    req_string: "here"
  }
}
`,
		wantMessage: &pb2.IndirectRequired{
			StrToNested: map[string]*pb2.NestedWithRequired{
				"missing": &pb2.NestedWithRequired{},
				"contains": &pb2.NestedWithRequired{
					ReqString: scalar.String("here"),
				},
			},
		},
		wantErr: true,
	}, {
		desc:         "ignore reserved field",
		inputMessage: &pb2.Nests{},
		inputText:    "reserved_field: 'ignore this'",
		wantMessage:  &pb2.Nests{},
	}, {
		desc:         "extensions of non-repeated fields",
		inputMessage: &pb2.Extensions{},
		inputText: `opt_string: "non-extension field"
[pb2.opt_ext_bool]: true
opt_bool: true
[pb2.opt_ext_nested]: {
  opt_string: "nested in an extension"
  opt_nested: {
    opt_string: "another nested in an extension"
  }
}
[pb2.opt_ext_string]: "extension field"
opt_int32: 42
[pb2.opt_ext_enum]: TENTH
`,
		wantMessage: func() proto.Message {
			m := &pb2.Extensions{
				OptString: scalar.String("non-extension field"),
				OptBool:   scalar.Bool(true),
				OptInt32:  scalar.Int32(42),
			}
			setExtension(m, pb2.E_OptExtBool, true)
			setExtension(m, pb2.E_OptExtString, "extension field")
			setExtension(m, pb2.E_OptExtEnum, pb2.Enum_TENTH)
			setExtension(m, pb2.E_OptExtNested, &pb2.Nested{
				OptString: scalar.String("nested in an extension"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("another nested in an extension"),
				},
			})
			return m
		}(),
	}, {
		desc:         "extensions of repeated fields",
		inputMessage: &pb2.Extensions{},
		inputText: `[pb2.rpt_ext_enum]: TENTH
[pb2.rpt_ext_enum]: 101
[pb2.rpt_ext_fixed32]: 42
[pb2.rpt_ext_enum]: FIRST
[pb2.rpt_ext_nested]: {
  opt_string: "one"
}
[pb2.rpt_ext_nested]: {
  opt_string: "two"
}
[pb2.rpt_ext_fixed32]: 47
[pb2.rpt_ext_nested]: {
  opt_string: "three"
}
`,
		wantMessage: func() proto.Message {
			m := &pb2.Extensions{}
			setExtension(m, pb2.E_RptExtEnum, &[]pb2.Enum{pb2.Enum_TENTH, 101, pb2.Enum_FIRST})
			setExtension(m, pb2.E_RptExtFixed32, &[]uint32{42, 47})
			setExtension(m, pb2.E_RptExtNested, &[]*pb2.Nested{
				&pb2.Nested{OptString: scalar.String("one")},
				&pb2.Nested{OptString: scalar.String("two")},
				&pb2.Nested{OptString: scalar.String("three")},
			})
			return m
		}(),
	}, {
		desc:         "extensions of non-repeated fields in another message",
		inputMessage: &pb2.Extensions{},
		inputText: `[pb2.ExtensionsContainer.opt_ext_bool]: true
[pb2.ExtensionsContainer.opt_ext_enum]: TENTH
[pb2.ExtensionsContainer.opt_ext_nested]: {
  opt_string: "nested in an extension"
  opt_nested: {
    opt_string: "another nested in an extension"
  }
}
[pb2.ExtensionsContainer.opt_ext_string]: "extension field"
`,
		wantMessage: func() proto.Message {
			m := &pb2.Extensions{}
			setExtension(m, pb2.E_ExtensionsContainer_OptExtBool, true)
			setExtension(m, pb2.E_ExtensionsContainer_OptExtString, "extension field")
			setExtension(m, pb2.E_ExtensionsContainer_OptExtEnum, pb2.Enum_TENTH)
			setExtension(m, pb2.E_ExtensionsContainer_OptExtNested, &pb2.Nested{
				OptString: scalar.String("nested in an extension"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("another nested in an extension"),
				},
			})
			return m
		}(),
	}, {
		desc:         "extensions of repeated fields in another message",
		inputMessage: &pb2.Extensions{},
		inputText: `opt_string: "non-extension field"
opt_bool: true
opt_int32: 42
[pb2.ExtensionsContainer.rpt_ext_nested]: {
  opt_string: "one"
}
[pb2.ExtensionsContainer.rpt_ext_enum]: TENTH
[pb2.ExtensionsContainer.rpt_ext_nested]: {
  opt_string: "two"
}
[pb2.ExtensionsContainer.rpt_ext_enum]: 101
[pb2.ExtensionsContainer.rpt_ext_string]: "hello"
[pb2.ExtensionsContainer.rpt_ext_enum]: FIRST
[pb2.ExtensionsContainer.rpt_ext_nested]: {
  opt_string: "three"
}
[pb2.ExtensionsContainer.rpt_ext_string]: "world"
`,
		wantMessage: func() proto.Message {
			m := &pb2.Extensions{
				OptString: scalar.String("non-extension field"),
				OptBool:   scalar.Bool(true),
				OptInt32:  scalar.Int32(42),
			}
			setExtension(m, pb2.E_ExtensionsContainer_RptExtEnum, &[]pb2.Enum{pb2.Enum_TENTH, 101, pb2.Enum_FIRST})
			setExtension(m, pb2.E_ExtensionsContainer_RptExtString, &[]string{"hello", "world"})
			setExtension(m, pb2.E_ExtensionsContainer_RptExtNested, &[]*pb2.Nested{
				&pb2.Nested{OptString: scalar.String("one")},
				&pb2.Nested{OptString: scalar.String("two")},
				&pb2.Nested{OptString: scalar.String("three")},
			})
			return m
		}(),
	}, {
		desc:         "invalid extension field name",
		inputMessage: &pb2.Extensions{},
		inputText:    "[pb2.invalid_message_field]: true",
		wantErr:      true,
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			err := textpb.Unmarshal(tt.inputMessage, []byte(tt.inputText))
			if err != nil && !tt.wantErr {
				t.Errorf("Unmarshal() returned error: %v\n\n", err)
			}
			if err == nil && tt.wantErr {
				t.Error("Unmarshal() got nil error, want error\n\n")
			}
			if tt.wantMessage != nil && !protoV1.Equal(tt.inputMessage.(protoV1.Message), tt.wantMessage.(protoV1.Message)) {
				t.Errorf("Unmarshal()\n<got>\n%v\n<want>\n%v\n", tt.inputMessage, tt.wantMessage)
			}
		})
	}
}
