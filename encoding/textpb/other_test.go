package textpb_test

import (
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/encoding/textpb"
	"github.com/golang/protobuf/v2/encoding/textpb/testprotos/pb2"
	"github.com/golang/protobuf/v2/proto"

	// The legacy package must be imported prior to use of any legacy messages.
	// TODO: Remove this when protoV1 registers these hooks for you.
	_ "github.com/golang/protobuf/v2/internal/legacy"

	anypb "github.com/golang/protobuf/ptypes/any"
	durpb "github.com/golang/protobuf/ptypes/duration"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	stpb "github.com/golang/protobuf/ptypes/struct"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	wpb "github.com/golang/protobuf/ptypes/wrappers"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		desc    string
		message proto.Message
	}{{
		desc: "well-known type fields set to empty messages",
		message: &pb2.KnownTypes{
			OptBool:      &wpb.BoolValue{},
			OptInt32:     &wpb.Int32Value{},
			OptInt64:     &wpb.Int64Value{},
			OptUint32:    &wpb.UInt32Value{},
			OptUint64:    &wpb.UInt64Value{},
			OptFloat:     &wpb.FloatValue{},
			OptDouble:    &wpb.DoubleValue{},
			OptString:    &wpb.StringValue{},
			OptBytes:     &wpb.BytesValue{},
			OptDuration:  &durpb.Duration{},
			OptTimestamp: &tspb.Timestamp{},
			OptStruct:    &stpb.Struct{},
			OptList:      &stpb.ListValue{},
			OptValue:     &stpb.Value{},
			OptEmpty:     &emptypb.Empty{},
			OptAny:       &anypb.Any{},
		},
	}, {
		desc: "well-known type scalar fields",
		message: &pb2.KnownTypes{
			OptBool: &wpb.BoolValue{
				Value: true,
			},
			OptInt32: &wpb.Int32Value{
				Value: -42,
			},
			OptInt64: &wpb.Int64Value{
				Value: -42,
			},
			OptUint32: &wpb.UInt32Value{
				Value: 0xff,
			},
			OptUint64: &wpb.UInt64Value{
				Value: 0xffff,
			},
			OptFloat: &wpb.FloatValue{
				Value: 1.234,
			},
			OptDouble: &wpb.DoubleValue{
				Value: 1.23e308,
			},
			OptString: &wpb.StringValue{
				Value: "谷歌",
			},
			OptBytes: &wpb.BytesValue{
				Value: []byte("\xe8\xb0\xb7\xe6\xad\x8c"),
			},
		},
	}, {
		desc: "well-known type time-related fields",
		message: &pb2.KnownTypes{
			OptDuration: &durpb.Duration{
				Seconds: -3600,
				Nanos:   -123,
			},
			OptTimestamp: &tspb.Timestamp{
				Seconds: 1257894000,
				Nanos:   123,
			},
		},
	}, {
		desc: "well-known type struct field and different Value types",
		message: &pb2.KnownTypes{
			OptStruct: &stpb.Struct{
				Fields: map[string]*stpb.Value{
					"bool": &stpb.Value{
						Kind: &stpb.Value_BoolValue{
							BoolValue: true,
						},
					},
					"double": &stpb.Value{
						Kind: &stpb.Value_NumberValue{
							NumberValue: 3.1415,
						},
					},
					"null": &stpb.Value{
						Kind: &stpb.Value_NullValue{
							NullValue: stpb.NullValue_NULL_VALUE,
						},
					},
					"string": &stpb.Value{
						Kind: &stpb.Value_StringValue{
							StringValue: "string",
						},
					},
					"struct": &stpb.Value{
						Kind: &stpb.Value_StructValue{
							StructValue: &stpb.Struct{
								Fields: map[string]*stpb.Value{
									"bool": &stpb.Value{
										Kind: &stpb.Value_BoolValue{
											BoolValue: false,
										},
									},
								},
							},
						},
					},
					"list": &stpb.Value{
						Kind: &stpb.Value_ListValue{
							ListValue: &stpb.ListValue{
								Values: []*stpb.Value{
									{
										Kind: &stpb.Value_BoolValue{
											BoolValue: false,
										},
									},
									{
										Kind: &stpb.Value_StringValue{
											StringValue: "hello",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			b, err := textpb.Marshal(tt.message)
			if err != nil {
				t.Errorf("Marshal() returned error: %v\n\n", err)
			}
			gotMessage := tt.message.ProtoReflect().Type().New()
			err = textpb.Unmarshal(gotMessage, b)
			if err != nil {
				t.Errorf("Unmarshal() returned error: %v\n\n", err)
			}
			if !protoV1.Equal(gotMessage.(protoV1.Message), tt.message.(protoV1.Message)) {
				t.Errorf("Unmarshal()\n<got>\n%v\n<want>\n%v\n", gotMessage, tt.message)
			}
		})
	}
}
