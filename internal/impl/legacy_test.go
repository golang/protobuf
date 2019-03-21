// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl_test

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	pack "github.com/golang/protobuf/v2/internal/encoding/pack"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	pragma "github.com/golang/protobuf/v2/internal/pragma"
	ptype "github.com/golang/protobuf/v2/internal/prototype"
	scalar "github.com/golang/protobuf/v2/internal/scalar"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	piface "github.com/golang/protobuf/v2/runtime/protoiface"
	cmp "github.com/google/go-cmp/cmp"
	cmpopts "github.com/google/go-cmp/cmp/cmpopts"

	// The legacy package must be imported prior to use of any legacy messages.
	// TODO: Remove this when protoV1 registers these hooks for you.
	plegacy "github.com/golang/protobuf/v2/internal/legacy"

	proto2_20180125 "github.com/golang/protobuf/v2/internal/testprotos/legacy/proto2.v1.0.0-20180125-92554152"
)

type legacyTestMessage struct {
	XXX_unrecognized       []byte
	XXX_InternalExtensions pimpl.ExtensionFieldsV1
}

func (*legacyTestMessage) Reset()         {}
func (*legacyTestMessage) String() string { return "" }
func (*legacyTestMessage) ProtoMessage()  {}
func (*legacyTestMessage) ExtensionRangeArray() []piface.ExtensionRangeV1 {
	return []piface.ExtensionRangeV1{{Start: 10, End: 20}, {Start: 40, End: 80}, {Start: 10000, End: 20000}}
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

	m := new(legacyTestMessage)
	fs := pimpl.Export{}.MessageOf(m).UnknownFields()

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

func mustMakeExtensionType(x *ptype.StandaloneExtension, v interface{}) pref.ExtensionType {
	xd, err := ptype.NewExtension(x)
	if err != nil {
		panic(xd)
	}
	return pimpl.Export{}.ExtensionTypeOf(xd, v)
}

var (
	parentType    = pimpl.Export{}.MessageTypeOf((*legacyTestMessage)(nil))
	enumV1Type    = pimpl.Export{}.EnumTypeOf(proto2_20180125.Message_ChildEnum(0))
	messageV1Type = pimpl.Export{}.MessageTypeOf((*proto2_20180125.Message_ChildMessage)(nil))
	enumV2Type    = enumProto2Type
	messageV2Type = enumMessagesType.PBType

	extensionTypes = []pref.ExtensionType{
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_bool",
			Number:       10000,
			Cardinality:  pref.Optional,
			Kind:         pref.BoolKind,
			Default:      pref.ValueOf(true),
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_int32",
			Number:       10001,
			Cardinality:  pref.Optional,
			Kind:         pref.Int32Kind,
			Default:      pref.ValueOf(int32(-12345)),
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_uint32",
			Number:       10002,
			Cardinality:  pref.Optional,
			Kind:         pref.Uint32Kind,
			Default:      pref.ValueOf(uint32(3200)),
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_float",
			Number:       10003,
			Cardinality:  pref.Optional,
			Kind:         pref.FloatKind,
			Default:      pref.ValueOf(float32(3.14159)),
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_string",
			Number:       10004,
			Cardinality:  pref.Optional,
			Kind:         pref.StringKind,
			Default:      pref.ValueOf(string("hello, \"world!\"\n")),
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_bytes",
			Number:       10005,
			Cardinality:  pref.Optional,
			Kind:         pref.BytesKind,
			Default:      pref.ValueOf([]byte("dead\xde\xad\xbe\xefbeef")),
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_enum_v1",
			Number:       10006,
			Cardinality:  pref.Optional,
			Kind:         pref.EnumKind,
			Default:      pref.ValueOf(pref.EnumNumber(0)),
			EnumType:     enumV1Type,
			ExtendedType: parentType,
		}, proto2_20180125.Message_ChildEnum(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_message_v1",
			Number:       10007,
			Cardinality:  pref.Optional,
			Kind:         pref.MessageKind,
			MessageType:  messageV1Type,
			ExtendedType: parentType,
		}, (*proto2_20180125.Message_ChildMessage)(nil)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_enum_v2",
			Number:       10008,
			Cardinality:  pref.Optional,
			Kind:         pref.EnumKind,
			Default:      pref.ValueOf(pref.EnumNumber(57005)),
			EnumType:     enumV2Type,
			ExtendedType: parentType,
		}, EnumProto2(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_message_v2",
			Number:       10009,
			Cardinality:  pref.Optional,
			Kind:         pref.MessageKind,
			MessageType:  messageV2Type,
			ExtendedType: parentType,
		}, (*EnumMessages)(nil)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_bool",
			Number:       10010,
			Cardinality:  pref.Repeated,
			Kind:         pref.BoolKind,
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_int32",
			Number:       10011,
			Cardinality:  pref.Repeated,
			Kind:         pref.Int32Kind,
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_uint32",
			Number:       10012,
			Cardinality:  pref.Repeated,
			Kind:         pref.Uint32Kind,
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_float",
			Number:       10013,
			Cardinality:  pref.Repeated,
			Kind:         pref.FloatKind,
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_string",
			Number:       10014,
			Cardinality:  pref.Repeated,
			Kind:         pref.StringKind,
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_bytes",
			Number:       10015,
			Cardinality:  pref.Repeated,
			Kind:         pref.BytesKind,
			ExtendedType: parentType,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_enum_v1",
			Number:       10016,
			Cardinality:  pref.Repeated,
			Kind:         pref.EnumKind,
			EnumType:     enumV1Type,
			ExtendedType: parentType,
		}, proto2_20180125.Message_ChildEnum(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_message_v1",
			Number:       10017,
			Cardinality:  pref.Repeated,
			Kind:         pref.MessageKind,
			MessageType:  messageV1Type,
			ExtendedType: parentType,
		}, (*proto2_20180125.Message_ChildMessage)(nil)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_enum_v2",
			Number:       10018,
			Cardinality:  pref.Repeated,
			Kind:         pref.EnumKind,
			EnumType:     enumV2Type,
			ExtendedType: parentType,
		}, EnumProto2(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_message_v2",
			Number:       10019,
			Cardinality:  pref.Repeated,
			Kind:         pref.MessageKind,
			MessageType:  messageV2Type,
			ExtendedType: parentType,
		}, (*EnumMessages)(nil)),
	}

	extensionDescs = []*piface.ExtensionDescV1{{
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*bool)(nil),
		Field:         10000,
		Name:          "fizz.buzz.optional_bool",
		Tag:           "varint,10000,opt,name=optional_bool,def=1",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*int32)(nil),
		Field:         10001,
		Name:          "fizz.buzz.optional_int32",
		Tag:           "varint,10001,opt,name=optional_int32,def=-12345",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*uint32)(nil),
		Field:         10002,
		Name:          "fizz.buzz.optional_uint32",
		Tag:           "varint,10002,opt,name=optional_uint32,def=3200",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*float32)(nil),
		Field:         10003,
		Name:          "fizz.buzz.optional_float",
		Tag:           "fixed32,10003,opt,name=optional_float,def=3.14159",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*string)(nil),
		Field:         10004,
		Name:          "fizz.buzz.optional_string",
		Tag:           "bytes,10004,opt,name=optional_string,def=hello, \"world!\"\n",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]byte)(nil),
		Field:         10005,
		Name:          "fizz.buzz.optional_bytes",
		Tag:           "bytes,10005,opt,name=optional_bytes,def=dead\\336\\255\\276\\357beef",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*proto2_20180125.Message_ChildEnum)(nil),
		Field:         10006,
		Name:          "fizz.buzz.optional_enum_v1",
		Tag:           "varint,10006,opt,name=optional_enum_v1,enum=google.golang.org.proto2_20180125.Message_ChildEnum,def=0",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*proto2_20180125.Message_ChildMessage)(nil),
		Field:         10007,
		Name:          "fizz.buzz.optional_message_v1",
		Tag:           "bytes,10007,opt,name=optional_message_v1",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*EnumProto2)(nil),
		Field:         10008,
		Name:          "fizz.buzz.optional_enum_v2",
		Tag:           "varint,10008,opt,name=optional_enum_v2,enum=EnumProto2,def=57005",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: (*EnumMessages)(nil),
		Field:         10009,
		Name:          "fizz.buzz.optional_message_v2",
		Tag:           "bytes,10009,opt,name=optional_message_v2",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]bool)(nil),
		Field:         10010,
		Name:          "fizz.buzz.repeated_bool",
		Tag:           "varint,10010,rep,name=repeated_bool",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]int32)(nil),
		Field:         10011,
		Name:          "fizz.buzz.repeated_int32",
		Tag:           "varint,10011,rep,name=repeated_int32",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]uint32)(nil),
		Field:         10012,
		Name:          "fizz.buzz.repeated_uint32",
		Tag:           "varint,10012,rep,name=repeated_uint32",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]float32)(nil),
		Field:         10013,
		Name:          "fizz.buzz.repeated_float",
		Tag:           "fixed32,10013,rep,name=repeated_float",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]string)(nil),
		Field:         10014,
		Name:          "fizz.buzz.repeated_string",
		Tag:           "bytes,10014,rep,name=repeated_string",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([][]byte)(nil),
		Field:         10015,
		Name:          "fizz.buzz.repeated_bytes",
		Tag:           "bytes,10015,rep,name=repeated_bytes",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]proto2_20180125.Message_ChildEnum)(nil),
		Field:         10016,
		Name:          "fizz.buzz.repeated_enum_v1",
		Tag:           "varint,10016,rep,name=repeated_enum_v1,enum=google.golang.org.proto2_20180125.Message_ChildEnum",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]*proto2_20180125.Message_ChildMessage)(nil),
		Field:         10017,
		Name:          "fizz.buzz.repeated_message_v1",
		Tag:           "bytes,10017,rep,name=repeated_message_v1",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]EnumProto2)(nil),
		Field:         10018,
		Name:          "fizz.buzz.repeated_enum_v2",
		Tag:           "varint,10018,rep,name=repeated_enum_v2,enum=EnumProto2",
	}, {
		ExtendedType:  (*legacyTestMessage)(nil),
		ExtensionType: ([]*EnumMessages)(nil),
		Field:         10019,
		Name:          "fizz.buzz.repeated_message_v2",
		Tag:           "bytes,10019,rep,name=repeated_message_v2",
	}}
)

