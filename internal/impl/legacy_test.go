// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/v2/internal/pragma"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
	"github.com/google/go-cmp/cmp"
)

func mustLoadFileDesc(b []byte) pref.FileDescriptor {
	fd, err := ptype.NewFileFromDescriptorProto(loadFileDesc(b), nil)
	if err != nil {
		panic(err)
	}
	return fd
}

var fileDescLP2 = mustLoadFileDesc(LP2FileDescriptor)
var fileDescLP3 = mustLoadFileDesc(LP3FileDescriptor)

func TestLegacy(t *testing.T) {
	tests := []struct {
		got  pref.Descriptor
		want pref.Descriptor
	}{{
		got:  loadEnumDesc(reflect.TypeOf(LP2MapEnum(0))),
		want: fileDescLP2.Enums().ByName("LP2MapEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(LP2SiblingEnum(0))),
		want: fileDescLP2.Enums().ByName("LP2SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(LP2Message_LP2ChildEnum(0))),
		want: fileDescLP2.Messages().ByName("LP2Message").Enums().ByName("LP2ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP2Message))),
		want: fileDescLP2.Messages().ByName("LP2Message"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP2Message_LP2ChildMessage))),
		want: fileDescLP2.Messages().ByName("LP2Message").Messages().ByName("LP2ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP2Message_LP2NamedGroup))),
		want: fileDescLP2.Messages().ByName("LP2Message").Messages().ByName("LP2NamedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP2Message_OptionalGroup))),
		want: fileDescLP2.Messages().ByName("LP2Message").Messages().ByName("OptionalGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP2Message_RequiredGroup))),
		want: fileDescLP2.Messages().ByName("LP2Message").Messages().ByName("RequiredGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP2Message_RepeatedGroup))),
		want: fileDescLP2.Messages().ByName("LP2Message").Messages().ByName("RepeatedGroup"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP2SiblingMessage))),
		want: fileDescLP2.Messages().ByName("LP2SiblingMessage"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(LP3SiblingEnum(0))),
		want: fileDescLP3.Enums().ByName("LP3SiblingEnum"),
	}, {
		got:  loadEnumDesc(reflect.TypeOf(LP3Message_LP3ChildEnum(0))),
		want: fileDescLP3.Messages().ByName("LP3Message").Enums().ByName("LP3ChildEnum"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP3Message))),
		want: fileDescLP3.Messages().ByName("LP3Message"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP3Message_LP3ChildMessage))),
		want: fileDescLP3.Messages().ByName("LP3Message").Messages().ByName("LP3ChildMessage"),
	}, {
		got:  loadMessageDesc(reflect.TypeOf(new(LP3SiblingMessage))),
		want: fileDescLP3.Messages().ByName("LP3SiblingMessage"),
	}}

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
					case "Messages", "Enums":
						// Ignore nested message and enum declarations since
						// legacy descriptors are all created standalone.
					case "OneofType", "ExtendedType", "MessageType", "EnumType":
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
