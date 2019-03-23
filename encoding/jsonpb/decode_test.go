// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonpb_test

import (
	"math"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/encoding/jsonpb"
	"github.com/golang/protobuf/v2/encoding/testprotos/pb2"
	"github.com/golang/protobuf/v2/encoding/testprotos/pb3"
	"github.com/golang/protobuf/v2/internal/scalar"
	"github.com/golang/protobuf/v2/proto"
	preg "github.com/golang/protobuf/v2/reflect/protoregistry"
	"github.com/golang/protobuf/v2/runtime/protoiface"

	knownpb "github.com/golang/protobuf/v2/types/known"
)

func init() {
	// TODO: remove these registerExtension calls when generated code registers
	// to V2 global registry.
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
	registerExtension(pb2.E_MessageSetExtension)
	registerExtension(pb2.E_MessageSetExtension_MessageSetExtension)
	registerExtension(pb2.E_MessageSetExtension_NotMessageSetExtension)
	registerExtension(pb2.E_MessageSetExtension_ExtNested)
	registerExtension(pb2.E_FakeMessageSetExtension_MessageSetExtension)
}

func registerExtension(xd *protoiface.ExtensionDescV1) {
	preg.GlobalTypes.Register(xd.Type)
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		desc         string
		umo          jsonpb.UnmarshalOptions
		inputMessage proto.Message
		inputText    string
		wantMessage  proto.Message
		// TODO: verify expected error message substring.
		wantErr bool
	}{{
		desc:         "proto2 empty message",
		inputMessage: &pb2.Scalars{},
		inputText:    "{}",
		wantMessage:  &pb2.Scalars{},
	}, {
		desc:         "unexpected value instead of EOF",
		inputMessage: &pb2.Scalars{},
		inputText:    "{} {}",
		wantErr:      true,
	}, {
		desc:         "proto2 optional scalars set to zero values",
		inputMessage: &pb2.Scalars{},
		inputText: `{
  "optBool": false,
  "optInt32": 0,
  "optInt64": 0,
  "optUint32": 0,
  "optUint64": 0,
  "optSint32": 0,
  "optSint64": 0,
  "optFixed32": 0,
  "optFixed64": 0,
  "optSfixed32": 0,
  "optSfixed64": 0,
  "optFloat": 0,
  "optDouble": 0,
  "optBytes": "",
  "optString": ""
}`,
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
		desc:         "proto3 scalars set to zero values",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sBool": false,
  "sInt32": 0,
  "sInt64": 0,
  "sUint32": 0,
  "sUint64": 0,
  "sSint32": 0,
  "sSint64": 0,
  "sFixed32": 0,
  "sFixed64": 0,
  "sSfixed32": 0,
  "sSfixed64": 0,
  "sFloat": 0,
  "sDouble": 0,
  "sBytes": "",
  "sString": ""
}`,
		wantMessage: &pb3.Scalars{},
	}, {
		desc:         "proto2 optional scalars set to null",
		inputMessage: &pb2.Scalars{},
		inputText: `{
  "optBool": null,
  "optInt32": null,
  "optInt64": null,
  "optUint32": null,
  "optUint64": null,
  "optSint32": null,
  "optSint64": null,
  "optFixed32": null,
  "optFixed64": null,
  "optSfixed32": null,
  "optSfixed64": null,
  "optFloat": null,
  "optDouble": null,
  "optBytes": null,
  "optString": null
}`,
		wantMessage: &pb2.Scalars{},
	}, {
		desc:         "proto3 scalars set to null",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sBool": null,
  "sInt32": null,
  "sInt64": null,
  "sUint32": null,
  "sUint64": null,
  "sSint32": null,
  "sSint64": null,
  "sFixed32": null,
  "sFixed64": null,
  "sSfixed32": null,
  "sSfixed64": null,
  "sFloat": null,
  "sDouble": null,
  "sBytes": null,
  "sString": null
}`,
		wantMessage: &pb3.Scalars{},
	}, {
		desc:         "boolean",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sBool": true}`,
		wantMessage: &pb3.Scalars{
			SBool: true,
		},
	}, {
		desc:         "not boolean",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sBool": "true"}`,
		wantErr:      true,
	}, {
		desc:         "float and double",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sFloat": 1.234,
  "sDouble": 5.678
}`,
		wantMessage: &pb3.Scalars{
			SFloat:  1.234,
			SDouble: 5.678,
		},
	}, {
		desc:         "float and double in string",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sFloat": "1.234",
  "sDouble": "5.678"
}`,
		wantMessage: &pb3.Scalars{
			SFloat:  1.234,
			SDouble: 5.678,
		},
	}, {
		desc:         "float and double in E notation",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sFloat": 12.34E-1,
  "sDouble": 5.678e4
}`,
		wantMessage: &pb3.Scalars{
			SFloat:  1.234,
			SDouble: 56780,
		},
	}, {
		desc:         "float and double in string E notation",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sFloat": "12.34E-1",
  "sDouble": "5.678e4"
}`,
		wantMessage: &pb3.Scalars{
			SFloat:  1.234,
			SDouble: 56780,
		},
	}, {
		desc:         "float exceeds limit",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sFloat": 3.4e39}`,
		wantErr:      true,
	}, {
		desc:         "float in string exceeds limit",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sFloat": "-3.4e39"}`,
		wantErr:      true,
	}, {
		desc:         "double exceeds limit",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sFloat": -1.79e+309}`,
		wantErr:      true,
	}, {
		desc:         "double in string exceeds limit",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sFloat": "1.79e+309"}`,
		wantErr:      true,
	}, {
		desc:         "infinites",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sFloat": "Infinity", "sDouble": "-Infinity"}`,
		wantMessage: &pb3.Scalars{
			SFloat:  float32(math.Inf(+1)),
			SDouble: math.Inf(-1),
		},
	}, {
		desc:         "not float",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sFloat": true}`,
		wantErr:      true,
	}, {
		desc:         "not double",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sDouble": "not a number"}`,
		wantErr:      true,
	}, {
		desc:         "integers",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sInt32": 1234,
  "sInt64": -1234,
  "sUint32": 1e2,
  "sUint64": 100E-2,
  "sSint32": 1.0,
  "sSint64": -1.0,
  "sFixed32": 1.234e+5,
  "sFixed64": 1200E-2,
  "sSfixed32": -1.234e05,
  "sSfixed64": -1200e-02
}`,
		wantMessage: &pb3.Scalars{
			SInt32:    1234,
			SInt64:    -1234,
			SUint32:   100,
			SUint64:   1,
			SSint32:   1,
			SSint64:   -1,
			SFixed32:  123400,
			SFixed64:  12,
			SSfixed32: -123400,
			SSfixed64: -12,
		},
	}, {
		desc:         "integers in string",
		inputMessage: &pb3.Scalars{},
		inputText: `{
  "sInt32": "1234",
  "sInt64": "-1234",
  "sUint32": "1e2",
  "sUint64": "100E-2",
  "sSint32": "1.0",
  "sSint64": "-1.0",
  "sFixed32": "1.234e+5",
  "sFixed64": "1200E-2",
  "sSfixed32": "-1.234e05",
  "sSfixed64": "-1200e-02"
}`,
		wantMessage: &pb3.Scalars{
			SInt32:    1234,
			SInt64:    -1234,
			SUint32:   100,
			SUint64:   1,
			SSint32:   1,
			SSint64:   -1,
			SFixed32:  123400,
			SFixed64:  12,
			SSfixed32: -123400,
			SSfixed64: -12,
		},
	}, {
		desc:         "integers in escaped string",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sInt32": "\u0031\u0032"}`,
		wantMessage: &pb3.Scalars{
			SInt32: 12,
		},
	}, {
		desc:         "number is not an integer",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sInt32": 1.001}`,
		wantErr:      true,
	}, {
		desc:         "32-bit int exceeds limit",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sInt32": 2e10}`,
		wantErr:      true,
	}, {
		desc:         "64-bit int exceeds limit",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sSfixed64": -9e19}`,
		wantErr:      true,
	}, {
		desc:         "not integer",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sInt32": "not a number"}`,
		wantErr:      true,
	}, {
		desc:         "not unsigned integer",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sUint32": "not a number"}`,
		wantErr:      true,
	}, {
		desc:         "number is not an unsigned integer",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sUint32": -1}`,
		wantErr:      true,
	}, {
		desc:         "string",
		inputMessage: &pb2.Scalars{},
		inputText:    `{"optString": "谷歌"}`,
		wantMessage: &pb2.Scalars{
			OptString: scalar.String("谷歌"),
		},
	}, {
		desc:         "string with invalid UTF-8",
		inputMessage: &pb3.Scalars{},
		inputText:    "{\"sString\": \"\xff\"}",
		wantMessage: &pb3.Scalars{
			SString: "\xff",
		},
		wantErr: true,
	}, {
		desc:         "not string",
		inputMessage: &pb2.Scalars{},
		inputText:    `{"optString": 42}`,
		wantErr:      true,
	}, {
		desc:         "bytes",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sBytes": "aGVsbG8gd29ybGQ"}`,
		wantMessage: &pb3.Scalars{
			SBytes: []byte("hello world"),
		},
	}, {
		desc:         "bytes padded",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sBytes": "aGVsbG8gd29ybGQ="}`,
		wantMessage: &pb3.Scalars{
			SBytes: []byte("hello world"),
		},
	}, {
		desc:         "not bytes",
		inputMessage: &pb3.Scalars{},
		inputText:    `{"sBytes": true}`,
		wantErr:      true,
	}, {
		desc:         "proto2 enum",
		inputMessage: &pb2.Enums{},
		inputText: `{
  "optEnum": "ONE",
  "optNestedEnum": "UNO"
}`,
		wantMessage: &pb2.Enums{
			OptEnum:       pb2.Enum_ONE.Enum(),
			OptNestedEnum: pb2.Enums_UNO.Enum(),
		},
	}, {
		desc:         "proto3 enum",
		inputMessage: &pb3.Enums{},
		inputText: `{
  "sEnum": "ONE",
  "sNestedEnum": "DIEZ"
}`,
		wantMessage: &pb3.Enums{
			SEnum:       pb3.Enum_ONE,
			SNestedEnum: pb3.Enums_DIEZ,
		},
	}, {
		desc:         "enum numeric value",
		inputMessage: &pb3.Enums{},
		inputText: `{
  "sEnum": 2,
  "sNestedEnum": 2
}`,
		wantMessage: &pb3.Enums{
			SEnum:       pb3.Enum_TWO,
			SNestedEnum: pb3.Enums_DOS,
		},
	}, {
		desc:         "enum unnamed numeric value",
		inputMessage: &pb3.Enums{},
		inputText: `{
  "sEnum": 101,
  "sNestedEnum": -101
}`,
		wantMessage: &pb3.Enums{
			SEnum:       101,
			SNestedEnum: -101,
		},
	}, {
		desc:         "enum set to number string",
		inputMessage: &pb3.Enums{},
		inputText: `{
  "sEnum": "1",
}`,
		wantErr: true,
	}, {
		desc:         "enum set to invalid named",
		inputMessage: &pb3.Enums{},
		inputText: `{
  "sEnum": "UNNAMED",
}`,
		wantErr: true,
	}, {
		desc:         "enum set to not enum",
		inputMessage: &pb3.Enums{},
		inputText: `{
  "sEnum": true,
}`,
		wantErr: true,
	}, {
		desc:         "proto name",
		inputMessage: &pb3.JSONNames{},
		inputText: `{
  "s_string": "proto name used"
}`,
		wantMessage: &pb3.JSONNames{
			SString: "proto name used",
		},
	}, {
		desc:         "json_name",
		inputMessage: &pb3.JSONNames{},
		inputText: `{
  "foo_bar": "json_name used"
}`,
		wantMessage: &pb3.JSONNames{
			SString: "json_name used",
		},
	}, {
		desc:         "camelCase name",
		inputMessage: &pb3.JSONNames{},
		inputText: `{
  "sString": "camelcase used"
}`,
		wantErr: true,
	}, {
		desc:         "proto name and json_name",
		inputMessage: &pb3.JSONNames{},
		inputText: `{
  "foo_bar": "json_name used",
  "s_string": "proto name used"
}`,
		wantErr: true,
	}, {
		desc:         "duplicate field names",
		inputMessage: &pb3.JSONNames{},
		inputText: `{
  "foo_bar": "one",
  "foo_bar": "two",
}`,
		wantErr: true,
	}, {
		desc:         "null message",
		inputMessage: &pb2.Nests{},
		inputText:    "null",
		wantErr:      true,
	}, {
		desc:         "proto2 nested message not set",
		inputMessage: &pb2.Nests{},
		inputText:    "{}",
		wantMessage:  &pb2.Nests{},
	}, {
		desc:         "proto2 nested message set to null",
		inputMessage: &pb2.Nests{},
		inputText: `{
  "optNested": null,
  "optgroup": null
}`,
		wantMessage: &pb2.Nests{},
	}, {
		desc:         "proto2 nested message set to empty",
		inputMessage: &pb2.Nests{},
		inputText: `{
  "optNested": {},
  "optgroup": {}
}`,
		wantMessage: &pb2.Nests{
			OptNested: &pb2.Nested{},
			Optgroup:  &pb2.Nests_OptGroup{},
		},
	}, {
		desc:         "proto2 nested messages",
		inputMessage: &pb2.Nests{},
		inputText: `{
  "optNested": {
    "optString": "nested message",
    "optNested": {
      "optString": "another nested message"
    }
  }
}`,
		wantMessage: &pb2.Nests{
			OptNested: &pb2.Nested{
				OptString: scalar.String("nested message"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("another nested message"),
				},
			},
		},
	}, {
		desc:         "proto2 groups",
		inputMessage: &pb2.Nests{},
		inputText: `{
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
		wantMessage: &pb2.Nests{
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
	}, {
		desc:         "proto3 nested message not set",
		inputMessage: &pb3.Nests{},
		inputText:    "{}",
		wantMessage:  &pb3.Nests{},
	}, {
		desc:         "proto3 nested message set to null",
		inputMessage: &pb3.Nests{},
		inputText:    `{"sNested": null}`,
		wantMessage:  &pb3.Nests{},
	}, {
		desc:         "proto3 nested message set to empty",
		inputMessage: &pb3.Nests{},
		inputText:    `{"sNested": {}}`,
		wantMessage: &pb3.Nests{
			SNested: &pb3.Nested{},
		},
	}, {
		desc:         "proto3 nested message",
		inputMessage: &pb3.Nests{},
		inputText: `{
  "sNested": {
    "sString": "nested message",
    "sNested": {
      "sString": "another nested message"
    }
  }
}`,
		wantMessage: &pb3.Nests{
			SNested: &pb3.Nested{
				SString: "nested message",
				SNested: &pb3.Nested{
					SString: "another nested message",
				},
			},
		},
	}, {
		desc:         "message set to non-message",
		inputMessage: &pb3.Nests{},
		inputText:    `"not valid"`,
		wantErr:      true,
	}, {
		desc:         "nested message set to non-message",
		inputMessage: &pb3.Nests{},
		inputText:    `{"sNested": true}`,
		wantErr:      true,
	}, {
		desc:         "oneof not set",
		inputMessage: &pb3.Oneofs{},
		inputText:    "{}",
		wantMessage:  &pb3.Oneofs{},
	}, {
		desc:         "oneof set to empty string",
		inputMessage: &pb3.Oneofs{},
		inputText:    `{"oneofString": ""}`,
		wantMessage: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofString{},
		},
	}, {
		desc:         "oneof set to string",
		inputMessage: &pb3.Oneofs{},
		inputText:    `{"oneofString": "hello"}`,
		wantMessage: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofString{
				OneofString: "hello",
			},
		},
	}, {
		desc:         "oneof set to enum",
		inputMessage: &pb3.Oneofs{},
		inputText:    `{"oneofEnum": "ZERO"}`,
		wantMessage: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofEnum{
				OneofEnum: pb3.Enum_ZERO,
			},
		},
	}, {
		desc:         "oneof set to empty message",
		inputMessage: &pb3.Oneofs{},
		inputText:    `{"oneofNested": {}}`,
		wantMessage: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofNested{
				OneofNested: &pb3.Nested{},
			},
		},
	}, {
		desc:         "oneof set to message",
		inputMessage: &pb3.Oneofs{},
		inputText: `{
  "oneofNested": {
    "sString": "nested message"
  }
}`,
		wantMessage: &pb3.Oneofs{
			Union: &pb3.Oneofs_OneofNested{
				OneofNested: &pb3.Nested{
					SString: "nested message",
				},
			},
		},
	}, {
		desc:         "repeated null fields",
		inputMessage: &pb2.Repeats{},
		inputText: `{
  "rptString": null,
  "rptInt32" : null,
  "rptFloat" : null,
  "rptBytes" : null
}`,
		wantMessage: &pb2.Repeats{},
	}, {
		desc:         "repeated scalars",
		inputMessage: &pb2.Repeats{},
		inputText: `{
  "rptString": ["hello", "world"],
  "rptInt32" : [-1, 0, 1],
  "rptBool"  : [false, true]
}`,
		wantMessage: &pb2.Repeats{
			RptString: []string{"hello", "world"},
			RptInt32:  []int32{-1, 0, 1},
			RptBool:   []bool{false, true},
		},
	}, {
		desc:         "repeated enums",
		inputMessage: &pb2.Enums{},
		inputText: `{
  "rptEnum"      : ["TEN", 1, 42],
  "rptNestedEnum": ["DOS", 2, -47]
}`,
		wantMessage: &pb2.Enums{
			RptEnum:       []pb2.Enum{pb2.Enum_TEN, pb2.Enum_ONE, 42},
			RptNestedEnum: []pb2.Enums_NestedEnum{pb2.Enums_DOS, pb2.Enums_DOS, -47},
		},
	}, {
		desc:         "repeated messages",
		inputMessage: &pb2.Nests{},
		inputText: `{
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
		desc:         "repeated groups",
		inputMessage: &pb2.Nests{},
		inputText: `{
  "rptgroup": [
    {
      "rptString": ["hello", "world"]
    },
    {}
  ]
}
`,
		wantMessage: &pb2.Nests{
			Rptgroup: []*pb2.Nests_RptGroup{
				{
					RptString: []string{"hello", "world"},
				},
				{},
			},
		},
	}, {
		desc:         "repeated string contains invalid UTF8",
		inputMessage: &pb2.Repeats{},
		inputText:    `{"rptString": ["` + "abc\xff" + `"]}`,
		wantMessage: &pb2.Repeats{
			RptString: []string{"abc\xff"},
		},
		wantErr: true,
	}, {
		desc:         "repeated messages contain invalid UTF8",
		inputMessage: &pb2.Nests{},
		inputText:    `{"rptNested": [{"optString": "` + "abc\xff" + `"}]}`,
		wantMessage: &pb2.Nests{
			RptNested: []*pb2.Nested{{OptString: scalar.String("abc\xff")}},
		},
		wantErr: true,
	}, {
		desc:         "repeated scalars contain invalid type",
		inputMessage: &pb2.Repeats{},
		inputText:    `{"rptString": ["hello", null, "world"]}`,
		wantErr:      true,
	}, {
		desc:         "repeated messages contain invalid type",
		inputMessage: &pb2.Nests{},
		inputText:    `{"rptNested": [{}, null]}`,
		wantErr:      true,
	}, {
		desc:         "map fields 1",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "int32ToStr": {
    "-101": "-101",
	"0"   : "zero",
	"255" : "0xff"
  },
  "boolToUint32": {
    "false": 101,
	"true" : "42"
  }
}`,
		wantMessage: &pb3.Maps{
			Int32ToStr: map[int32]string{
				-101: "-101",
				0xff: "0xff",
				0:    "zero",
			},
			BoolToUint32: map[bool]uint32{
				true:  42,
				false: 101,
			},
		},
	}, {
		desc:         "map fields 2",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "uint64ToEnum": {
    "1" : "ONE",
	"2" : 2,
	"10": 101
  }
}`,
		wantMessage: &pb3.Maps{
			Uint64ToEnum: map[uint64]pb3.Enum{
				1:  pb3.Enum_ONE,
				2:  pb3.Enum_TWO,
				10: 101,
			},
		},
	}, {
		desc:         "map fields 3",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "strToNested": {
    "nested_one": {
	  "sString": "nested in a map"
    },
    "nested_two": {}
  }
}`,
		wantMessage: &pb3.Maps{
			StrToNested: map[string]*pb3.Nested{
				"nested_one": {
					SString: "nested in a map",
				},
				"nested_two": {},
			},
		},
	}, {
		desc:         "map fields 4",
		inputMessage: &pb3.Maps{},
		inputText: `{
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
		wantMessage: &pb3.Maps{
			StrToOneofs: map[string]*pb3.Oneofs{
				"string": {
					Union: &pb3.Oneofs_OneofString{
						OneofString: "hello",
					},
				},
				"nested": {
					Union: &pb3.Oneofs_OneofNested{
						OneofNested: &pb3.Nested{
							SString: "nested oneof in map field value",
						},
					},
				},
			},
		},
	}, {
		desc:         "map contains duplicate keys",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "int32ToStr": {
    "0": "cero",
	"0": "zero"
  }
}
`,
		wantErr: true,
	}, {
		desc:         "map key empty string",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "strToNested": {
    "": {}
  }
}`,
		wantMessage: &pb3.Maps{
			StrToNested: map[string]*pb3.Nested{
				"": {},
			},
		},
	}, {
		desc:         "map contains invalid key 1",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "int32ToStr": {
    "invalid": "cero"
}`,
		wantErr: true,
	}, {
		desc:         "map contains invalid key 2",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "int32ToStr": {
    "1.02": "float"
}`,
		wantErr: true,
	}, {
		desc:         "map contains invalid key 3",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "int32ToStr": {
    "2147483648": "exceeds 32-bit integer max limit"
}`,
		wantErr: true,
	}, {
		desc:         "map contains invalid key 4",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "uint64ToEnum": {
    "-1": 0
  }
}`,
		wantErr: true,
	}, {
		desc:         "map contains invalid value",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "int32ToStr": {
    "101": true
}`,
		wantErr: true,
	}, {
		desc:         "map contains null for scalar value",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "int32ToStr": {
    "101": null
}`,
		wantErr: true,
	}, {
		desc:         "map contains null for message value",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "strToNested": {
    "hello": null
  }
}`,
		wantErr: true,
	}, {
		desc:         "map contains contains message value with invalid UTF8",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "strToNested": {
    "hello": {
      "sString": "` + "abc\xff" + `"
	}
  }
}`,
		wantMessage: &pb3.Maps{
			StrToNested: map[string]*pb3.Nested{
				"hello": {SString: "abc\xff"},
			},
		},
		wantErr: true,
	}, {
		desc:         "map key contains invalid UTF8",
		inputMessage: &pb3.Maps{},
		inputText: `{
  "strToNested": {
    "` + "abc\xff" + `": {}
  }
}`,
		wantMessage: &pb3.Maps{
			StrToNested: map[string]*pb3.Nested{
				"abc\xff": {},
			},
		},
		wantErr: true,
	}, {
		desc:         "extensions of non-repeated fields",
		inputMessage: &pb2.Extensions{},
		inputText: `{
  "optString": "non-extension field",
  "optBool": true,
  "optInt32": 42,
  "[pb2.opt_ext_bool]": true,
  "[pb2.opt_ext_nested]": {
    "optString": "nested in an extension",
    "optNested": {
      "optString": "another nested in an extension"
    }
  },
  "[pb2.opt_ext_string]": "extension field",
  "[pb2.opt_ext_enum]": "TEN"
}`,
		wantMessage: func() proto.Message {
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
	}, {
		desc:         "extensions of repeated fields",
		inputMessage: &pb2.Extensions{},
		inputText: `{
  "[pb2.rpt_ext_enum]": ["TEN", 101, "ONE"],
  "[pb2.rpt_ext_fixed32]": [42, 47],
  "[pb2.rpt_ext_nested]": [
    {"optString": "one"},
	{"optString": "two"},
	{"optString": "three"}
  ]
}`,
		wantMessage: func() proto.Message {
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
	}, {
		desc:         "extensions of non-repeated fields in another message",
		inputMessage: &pb2.Extensions{},
		inputText: `{
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
		wantMessage: func() proto.Message {
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
	}, {
		desc:         "extensions of repeated fields in another message",
		inputMessage: &pb2.Extensions{},
		inputText: `{
  "optString": "non-extension field",
  "optBool": true,
  "optInt32": 42,
  "[pb2.ExtensionsContainer.rpt_ext_nested]": [
    {"optString": "one"},
    {"optString": "two"},
    {"optString": "three"}
  ],
  "[pb2.ExtensionsContainer.rpt_ext_enum]": ["TEN", 101, "ONE"],
  "[pb2.ExtensionsContainer.rpt_ext_string]": ["hello", "world"]
}`,
		wantMessage: func() proto.Message {
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
	}, {
		desc:         "invalid extension field name",
		inputMessage: &pb2.Extensions{},
		inputText:    `{ "[pb2.invalid_message_field]": true }`,
		wantErr:      true,
	}, {
		desc:         "MessageSet",
		inputMessage: &pb2.MessageSet{},
		inputText: `{
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
		wantMessage: func() proto.Message {
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
	}, {
		desc:         "extension field set to null",
		inputMessage: &pb2.Extensions{},
		inputText: `{
  "[pb2.ExtensionsContainer.opt_ext_bool]": null,
  "[pb2.ExtensionsContainer.opt_ext_nested]": null
}`,
		wantMessage: func() proto.Message {
			m := &pb2.Extensions{}
			setExtension(m, pb2.E_ExtensionsContainer_OptExtBool, nil)
			setExtension(m, pb2.E_ExtensionsContainer_OptExtNested, nil)
			return m
		}(),
	}, {
		desc:         "extensions of repeated field contains null",
		inputMessage: &pb2.Extensions{},
		inputText: `{
  "[pb2.ExtensionsContainer.rpt_ext_nested]": [
    {"optString": "one"},
	null,
    {"optString": "three"}
  ],
}`,
		wantErr: true,
	}, {
		desc:         "not real MessageSet 1",
		inputMessage: &pb2.FakeMessageSet{},
		inputText: `{
  "[pb2.FakeMessageSetExtension.message_set_extension]": {
    "optString": "not a messageset extension"
  }
}`,
		wantMessage: func() proto.Message {
			m := &pb2.FakeMessageSet{}
			setExtension(m, pb2.E_FakeMessageSetExtension_MessageSetExtension, &pb2.FakeMessageSetExtension{
				OptString: scalar.String("not a messageset extension"),
			})
			return m
		}(),
	}, {
		desc:         "not real MessageSet 2",
		inputMessage: &pb2.FakeMessageSet{},
		inputText: `{
  "[pb2.FakeMessageSetExtension]": {
    "optString": "not a messageset extension"
  }
}`,
		wantErr: true,
	}, {
		desc:         "not real MessageSet 3",
		inputMessage: &pb2.MessageSet{},
		inputText: `{
  "[pb2.message_set_extension]": {
    "optString": "another not a messageset extension"
  }
}`,
		wantMessage: func() proto.Message {
			m := &pb2.MessageSet{}
			setExtension(m, pb2.E_MessageSetExtension, &pb2.FakeMessageSetExtension{
				OptString: scalar.String("another not a messageset extension"),
			})
			return m
		}(),
	}, {
		desc:         "Empty",
		inputMessage: &knownpb.Empty{},
		inputText:    `{}`,
		wantMessage:  &knownpb.Empty{},
	}, {
		desc:         "Empty contains unknown",
		inputMessage: &knownpb.Empty{},
		inputText:    `{"unknown": null}`,
		wantErr:      true,
	}, {
		desc:         "BoolValue false",
		inputMessage: &knownpb.BoolValue{},
		inputText:    `false`,
		wantMessage:  &knownpb.BoolValue{},
	}, {
		desc:         "BoolValue true",
		inputMessage: &knownpb.BoolValue{},
		inputText:    `true`,
		wantMessage:  &knownpb.BoolValue{Value: true},
	}, {
		desc:         "BoolValue invalid value",
		inputMessage: &knownpb.BoolValue{},
		inputText:    `{}`,
		wantErr:      true,
	}, {
		desc:         "Int32Value",
		inputMessage: &knownpb.Int32Value{},
		inputText:    `42`,
		wantMessage:  &knownpb.Int32Value{Value: 42},
	}, {
		desc:         "Int32Value in JSON string",
		inputMessage: &knownpb.Int32Value{},
		inputText:    `"1.23e3"`,
		wantMessage:  &knownpb.Int32Value{Value: 1230},
	}, {
		desc:         "Int64Value",
		inputMessage: &knownpb.Int64Value{},
		inputText:    `"42"`,
		wantMessage:  &knownpb.Int64Value{Value: 42},
	}, {
		desc:         "UInt32Value",
		inputMessage: &knownpb.UInt32Value{},
		inputText:    `42`,
		wantMessage:  &knownpb.UInt32Value{Value: 42},
	}, {
		desc:         "UInt64Value",
		inputMessage: &knownpb.UInt64Value{},
		inputText:    `"42"`,
		wantMessage:  &knownpb.UInt64Value{Value: 42},
	}, {
		desc:         "FloatValue",
		inputMessage: &knownpb.FloatValue{},
		inputText:    `1.02`,
		wantMessage:  &knownpb.FloatValue{Value: 1.02},
	}, {
		desc:         "FloatValue exceeds max limit",
		inputMessage: &knownpb.FloatValue{},
		inputText:    `1.23+40`,
		wantErr:      true,
	}, {
		desc:         "FloatValue Infinity",
		inputMessage: &knownpb.FloatValue{},
		inputText:    `"-Infinity"`,
		wantMessage:  &knownpb.FloatValue{Value: float32(math.Inf(-1))},
	}, {
		desc:         "DoubleValue",
		inputMessage: &knownpb.DoubleValue{},
		inputText:    `1.02`,
		wantMessage:  &knownpb.DoubleValue{Value: 1.02},
	}, {
		desc:         "DoubleValue Infinity",
		inputMessage: &knownpb.DoubleValue{},
		inputText:    `"Infinity"`,
		wantMessage:  &knownpb.DoubleValue{Value: math.Inf(+1)},
	}, {
		desc:         "StringValue empty",
		inputMessage: &knownpb.StringValue{},
		inputText:    `""`,
		wantMessage:  &knownpb.StringValue{},
	}, {
		desc:         "StringValue",
		inputMessage: &knownpb.StringValue{},
		inputText:    `"谷歌"`,
		wantMessage:  &knownpb.StringValue{Value: "谷歌"},
	}, {
		desc:         "StringValue with invalid UTF8 error",
		inputMessage: &knownpb.StringValue{},
		inputText:    "\"abc\xff\"",
		wantMessage:  &knownpb.StringValue{Value: "abc\xff"},
		wantErr:      true,
	}, {
		desc:         "StringValue field with invalid UTF8 error",
		inputMessage: &pb2.KnownTypes{},
		inputText:    "{\n  \"optString\": \"abc\xff\"\n}",
		wantMessage: &pb2.KnownTypes{
			OptString: &knownpb.StringValue{Value: "abc\xff"},
		},
		wantErr: true,
	}, {
		desc:         "BytesValue",
		inputMessage: &knownpb.BytesValue{},
		inputText:    `"aGVsbG8="`,
		wantMessage:  &knownpb.BytesValue{Value: []byte("hello")},
	}, {
		desc:         "Value null",
		inputMessage: &knownpb.Value{},
		inputText:    `null`,
		wantMessage:  &knownpb.Value{Kind: &knownpb.Value_NullValue{}},
	}, {
		desc:         "Value field null",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optValue": null
}`,
		wantMessage: &pb2.KnownTypes{
			OptValue: &knownpb.Value{Kind: &knownpb.Value_NullValue{}},
		},
	}, {
		desc:         "Value bool",
		inputMessage: &knownpb.Value{},
		inputText:    `false`,
		wantMessage:  &knownpb.Value{Kind: &knownpb.Value_BoolValue{}},
	}, {
		desc:         "Value field bool",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optValue": true
}`,
		wantMessage: &pb2.KnownTypes{
			OptValue: &knownpb.Value{Kind: &knownpb.Value_BoolValue{true}},
		},
	}, {
		desc:         "Value number",
		inputMessage: &knownpb.Value{},
		inputText:    `1.02`,
		wantMessage:  &knownpb.Value{Kind: &knownpb.Value_NumberValue{1.02}},
	}, {
		desc:         "Value field number",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optValue": 1.02
}`,
		wantMessage: &pb2.KnownTypes{
			OptValue: &knownpb.Value{Kind: &knownpb.Value_NumberValue{1.02}},
		},
	}, {
		desc:         "Value string",
		inputMessage: &knownpb.Value{},
		inputText:    `"hello"`,
		wantMessage:  &knownpb.Value{Kind: &knownpb.Value_StringValue{"hello"}},
	}, {
		desc:         "Value string with invalid UTF8",
		inputMessage: &knownpb.Value{},
		inputText:    "\"\xff\"",
		wantMessage:  &knownpb.Value{Kind: &knownpb.Value_StringValue{"\xff"}},
		wantErr:      true,
	}, {
		desc:         "Value field string",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optValue": "NaN"
}`,
		wantMessage: &pb2.KnownTypes{
			OptValue: &knownpb.Value{Kind: &knownpb.Value_StringValue{"NaN"}},
		},
	}, {
		desc:         "Value field string with invalid UTF8",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optValue": "` + "\xff" + `"
}`,
		wantMessage: &pb2.KnownTypes{
			OptValue: &knownpb.Value{Kind: &knownpb.Value_StringValue{"\xff"}},
		},
		wantErr: true,
	}, {
		desc:         "Value empty struct",
		inputMessage: &knownpb.Value{},
		inputText:    `{}`,
		wantMessage: &knownpb.Value{
			Kind: &knownpb.Value_StructValue{
				&knownpb.Struct{Fields: map[string]*knownpb.Value{}},
			},
		},
	}, {
		desc:         "Value struct",
		inputMessage: &knownpb.Value{},
		inputText: `{
  "string": "hello",
  "number": 123,
  "null": null,
  "bool": false,
  "struct": {
    "string": "world"
  },
  "list": []
}`,
		wantMessage: &knownpb.Value{
			Kind: &knownpb.Value_StructValue{
				&knownpb.Struct{
					Fields: map[string]*knownpb.Value{
						"string": {Kind: &knownpb.Value_StringValue{"hello"}},
						"number": {Kind: &knownpb.Value_NumberValue{123}},
						"null":   {Kind: &knownpb.Value_NullValue{}},
						"bool":   {Kind: &knownpb.Value_BoolValue{false}},
						"struct": {
							Kind: &knownpb.Value_StructValue{
								&knownpb.Struct{
									Fields: map[string]*knownpb.Value{
										"string": {Kind: &knownpb.Value_StringValue{"world"}},
									},
								},
							},
						},
						"list": {
							Kind: &knownpb.Value_ListValue{&knownpb.ListValue{}},
						},
					},
				},
			},
		},
	}, {
		desc:         "Value struct with invalid UTF8 string",
		inputMessage: &knownpb.Value{},
		inputText:    "{\"string\": \"abc\xff\"}",
		wantMessage: &knownpb.Value{
			Kind: &knownpb.Value_StructValue{
				&knownpb.Struct{
					Fields: map[string]*knownpb.Value{
						"string": {Kind: &knownpb.Value_StringValue{"abc\xff"}},
					},
				},
			},
		},
		wantErr: true,
	}, {
		desc:         "Value field struct",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optValue": {
    "string": "hello"
  }
}`,
		wantMessage: &pb2.KnownTypes{
			OptValue: &knownpb.Value{
				Kind: &knownpb.Value_StructValue{
					&knownpb.Struct{
						Fields: map[string]*knownpb.Value{
							"string": {Kind: &knownpb.Value_StringValue{"hello"}},
						},
					},
				},
			},
		},
	}, {
		desc:         "Value empty list",
		inputMessage: &knownpb.Value{},
		inputText:    `[]`,
		wantMessage: &knownpb.Value{
			Kind: &knownpb.Value_ListValue{
				&knownpb.ListValue{Values: []*knownpb.Value{}},
			},
		},
	}, {
		desc:         "Value list",
		inputMessage: &knownpb.Value{},
		inputText: `[
  "string",
  123,
  null,
  true,
  {},
  [
    "string",
	1.23,
	null,
	false
  ]
]`,
		wantMessage: &knownpb.Value{
			Kind: &knownpb.Value_ListValue{
				&knownpb.ListValue{
					Values: []*knownpb.Value{
						{Kind: &knownpb.Value_StringValue{"string"}},
						{Kind: &knownpb.Value_NumberValue{123}},
						{Kind: &knownpb.Value_NullValue{}},
						{Kind: &knownpb.Value_BoolValue{true}},
						{Kind: &knownpb.Value_StructValue{&knownpb.Struct{}}},
						{
							Kind: &knownpb.Value_ListValue{
								&knownpb.ListValue{
									Values: []*knownpb.Value{
										{Kind: &knownpb.Value_StringValue{"string"}},
										{Kind: &knownpb.Value_NumberValue{1.23}},
										{Kind: &knownpb.Value_NullValue{}},
										{Kind: &knownpb.Value_BoolValue{false}},
									},
								},
							},
						},
					},
				},
			},
		},
	}, {
		desc:         "Value list with invalid UTF8 string",
		inputMessage: &knownpb.Value{},
		inputText:    "[\"abc\xff\"]",
		wantMessage: &knownpb.Value{
			Kind: &knownpb.Value_ListValue{
				&knownpb.ListValue{
					Values: []*knownpb.Value{
						{Kind: &knownpb.Value_StringValue{"abc\xff"}},
					},
				},
			},
		},
		wantErr: true,
	}, {
		desc:         "Value field list with invalid UTF8 string",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optValue": [ "` + "abc\xff" + `"]
}`,
		wantMessage: &pb2.KnownTypes{
			OptValue: &knownpb.Value{
				Kind: &knownpb.Value_ListValue{
					&knownpb.ListValue{
						Values: []*knownpb.Value{
							{Kind: &knownpb.Value_StringValue{"abc\xff"}},
						},
					},
				},
			},
		},
		wantErr: true,
	}, {
		desc:         "FieldMask empty",
		inputMessage: &knownpb.FieldMask{},
		inputText:    `""`,
		wantMessage:  &knownpb.FieldMask{Paths: []string{}},
	}, {
		desc:         "FieldMask",
		inputMessage: &knownpb.FieldMask{},
		inputText:    `"foo,fooBar , foo.barQux ,Foo"`,
		wantMessage: &knownpb.FieldMask{
			Paths: []string{
				"foo",
				"foo_bar",
				"foo.bar_qux",
				"_foo",
			},
		},
	}, {
		desc:         "FieldMask field",
		inputMessage: &pb2.KnownTypes{},
		inputText: `{
  "optFieldmask": "foo, qux.fooBar"
}`,
		wantMessage: &pb2.KnownTypes{
			OptFieldmask: &knownpb.FieldMask{
				Paths: []string{
					"foo",
					"qux.foo_bar",
				},
			},
		},
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.umo.Unmarshal(tt.inputMessage, []byte(tt.inputText))
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
