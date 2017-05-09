package descriptor_test

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/descriptor"
	tpb "github.com/golang/protobuf/proto/testdata"
	protobuf "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func TestMessage(t *testing.T) {
	var msg *protobuf.DescriptorProto
	fd, md := descriptor.ForMessage(msg)
	if pkg, want := fd.GetPackage(), "google.protobuf"; pkg != want {
		t.Errorf("descriptor.ForMessage(%T).GetPackage() = %q; want %q", msg, pkg, want)
	}
	if name, want := md.GetName(), "DescriptorProto"; name != want {
		t.Fatalf("descriptor.ForMessage(%T).GetName() = %q; want %q", msg, name, want)
	}
}

func TestEnum(t *testing.T) {
	var enum *protobuf.FieldDescriptorProto_Type
	fd, ed := descriptor.ForEnum(enum)
	if pkg, want := fd.GetPackage(), "google.protobuf"; pkg != want {
		t.Errorf("descriptor.ForEnum(%T).GetPackage() = %q; want %q", enum, pkg, want)
	}
	if name, want := ed.GetName(), "Type"; name != want {
		t.Errorf("descriptor.ForEnum(%T).GetName() = %q; want %q", enum, name, want)
	}
	if value, want := ed.GetValue()[0].GetName(), "TYPE_DOUBLE"; value != want {
		t.Errorf("descriptor.ForEnum(%T).GetValue()[0].GetName() = %v; want %v", enum, value, want)
	}
}

func Example_Options() {
	var msg *tpb.MyMessageSet
	_, md := descriptor.ForMessage(msg)
	if md.GetOptions().GetMessageSetWireFormat() {
		fmt.Printf("%v uses option message_set_wire_format.\n", md.GetName())
	}

	// Output:
	// MyMessageSet uses option message_set_wire_format.
}

func Example_EnumOptions() {
	var enum *tpb.EnumAllowingAlias_NUMBER
	_, ed := descriptor.ForEnum(enum)
	if ed.GetOptions().GetAllowAlias() {
		fmt.Printf("%v uses option allow_alias.\n", ed.GetName())
	}

	// Output:
	// NUMBER uses option allow_alias.
}
