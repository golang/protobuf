// Code generated by protoc-gen-go.
// source: test_objects.proto
// DO NOT EDIT!

package jsonpb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Widget_Color int32

const (
	Widget_RED   Widget_Color = 0
	Widget_GREEN Widget_Color = 1
	Widget_BLUE  Widget_Color = 2
)

var Widget_Color_name = map[int32]string{
	0: "RED",
	1: "GREEN",
	2: "BLUE",
}
var Widget_Color_value = map[string]int32{
	"RED":   0,
	"GREEN": 1,
	"BLUE":  2,
}

func (x Widget_Color) Enum() *Widget_Color {
	p := new(Widget_Color)
	*p = x
	return p
}
func (x Widget_Color) String() string {
	return proto.EnumName(Widget_Color_name, int32(x))
}
func (x *Widget_Color) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Widget_Color_value, data, "Widget_Color")
	if err != nil {
		return err
	}
	*x = Widget_Color(value)
	return nil
}
func (Widget_Color) EnumDescriptor() ([]byte, []int) { return fileDescriptor1, []int{2, 0} }

// Test message for holding primitive types.
type Simple struct {
	OBool            *bool    `protobuf:"varint,1,opt,name=o_bool" json:"o_bool,omitempty"`
	OInt32           *int32   `protobuf:"varint,2,opt,name=o_int32" json:"o_int32,omitempty"`
	OInt64           *int64   `protobuf:"varint,3,opt,name=o_int64" json:"o_int64,omitempty"`
	OUint32          *uint32  `protobuf:"varint,4,opt,name=o_uint32" json:"o_uint32,omitempty"`
	OUint64          *uint64  `protobuf:"varint,5,opt,name=o_uint64" json:"o_uint64,omitempty"`
	OSint32          *int32   `protobuf:"zigzag32,6,opt,name=o_sint32" json:"o_sint32,omitempty"`
	OSint64          *int64   `protobuf:"zigzag64,7,opt,name=o_sint64" json:"o_sint64,omitempty"`
	OFloat           *float32 `protobuf:"fixed32,8,opt,name=o_float" json:"o_float,omitempty"`
	ODouble          *float64 `protobuf:"fixed64,9,opt,name=o_double" json:"o_double,omitempty"`
	OString          *string  `protobuf:"bytes,10,opt,name=o_string" json:"o_string,omitempty"`
	OBytes           []byte   `protobuf:"bytes,11,opt,name=o_bytes" json:"o_bytes,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Simple) Reset()                    { *m = Simple{} }
func (m *Simple) String() string            { return proto.CompactTextString(m) }
func (*Simple) ProtoMessage()               {}
func (*Simple) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *Simple) GetOBool() bool {
	if m != nil && m.OBool != nil {
		return *m.OBool
	}
	return false
}

func (m *Simple) GetOInt32() int32 {
	if m != nil && m.OInt32 != nil {
		return *m.OInt32
	}
	return 0
}

func (m *Simple) GetOInt64() int64 {
	if m != nil && m.OInt64 != nil {
		return *m.OInt64
	}
	return 0
}

func (m *Simple) GetOUint32() uint32 {
	if m != nil && m.OUint32 != nil {
		return *m.OUint32
	}
	return 0
}

func (m *Simple) GetOUint64() uint64 {
	if m != nil && m.OUint64 != nil {
		return *m.OUint64
	}
	return 0
}

func (m *Simple) GetOSint32() int32 {
	if m != nil && m.OSint32 != nil {
		return *m.OSint32
	}
	return 0
}

func (m *Simple) GetOSint64() int64 {
	if m != nil && m.OSint64 != nil {
		return *m.OSint64
	}
	return 0
}

func (m *Simple) GetOFloat() float32 {
	if m != nil && m.OFloat != nil {
		return *m.OFloat
	}
	return 0
}

func (m *Simple) GetODouble() float64 {
	if m != nil && m.ODouble != nil {
		return *m.ODouble
	}
	return 0
}

func (m *Simple) GetOString() string {
	if m != nil && m.OString != nil {
		return *m.OString
	}
	return ""
}

func (m *Simple) GetOBytes() []byte {
	if m != nil {
		return m.OBytes
	}
	return nil
}

// Test message for holding repeated primitives.
type Repeats struct {
	RBool            []bool    `protobuf:"varint,1,rep,name=r_bool" json:"r_bool,omitempty"`
	RInt32           []int32   `protobuf:"varint,2,rep,name=r_int32" json:"r_int32,omitempty"`
	RInt64           []int64   `protobuf:"varint,3,rep,name=r_int64" json:"r_int64,omitempty"`
	RUint32          []uint32  `protobuf:"varint,4,rep,name=r_uint32" json:"r_uint32,omitempty"`
	RUint64          []uint64  `protobuf:"varint,5,rep,name=r_uint64" json:"r_uint64,omitempty"`
	RSint32          []int32   `protobuf:"zigzag32,6,rep,name=r_sint32" json:"r_sint32,omitempty"`
	RSint64          []int64   `protobuf:"zigzag64,7,rep,name=r_sint64" json:"r_sint64,omitempty"`
	RFloat           []float32 `protobuf:"fixed32,8,rep,name=r_float" json:"r_float,omitempty"`
	RDouble          []float64 `protobuf:"fixed64,9,rep,name=r_double" json:"r_double,omitempty"`
	RString          []string  `protobuf:"bytes,10,rep,name=r_string" json:"r_string,omitempty"`
	RBytes           [][]byte  `protobuf:"bytes,11,rep,name=r_bytes" json:"r_bytes,omitempty"`
	XXX_unrecognized []byte    `json:"-"`
}

func (m *Repeats) Reset()                    { *m = Repeats{} }
func (m *Repeats) String() string            { return proto.CompactTextString(m) }
func (*Repeats) ProtoMessage()               {}
func (*Repeats) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func (m *Repeats) GetRBool() []bool {
	if m != nil {
		return m.RBool
	}
	return nil
}

func (m *Repeats) GetRInt32() []int32 {
	if m != nil {
		return m.RInt32
	}
	return nil
}

func (m *Repeats) GetRInt64() []int64 {
	if m != nil {
		return m.RInt64
	}
	return nil
}

func (m *Repeats) GetRUint32() []uint32 {
	if m != nil {
		return m.RUint32
	}
	return nil
}

func (m *Repeats) GetRUint64() []uint64 {
	if m != nil {
		return m.RUint64
	}
	return nil
}

func (m *Repeats) GetRSint32() []int32 {
	if m != nil {
		return m.RSint32
	}
	return nil
}

func (m *Repeats) GetRSint64() []int64 {
	if m != nil {
		return m.RSint64
	}
	return nil
}

func (m *Repeats) GetRFloat() []float32 {
	if m != nil {
		return m.RFloat
	}
	return nil
}

func (m *Repeats) GetRDouble() []float64 {
	if m != nil {
		return m.RDouble
	}
	return nil
}

func (m *Repeats) GetRString() []string {
	if m != nil {
		return m.RString
	}
	return nil
}

func (m *Repeats) GetRBytes() [][]byte {
	if m != nil {
		return m.RBytes
	}
	return nil
}

// Test message for holding enums and nested messages.
type Widget struct {
	Color            *Widget_Color  `protobuf:"varint,1,opt,name=color,enum=jsonpb.Widget_Color" json:"color,omitempty"`
	RColor           []Widget_Color `protobuf:"varint,2,rep,name=r_color,enum=jsonpb.Widget_Color" json:"r_color,omitempty"`
	Simple           *Simple        `protobuf:"bytes,10,opt,name=simple" json:"simple,omitempty"`
	RSimple          []*Simple      `protobuf:"bytes,11,rep,name=r_simple" json:"r_simple,omitempty"`
	Repeats          *Repeats       `protobuf:"bytes,20,opt,name=repeats" json:"repeats,omitempty"`
	RRepeats         []*Repeats     `protobuf:"bytes,21,rep,name=r_repeats" json:"r_repeats,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
}