func TestLegacyExtensions(t *testing.T) {
	opts := cmp.Options{cmp.Comparer(func(x, y *proto2_20180125.Message_ChildMessage) bool {
		return x == y // pointer compare messages for object identity
	})}

	m := new(legacyTestMessage)
	fs := pimpl.Export{}.MessageOf(m).KnownFields()
	ts := fs.ExtensionTypes()

	if n := fs.Len(); n != 0 {
		t.Errorf("KnownFields.Len() = %v, want 0", n)
	}
	if n := ts.Len(); n != 0 {
		t.Errorf("ExtensionFieldTypes.Len() = %v, want 0", n)
	}

	// Register all the extension types.
	for _, xt := range extensionTypes {
		ts.Register(xt)
	}

	// Check that getting the zero value returns the default value for scalars,
	// nil for singular messages, and an empty list for repeated fields.
	defaultValues := []interface{}{
		bool(true),
		int32(-12345),
		uint32(3200),
		float32(3.14159),
		string("hello, \"world!\"\n"),
		[]byte("dead\xde\xad\xbe\xefbeef"),
		proto2_20180125.Message_ALPHA,
		nil,
		EnumProto2(0xdead),
		nil,
		new([]bool),
		new([]int32),
		new([]uint32),
		new([]float32),
		new([]string),
		new([][]byte),
		new([]proto2_20180125.Message_ChildEnum),
		new([]*proto2_20180125.Message_ChildMessage),
		new([]EnumProto2),
		new([]*EnumMessages),
	}
	for i, xt := range extensionTypes {
		var got interface{}
		if v := fs.Get(xt.Number()); v.IsValid() {
			got = xt.InterfaceOf(v)
		}
		want := defaultValues[i]
		if diff := cmp.Diff(want, got, opts); diff != "" {
			t.Errorf("KnownFields.Get(%d) mismatch (-want +got):\n%v", xt.Number(), diff)
		}
	}

	// All fields should be unpopulated.
	for _, xt := range extensionTypes {
		if fs.Has(xt.Number()) {
			t.Errorf("KnownFields.Has(%d) = true, want false", xt.Number())
		}
	}

	// Set some values and append to values to the lists.
	m1a := &proto2_20180125.Message_ChildMessage{F1: scalar.String("m1a")}
	m1b := &proto2_20180125.Message_ChildMessage{F1: scalar.String("m2b")}
	m2a := &EnumMessages{EnumP2: EnumProto2(0x1b).Enum()}
	m2b := &EnumMessages{EnumP2: EnumProto2(0x2b).Enum()}
	setValues := []interface{}{
		bool(false),
		int32(-54321),
		uint32(6400),
		float32(2.71828),
		string("goodbye, \"world!\"\n"),
		[]byte("live\xde\xad\xbe\xefchicken"),
		proto2_20180125.Message_CHARLIE,
		m1a,
		EnumProto2(0xbeef),
		m2a,
		&[]bool{true},
		&[]int32{-1000},
		&[]uint32{1280},
		&[]float32{1.6180},
		&[]string{"zero"},
		&[][]byte{[]byte("zero")},
		&[]proto2_20180125.Message_ChildEnum{proto2_20180125.Message_BRAVO},
		&[]*proto2_20180125.Message_ChildMessage{m1b},
		&[]EnumProto2{0xdead},
		&[]*EnumMessages{m2b},
	}
	for i, xt := range extensionTypes {
		fs.Set(xt.Number(), xt.ValueOf(setValues[i]))
	}
	for i, xt := range extensionTypes[len(extensionTypes)/2:] {
		v := extensionTypes[i].ValueOf(setValues[i])
		fs.Get(xt.Number()).List().Append(v)
	}

	// Get the values and check for equality.
	getValues := []interface{}{
		bool(false),
		int32(-54321),
		uint32(6400),
		float32(2.71828),
		string("goodbye, \"world!\"\n"),
		[]byte("live\xde\xad\xbe\xefchicken"),
		proto2_20180125.Message_ChildEnum(proto2_20180125.Message_CHARLIE),
		m1a,
		EnumProto2(0xbeef),
		m2a,
		&[]bool{true, false},
		&[]int32{-1000, -54321},
		&[]uint32{1280, 6400},
		&[]float32{1.6180, 2.71828},
		&[]string{"zero", "goodbye, \"world!\"\n"},
		&[][]byte{[]byte("zero"), []byte("live\xde\xad\xbe\xefchicken")},
		&[]proto2_20180125.Message_ChildEnum{proto2_20180125.Message_BRAVO, proto2_20180125.Message_CHARLIE},
		&[]*proto2_20180125.Message_ChildMessage{m1b, m1a},
		&[]EnumProto2{0xdead, 0xbeef},
		&[]*EnumMessages{m2b, m2a},
	}
	for i, xt := range extensionTypes {
		got := xt.InterfaceOf(fs.Get(xt.Number()))
		want := getValues[i]
		if diff := cmp.Diff(want, got, opts); diff != "" {
			t.Errorf("KnownFields.Get(%d) mismatch (-want +got):\n%v", xt.Number(), diff)
		}
	}

	if n := fs.Len(); n != 20 {
		t.Errorf("KnownFields.Len() = %v, want 0", n)
	}
	if n := ts.Len(); n != 20 {
		t.Errorf("ExtensionFieldTypes.Len() = %v, want 20", n)
	}

	// Clear the field for all extension types.
	for _, xt := range extensionTypes[:len(extensionTypes)/2] {
		fs.Clear(xt.Number())
	}
	for i, xt := range extensionTypes[len(extensionTypes)/2:] {
		if i%2 == 0 {
			fs.Clear(xt.Number())
		} else {
			fs.Get(xt.Number()).List().Truncate(0)
		}
	}
	if n := fs.Len(); n != 0 {
		t.Errorf("KnownFields.Len() = %v, want 0", n)
	}
	if n := ts.Len(); n != 20 {
		t.Errorf("ExtensionFieldTypes.Len() = %v, want 20", n)
	}

	// De-register all extension types.
	for _, xt := range extensionTypes {
		ts.Remove(xt)
	}
	if n := fs.Len(); n != 0 {
		t.Errorf("KnownFields.Len() = %v, want 0", n)
	}
	if n := ts.Len(); n != 0 {
		t.Errorf("ExtensionFieldTypes.Len() = %v, want 0", n)
	}
}

