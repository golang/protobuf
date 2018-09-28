// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/protobuf/struct.proto

package structpb

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// `NullValue` is a singleton enumeration to represent the null value for the
// `Value` type union.
//
//  The JSON representation for `NullValue` is JSON `null`.
type NullValue int32

const (
	// Null value.
	NullValue_NULL_VALUE NullValue = 0
)

var NullValue_name = map[int32]string{
	0: "NULL_VALUE",
}

var NullValue_value = map[string]int32{
	"NULL_VALUE": 0,
}

func (x NullValue) String() string {
	return proto.EnumName(NullValue_name, int32(x))
}

func (NullValue) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_df322afd6c9fb402, []int{0}
}

func (NullValue) XXX_WellKnownType() string { return "NullValue" }

// `Struct` represents a structured data value, consisting of fields
// which map to dynamically typed values. In some languages, `Struct`
// might be supported by a native representation. For example, in
// scripting languages like JS a struct is represented as an
// object. The details of that representation are described together
// with the proto support for the language.
//
// The JSON representation for `Struct` is JSON object.
type Struct struct {
	// Unordered map of dynamically typed values.
	Fields               map[string]*Value `protobuf:"bytes,1,rep,name=fields,proto3" json:"fields,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Struct) Reset()         { *m = Struct{} }
func (m *Struct) String() string { return proto.CompactTextString(m) }
func (*Struct) ProtoMessage()    {}
func (*Struct) Descriptor() ([]byte, []int) {
	return fileDescriptor_df322afd6c9fb402, []int{0}
}

func (*Struct) XXX_WellKnownType() string { return "Struct" }

func (m *Struct) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Struct.Unmarshal(m, b)
}
func (m *Struct) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Struct.Marshal(b, m, deterministic)
}
func (m *Struct) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Struct.Merge(m, src)
}
func (m *Struct) XXX_Size() int {
	return xxx_messageInfo_Struct.Size(m)
}
func (m *Struct) XXX_DiscardUnknown() {
	xxx_messageInfo_Struct.DiscardUnknown(m)
}

var xxx_messageInfo_Struct proto.InternalMessageInfo

func (m *Struct) GetFields() map[string]*Value {
	if m != nil {
		return m.Fields
	}
	return nil
}

// `Value` represents a dynamically typed value which can be either
// null, a number, a string, a boolean, a recursive struct value, or a
// list of values. A producer of value is expected to set one of that
// variants, absence of any variant indicates an error.
//
// The JSON representation for `Value` is JSON value.
type Value struct {
	// The kind of value.
	//
	// Types that are valid to be assigned to Kind:
	//	*Value_NullValue
	//	*Value_NumberValue
	//	*Value_StringValue
	//	*Value_BoolValue
	//	*Value_StructValue
	//	*Value_ListValue
	Kind                 isValue_Kind `protobuf_oneof:"kind"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *Value) Reset()         { *m = Value{} }
func (m *Value) String() string { return proto.CompactTextString(m) }
func (*Value) ProtoMessage()    {}
func (*Value) Descriptor() ([]byte, []int) {
	return fileDescriptor_df322afd6c9fb402, []int{1}
}

func (*Value) XXX_WellKnownType() string { return "Value" }

func (m *Value) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Value.Unmarshal(m, b)
}
func (m *Value) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Value.Marshal(b, m, deterministic)
}
func (m *Value) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Value.Merge(m, src)
}
func (m *Value) XXX_Size() int {
	return xxx_messageInfo_Value.Size(m)
}
func (m *Value) XXX_DiscardUnknown() {
	xxx_messageInfo_Value.DiscardUnknown(m)
}

var xxx_messageInfo_Value proto.InternalMessageInfo

type isValue_Kind interface {
	isValue_Kind()
}

type Value_NullValue struct {
	NullValue NullValue `protobuf:"varint,1,opt,name=null_value,json=nullValue,proto3,enum=google.protobuf.NullValue,oneof"`
}

type Value_NumberValue struct {
	NumberValue float64 `protobuf:"fixed64,2,opt,name=number_value,json=numberValue,proto3,oneof"`
}

