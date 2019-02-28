package textpb_test

import (
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/encoding/textpb"
	"github.com/golang/protobuf/v2/internal/impl"
	"github.com/golang/protobuf/v2/internal/scalar"
	"github.com/golang/protobuf/v2/proto"
	preg "github.com/golang/protobuf/v2/reflect/protoregistry"

	"github.com/golang/protobuf/v2/encoding/testprotos/pb2"
	knownpb "github.com/golang/protobuf/v2/types/known"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct {
		desc     string
		resolver *preg.Types
		message  proto.Message
	}{{
		desc: "well-known type fields set to empty messages",
		message: &pb2.KnownTypes{
			OptBool:      &knownpb.BoolValue{},
			OptInt32:     &knownpb.Int32Value{},
			OptInt64:     &knownpb.Int64Value{},
			OptUint32:    &knownpb.UInt32Value{},
			OptUint64:    &knownpb.UInt64Value{},
			OptFloat:     &knownpb.FloatValue{},
			OptDouble:    &knownpb.DoubleValue{},
			OptString:    &knownpb.StringValue{},
			OptBytes:     &knownpb.BytesValue{},
			OptDuration:  &knownpb.Duration{},
			OptTimestamp: &knownpb.Timestamp{},
			OptStruct:    &knownpb.Struct{},
			OptList:      &knownpb.ListValue{},
			OptValue:     &knownpb.Value{},
			OptEmpty:     &knownpb.Empty{},
			OptAny:       &knownpb.Any{},
		},
	}, {
		desc: "well-known type scalar fields",
		message: &pb2.KnownTypes{
			OptBool: &knownpb.BoolValue{
				Value: true,
			},
			OptInt32: &knownpb.Int32Value{
				Value: -42,
			},
			OptInt64: &knownpb.Int64Value{
				Value: -42,
			},
			OptUint32: &knownpb.UInt32Value{
				Value: 0xff,
			},
			OptUint64: &knownpb.UInt64Value{
				Value: 0xffff,
			},
			OptFloat: &knownpb.FloatValue{
				Value: 1.234,
			},
			OptDouble: &knownpb.DoubleValue{
				Value: 1.23e308,
			},
			OptString: &knownpb.StringValue{
				Value: "谷歌",
			},
			OptBytes: &knownpb.BytesValue{
				Value: []byte("\xe8\xb0\xb7\xe6\xad\x8c"),
			},
		},
	}, {
		desc: "well-known type time-related fields",
		message: &pb2.KnownTypes{
			OptDuration: &knownpb.Duration{
				Seconds: -3600,
				Nanos:   -123,
			},
			OptTimestamp: &knownpb.Timestamp{
				Seconds: 1257894000,
				Nanos:   123,
			},
		},
	}, {
		desc: "Struct field and different Value types",
		message: &pb2.KnownTypes{
			OptStruct: &knownpb.Struct{
				Fields: map[string]*knownpb.Value{
					"bool": &knownpb.Value{
						Kind: &knownpb.Value_BoolValue{
							BoolValue: true,
						},
					},
					"double": &knownpb.Value{
						Kind: &knownpb.Value_NumberValue{
							NumberValue: 3.1415,
						},
					},
					"null": &knownpb.Value{
						Kind: &knownpb.Value_NullValue{
							NullValue: knownpb.NullValue_NULL_VALUE,
						},
					},
					"string": &knownpb.Value{
						Kind: &knownpb.Value_StringValue{
							StringValue: "string",
						},
					},
					"struct": &knownpb.Value{
						Kind: &knownpb.Value_StructValue{
							StructValue: &knownpb.Struct{
								Fields: map[string]*knownpb.Value{
									"bool": &knownpb.Value{
										Kind: &knownpb.Value_BoolValue{
											BoolValue: false,
										},
									},
								},
							},
						},
					},
					"list": &knownpb.Value{
						Kind: &knownpb.Value_ListValue{
							ListValue: &knownpb.ListValue{
								Values: []*knownpb.Value{
									{
										Kind: &knownpb.Value_BoolValue{
											BoolValue: false,
										},
									},
									{
										Kind: &knownpb.Value_StringValue{
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
				OptAny: &knownpb.Any{
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
				OptAny: &knownpb.Any{
					TypeUrl: string(m.ProtoReflect().Type().FullName()),
					Value:   b,
				},
			}
		}(),
	}, {
		desc: "Any field containing Any message",
		resolver: func() *preg.Types {
			mt1 := (&pb2.Nested{}).ProtoReflect().Type()
			mt2 := impl.Export{}.MessageTypeOf(&knownpb.Any{})
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
			m2 := &knownpb.Any{
				TypeUrl: "pb2.Nested",
				Value:   b1,
			}
			// TODO: Switch to V2 marshal when ready.
			b2, err := protoV1.Marshal(m2)
			if err != nil {
				t.Fatalf("error in binary marshaling message for Any.value: %v", err)
			}
			return &pb2.KnownTypes{
				OptAny: &knownpb.Any{
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
