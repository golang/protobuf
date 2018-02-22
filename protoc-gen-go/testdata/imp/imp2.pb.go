// Code generated by protoc-gen-go. DO NOT EDIT.
// source: imp/imp2.proto

package imp

/*
This file includes these top-level messages:
	PubliclyImportedMessage
*/

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type PubliclyImportedEnum int32

const (
	PubliclyImportedEnum_GLASSES PubliclyImportedEnum = 1
	PubliclyImportedEnum_HAIR    PubliclyImportedEnum = 2
)

var PubliclyImportedEnum_name = map[int32]string{
	1: "GLASSES",
	2: "HAIR",
}
var PubliclyImportedEnum_value = map[string]int32{
	"GLASSES": 1,
	"HAIR":    2,
}

func (x PubliclyImportedEnum) Enum() *PubliclyImportedEnum {
	p := new(PubliclyImportedEnum)
	*p = x
	return p
}
func (x PubliclyImportedEnum) String() string {
	return proto.EnumName(PubliclyImportedEnum_name, int32(x))
}
func (x *PubliclyImportedEnum) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(PubliclyImportedEnum_value, data, "PubliclyImportedEnum")
	if err != nil {
		return err
	}
	*x = PubliclyImportedEnum(value)
	return nil
}
func (PubliclyImportedEnum) EnumDescriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

type PubliclyImportedMessage struct {
	Field                *int64   `protobuf:"varint,1,opt,name=field" json:"field,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PubliclyImportedMessage) Reset()                    { *m = PubliclyImportedMessage{} }
func (m *PubliclyImportedMessage) String() string            { return proto.CompactTextString(m) }
func (*PubliclyImportedMessage) ProtoMessage()               {}
func (*PubliclyImportedMessage) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }
func (m *PubliclyImportedMessage) Unmarshal(b []byte) error {
	return xxx_messageInfo_PubliclyImportedMessage.Unmarshal(m, b)
}
func (m *PubliclyImportedMessage) Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PubliclyImportedMessage.Marshal(b, m, deterministic)
}
func (dst *PubliclyImportedMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PubliclyImportedMessage.Merge(dst, src)
}
func (m *PubliclyImportedMessage) XXX_Size() int {
	return xxx_messageInfo_PubliclyImportedMessage.Size(m)
}
func (m *PubliclyImportedMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_PubliclyImportedMessage.DiscardUnknown(m)
}

var xxx_messageInfo_PubliclyImportedMessage proto.InternalMessageInfo

func (m *PubliclyImportedMessage) GetField() int64 {
	if m != nil && m.Field != nil {
		return *m.Field
	}
	return 0
}

func init() {
	proto.RegisterType((*PubliclyImportedMessage)(nil), "imp.PubliclyImportedMessage")
	proto.RegisterEnum("imp.PubliclyImportedEnum", PubliclyImportedEnum_name, PubliclyImportedEnum_value)
}

func init() { proto.RegisterFile("imp/imp2.proto", fileDescriptor1) }

var fileDescriptor1 = []byte{
	// 171 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xcb, 0xcc, 0x2d, 0xd0,
	0xcf, 0xcc, 0x2d, 0x30, 0xd2, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0xce, 0xcc, 0x2d, 0x50,
	0xd2, 0xe7, 0x12, 0x0f, 0x28, 0x4d, 0xca, 0xc9, 0x4c, 0xce, 0xa9, 0xf4, 0xcc, 0x2d, 0xc8, 0x2f,
	0x2a, 0x49, 0x4d, 0xf1, 0x4d, 0x2d, 0x2e, 0x4e, 0x4c, 0x4f, 0x15, 0x12, 0xe1, 0x62, 0x4d, 0xcb,
	0x4c, 0xcd, 0x49, 0x91, 0x60, 0x54, 0x60, 0xd4, 0x60, 0x0e, 0x82, 0x70, 0xb4, 0x74, 0xb9, 0x44,
	0xd0, 0x35, 0xb8, 0xe6, 0x95, 0xe6, 0x0a, 0x71, 0x73, 0xb1, 0xbb, 0xfb, 0x38, 0x06, 0x07, 0xbb,
	0x06, 0x0b, 0x30, 0x0a, 0x71, 0x70, 0xb1, 0x78, 0x38, 0x7a, 0x06, 0x09, 0x30, 0x39, 0x99, 0x47,
	0x99, 0xa6, 0x67, 0x96, 0x64, 0x94, 0x26, 0xe9, 0x25, 0xe7, 0xe7, 0xea, 0xa7, 0xe7, 0xe7, 0x24,
	0xe6, 0xa5, 0xeb, 0x83, 0xed, 0x4f, 0x2a, 0x4d, 0x83, 0x30, 0x92, 0x75, 0xd3, 0x53, 0xf3, 0x74,
	0xd3, 0xf3, 0xf5, 0x4b, 0x52, 0x8b, 0x4b, 0x52, 0x12, 0x4b, 0x12, 0x41, 0x8e, 0x04, 0x04, 0x00,
	0x00, 0xff, 0xff, 0x32, 0x18, 0x4d, 0x15, 0xae, 0x00, 0x00, 0x00,
}
