// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	pack "github.com/golang/protobuf/v2/internal/encoding/pack"
	pragma "github.com/golang/protobuf/v2/internal/pragma"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
	cmp "github.com/google/go-cmp/cmp"

	proto2_20160225 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v0.0.0-20160225-2fc053c5"
	proto2_20160519 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v0.0.0-20160519-a4ab9ec5"
	proto2_20180125 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v1.0.0-20180125-92554152"
	proto2_20180430 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v1.1.0-20180430-b4deda09"
	proto2_20180814 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v1.2.0-20180814-aa810b61"
	proto3_20160225 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto3.v0.0.0-20160225-2fc053c5"
	proto3_20160519 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto3.v0.0.0-20160519-a4ab9ec5"
	proto3_20180125 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto3.v1.0.0-20180125-92554152"
	proto3_20180430 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto3.v1.1.0-20180430-b4deda09"
	proto3_20180814 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto3.v1.2.0-20180814-aa810b61"
)

func mustLoadFileDesc(b []byte, _ []int) pref.FileDescriptor {
	fd, err := ptype.NewFileFromDescriptorProto(loadFileDesc(b), nil)
	if err != nil {
		panic(err)
	}
	return fd
}

func TestLegacyDescriptor(t *testing.T) {
	var tests []struct{ got, want pref.Descriptor }

	fileDescP2_20160225 := mustLoadFileDesc(new(proto2_20160225.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto2_20160225.SiblingEnum(0))),
		want: fileDescP2_20160225.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto2_20160225.Message_ChildEnum(0))),
		want: fileDescP2_20160225.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.SiblingMessage))),
		want: fileDescP2_20160225.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_ChildMessage))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message))),
		want: fileDescP2_20160225.Messages().ByName("Message"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_NamedGroup))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("NamedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_OptionalGroup))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("OptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_RequiredGroup))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("RequiredGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_RepeatedGroup))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("RepeatedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_OneofGroup))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("OneofGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_ExtensionOptionalGroup))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("ExtensionOptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160225.Message_ExtensionRepeatedGroup))),
		want: fileDescP2_20160225.Messages().ByName("Message").Messages().ByName("ExtensionRepeatedGroup"),
	}}...)

	fileDescP3_20160225 := mustLoadFileDesc(new(proto3_20160225.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto3_20160225.SiblingEnum(0))),
		want: fileDescP3_20160225.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto3_20160225.Message_ChildEnum(0))),
		want: fileDescP3_20160225.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20160225.SiblingMessage))),
		want: fileDescP3_20160225.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20160225.Message_ChildMessage))),
		want: fileDescP3_20160225.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20160225.Message))),
		want: fileDescP3_20160225.Messages().ByName("Message"),
	}}...)

	fileDescP2_20160519 := mustLoadFileDesc(new(proto2_20160519.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto2_20160519.SiblingEnum(0))),
		want: fileDescP2_20160519.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto2_20160519.Message_ChildEnum(0))),
		want: fileDescP2_20160519.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.SiblingMessage))),
		want: fileDescP2_20160519.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_ChildMessage))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message))),
		want: fileDescP2_20160519.Messages().ByName("Message"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_NamedGroup))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("NamedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_OptionalGroup))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("OptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_RequiredGroup))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("RequiredGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_RepeatedGroup))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("RepeatedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_OneofGroup))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("OneofGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_ExtensionOptionalGroup))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("ExtensionOptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20160519.Message_ExtensionRepeatedGroup))),
		want: fileDescP2_20160519.Messages().ByName("Message").Messages().ByName("ExtensionRepeatedGroup"),
	}}...)

	fileDescP3_20160519 := mustLoadFileDesc(new(proto3_20160519.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto3_20160519.SiblingEnum(0))),
		want: fileDescP3_20160519.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto3_20160519.Message_ChildEnum(0))),
		want: fileDescP3_20160519.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20160519.SiblingMessage))),
		want: fileDescP3_20160519.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20160519.Message_ChildMessage))),
		want: fileDescP3_20160519.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20160519.Message))),
		want: fileDescP3_20160519.Messages().ByName("Message"),
	}}...)

	fileDescP2_20180125 := mustLoadFileDesc(new(proto2_20180125.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto2_20180125.SiblingEnum(0))),
		want: fileDescP2_20180125.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto2_20180125.Message_ChildEnum(0))),
		want: fileDescP2_20180125.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.SiblingMessage))),
		want: fileDescP2_20180125.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_ChildMessage))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message))),
		want: fileDescP2_20180125.Messages().ByName("Message"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_NamedGroup))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("NamedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_OptionalGroup))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("OptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_RequiredGroup))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("RequiredGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_RepeatedGroup))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("RepeatedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_OneofGroup))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("OneofGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_ExtensionOptionalGroup))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("ExtensionOptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180125.Message_ExtensionRepeatedGroup))),
		want: fileDescP2_20180125.Messages().ByName("Message").Messages().ByName("ExtensionRepeatedGroup"),
	}}...)

	fileDescP3_20180125 := mustLoadFileDesc(new(proto3_20180125.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto3_20180125.SiblingEnum(0))),
		want: fileDescP3_20180125.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto3_20180125.Message_ChildEnum(0))),
		want: fileDescP3_20180125.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180125.SiblingMessage))),
		want: fileDescP3_20180125.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180125.Message_ChildMessage))),
		want: fileDescP3_20180125.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180125.Message))),
		want: fileDescP3_20180125.Messages().ByName("Message"),
	}}...)

	fileDescP2_20180430 := mustLoadFileDesc(new(proto2_20180430.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto2_20180430.SiblingEnum(0))),
		want: fileDescP2_20180430.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto2_20180430.Message_ChildEnum(0))),
		want: fileDescP2_20180430.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.SiblingMessage))),
		want: fileDescP2_20180430.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_ChildMessage))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message))),
		want: fileDescP2_20180430.Messages().ByName("Message"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_NamedGroup))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("NamedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_OptionalGroup))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("OptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_RequiredGroup))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("RequiredGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_RepeatedGroup))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("RepeatedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_OneofGroup))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("OneofGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_ExtensionOptionalGroup))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("ExtensionOptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180430.Message_ExtensionRepeatedGroup))),
		want: fileDescP2_20180430.Messages().ByName("Message").Messages().ByName("ExtensionRepeatedGroup"),
	}}...)

	fileDescP3_20180430 := mustLoadFileDesc(new(proto3_20180430.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto3_20180430.SiblingEnum(0))),
		want: fileDescP3_20180430.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto3_20180430.Message_ChildEnum(0))),
		want: fileDescP3_20180430.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180430.SiblingMessage))),
		want: fileDescP3_20180430.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180430.Message_ChildMessage))),
		want: fileDescP3_20180430.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180430.Message))),
		want: fileDescP3_20180430.Messages().ByName("Message"),
	}}...)

	fileDescP2_20180814 := mustLoadFileDesc(new(proto2_20180814.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto2_20180814.SiblingEnum(0))),
		want: fileDescP2_20180814.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto2_20180814.Message_ChildEnum(0))),
		want: fileDescP2_20180814.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.SiblingMessage))),
		want: fileDescP2_20180814.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_ChildMessage))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message))),
		want: fileDescP2_20180814.Messages().ByName("Message"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_NamedGroup))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("NamedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_OptionalGroup))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("OptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_RequiredGroup))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("RequiredGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_RepeatedGroup))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("RepeatedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_OneofGroup))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("OneofGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_ExtensionOptionalGroup))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("ExtensionOptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto2_20180814.Message_ExtensionRepeatedGroup))),
		want: fileDescP2_20180814.Messages().ByName("Message").Messages().ByName("ExtensionRepeatedGroup"),
	}}...)

	fileDescP3_20180814 := mustLoadFileDesc(new(proto3_20180814.Message).Descriptor())
	tests = append(tests, []struct{ got, want pref.Descriptor }{{
		got:  loadEnumDesc(reflect.TypeOf(proto3_20180814.SiblingEnum(0))),
		want: fileDescP3_20180814.Enums().ByName("SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(proto3_20180814.Message_ChildEnum(0))),
		want: fileDescP3_20180814.Messages().ByName("Message").Enums().ByName("ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180814.SiblingMessage))),
		want: fileDescP3_20180814.Messages().ByName("SiblingMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180814.Message_ChildMessage))),
		want: fileDescP3_20180814.Messages().ByName("Message").Messages().ByName("ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(proto3_20180814.Message))),
		want: fileDescP3_20180814.Messages().ByName("Message"),
	}}...)

	type list interface {
		Len() int
		pragma.DoNotImplement
	}
	opts := cmp.Options{
		cmp.Transformer("", func(x list) []interface{} {
			out := make([]interface{}, x.Len())
			v := reflect.ValueOf(x)
			for i := 0; i < x.Len(); i++ {
				m := v.MethodByName("Get")
				out[i] = m.Call([]reflect.Value{reflect.ValueOf(i)})[0].Interface()
			}
			return out
		}),
		cmp.Transformer("", func(x pref.Descriptor) map[string]interface{} {
			out := make(map[string]interface{})
			v := reflect.ValueOf(x)
			for i := 0; i < v.NumMethod(); i++ {
				name := v.Type().Method(i).Name
				if m := v.Method(i); m.Type().NumIn() == 0 && m.Type().NumOut() == 1 {
					switch name {
					case "Index":
						// Ignore index since legacy descriptors have no parent.
					case "Options":
						// Ignore descriptor options since protos are not cmperable.
					case "Enums", "Messages", "Extensions":
						// Ignore nested message and enum declarations since
						// legacy descriptors are all created standalone.
					case "OneofType", "ExtendedType", "EnumType", "MessageType":
						// Avoid descending into a dependency to avoid a cycle.
						// Just record the full name if available.
						//
						// TODO: Cycle support in cmp would be useful here.
						v := m.Call(nil)[0]
						if !v.IsNil() {
							out[name] = v.Interface().(pref.Descriptor).FullName()
						}
					default:
						out[name] = m.Call(nil)[0].Interface()
					}
				}
			}
			return out
		}),
		cmp.Transformer("", func(v pref.Value) interface{} {
			return v.Interface()
		}),
	}

	for _, tt := range tests {
		t.Run(string(tt.want.FullName()), func(t *testing.T) {
			if diff := cmp.Diff(&tt.want, &tt.got, opts); diff != "" {
				t.Errorf("descriptor mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

type legacyUnknownMessage struct {
	XXX_unrecognized []byte
	protoV1.XXX_InternalExtensions
}

func (*legacyUnknownMessage) ExtensionRangeArray() []protoV1.ExtensionRange {
	return []protoV1.ExtensionRange{{Start: 10, End: 20}, {Start: 40, End: 80}}
}

func TestLegacyUnknown(t *testing.T) {
	rawOf := func(toks ...pack.Token) pref.RawFields {
		return pref.RawFields(pack.Message(toks).Marshal())
	}
	raw1a := rawOf(pack.Tag{1, pack.VarintType}, pack.Svarint(-4321))                // 08c143
	raw1b := rawOf(pack.Tag{1, pack.Fixed32Type}, pack.Uint32(0xdeadbeef))           // 0defbeadde
	raw1c := rawOf(pack.Tag{1, pack.Fixed64Type}, pack.Float64(math.Pi))             // 09182d4454fb210940
	raw2a := rawOf(pack.Tag{2, pack.BytesType}, pack.String("hello, world!"))        // 120d68656c6c6f2c20776f726c6421
	raw2b := rawOf(pack.Tag{2, pack.VarintType}, pack.Uvarint(1234))                 // 10d209
	raw3a := rawOf(pack.Tag{3, pack.StartGroupType}, pack.Tag{3, pack.EndGroupType}) // 1b1c
	raw3b := rawOf(pack.Tag{3, pack.BytesType}, pack.Bytes("\xde\xad\xbe\xef"))      // 1a04deadbeef

	raw1 := rawOf(pack.Tag{1, pack.BytesType}, pack.Bytes("1"))    // 0a0131
	raw3 := rawOf(pack.Tag{3, pack.BytesType}, pack.Bytes("3"))    // 1a0133
	raw10 := rawOf(pack.Tag{10, pack.BytesType}, pack.Bytes("10")) // 52023130 - extension
	raw15 := rawOf(pack.Tag{15, pack.BytesType}, pack.Bytes("15")) // 7a023135 - extension
	raw26 := rawOf(pack.Tag{26, pack.BytesType}, pack.Bytes("26")) // d201023236
	raw32 := rawOf(pack.Tag{32, pack.BytesType}, pack.Bytes("32")) // 8202023332
	raw45 := rawOf(pack.Tag{45, pack.BytesType}, pack.Bytes("45")) // ea02023435 - extension
	raw46 := rawOf(pack.Tag{45, pack.BytesType}, pack.Bytes("46")) // ea02023436 - extension
	raw47 := rawOf(pack.Tag{45, pack.BytesType}, pack.Bytes("47")) // ea02023437 - extension
	raw99 := rawOf(pack.Tag{99, pack.BytesType}, pack.Bytes("99")) // 9a06023939

	joinRaw := func(bs ...pref.RawFields) (out []byte) {
		for _, b := range bs {
			out = append(out, b...)
		}
		return out
	}

	m := new(legacyUnknownMessage)
	fs := new(MessageType).MessageOf(m).UnknownFields()

	if got, want := fs.Len(), 0; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	fs.Set(1, raw1a)
	fs.Set(1, append(fs.Get(1), raw1b...))
	fs.Set(1, append(fs.Get(1), raw1c...))
	if got, want := fs.Len(), 1; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw1a, raw1b, raw1c); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	fs.Set(2, raw2a)
	if got, want := fs.Len(), 2; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw1a, raw1b, raw1c, raw2a); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	if got, want := fs.Get(1), joinRaw(raw1a, raw1b, raw1c); !bytes.Equal(got, want) {
		t.Errorf("Get(%d) = %x, want %x", 1, got, want)
	}
	if got, want := fs.Get(2), joinRaw(raw2a); !bytes.Equal(got, want) {
		t.Errorf("Get(%d) = %x, want %x", 2, got, want)
	}
	if got, want := fs.Get(3), joinRaw(); !bytes.Equal(got, want) {
		t.Errorf("Get(%d) = %x, want %x", 3, got, want)
	}

	fs.Set(1, nil) // remove field 1
	if got, want := fs.Len(), 1; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw2a); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	// Simulate manual appending of raw field data.
	m.XXX_unrecognized = append(m.XXX_unrecognized, joinRaw(raw3a, raw1a, raw1b, raw2b, raw3b, raw1c)...)
	if got, want := fs.Len(), 3; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}

	// Verify range iteration order.
	var i int
	want := []struct {
		num pref.FieldNumber
		raw pref.RawFields
	}{
		{2, joinRaw(raw2a, raw2b)},
		{3, joinRaw(raw3a, raw3b)},
		{1, joinRaw(raw1a, raw1b, raw1c)},
	}
	fs.Range(func(num pref.FieldNumber, raw pref.RawFields) bool {
		if i < len(want) {
			if num != want[i].num || !bytes.Equal(raw, want[i].raw) {
				t.Errorf("Range(%d) = (%d, %x), want (%d, %x)", i, num, raw, want[i].num, want[i].raw)
			}
		} else {
			t.Errorf("unexpected Range iteration: %d", i)
		}
		i++
		return true
	})

	fs.Set(2, fs.Get(2)) // moves field 2 to the end
	if got, want := fs.Len(), 3; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw3a, raw1a, raw1b, raw3b, raw1c, raw2a, raw2b); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}
	fs.Set(1, nil) // remove field 1
	if got, want := fs.Len(), 2; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw3a, raw3b, raw2a, raw2b); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	// Remove all fields.
	fs.Range(func(n pref.FieldNumber, b pref.RawFields) bool {
		fs.Set(n, nil)
		return true
	})
	if got, want := fs.Len(), 0; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	fs.Set(1, raw1)
	if got, want := fs.Len(), 1; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw1); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	fs.Set(45, raw45)
	fs.Set(10, raw10) // extension
	fs.Set(32, raw32)
	fs.Set(1, nil) // deletion
	fs.Set(26, raw26)
	fs.Set(47, raw47) // extension
	fs.Set(46, raw46) // extension
	if got, want := fs.Len(), 6; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw32, raw26); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	// Verify iteration order.
	i = 0
	want = []struct {
		num pref.FieldNumber
		raw pref.RawFields
	}{
		{32, raw32},
		{26, raw26},
		{10, raw10}, // extension
		{45, raw45}, // extension
		{46, raw46}, // extension
		{47, raw47}, // extension
	}
	fs.Range(func(num pref.FieldNumber, raw pref.RawFields) bool {
		if i < len(want) {
			if num != want[i].num || !bytes.Equal(raw, want[i].raw) {
				t.Errorf("Range(%d) = (%d, %x), want (%d, %x)", i, num, raw, want[i].num, want[i].raw)
			}
		} else {
			t.Errorf("unexpected Range iteration: %d", i)
		}
		i++
		return true
	})

	// Perform partial deletion while iterating.
	i = 0
	fs.Range(func(num pref.FieldNumber, raw pref.RawFields) bool {
		if i%2 == 0 {
			fs.Set(num, nil)
		}
		i++
		return true
	})

	if got, want := fs.Len(), 3; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw26); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	fs.Set(15, raw15) // extension
	fs.Set(3, raw3)
	fs.Set(99, raw99)
	if got, want := fs.Len(), 6; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := m.XXX_unrecognized, joinRaw(raw26, raw3, raw99); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	// Perform partial iteration.
	i = 0
	want = []struct {
		num pref.FieldNumber
		raw pref.RawFields
	}{
		{26, raw26},
		{3, raw3},
	}
	fs.Range(func(num pref.FieldNumber, raw pref.RawFields) bool {
		if i < len(want) {
			if num != want[i].num || !bytes.Equal(raw, want[i].raw) {
				t.Errorf("Range(%d) = (%d, %x), want (%d, %x)", i, num, raw, want[i].num, want[i].raw)
			}
		} else {
			t.Errorf("unexpected Range iteration: %d", i)
		}
		i++
		return i < 2
	})
}