func (m *Widget) Reset()                    { *m = Widget{} }
func (m *Widget) String() string            { return proto.CompactTextString(m) }
func (*Widget) ProtoMessage()               {}
func (*Widget) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{2} }

func (m *Widget) GetColor() Widget_Color {
	if m != nil && m.Color != nil {
		return *m.Color
	}
	return Widget_RED
}

func (m *Widget) GetRColor() []Widget_Color {
	if m != nil {
		return m.RColor
	}
	return nil
}

func (m *Widget) GetSimple() *Simple {
	if m != nil {
		return m.Simple
	}
	return nil
}

func (m *Widget) GetRSimple() []*Simple {
	if m != nil {
		return m.RSimple
	}
	return nil
}

func (m *Widget) GetRepeats() *Repeats {
	if m != nil {
		return m.Repeats
	}
	return nil
}

func (m *Widget) GetRRepeats() []*Repeats {
	if m != nil {
		return m.RRepeats
	}
	return nil
}

type Maps struct {
	MInt64Str        map[int64]string `protobuf:"bytes,1,rep,name=m_int64_str" json:"m_int64_str,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	MBoolSimple      map[bool]*Simple `protobuf:"bytes,2,rep,name=m_bool_simple" json:"m_bool_simple,omitempty" protobuf_key:"varint,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_unrecognized []byte           `json:"-"`
}

