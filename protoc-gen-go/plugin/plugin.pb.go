// Code generated by protoc-gen-go from "google/protobuf/compiler/plugin.proto"
// DO NOT EDIT!

package google_protobuf_compiler

import proto "code.google.com/p/goprotobuf/proto"
import "math"
import google_protobuf "code.google.com/p/goprotobuf/protoc-gen-go/descriptor"

// Reference proto and math imports to suppress error if they are not otherwise used.
var _ = proto.GetString
var _ = math.Inf

type CodeGeneratorRequest struct {
	FileToGenerate   []string                               `protobuf:"bytes,1,rep,name=file_to_generate" json:"file_to_generate,omitempty"`
	Parameter        *string                                `protobuf:"bytes,2,opt,name=parameter" json:"parameter,omitempty"`
	ProtoFile        []*google_protobuf.FileDescriptorProto `protobuf:"bytes,15,rep,name=proto_file" json:"proto_file,omitempty"`
	XXX_unrecognized []byte                                 `json:"-"`
}

func (this *CodeGeneratorRequest) Reset()         { *this = CodeGeneratorRequest{} }
func (this *CodeGeneratorRequest) String() string { return proto.CompactTextString(this) }
func (*CodeGeneratorRequest) ProtoMessage()       {}

func (this *CodeGeneratorRequest) GetParameter() string {
	if this != nil && this.Parameter != nil {
		return *this.Parameter
	}
	return ""
}

type CodeGeneratorResponse struct {
	Error            *string                       `protobuf:"bytes,1,opt,name=error" json:"error,omitempty"`
	File             []*CodeGeneratorResponse_File `protobuf:"bytes,15,rep,name=file" json:"file,omitempty"`
	XXX_unrecognized []byte                        `json:"-"`
}

func (this *CodeGeneratorResponse) Reset()         { *this = CodeGeneratorResponse{} }
func (this *CodeGeneratorResponse) String() string { return proto.CompactTextString(this) }
func (*CodeGeneratorResponse) ProtoMessage()       {}

func (this *CodeGeneratorResponse) GetError() string {
	if this != nil && this.Error != nil {
		return *this.Error
	}
	return ""
}

type CodeGeneratorResponse_File struct {
	Name             *string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	InsertionPoint   *string `protobuf:"bytes,2,opt,name=insertion_point" json:"insertion_point,omitempty"`
	Content          *string `protobuf:"bytes,15,opt,name=content" json:"content,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (this *CodeGeneratorResponse_File) Reset()         { *this = CodeGeneratorResponse_File{} }
func (this *CodeGeneratorResponse_File) String() string { return proto.CompactTextString(this) }
func (*CodeGeneratorResponse_File) ProtoMessage()       {}

func (this *CodeGeneratorResponse_File) GetName() string {
	if this != nil && this.Name != nil {
		return *this.Name
	}
	return ""
}

func (this *CodeGeneratorResponse_File) GetInsertionPoint() string {
	if this != nil && this.InsertionPoint != nil {
		return *this.InsertionPoint
	}
	return ""
}

func (this *CodeGeneratorResponse_File) GetContent() string {
	if this != nil && this.Content != nil {
		return *this.Content
	}
	return ""
}

func init() {
}
