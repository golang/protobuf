// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	"github.com/golang/protobuf/v2/internal/encoding/pack"
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

func TestLegacyDescriptor(t *testing.T) {
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
					case "Options":
						// Ignore descriptor options since protos are not cmperable.
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

	joinRaw := func(bs ...pref.RawFields) (out []byte) {
		for _, b := range bs {
			out = append(out, b...)
		}
		return out
	}

	var fs legacyUnknownBytes
	if got, want := fs.Len(), 0; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := []byte(fs), joinRaw(); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	fs.Set(1, raw1a)
	fs.Set(1, append(fs.Get(1), raw1b...))
	fs.Set(1, append(fs.Get(1), raw1c...))
	if got, want := fs.Len(), 1; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := []byte(fs), joinRaw(raw1a, raw1b, raw1c); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	fs.Set(2, raw2a)
	if got, want := fs.Len(), 2; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := []byte(fs), joinRaw(raw1a, raw1b, raw1c, raw2a); !bytes.Equal(got, want) {
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
	if got, want := []byte(fs), joinRaw(raw2a); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}

	// Simulate manual appending of raw field data.
	fs = append(fs, joinRaw(raw3a, raw1a, raw1b, raw2b, raw3b, raw1c)...)
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
	if got, want := []byte(fs), joinRaw(raw3a, raw1a, raw1b, raw3b, raw1c, raw2a, raw2b); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}
	fs.Set(1, nil) // remove field 1
	if got, want := fs.Len(), 2; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := []byte(fs), joinRaw(raw3a, raw3b, raw2a, raw2b); !bytes.Equal(got, want) {
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
	if got, want := []byte(fs), joinRaw(); !bytes.Equal(got, want) {
		t.Errorf("data mismatch:\ngot:  %x\nwant: %x", got, want)
	}
}