func (m *Maps) Reset()                    { *m = Maps{} }
func (m *Maps) String() string            { return proto.CompactTextString(m) }
func (*Maps) ProtoMessage()               {}
func (*Maps) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{3} }

func (m *Maps) GetMInt64Str() map[int64]string {
	if m != nil {
		return m.MInt64Str
	}
	return nil
}

func (m *Maps) GetMBoolSimple() map[bool]*Simple {
	if m != nil {
		return m.MBoolSimple
	}
	return nil
}

type MsgWithOneof struct {
	// Types that are valid to be assigned to Union:
	//	*MsgWithOneof_Title
	//	*MsgWithOneof_Salary
	Union            isMsgWithOneof_Union `protobuf_oneof:"union"`
	XXX_unrecognized []byte               `json:"-"`
}

func (m *MsgWithOneof) Reset()                    { *m = MsgWithOneof{} }
func (m *MsgWithOneof) String() string            { return proto.CompactTextString(m) }
func (*MsgWithOneof) ProtoMessage()               {}
func (*MsgWithOneof) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{4} }

type isMsgWithOneof_Union interface {
	isMsgWithOneof_Union()
}

type MsgWithOneof_Title struct {
	Title string `protobuf:"bytes,1,opt,name=title,oneof"`
}
type MsgWithOneof_Salary struct {
	Salary int64 `protobuf:"varint,2,opt,name=salary,oneof"`
}

func (*MsgWithOneof_Title) isMsgWithOneof_Union()  {}
func (*MsgWithOneof_Salary) isMsgWithOneof_Union() {}

func (m *MsgWithOneof) GetUnion() isMsgWithOneof_Union {
	if m != nil {
		return m.Union
	}
	return nil
}

func (m *MsgWithOneof) GetTitle() string {
	if x, ok := m.GetUnion().(*MsgWithOneof_Title); ok {
		return x.Title
	}
	return ""
}

func (m *MsgWithOneof) GetSalary() int64 {
	if x, ok := m.GetUnion().(*MsgWithOneof_Salary); ok {
		return x.Salary
	}
	return 0
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*MsgWithOneof) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), []interface{}) {
	return _MsgWithOneof_OneofMarshaler, _MsgWithOneof_OneofUnmarshaler, []interface{}{
		(*MsgWithOneof_Title)(nil),
		(*MsgWithOneof_Salary)(nil),
	}
}

func _MsgWithOneof_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*MsgWithOneof)
	// union
	switch x := m.Union.(type) {
	case *MsgWithOneof_Title:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.Title)
	case *MsgWithOneof_Salary:
		b.EncodeVarint(2<<3 | proto.WireVarint)
		b.EncodeVarint(uint64(x.Salary))
	case nil:
	default:
		return fmt.Errorf("MsgWithOneof.Union has unexpected type %T", x)
	}
	return nil
}

