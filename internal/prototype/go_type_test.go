package prototype_test

import (
	"fmt"
	"reflect"
	"testing"

	"google.golang.org/protobuf/internal/prototype"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	testpb "google.golang.org/protobuf/internal/testprotos/test"
)

func TestGoEnum(t *testing.T) {
	enumDescs := []protoreflect.EnumDescriptor{
		testpb.ForeignEnum(0).Descriptor(),
		testpb.TestAllTypes_NestedEnum(0).Descriptor(),
	}
	for _, ed := range enumDescs {
		et := prototype.GoEnum(ed, newEnum)
		if gotED := et.Descriptor(); gotED != ed {
			fmt.Errorf("GoEnum(ed (%v), newEnum).Descriptor() != ed", ed.FullName())
		}
		e := et.New(0)
		if gotED := e.Descriptor(); gotED != ed {
			fmt.Errorf("GoEnum(ed (%v), newEnum).New(0).Descriptor() != ed", ed.FullName())
		}
		if n := e.Number(); n != 0 {
			fmt.Errorf("GoEnum(ed (%v), newEnum).New(0).Number() = %v; want 0", ed.FullName(), n)
		}
		if _, ok := e.(fakeEnum); !ok {
			fmt.Errorf("GoEnum(ed (%v), newEnum).New(0) type is %T; want fakeEnum", ed.FullName(), e)
		}
	}
}

func TestGoMessage(t *testing.T) {
	msgDescs := []protoreflect.MessageDescriptor{
		((*testpb.TestAllTypes)(nil)).ProtoReflect().Descriptor(),
		((*testpb.TestAllTypes_NestedMessage)(nil)).ProtoReflect().Descriptor(),
	}
	for _, md := range msgDescs {
		mt := prototype.GoMessage(md, newMessage)
		if gotMD := mt.Descriptor(); gotMD != md {
			fmt.Errorf("GoMessage(md (%v), newMessage).Descriptor() != md", md.FullName())
		}
		m := mt.New()
		if gotMD := m.Descriptor(); gotMD != md {
			fmt.Errorf("GoMessage(md (%v), newMessage).New().Descriptor() != md", md.FullName())
		}
		if _, ok := m.(*fakeMessage); !ok {
			fmt.Errorf("GoMessage(md (%v), newMessage).New() type is %T; want *fakeMessage", md.FullName(), m)
		}
	}
}

func TestGoExtension(t *testing.T) {
	testCases := []struct {
		extName     protoreflect.FullName
		wantNewType reflect.Type
	}{{
		extName:     "goproto.proto.test.optional_int32_extension",
		wantNewType: reflect.TypeOf(int32(0)),
	}, {
		extName:     "goproto.proto.test.optional_string_extension",
		wantNewType: reflect.TypeOf(""),
	}, {
		extName:     "goproto.proto.test.repeated_int32_extension",
		wantNewType: reflect.TypeOf((*[]int32)(nil)),
	}, {
		extName:     "goproto.proto.test.repeated_string_extension",
		wantNewType: reflect.TypeOf((*[]string)(nil)),
	}, {
		extName:     "goproto.proto.test.repeated_string_extension",
		wantNewType: reflect.TypeOf((*[]string)(nil)),
	}, {
		extName:     "goproto.proto.test.optional_nested_enum_extension",
		wantNewType: reflect.TypeOf((*fakeEnum)(nil)).Elem(),
	}, {
		extName:     "goproto.proto.test.optional_nested_message_extension",
		wantNewType: reflect.TypeOf((*fakeMessageImpl)(nil)),
	}, {
		extName:     "goproto.proto.test.repeated_nested_enum_extension",
		wantNewType: reflect.TypeOf((*[]fakeEnum)(nil)),
	}, {
		extName:     "goproto.proto.test.repeated_nested_message_extension",
		wantNewType: reflect.TypeOf((*[]*fakeMessageImpl)(nil)),
	}}
	for _, tc := range testCases {
		xd, err := protoregistry.GlobalFiles.FindExtensionByName(tc.extName)
		if err != nil {
			t.Errorf("GlobalFiles.FindExtensionByName(%q) = _, %v; want _, <nil>", tc.extName, err)
			continue
		}
		var et protoreflect.EnumType
		if ed := xd.Enum(); ed != nil {
			et = prototype.GoEnum(ed, newEnum)
		}
		var mt protoreflect.MessageType
		if md := xd.Message(); md != nil {
			mt = prototype.GoMessage(md, newMessage)
		}
		xt := prototype.GoExtension(xd, et, mt)
		v := xt.InterfaceOf(xt.New())
		if typ := reflect.TypeOf(v); typ != tc.wantNewType {
			t.Errorf("GoExtension(xd (%v), et, mt).New() type unwraps to %v; want %v", tc.extName, typ, tc.wantNewType)
		}
	}
}

type fakeMessage struct {
	imp *fakeMessageImpl
	protoreflect.Message
}

func (m *fakeMessage) Type() protoreflect.MessageType             { return m.imp.typ }
func (m *fakeMessage) Descriptor() protoreflect.MessageDescriptor { return m.imp.typ.Descriptor() }
func (m *fakeMessage) Interface() protoreflect.ProtoMessage       { return m.imp }

type fakeMessageImpl struct{ typ protoreflect.MessageType }

func (m *fakeMessageImpl) ProtoReflect() protoreflect.Message { return &fakeMessage{imp: m} }

func newMessage(typ protoreflect.MessageType) protoreflect.Message {
	return (&fakeMessageImpl{typ: typ}).ProtoReflect()
}

type fakeEnum struct {
	typ protoreflect.EnumType
	num protoreflect.EnumNumber
}

func (e fakeEnum) Descriptor() protoreflect.EnumDescriptor { return e.typ.Descriptor() }
func (e fakeEnum) Type() protoreflect.EnumType             { return e.typ }
func (e fakeEnum) Number() protoreflect.EnumNumber         { return e.num }

func newEnum(typ protoreflect.EnumType, num protoreflect.EnumNumber) protoreflect.Enum {
	return fakeEnum{
		typ: typ,
		num: num,
	}
}
