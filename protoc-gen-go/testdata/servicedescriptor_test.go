package testdata

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	pb "github.com/golang/protobuf/protoc-gen-go/testdata/grpc"
)

func TestServiceDescriptorExposed(t *testing.T) {
	data, idx := pb.GetTestServiceDescriptor()
	if len(idx) != 1 || idx[0] != 0 {
		t.Errorf("Expected idx to be [0], but was %v", idx)
	}
	decompressor, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := ioutil.ReadAll(decompressor)
	if err != nil {
		t.Fatal(err)
	}
	desc := &descriptor.FileDescriptorProto{}
	if err := proto.Unmarshal(decompressed, desc); err != nil {
		t.Fatal(err)
	}
	if len(desc.Service) != 1 {
		t.Errorf("Expected 1 service, but found %v", len(desc.Service))
	}
	svc := desc.Service[0]
	if svc.GetName() != "Test" {
		t.Errorf("Expected service name to be Test, but was %s", svc.GetName())
	}
}
