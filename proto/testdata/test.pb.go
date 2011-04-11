// Code generated by protoc-gen-go from "test.proto"
// DO NOT EDIT!

package test_proto

import proto "goprotobuf.googlecode.com/hg/proto"
import "math"
import "os"

// Reference proto, math & os imports to suppress error if they are not otherwise used.
var _ = proto.GetString
var _ = math.Inf
var _ os.Error


type FOO int32

const (
	FOO_FOO1 = 1
)

var FOO_name = map[int32]string{
	1: "FOO1",
}
var FOO_value = map[string]int32{
	"FOO1": 1,
}

func NewFOO(x int32) *FOO {
	e := FOO(x)
	return &e
}
func (x FOO) String() string {
	return proto.EnumName(FOO_name, int32(x))
}

type GoTest_KIND int32

const (
	GoTest_VOID		= 0
	GoTest_BOOL		= 1
	GoTest_BYTES		= 2
	GoTest_FINGERPRINT	= 3
	GoTest_FLOAT		= 4
	GoTest_INT		= 5
	GoTest_STRING		= 6
	GoTest_TIME		= 7
	GoTest_TUPLE		= 8
	GoTest_ARRAY		= 9
	GoTest_MAP		= 10
	GoTest_TABLE		= 11
	GoTest_FUNCTION		= 12
)

var GoTest_KIND_name = map[int32]string{
	0:	"VOID",
	1:	"BOOL",
	2:	"BYTES",
	3:	"FINGERPRINT",
	4:	"FLOAT",
	5:	"INT",
	6:	"STRING",
	7:	"TIME",
	8:	"TUPLE",
	9:	"ARRAY",
	10:	"MAP",
	11:	"TABLE",
	12:	"FUNCTION",
}
var GoTest_KIND_value = map[string]int32{
	"VOID":		0,
	"BOOL":		1,
	"BYTES":	2,
	"FINGERPRINT":	3,
	"FLOAT":	4,
	"INT":		5,
	"STRING":	6,
	"TIME":		7,
	"TUPLE":	8,
	"ARRAY":	9,
	"MAP":		10,
	"TABLE":	11,
	"FUNCTION":	12,
}

func NewGoTest_KIND(x int32) *GoTest_KIND {
	e := GoTest_KIND(x)
	return &e
}
func (x GoTest_KIND) String() string {
	return proto.EnumName(GoTest_KIND_name, int32(x))
}

type MyMessage_Color int32

const (
	MyMessage_RED	= 0
	MyMessage_GREEN	= 1
	MyMessage_BLUE	= 2
)

var MyMessage_Color_name = map[int32]string{
	0:	"RED",
	1:	"GREEN",
	2:	"BLUE",
}
var MyMessage_Color_value = map[string]int32{
	"RED":		0,
	"GREEN":	1,
	"BLUE":		2,
}

func NewMyMessage_Color(x int32) *MyMessage_Color {
	e := MyMessage_Color(x)
	return &e
}
func (x MyMessage_Color) String() string {
	return proto.EnumName(MyMessage_Color_name, int32(x))
}

type GoEnum struct {
	Foo			*FOO	"PB(varint,1,req,name=foo,enum=test_proto.FOO)"
	XXX_unrecognized	[]byte
}

func (this *GoEnum) Reset() {
	*this = GoEnum{}
}

type GoTestField struct {
	Label			*string	"PB(bytes,1,req)"
	Type			*string	"PB(bytes,2,req)"
	XXX_unrecognized	[]byte
}

func (this *GoTestField) Reset() {
	*this = GoTestField{}
}