type Value_StringValue struct {
	StringValue string `protobuf:"bytes,3,opt,name=string_value,json=stringValue,proto3,oneof"`
}

type Value_BoolValue struct {
	BoolValue bool `protobuf:"varint,4,opt,name=bool_value,json=boolValue,proto3,oneof"`
}

type Value_StructValue struct {
	StructValue *Struct `protobuf:"bytes,5,opt,name=struct_value,json=structValue,proto3,oneof"`
}

type Value_ListValue struct {
	ListValue *ListValue `protobuf:"bytes,6,opt,name=list_value,json=listValue,proto3,oneof"`
}

func (*Value_NullValue) isValue_Kind() {}

func (*Value_NumberValue) isValue_Kind() {}

func (*Value_StringValue) isValue_Kind() {}

func (*Value_BoolValue) isValue_Kind() {}

func (*Value_StructValue) isValue_Kind() {}

func (*Value_ListValue) isValue_Kind() {}

func (m *Value) GetKind() isValue_Kind {
	if m != nil {
		return m.Kind
	}
	return nil
}

func (m *Value) GetNullValue() NullValue {
	if x, ok := m.GetKind().(*Value_NullValue); ok {
		return x.NullValue
	}
	return NullValue_NULL_VALUE
}

func (m *Value) GetNumberValue() float64 {
	if x, ok := m.GetKind().(*Value_NumberValue); ok {
		return x.NumberValue
	}
	return 0
}

func (m *Value) GetStringValue() string {
	if x, ok := m.GetKind().(*Value_StringValue); ok {
		return x.StringValue
	}
	return ""
}

func (m *Value) GetBoolValue() bool {
	if x, ok := m.GetKind().(*Value_BoolValue); ok {
		return x.BoolValue
	}
	return false
}

func (m *Value) GetStructValue() *Struct {
	if x, ok := m.GetKind().(*Value_StructValue); ok {
		return x.StructValue
	}
	return nil
}

func (m *Value) GetListValue() *ListValue {
	if x, ok := m.GetKind().(*Value_ListValue); ok {
		return x.ListValue
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Value) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Value_OneofMarshaler, _Value_OneofUnmarshaler, _Value_OneofSizer, []interface{}{
		(*Value_NullValue)(nil),
		(*Value_NumberValue)(nil),
		(*Value_StringValue)(nil),
		(*Value_BoolValue)(nil),
		(*Value_StructValue)(nil),
		(*Value_ListValue)(nil),
	}
}

func _Value_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Value)
	// kind
	switch x := m.Kind.(type) {
	case *Value_NullValue:
		b.EncodeVarint(1<<3 | proto.WireVarint)
		b.EncodeVarint(uint64(x.NullValue))
	case *Value_NumberValue:
		b.EncodeVarint(2<<3 | proto.WireFixed64)
		b.EncodeFixed64(math.Float64bits(x.NumberValue))
	case *Value_StringValue:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.StringValue)
	case *Value_BoolValue:
		t := uint64(0)
		if x.BoolValue {
			t = 1
		}
		b.EncodeVarint(4<<3 | proto.WireVarint)
		b.EncodeVarint(t)
	case *Value_StructValue:
		b.EncodeVarint(5<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.StructValue); err != nil {
			return err
		}
	case *Value_ListValue:
		b.EncodeVarint(6<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.ListValue); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("Value.Kind has unexpected type %T", x)
	}
	return nil
}

func _Value_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Value)
	switch tag {
	case 1: // kind.null_value
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.Kind = &Value_NullValue{NullValue(x)}
		return true, err
	case 2: // kind.number_value
		if wire != proto.WireFixed64 {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeFixed64()
		m.Kind = &Value_NumberValue{math.Float64frombits(x)}
		return true, err
	case 3: // kind.string_value
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Kind = &Value_StringValue{x}
		return true, err
	case 4: // kind.bool_value
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.Kind = &Value_BoolValue{x != 0}
		return true, err
	case 5: // kind.struct_value
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(Struct)
		err := b.DecodeMessage(msg)
		m.Kind = &Value_StructValue{msg}
		return true, err
	case 6: // kind.list_value
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(ListValue)
		err := b.DecodeMessage(msg)
		m.Kind = &Value_ListValue{msg}
		return true, err
	default:
		return false, nil
	}
}