func _MsgWithOneof_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*MsgWithOneof)
	switch tag {
	case 1: // union.title
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Union = &MsgWithOneof_Title{x}
		return true, err
	case 2: // union.salary
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.Union = &MsgWithOneof_Salary{int64(x)}
		return true, err
	default:
		return false, nil
	}
}

type Real struct {
	Value            *float64                  `protobuf:"fixed64,1,opt,name=value" json:"value,omitempty"`
	XXX_extensions   map[int32]proto.Extension `json:"-"`
	XXX_unrecognized []byte                    `json:"-"`
}

func (m *Real) Reset()                    { *m = Real{} }
func (m *Real) String() string            { return proto.CompactTextString(m) }
func (*Real) ProtoMessage()               {}
func (*Real) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{5} }

var extRange_Real = []proto.ExtensionRange{
	{100, 536870911},
}

func (*Real) ExtensionRangeArray() []proto.ExtensionRange {
	return extRange_Real
}
func (m *Real) ExtensionMap() map[int32]proto.Extension {
	if m.XXX_extensions == nil {
		m.XXX_extensions = make(map[int32]proto.Extension)
	}
	return m.XXX_extensions
}

func (m *Real) GetValue() float64 {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return 0
}

type Complex struct {
	Imaginary        *float64                  `protobuf:"fixed64,1,opt,name=imaginary" json:"imaginary,omitempty"`
	XXX_extensions   map[int32]proto.Extension `json:"-"`
	XXX_unrecognized []byte                    `json:"-"`
}

func (m *Complex) Reset()                    { *m = Complex{} }
func (m *Complex) String() string            { return proto.CompactTextString(m) }
func (*Complex) ProtoMessage()               {}
func (*Complex) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{6} }

var extRange_Complex = []proto.ExtensionRange{
	{100, 536870911},
}

func (*Complex) ExtensionRangeArray() []proto.ExtensionRange {
	return extRange_Complex
}
func (m *Complex) ExtensionMap() map[int32]proto.Extension {
	if m.XXX_extensions == nil {
		m.XXX_extensions = make(map[int32]proto.Extension)
	}
	return m.XXX_extensions
}

func (m *Complex) GetImaginary() float64 {
	if m != nil && m.Imaginary != nil {
		return *m.Imaginary
	}
	return 0
}

var E_Complex_RealExtension = &proto.ExtensionDesc{
	ExtendedType:  (*Real)(nil),
	ExtensionType: (*Complex)(nil),
	Field:         123,
	Name:          "jsonpb.Complex.real_extension",
	Tag:           "bytes,123,opt,name=real_extension",
}

var E_Name = &proto.ExtensionDesc{
	ExtendedType:  (*Real)(nil),
	ExtensionType: (*string)(nil),
	Field:         124,
	Name:          "jsonpb.name",
	Tag:           "bytes,124,opt,name=name",
}

func init() {
	proto.RegisterType((*Simple)(nil), "jsonpb.Simple")
	proto.RegisterType((*Repeats)(nil), "jsonpb.Repeats")
	proto.RegisterType((*Widget)(nil), "jsonpb.Widget")
	proto.RegisterType((*Maps)(nil), "jsonpb.Maps")
	proto.RegisterType((*MsgWithOneof)(nil), "jsonpb.MsgWithOneof")
	proto.RegisterType((*Real)(nil), "jsonpb.Real")
	proto.RegisterType((*Complex)(nil), "jsonpb.Complex")
	proto.RegisterEnum("jsonpb.Widget_Color", Widget_Color_name, Widget_Color_value)
	proto.RegisterExtension(E_Complex_RealExtension)
	proto.RegisterExtension(E_Name)
}