type GoTest struct {
	Kind			*int32			"PB(varint,1,req)"
	Table			*string			"PB(bytes,2,opt)"
	Param			*int32			"PB(varint,3,opt)"
	RequiredField		*GoTestField		"PB(bytes,4,req)"
	RepeatedField		[]*GoTestField		"PB(bytes,5,rep)"
	OptionalField		*GoTestField		"PB(bytes,6,opt)"
	F_BoolRequired		*bool			"PB(varint,10,req,name=F_Bool_required)"
	F_Int32Required		*int32			"PB(varint,11,req,name=F_Int32_required)"
	F_Int64Required		*int64			"PB(varint,12,req,name=F_Int64_required)"
	F_Fixed32Required	*uint32			"PB(fixed32,13,req,name=F_Fixed32_required)"
	F_Fixed64Required	*uint64			"PB(fixed64,14,req,name=F_Fixed64_required)"
	F_Uint32Required	*uint32			"PB(varint,15,req,name=F_Uint32_required)"
	F_Uint64Required	*uint64			"PB(varint,16,req,name=F_Uint64_required)"
	F_FloatRequired		*float32		"PB(fixed32,17,req,name=F_Float_required)"
	F_DoubleRequired	*float64		"PB(fixed64,18,req,name=F_Double_required)"
	F_StringRequired	*string			"PB(bytes,19,req,name=F_String_required)"
	F_BytesRequired		[]byte			"PB(bytes,101,req,name=F_Bytes_required)"
	F_Sint32Required	*int32			"PB(zigzag32,102,req,name=F_Sint32_required)"
	F_Sint64Required	*int64			"PB(zigzag64,103,req,name=F_Sint64_required)"
	F_BoolRepeated		[]bool			"PB(varint,20,rep,name=F_Bool_repeated)"
	F_Int32Repeated		[]int32			"PB(varint,21,rep,name=F_Int32_repeated)"
	F_Int64Repeated		[]int64			"PB(varint,22,rep,name=F_Int64_repeated)"
	F_Fixed32Repeated	[]uint32		"PB(fixed32,23,rep,name=F_Fixed32_repeated)"
	F_Fixed64Repeated	[]uint64		"PB(fixed64,24,rep,name=F_Fixed64_repeated)"
	F_Uint32Repeated	[]uint32		"PB(varint,25,rep,name=F_Uint32_repeated)"
	F_Uint64Repeated	[]uint64		"PB(varint,26,rep,name=F_Uint64_repeated)"
	F_FloatRepeated		[]float32		"PB(fixed32,27,rep,name=F_Float_repeated)"
	F_DoubleRepeated	[]float64		"PB(fixed64,28,rep,name=F_Double_repeated)"
	F_StringRepeated	[]string		"PB(bytes,29,rep,name=F_String_repeated)"
	F_BytesRepeated		[][]byte		"PB(bytes,201,rep,name=F_Bytes_repeated)"
	F_Sint32Repeated	[]int32			"PB(zigzag32,202,rep,name=F_Sint32_repeated)"
	F_Sint64Repeated	[]int64			"PB(zigzag64,203,rep,name=F_Sint64_repeated)"
	F_BoolOptional		*bool			"PB(varint,30,opt,name=F_Bool_optional)"
	F_Int32Optional		*int32			"PB(varint,31,opt,name=F_Int32_optional)"
	F_Int64Optional		*int64			"PB(varint,32,opt,name=F_Int64_optional)"
	F_Fixed32Optional	*uint32			"PB(fixed32,33,opt,name=F_Fixed32_optional)"
	F_Fixed64Optional	*uint64			"PB(fixed64,34,opt,name=F_Fixed64_optional)"
	F_Uint32Optional	*uint32			"PB(varint,35,opt,name=F_Uint32_optional)"
	F_Uint64Optional	*uint64			"PB(varint,36,opt,name=F_Uint64_optional)"
	F_FloatOptional		*float32		"PB(fixed32,37,opt,name=F_Float_optional)"
	F_DoubleOptional	*float64		"PB(fixed64,38,opt,name=F_Double_optional)"
	F_StringOptional	*string			"PB(bytes,39,opt,name=F_String_optional)"
	F_BytesOptional		[]byte			"PB(bytes,301,opt,name=F_Bytes_optional)"
	F_Sint32Optional	*int32			"PB(zigzag32,302,opt,name=F_Sint32_optional)"
	F_Sint64Optional	*int64			"PB(zigzag64,303,opt,name=F_Sint64_optional)"
	F_BoolDefaulted		*bool			"PB(varint,40,opt,name=F_Bool_defaulted,def=1)"
	F_Int32Defaulted	*int32			"PB(varint,41,opt,name=F_Int32_defaulted,def=32)"
	F_Int64Defaulted	*int64			"PB(varint,42,opt,name=F_Int64_defaulted,def=64)"
	F_Fixed32Defaulted	*uint32			"PB(fixed32,43,opt,name=F_Fixed32_defaulted,def=320)"
	F_Fixed64Defaulted	*uint64			"PB(fixed64,44,opt,name=F_Fixed64_defaulted,def=640)"
	F_Uint32Defaulted	*uint32			"PB(varint,45,opt,name=F_Uint32_defaulted,def=3200)"
	F_Uint64Defaulted	*uint64			"PB(varint,46,opt,name=F_Uint64_defaulted,def=6400)"
	F_FloatDefaulted	*float32		"PB(fixed32,47,opt,name=F_Float_defaulted,def=314159)"
	F_DoubleDefaulted	*float64		"PB(fixed64,48,opt,name=F_Double_defaulted,def=271828)"
	F_StringDefaulted	*string			"PB(bytes,49,opt,name=F_String_defaulted,def=hello, \\\"world!\\\"\\n)"
	F_BytesDefaulted	[]byte			"PB(bytes,401,opt,name=F_Bytes_defaulted,def=Bignose)"
	F_Sint32Defaulted	*int32			"PB(zigzag32,402,opt,name=F_Sint32_defaulted,def=-32)"
	F_Sint64Defaulted	*int64			"PB(zigzag64,403,opt,name=F_Sint64_defaulted,def=-64)"
	F_BoolRepeatedPacked	[]bool			"PB(varint,50,rep,packed,name=F_Bool_repeated_packed)"
	F_Int32RepeatedPacked	[]int32			"PB(varint,51,rep,packed,name=F_Int32_repeated_packed)"
	F_Int64RepeatedPacked	[]int64			"PB(varint,52,rep,packed,name=F_Int64_repeated_packed)"
	F_Fixed32RepeatedPacked	[]uint32		"PB(fixed32,53,rep,packed,name=F_Fixed32_repeated_packed)"
	F_Fixed64RepeatedPacked	[]uint64		"PB(fixed64,54,rep,packed,name=F_Fixed64_repeated_packed)"
	F_Uint32RepeatedPacked	[]uint32		"PB(varint,55,rep,packed,name=F_Uint32_repeated_packed)"
	F_Uint64RepeatedPacked	[]uint64		"PB(varint,56,rep,packed,name=F_Uint64_repeated_packed)"
	F_FloatRepeatedPacked	[]float32		"PB(fixed32,57,rep,packed,name=F_Float_repeated_packed)"
	F_DoubleRepeatedPacked	[]float64		"PB(fixed64,58,rep,packed,name=F_Double_repeated_packed)"
	F_Sint32RepeatedPacked	[]int32			"PB(zigzag32,502,rep,packed,name=F_Sint32_repeated_packed)"
	F_Sint64RepeatedPacked	[]int64			"PB(zigzag64,503,rep,packed,name=F_Sint64_repeated_packed)"
	Requiredgroup		*GoTest_RequiredGroup	"PB(group,70,req,name=requiredgroup)"
	Repeatedgroup		[]*GoTest_RepeatedGroup	"PB(group,80,rep,name=repeatedgroup)"
	Optionalgroup		*GoTest_OptionalGroup	"PB(group,90,opt,name=optionalgroup)"
	XXX_unrecognized	[]byte
}