func TestExtensionConvert(t *testing.T) {
	for i := range extensionTypes {
		i := i
		t.Run("", func(t *testing.T) {
			t.Parallel()

			wantType := extensionTypes[i]
			wantDesc := extensionDescs[i]
			gotType := plegacy.Export{}.ExtensionTypeFromDesc(wantDesc)
			gotDesc := plegacy.Export{}.ExtensionDescFromType(wantType)

			// TODO: We need a test package to compare descriptors.
			type list interface {
				Len() int
				pragma.DoNotImplement
			}
			opts := cmp.Options{
				cmp.Comparer(func(x, y reflect.Type) bool {
					return x == y
				}),
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
							case "New":
								// Ignore New since it a constructor.
							case "Options":
								// Ignore descriptor options since protos are not cmperable.
							case "EnumType", "MessageType", "ExtendedType":
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
			if diff := cmp.Diff(&wantType, &gotType, opts); diff != "" {
				t.Errorf("ExtensionType mismatch (-want, +got):\n%v", diff)
			}

			opts = cmp.Options{
				cmpopts.IgnoreFields(piface.ExtensionDescV1{}, "Type"),
			}
			if diff := cmp.Diff(wantDesc, gotDesc, opts); diff != "" {
				t.Errorf("ExtensionDesc mismatch (-want, +got):\n%v", diff)
			}
		})
	}
}
