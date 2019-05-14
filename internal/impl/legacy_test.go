// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl_test

import (
	"reflect"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pimpl "google.golang.org/protobuf/internal/impl"
	pragma "google.golang.org/protobuf/internal/pragma"
	ptype "google.golang.org/protobuf/internal/prototype"
	"google.golang.org/protobuf/internal/scalar"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
	piface "google.golang.org/protobuf/runtime/protoiface"

	proto2_20180125 "google.golang.org/protobuf/internal/testprotos/legacy/proto2.v1.0.0-20180125-92554152"
)

type legacyTestMessage struct {
	XXX_unrecognized       []byte
	XXX_InternalExtensions map[int32]pimpl.ExtensionField
}

func (*legacyTestMessage) Reset()         {}
func (*legacyTestMessage) String() string { return "" }
func (*legacyTestMessage) ProtoMessage()  {}
func (*legacyTestMessage) ExtensionRangeArray() []piface.ExtensionRangeV1 {
	return []piface.ExtensionRangeV1{{Start: 10, End: 20}, {Start: 40, End: 80}, {Start: 10000, End: 20000}}
}

func init() {
	mt := pimpl.Export{}.MessageTypeOf(&legacyTestMessage{})
	preg.GlobalTypes.Register(mt)
}

var (
	testParentDesc    = pimpl.Export{}.MessageDescriptorOf((*legacyTestMessage)(nil))
	testEnumV1Desc    = pimpl.Export{}.EnumDescriptorOf(proto2_20180125.Message_ChildEnum(0))
	testMessageV1Desc = pimpl.Export{}.MessageDescriptorOf((*proto2_20180125.Message_ChildMessage)(nil))
	testEnumV2Desc    = enumProto2Type.Descriptor()
	testMessageV2Desc = enumMessagesType.PBType.Descriptor()

	extensionTypes = []pref.ExtensionType{
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_bool",
			Number:       10000,
			Cardinality:  pref.Optional,
			Kind:         pref.BoolKind,
			Default:      pref.ValueOf(true),
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_int32",
			Number:       10001,
			Cardinality:  pref.Optional,
			Kind:         pref.Int32Kind,
			Default:      pref.ValueOf(int32(-12345)),
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_uint32",
			Number:       10002,
			Cardinality:  pref.Optional,
			Kind:         pref.Uint32Kind,
			Default:      pref.ValueOf(uint32(3200)),
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_float",
			Number:       10003,
			Cardinality:  pref.Optional,
			Kind:         pref.FloatKind,
			Default:      pref.ValueOf(float32(3.14159)),
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_string",
			Number:       10004,
			Cardinality:  pref.Optional,
			Kind:         pref.StringKind,
			Default:      pref.ValueOf(string("hello, \"world!\"\n")),
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_bytes",
			Number:       10005,
			Cardinality:  pref.Optional,
			Kind:         pref.BytesKind,
			Default:      pref.ValueOf([]byte("dead\xde\xad\xbe\xefbeef")),
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_enum_v1",
			Number:       10006,
			Cardinality:  pref.Optional,
			Kind:         pref.EnumKind,
			Default:      pref.ValueOf(pref.EnumNumber(0)),
			EnumType:     testEnumV1Desc,
			ExtendedType: testParentDesc,
		}, proto2_20180125.Message_ChildEnum(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_message_v1",
			Number:       10007,
			Cardinality:  pref.Optional,
			Kind:         pref.MessageKind,
			MessageType:  testMessageV1Desc,
			ExtendedType: testParentDesc,
		}, (*proto2_20180125.Message_ChildMessage)(nil)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_enum_v2",
			Number:       10008,
			Cardinality:  pref.Optional,
			Kind:         pref.EnumKind,
			Default:      pref.ValueOf(pref.EnumNumber(57005)),
			EnumType:     testEnumV2Desc,
			ExtendedType: testParentDesc,
		}, EnumProto2(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.optional_message_v2",
			Number:       10009,
			Cardinality:  pref.Optional,
			Kind:         pref.MessageKind,
			MessageType:  testMessageV2Desc,
			ExtendedType: testParentDesc,
		}, (*EnumMessages)(nil)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_bool",
			Number:       10010,
			Cardinality:  pref.Repeated,
			Kind:         pref.BoolKind,
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_int32",
			Number:       10011,
			Cardinality:  pref.Repeated,
			Kind:         pref.Int32Kind,
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_uint32",
			Number:       10012,
			Cardinality:  pref.Repeated,
			Kind:         pref.Uint32Kind,
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_float",
			Number:       10013,
			Cardinality:  pref.Repeated,
			Kind:         pref.FloatKind,
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_string",
			Number:       10014,
			Cardinality:  pref.Repeated,
			Kind:         pref.StringKind,
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_bytes",
			Number:       10015,
			Cardinality:  pref.Repeated,
			Kind:         pref.BytesKind,
			ExtendedType: testParentDesc,
		}, nil),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_enum_v1",
			Number:       10016,
			Cardinality:  pref.Repeated,
			Kind:         pref.EnumKind,
			EnumType:     testEnumV1Desc,
			ExtendedType: testParentDesc,
		}, proto2_20180125.Message_ChildEnum(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_message_v1",
			Number:       10017,
			Cardinality:  pref.Repeated,
			Kind:         pref.MessageKind,
			MessageType:  testMessageV1Desc,
			ExtendedType: testParentDesc,
		}, (*proto2_20180125.Message_ChildMessage)(nil)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_enum_v2",
			Number:       10018,
			Cardinality:  pref.Repeated,
			Kind:         pref.EnumKind,
			EnumType:     testEnumV2Desc,
			ExtendedType: testParentDesc,
		}, EnumProto2(0)),
		mustMakeExtensionType(&ptype.StandaloneExtension{
			FullName:     "fizz.buzz.repeated_message_v2",
			Number:       10019,
			Cardinality:  pref.Repeated,
			Kind:         pref.MessageKind,
			MessageType:  testMessageV2Desc,
			ExtendedType: testParentDesc,
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

	m := pimpl.Export{}.MessageOf(new(legacyTestMessage))

	if n := m.Len(); n != 0 {
		t.Errorf("KnownFields.Len() = %v, want 0", n)
	}

	// Check that getting the zero value returns the default value for scalars,
	// nil for singular messages, and an empty list for repeated fields.
	defaultValues := map[int]interface{}{
		0: bool(true),
		1: int32(-12345),
		2: uint32(3200),
		3: float32(3.14159),
		4: string("hello, \"world!\"\n"),
		5: []byte("dead\xde\xad\xbe\xefbeef"),
		6: proto2_20180125.Message_ALPHA,
		7: nil,
		8: EnumProto2(0xdead),
		9: nil,
	}
	for i, xt := range extensionTypes {
		var got interface{}
		if !(xt.IsList() || xt.IsMap() || xt.Message() != nil) {
			got = xt.InterfaceOf(m.Get(xt))
		}
		want := defaultValues[i]
		if diff := cmp.Diff(want, got, opts); diff != "" {
			t.Errorf("Message.Get(%d) mismatch (-want +got):\n%v", xt.Number(), diff)
		}
	}

	// All fields should be unpopulated.
	for _, xt := range extensionTypes {
		if m.Has(xt) {
			t.Errorf("Message.Has(%d) = true, want false", xt.Number())
		}
	}

	// Set some values and append to values to the lists.
	m1a := &proto2_20180125.Message_ChildMessage{F1: scalar.String("m1a")}
	m1b := &proto2_20180125.Message_ChildMessage{F1: scalar.String("m2b")}
	m2a := &EnumMessages{EnumP2: EnumProto2(0x1b).Enum()}
	m2b := &EnumMessages{EnumP2: EnumProto2(0x2b).Enum()}
	setValues := map[int]interface{}{
		0:  bool(false),
		1:  int32(-54321),
		2:  uint32(6400),
		3:  float32(2.71828),
		4:  string("goodbye, \"world!\"\n"),
		5:  []byte("live\xde\xad\xbe\xefchicken"),
		6:  proto2_20180125.Message_CHARLIE,
		7:  m1a,
		8:  EnumProto2(0xbeef),
		9:  m2a,
		10: &[]bool{true},
		11: &[]int32{-1000},
		12: &[]uint32{1280},
		13: &[]float32{1.6180},
		14: &[]string{"zero"},
		15: &[][]byte{[]byte("zero")},
		16: &[]proto2_20180125.Message_ChildEnum{proto2_20180125.Message_BRAVO},
		17: &[]*proto2_20180125.Message_ChildMessage{m1b},
		18: &[]EnumProto2{0xdead},
		19: &[]*EnumMessages{m2b},
	}
	for i, xt := range extensionTypes {
		m.Set(xt, xt.ValueOf(setValues[i]))
	}
	for i, xt := range extensionTypes[len(extensionTypes)/2:] {
		v := extensionTypes[i].ValueOf(setValues[i])
		m.Get(xt).List().Append(v)
	}

	// Get the values and check for equality.
	getValues := map[int]interface{}{
		0:  bool(false),
		1:  int32(-54321),
		2:  uint32(6400),
		3:  float32(2.71828),
		4:  string("goodbye, \"world!\"\n"),
		5:  []byte("live\xde\xad\xbe\xefchicken"),
		6:  proto2_20180125.Message_ChildEnum(proto2_20180125.Message_CHARLIE),
		7:  m1a,
		8:  EnumProto2(0xbeef),
		9:  m2a,
		10: &[]bool{true, false},
		11: &[]int32{-1000, -54321},
		12: &[]uint32{1280, 6400},
		13: &[]float32{1.6180, 2.71828},
		14: &[]string{"zero", "goodbye, \"world!\"\n"},
		15: &[][]byte{[]byte("zero"), []byte("live\xde\xad\xbe\xefchicken")},
		16: &[]proto2_20180125.Message_ChildEnum{proto2_20180125.Message_BRAVO, proto2_20180125.Message_CHARLIE},
		17: &[]*proto2_20180125.Message_ChildMessage{m1b, m1a},
		18: &[]EnumProto2{0xdead, 0xbeef},
		19: &[]*EnumMessages{m2b, m2a},
	}
	for i, xt := range extensionTypes {
		got := xt.InterfaceOf(m.Get(xt))
		want := getValues[i]
		if diff := cmp.Diff(want, got, opts); diff != "" {
			t.Errorf("Message.Get(%d) mismatch (-want +got):\n%v", xt.Number(), diff)
		}
	}

	if n := m.Len(); n != 20 {
		t.Errorf("Message.Len() = %v, want 0", n)
	}

	// Clear all singular fields and truncate all repeated fields.
	for _, xt := range extensionTypes[:len(extensionTypes)/2] {
		m.Clear(xt)
	}
	for _, xt := range extensionTypes[len(extensionTypes)/2:] {
		m.Get(xt).List().Truncate(0)
	}
	if n := m.Len(); n != 10 {
		t.Errorf("Message.Len() = %v, want 10", n)
	}

	// Clear all repeated fields.
	for _, xt := range extensionTypes[len(extensionTypes)/2:] {
		m.Clear(xt)
	}
	if n := m.Len(); n != 0 {
		t.Errorf("Message.Len() = %v, want 0", n)
	}
}

func TestExtensionConvert(t *testing.T) {
	for i := range extensionTypes {
		i := i
		t.Run("", func(t *testing.T) {
			t.Parallel()

			wantType := extensionTypes[i]
			wantDesc := extensionDescs[i]
			gotType := pimpl.Export{}.ExtensionTypeFromDesc(wantDesc)
			gotDesc := pimpl.Export{}.ExtensionDescFromType(wantType)

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
				// TODO: Add this when ExtensionType no longer implements
				// ExtensionDescriptor.
				/*
					cmp.Transformer("", func(x pref.ExtensionType) pref.ExtensionDescriptor {
						return x.Descriptor()
					}),
				*/
				cmp.Transformer("", func(x pref.Descriptor) map[string]interface{} {
					out := make(map[string]interface{})
					v := reflect.ValueOf(x)
					for i := 0; i < v.NumMethod(); i++ {
						name := v.Type().Method(i).Name
						if m := v.Method(i); m.Type().NumIn() == 0 && m.Type().NumOut() == 1 {
							switch name {
							case "ParentFile", "Parent":
							// Ignore parents to avoid recursive cycle.
							case "New":
								// Ignore New since it a constructor.
							case "Options":
								// Ignore descriptor options since protos are not cmperable.
							case "ContainingOneof", "ContainingMessage", "Enum", "Message":
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

type (
	MessageA struct {
		A1 *MessageA `protobuf:"bytes,1,req,name=a1"`
		A2 *MessageB `protobuf:"bytes,2,req,name=a2"`
		A3 Enum      `protobuf:"varint,3,opt,name=a3,enum=legacy.Enum"`
	}
	MessageB struct {
		B1 *MessageA `protobuf:"bytes,1,req,name=b1"`
		B2 *MessageB `protobuf:"bytes,2,req,name=b2"`
		B3 Enum      `protobuf:"varint,3,opt,name=b3,enum=legacy.Enum"`
	}
	Enum int32
)

// TestConcurrentInit tests that concurrent wrapping of multiple legacy types
// results in the exact same descriptor being created.
func TestConcurrentInit(t *testing.T) {
	const numParallel = 5
	var messageATypes [numParallel]pref.MessageType
	var messageBTypes [numParallel]pref.MessageType
	var enumDescs [numParallel]pref.EnumDescriptor

	// Concurrently load message and enum types.
	var wg sync.WaitGroup
	for i := 0; i < numParallel; i++ {
		i := i
		wg.Add(3)
		go func() {
			defer wg.Done()
			messageATypes[i] = pimpl.Export{}.MessageTypeOf((*MessageA)(nil))
		}()
		go func() {
			defer wg.Done()
			messageBTypes[i] = pimpl.Export{}.MessageTypeOf((*MessageB)(nil))
		}()
		go func() {
			defer wg.Done()
			enumDescs[i] = pimpl.Export{}.EnumDescriptorOf(Enum(0))
		}()
	}
	wg.Wait()

	var (
		wantMTA = messageATypes[0]
		wantMDA = messageATypes[0].Descriptor().Fields().ByNumber(1).Message()
		wantMTB = messageBTypes[0]
		wantMDB = messageBTypes[0].Descriptor().Fields().ByNumber(2).Message()
		wantED  = messageATypes[0].Descriptor().Fields().ByNumber(3).Enum()
	)

	for _, gotMT := range messageATypes[1:] {
		if gotMT != wantMTA {
			t.Error("MessageType(MessageA) mismatch")
		}
		if gotMDA := gotMT.Descriptor().Fields().ByNumber(1).Message(); gotMDA != wantMDA {
			t.Error("MessageDescriptor(MessageA) mismatch")
		}
		if gotMDB := gotMT.Descriptor().Fields().ByNumber(2).Message(); gotMDB != wantMDB {
			t.Error("MessageDescriptor(MessageB) mismatch")
		}
		if gotED := gotMT.Descriptor().Fields().ByNumber(3).Enum(); gotED != wantED {
			t.Error("EnumDescriptor(Enum) mismatch")
		}
	}
	for _, gotMT := range messageBTypes[1:] {
		if gotMT != wantMTB {
			t.Error("MessageType(MessageB) mismatch")
		}
		if gotMDA := gotMT.Descriptor().Fields().ByNumber(1).Message(); gotMDA != wantMDA {
			t.Error("MessageDescriptor(MessageA) mismatch")
		}
		if gotMDB := gotMT.Descriptor().Fields().ByNumber(2).Message(); gotMDB != wantMDB {
			t.Error("MessageDescriptor(MessageB) mismatch")
		}
		if gotED := gotMT.Descriptor().Fields().ByNumber(3).Enum(); gotED != wantED {
			t.Error("EnumDescriptor(Enum) mismatch")
		}
	}
	for _, gotED := range enumDescs[1:] {
		if gotED != wantED {
			t.Error("EnumType(Enum) mismatch")
		}
	}
}