func (this *GoTest) Reset() {
	*this = GoTest{}
}

const Default_GoTest_F_BoolDefaulted bool = true
const Default_GoTest_F_Int32Defaulted int32 = 32
const Default_GoTest_F_Int64Defaulted int64 = 64
const Default_GoTest_F_Fixed32Defaulted uint32 = 320
const Default_GoTest_F_Fixed64Defaulted uint64 = 640
const Default_GoTest_F_Uint32Defaulted uint32 = 3200
const Default_GoTest_F_Uint64Defaulted uint64 = 6400
const Default_GoTest_F_FloatDefaulted float32 = 314159
const Default_GoTest_F_DoubleDefaulted float64 = 271828
const Default_GoTest_F_StringDefaulted string = "hello, \"world!\"\n"

var Default_GoTest_F_BytesDefaulted []byte = []byte("Bignose")

const Default_GoTest_F_Sint32Defaulted int32 = -32
const Default_GoTest_F_Sint64Defaulted int64 = -64

type GoTest_RequiredGroup struct {
	RequiredField		*string	"PB(bytes,71,req)"
	XXX_unrecognized	[]byte
}

func (this *GoTest_RequiredGroup) Reset() {
	*this = GoTest_RequiredGroup{}
}

type GoTest_RepeatedGroup struct {
	RequiredField		*string	"PB(bytes,81,req)"
	XXX_unrecognized	[]byte
}