func _Value_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Value)
	// kind
	switch x := m.Kind.(type) {
	case *Value_NullValue:
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(x.NullValue))
	case *Value_NumberValue:
		n += 1 // tag and wire
		n += 8
	case *Value_StringValue:
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(len(x.StringValue)))
		n += len(x.StringValue)
	case *Value_BoolValue:
		n += 1 // tag and wire
		n += 1
	case *Value_StructValue:
		s := proto.Size(x.StructValue)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case *Value_ListValue:
		s := proto.Size(x.ListValue)
		n += 1 // tag and wire
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

// `ListValue` is a wrapper around a repeated field of values.
//
// The JSON representation for `ListValue` is JSON array.
type ListValue struct {
	// Repeated field of dynamically typed values.
	Values               []*Value `protobuf:"bytes,1,rep,name=values,proto3" json:"values,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ListValue) Reset()         { *m = ListValue{} }
func (m *ListValue) String() string { return proto.CompactTextString(m) }
func (*ListValue) ProtoMessage()    {}
func (*ListValue) Descriptor() ([]byte, []int) {
	return fileDescriptor_df322afd6c9fb402, []int{2}
}

func (*ListValue) XXX_WellKnownType() string { return "ListValue" }

func (m *ListValue) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ListValue.Unmarshal(m, b)
}
func (m *ListValue) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ListValue.Marshal(b, m, deterministic)
}
func (m *ListValue) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListValue.Merge(m, src)
}
func (m *ListValue) XXX_Size() int {
	return xxx_messageInfo_ListValue.Size(m)
}
func (m *ListValue) XXX_DiscardUnknown() {
	xxx_messageInfo_ListValue.DiscardUnknown(m)
}

var xxx_messageInfo_ListValue proto.InternalMessageInfo

func (m *ListValue) GetValues() []*Value {
	if m != nil {
		return m.Values
	}
	return nil
}

func init() {
	proto.RegisterEnum("google.protobuf.NullValue", NullValue_name, NullValue_value)
	proto.RegisterType((*Struct)(nil), "google.protobuf.Struct")
	proto.RegisterMapType((map[string]*Value)(nil), "google.protobuf.Struct.FieldsEntry")
	proto.RegisterType((*Value)(nil), "google.protobuf.Value")
	proto.RegisterType((*ListValue)(nil), "google.protobuf.ListValue")
}

func init() { proto.RegisterFile("google/protobuf/struct.proto", fileDescriptor_df322afd6c9fb402) }

var fileDescriptor_df322afd6c9fb402 = []byte{
	// 417 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x92, 0x41, 0x8b, 0xd3, 0x40,
	0x14, 0xc7, 0x3b, 0xc9, 0x36, 0x98, 0x17, 0x59, 0x97, 0x11, 0xb4, 0xac, 0xa2, 0xa1, 0x7b, 0x09,
	0x22, 0x29, 0xd6, 0x8b, 0x18, 0x2f, 0x06, 0xd6, 0x5d, 0x30, 0x2c, 0x31, 0xba, 0x15, 0xbc, 0x94,
	0x26, 0x4d, 0x63, 0xe8, 0x74, 0x26, 0x24, 0x33, 0x4a, 0x8f, 0x7e, 0x0b, 0xcf, 0x1e, 0x3d, 0xfa,
	0xe9, 0x3c, 0xca, 0xcc, 0x24, 0xa9, 0xb4, 0xf4, 0x94, 0xbc, 0xf7, 0x7e, 0xef, 0x3f, 0xef, 0xff,
	0x66, 0xe0, 0x71, 0xc1, 0x58, 0x41, 0xf2, 0x49, 0x55, 0x33, 0xce, 0x52, 0xb1, 0x9a, 0x34, 0xbc,
	0x16, 0x19, 0xf7, 0x55, 0x8c, 0xef, 0xe9, 0xaa, 0xdf, 0x55, 0xc7, 0x3f, 0x11, 0x58, 0x1f, 0x15,
	0x81, 0x03, 0xb0, 0x56, 0x65, 0x4e, 0x96, 0xcd, 0x08, 0xb9, 0xa6, 0xe7, 0x4c, 0x2f, 0xfc, 0x3d,
	0xd8, 0xd7, 0xa0, 0xff, 0x4e, 0x51, 0x97, 0x94, 0xd7, 0xdb, 0xa4, 0x6d, 0x39, 0xff, 0x00, 0xce,
	0x7f, 0x69, 0x7c, 0x06, 0xe6, 0x3a, 0xdf, 0x8e, 0x90, 0x8b, 0x3c, 0x3b, 0x91, 0xbf, 0xf8, 0x39,
	0x0c, 0xbf, 0x2d, 0x88, 0xc8, 0x47, 0x86, 0x8b, 0x3c, 0x67, 0xfa, 0xe0, 0x40, 0x7c, 0x26, 0xab,
	0x89, 0x86, 0x5e, 0x1b, 0xaf, 0xd0, 0xf8, 0x8f, 0x01, 0x43, 0x95, 0xc4, 0x01, 0x00, 0x15, 0x84,
	0xcc, 0xb5, 0x80, 0x14, 0x3d, 0x9d, 0x9e, 0x1f, 0x08, 0xdc, 0x08, 0x42, 0x14, 0x7f, 0x3d, 0x48,
	0x6c, 0xda, 0x05, 0xf8, 0x02, 0xee, 0x52, 0xb1, 0x49, 0xf3, 0x7a, 0xbe, 0x3b, 0x1f, 0x5d, 0x0f,
	0x12, 0x47, 0x67, 0x7b, 0xa8, 0xe1, 0x75, 0x49, 0x8b, 0x16, 0x32, 0xe5, 0xe0, 0x12, 0xd2, 0x59,
	0x0d, 0x3d, 0x05, 0x48, 0x19, 0xeb, 0xc6, 0x38, 0x71, 0x91, 0x77, 0x47, 0x1e, 0x25, 0x73, 0x1a,
	0x78, 0xa3, 0x54, 0x44, 0xc6, 0x5b, 0x64, 0xa8, 0xac, 0x3e, 0x3c, 0xb2, 0xc7, 0x56, 0x5e, 0x64,
	0xbc, 0x77, 0x49, 0xca, 0xa6, 0xeb, 0xb5, 0x54, 0xef, 0xa1, 0xcb, 0xa8, 0x6c, 0x78, 0xef, 0x92,
	0x74, 0x41, 0x68, 0xc1, 0xc9, 0xba, 0xa4, 0xcb, 0x71, 0x00, 0x76, 0x4f, 0x60, 0x1f, 0x2c, 0x25,
	0xd6, 0xdd, 0xe8, 0xb1, 0xa5, 0xb7, 0xd4, 0xb3, 0x47, 0x60, 0xf7, 0x4b, 0xc4, 0xa7, 0x00, 0x37,
	0xb7, 0x51, 0x34, 0x9f, 0xbd, 0x8d, 0x6e, 0x2f, 0xcf, 0x06, 0xe1, 0x0f, 0x04, 0xf7, 0x33, 0xb6,
	0xd9, 0x97, 0x08, 0x1d, 0xed, 0x26, 0x96, 0x71, 0x8c, 0xbe, 0xbc, 0x28, 0x4a, 0xfe, 0x55, 0xa4,
	0x7e, 0xc6, 0x36, 0x93, 0x82, 0x91, 0x05, 0x2d, 0x76, 0x4f, 0xb1, 0xe2, 0xdb, 0x2a, 0x6f, 0xda,
	0x17, 0x19, 0xe8, 0x4f, 0x95, 0xfe, 0x45, 0xe8, 0x97, 0x61, 0x5e, 0xc5, 0xe1, 0x6f, 0xe3, 0xc9,
	0x95, 0x16, 0x8f, 0xbb, 0xf9, 0x3e, 0xe7, 0x84, 0xbc, 0xa7, 0xec, 0x3b, 0xfd, 0x24, 0x3b, 0x53,
	0x4b, 0x49, 0xbd, 0xfc, 0x17, 0x00, 0x00, 0xff, 0xff, 0xe8, 0x1b, 0x59, 0xf8, 0xe5, 0x02, 0x00,
	0x00,
}