var fileDescriptor1 = []byte{
	// 598 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x74, 0x93, 0x5f, 0x6b, 0x13, 0x4d,
	0x14, 0xc6, 0xbb, 0x3b, 0xfb, 0xf7, 0xa4, 0x4d, 0xb7, 0x43, 0x5f, 0x58, 0xfa, 0x52, 0x2d, 0x2b,
	0x05, 0xf1, 0x22, 0x94, 0x58, 0x45, 0x72, 0x99, 0x1a, 0xac, 0x60, 0x14, 0x52, 0xa4, 0x57, 0xb2,
	0x6c, 0x9a, 0x69, 0xdc, 0xba, 0xd9, 0x09, 0xb3, 0x13, 0x69, 0xd0, 0x0b, 0xc1, 0x2f, 0xa7, 0xdf,
	0xc3, 0x0f, 0xe2, 0xcc, 0x9c, 0x6c, 0xb6, 0x8d, 0x9a, 0xcb, 0x67, 0x9e, 0xf3, 0xe4, 0xfc, 0xce,
	0x39, 0x0b, 0x54, 0xb2, 0x4a, 0xa6, 0x7c, 0x7c, 0xc3, 0xae, 0x64, 0xd5, 0x99, 0x0b, 0x2e, 0x39,
	0xf5, 0x6e, 0x2a, 0x5e, 0xce, 0xc7, 0xc9, 0x0f, 0x0b, 0xbc, 0x8b, 0x7c, 0x36, 0x2f, 0x18, 0x6d,
	0x83, 0xc7, 0xd3, 0x31, 0xe7, 0x45, 0x6c, 0x1d, 0x59, 0x8f, 0x03, 0xba, 0x0b, 0x3e, 0x4f, 0xf3,
	0x52, 0x3e, 0xed, 0xc6, 0xb6, 0x12, 0xdc, 0xb5, 0xf0, 0xfc, 0x34, 0x26, 0x4a, 0x20, 0x34, 0x82,
	0x80, 0xa7, 0x0b, 0xb4, 0x38, 0x4a, 0xd9, 0x69, 0x14, 0xe5, 0x71, 0x95, 0xe2, 0xa0, 0x52, 0xa1,
	0xc7, 0x53, 0xca, 0x5e, 0xa3, 0x28, 0x8f, 0xaf, 0x14, 0x8a, 0xc1, 0xd7, 0x05, 0xcf, 0x64, 0x1c,
	0x28, 0xc1, 0x46, 0xcb, 0x84, 0x2f, 0xc6, 0x05, 0x8b, 0x43, 0xa5, 0x58, 0xab, 0x22, 0x29, 0xf2,
	0x72, 0x1a, 0x83, 0x52, 0x42, 0x2c, 0x1a, 0x2f, 0x15, 0x5b, 0xdc, 0x52, 0xc2, 0x76, 0xf2, 0xd3,
	0x02, 0x7f, 0xc4, 0xe6, 0x2c, 0x93, 0x95, 0x66, 0x11, 0x35, 0x0b, 0x41, 0x16, 0xb1, 0x66, 0x21,
	0xc8, 0x22, 0xd6, 0x2c, 0x04, 0x59, 0x44, 0xc3, 0x42, 0x90, 0x45, 0x34, 0x2c, 0x04, 0x59, 0x44,
	0xc3, 0x42, 0x90, 0x45, 0x34, 0x2c, 0x04, 0x59, 0xc4, 0x9a, 0x85, 0x20, 0x8b, 0x68, 0x58, 0x08,
	0xb2, 0x88, 0x86, 0x85, 0x20, 0x8b, 0x58, 0xb3, 0x10, 0xc5, 0xf2, 0xdd, 0x06, 0xef, 0x32, 0x9f,
	0x4c, 0x99, 0xa4, 0x8f, 0xc0, 0xbd, 0xe2, 0x05, 0x17, 0x66, 0x2b, 0xed, 0xee, 0x7e, 0x07, 0x37,
	0xd7, 0xc1, 0xe7, 0xce, 0x99, 0x7e, 0xa3, 0xc7, 0x3a, 0x00, 0x6d, 0x9a, 0xef, 0x5f, 0xb6, 0x07,
	0xe0, 0x55, 0x66, 0xd9, 0x66, 0x86, 0xad, 0x6e, 0xbb, 0x76, 0xad, 0x4e, 0xe0, 0x08, 0x71, 0x8c,
	0x43, 0x37, 0xf2, 0x37, 0x87, 0x2f, 0x70, 0xc6, 0xf1, 0xbe, 0x89, 0xd8, 0xad, 0x0d, 0xf5, 0xe8,
	0x13, 0x08, 0x45, 0x5a, 0x7b, 0xfe, 0x33, 0x21, 0x9b, 0x9e, 0xe4, 0x18, 0x5c, 0x6c, 0xc8, 0x07,
	0x32, 0x1a, 0xbc, 0x8c, 0xb6, 0x68, 0x08, 0xee, 0xab, 0xd1, 0x60, 0xf0, 0x36, 0xb2, 0x68, 0x00,
	0x4e, 0xff, 0xcd, 0xfb, 0x41, 0x64, 0x27, 0xbf, 0x2c, 0x70, 0x86, 0xd9, 0xbc, 0xa2, 0x27, 0xd0,
	0x9a, 0xe1, 0xb6, 0xf4, 0xdc, 0xcc, 0x4e, 0x5b, 0xdd, 0xff, 0xeb, 0x54, 0x6d, 0xe9, 0x0c, 0x5f,
	0xeb, 0xe7, 0x0b, 0x29, 0x06, 0xa5, 0x14, 0x4b, 0x7a, 0x0a, 0x3b, 0x33, 0x73, 0x00, 0x35, 0x8e,
	0x6d, 0x6a, 0x0e, 0xef, 0xd7, 0xf4, 0x95, 0x01, 0xc1, 0x4c, 0xd5, 0xc1, 0x09, 0xb4, 0x37, 0x72,
	0x5a, 0x40, 0x3e, 0xb1, 0xa5, 0x99, 0x3d, 0xa1, 0x3b, 0xe0, 0x7e, 0xce, 0x8a, 0x05, 0x33, 0xdf,
	0x43, 0xd8, 0xb3, 0x5f, 0x58, 0x07, 0x7d, 0x88, 0x36, 0x53, 0xee, 0xd6, 0x04, 0xf4, 0xf0, 0x6e,
	0xcd, 0x1f, 0xf3, 0xd4, 0x19, 0x49, 0x0f, 0xb6, 0x87, 0xd5, 0xf4, 0x32, 0x97, 0x1f, 0xdf, 0x95,
	0x8c, 0x5f, 0xab, 0x6b, 0x70, 0x65, 0x2e, 0x55, 0xcf, 0x3a, 0x21, 0x3c, 0xdf, 0x52, 0x07, 0xe3,
	0x55, 0x59, 0x91, 0x89, 0xa5, 0x09, 0x21, 0xe7, 0x5b, 0x7d, 0x1f, 0xdc, 0x45, 0x99, 0xf3, 0x32,
	0x79, 0x08, 0xce, 0x88, 0x65, 0x45, 0xd3, 0x9a, 0xae, 0xb1, 0x9e, 0x04, 0xc1, 0x24, 0xfa, 0xa6,
	0x7e, 0x76, 0xf2, 0x01, 0xfc, 0x33, 0xae, 0xff, 0xea, 0x96, 0xee, 0x41, 0x98, 0xcf, 0xb2, 0x69,
	0x5e, 0xea, 0xa4, 0x0d, 0x5f, 0xf7, 0x19, 0xb4, 0x85, 0x0a, 0x4a, 0xd9, 0xad, 0x64, 0x65, 0xa5,
	0xa2, 0xe9, 0x76, 0xb3, 0xb5, 0xac, 0x88, 0xbf, 0xdc, 0xdf, 0xf6, 0x2a, 0xb3, 0x77, 0x00, 0x4e,
	0x99, 0xcd, 0xd8, 0x86, 0xf9, 0xab, 0x6e, 0xfc, 0x77, 0x00, 0x00, 0x00, 0xff, 0xff, 0xfb, 0xc2,
	0xb2, 0xf6, 0x78, 0x04, 0x00, 0x00,
}