func (this *GoTest_RepeatedGroup) Reset() {
	*this = GoTest_RepeatedGroup{}
}

type GoTest_OptionalGroup struct {
	RequiredField		*string	"PB(bytes,91,req)"
	XXX_unrecognized	[]byte
}

func (this *GoTest_OptionalGroup) Reset() {
	*this = GoTest_OptionalGroup{}
}

type GoSkipTest struct {
	SkipInt32		*int32			"PB(varint,11,req,name=skip_int32)"
	SkipFixed32		*uint32			"PB(fixed32,12,req,name=skip_fixed32)"
	SkipFixed64		*uint64			"PB(fixed64,13,req,name=skip_fixed64)"
	SkipString		*string			"PB(bytes,14,req,name=skip_string)"
	Skipgroup		*GoSkipTest_SkipGroup	"PB(group,15,req,name=skipgroup)"
	XXX_unrecognized	[]byte
}

func (this *GoSkipTest) Reset() {
	*this = GoSkipTest{}
}

type GoSkipTest_SkipGroup struct {
	GroupInt32		*int32	"PB(varint,16,req,name=group_int32)"
	GroupString		*string	"PB(bytes,17,req,name=group_string)"
	XXX_unrecognized	[]byte
}

func (this *GoSkipTest_SkipGroup) Reset() {
	*this = GoSkipTest_SkipGroup{}
}

type NonPackedTest struct {
	A			[]int32	"PB(varint,1,rep,name=a)"
	XXX_unrecognized	[]byte
}

func (this *NonPackedTest) Reset() {
	*this = NonPackedTest{}
}

type PackedTest struct {
	B			[]int32	"PB(varint,1,rep,packed,name=b)"
	XXX_unrecognized	[]byte
}

func (this *PackedTest) Reset() {
	*this = PackedTest{}
}

type InnerMessage struct {
	Host			*string	"PB(bytes,1,req,name=host)"
	Port			*int32	"PB(varint,2,opt,name=port,def=4000)"
	Connected		*bool	"PB(varint,3,opt,name=connected)"
	XXX_unrecognized	[]byte
}

func (this *InnerMessage) Reset() {
	*this = InnerMessage{}
}

const Default_InnerMessage_Port int32 = 4000

type OtherMessage struct {
	Key			*int64		"PB(varint,1,opt,name=key)"
	Value			[]byte		"PB(bytes,2,opt,name=value)"
	Weight			*float32	"PB(fixed32,3,opt,name=weight)"
	Inner			*InnerMessage	"PB(bytes,4,opt,name=inner)"
	XXX_unrecognized	[]byte
}

func (this *OtherMessage) Reset() {
	*this = OtherMessage{}
}

type MyMessage struct {
	Count			*int32			"PB(varint,1,req,name=count)"
	Name			*string			"PB(bytes,2,opt,name=name)"
	Quote			*string			"PB(bytes,3,opt,name=quote)"
	Pet			[]string		"PB(bytes,4,rep,name=pet)"
	Inner			*InnerMessage		"PB(bytes,5,opt,name=inner)"
	Others			[]*OtherMessage		"PB(bytes,6,rep,name=others)"
	Bikeshed		*MyMessage_Color	"PB(varint,7,opt,name=bikeshed,enum=test_proto.MyMessage_Color)"
	XXX_unrecognized	[]byte
}

func (this *MyMessage) Reset() {
	*this = MyMessage{}
}

type MessageList struct {
	Message			[]*MessageList_Message	"PB(group,1,rep,name=message)"
	XXX_unrecognized	[]byte
}

func (this *MessageList) Reset() {
	*this = MessageList{}
}

type MessageList_Message struct {
	Name			*string	"PB(bytes,2,req,name=name)"
	Count			*int32	"PB(varint,3,req,name=count)"
	XXX_unrecognized	[]byte
}

func (this *MessageList_Message) Reset() {
	*this = MessageList_Message{}
}

func init() {
	proto.RegisterEnum("test_proto.FOO", FOO_name, FOO_value)
	proto.RegisterEnum("test_proto.GoTest_KIND", GoTest_KIND_name, GoTest_KIND_value)
	proto.RegisterEnum("test_proto.MyMessage_Color", MyMessage_Color_name, MyMessage_Color_value)
}
