package textpb_test

import (
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/encoding/textpb"
	"github.com/golang/protobuf/v2/encoding/textpb/testprotos/pb2"
	"github.com/golang/protobuf/v2/proto"
	preg "github.com/golang/protobuf/v2/reflect/protoregistry"

	// The legacy package must be imported prior to use of any legacy messages.
	// TODO: Remove this when protoV1 registers these hooks for you.
	"github.com/golang/protobuf/v2/internal/impl"
	_ "github.com/golang/protobuf/v2/internal/legacy"
	"github.com/golang/protobuf/v2/internal/scalar"

	anypb "github.com/golang/protobuf/ptypes/any"
	durpb "github.com/golang/protobuf/ptypes/duration"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	stpb "github.com/golang/protobuf/ptypes/struct"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	wpb "github.com/golang/protobuf/ptypes/wrappers"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		desc     string
		resolver *preg.Types
		message  proto.Message
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
		desc: "Struct field and different Value types",
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
	}, {
		desc:     "Any field without registered type",
		resolver: preg.NewTypes(),
		message: func() proto.Message {
			m := &pb2.Nested{
				OptString: scalar.String("embedded inside Any"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("inception"),
				},
			}
			// TODO: Switch to V2 marshal when ready.
			b, err := protoV1.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &pb2.KnownTypes{
				OptAny: &anypb.Any{
					TypeUrl: string(m.ProtoReflect().Type().FullName()),
					Value:   b,
				},
			}
		}(),
	}, {
		desc:     "Any field with registered type",
		resolver: preg.NewTypes((&pb2.Nested{}).ProtoReflect().Type()),
		message: func() proto.Message {
			m := &pb2.Nested{
				OptString: scalar.String("embedded inside Any"),
				OptNested: &pb2.Nested{
					OptString: scalar.String("inception"),
				},
			}
			// TODO: Switch to V2 marshal when ready.
			b, err := protoV1.Marshal(m)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &pb2.KnownTypes{
				OptAny: &anypb.Any{
					TypeUrl: string(m.ProtoReflect().Type().FullName()),
					Value:   b,
				},
			}
		}(),
	}, {
		desc: "Any field containing Any message",
		resolver: func() *preg.Types {
			mt1 := (&pb2.Nested{}).ProtoReflect().Type()
			mt2 := impl.Export{}.MessageTypeOf(&anypb.Any{})
			return preg.NewTypes(mt1, mt2)
		}(),
		message: func() proto.Message {
			m1 := &pb2.Nested{
				OptString: scalar.String("message inside Any of another Any field"),
			}
			// TODO: Switch to V2 marshal when ready.
			b1, err := protoV1.Marshal(m1)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			m2 := &anypb.Any{
				TypeUrl: "pb2.Nested",
				Value:   b1,
			}
			// TODO: Switch to V2 marshal when ready.
			b2, err := protoV1.Marshal(m2)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &pb2.KnownTypes{
				OptAny: &anypb.Any{
					TypeUrl: "google.protobuf.Any",
					Value:   b2,
				},
			}
		}(),
	}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()
			mo := textpb.MarshalOptions{Resolver: tt.resolver}
			umo := textpb.UnmarshalOptions{Resolver: tt.resolver}

			b, err := mo.Marshal(tt.message)
			if err != nil {
				t.Errorf("Marshal() returned error: %v\n\n", err)
			}
			gotMessage := tt.message.ProtoReflect().Type().New().Interface()
			err = umo.Unmarshal(gotMessage, b)
			if err != nil {
				t.Errorf("Unmarshal() returned error: %v\n\n", err)
			}

			if !protoV1.Equal(gotMessage.(protoV1.Message), tt.message.(protoV1.Message)) {
				t.Errorf("Unmarshal()\n<got>\n%v\n<want>\n%v\n", gotMessage, tt.message)
			}
		})
	}
}
