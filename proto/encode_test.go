package proto_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"

	test3pb "google.golang.org/protobuf/internal/testprotos/test3"
)

func TestEncode(t *testing.T) {
	for _, test := range testProtos {
		for _, want := range test.decodeTo {
			t.Run(fmt.Sprintf("%s (%T)", test.desc, want), func(t *testing.T) {
				opts := proto.MarshalOptions{
					AllowPartial: test.partial,
				}
				wire, err := opts.Marshal(want)
				if err != nil {
					t.Fatalf("Marshal error: %v\nMessage:\n%v", err, marshalText(want))
				}

				size := proto.Size(want)
				if size != len(wire) {
					t.Errorf("Size and marshal disagree: Size(m)=%v; len(Marshal(m))=%v\nMessage:\n%v", size, len(wire), marshalText(want))
				}

				got := newMessage(want)
				uopts := proto.UnmarshalOptions{
					AllowPartial: test.partial,
				}
				if err := uopts.Unmarshal(wire, got); err != nil {
					t.Errorf("Unmarshal error: %v\nMessage:\n%v", err, protoV1.MarshalTextString(want.(protoV1.Message)))
					return
				}

				if test.invalidExtensions {
					// Equal doesn't work on messages containing invalid extension data.
					return
				}
				if !proto.Equal(got, want) {
					t.Errorf("Unmarshal returned unexpected result; got:\n%v\nwant:\n%v", protoV1.MarshalTextString(got.(protoV1.Message)), protoV1.MarshalTextString(want.(protoV1.Message)))
				}
			})
		}
	}
}

func TestEncodeDeterministic(t *testing.T) {
	for _, test := range testProtos {
		for _, want := range test.decodeTo {
			t.Run(fmt.Sprintf("%s (%T)", test.desc, want), func(t *testing.T) {
				opts := proto.MarshalOptions{
					Deterministic: true,
					AllowPartial:  test.partial,
				}
				wire, err := opts.Marshal(want)
				if err != nil {
					t.Fatalf("Marshal error: %v\nMessage:\n%v", err, marshalText(want))
				}
				wire2, err := opts.Marshal(want)
				if err != nil {
					t.Fatalf("Marshal error: %v\nMessage:\n%v", err, marshalText(want))
				}
				if !bytes.Equal(wire, wire2) {
					t.Fatalf("deterministic marshal returned varying results:\n%v", cmp.Diff(wire, wire2))
				}

				got := newMessage(want)
				uopts := proto.UnmarshalOptions{
					AllowPartial: test.partial,
				}
				if err := uopts.Unmarshal(wire, got); err != nil {
					t.Errorf("Unmarshal error: %v\nMessage:\n%v", err, marshalText(want))
					return
				}

				if test.invalidExtensions {
					// Equal doesn't work on messages containing invalid extension data.
					return
				}
				if !proto.Equal(got, want) {
					t.Errorf("Unmarshal returned unexpected result; got:\n%v\nwant:\n%v", marshalText(got), marshalText(want))
				}
			})
		}
	}
}

func TestEncodeInvalidUTF8(t *testing.T) {
	for _, test := range invalidUTF8TestProtos {
		for _, want := range test.decodeTo {
			t.Run(fmt.Sprintf("%s (%T)", test.desc, want), func(t *testing.T) {
				wire, err := proto.Marshal(want)
				if !isErrInvalidUTF8(err) {
					t.Errorf("Marshal did not return expected error for invalid UTF8: %v\nMessage:\n%v", err, marshalText(want))
				}
				got := newMessage(want)
				if err := proto.Unmarshal(wire, got); !isErrInvalidUTF8(err) {
					t.Errorf("Unmarshal error: %v\nMessage:\n%v", err, marshalText(want))
					return
				}
				if !proto.Equal(got, want) {
					t.Errorf("Unmarshal returned unexpected result; got:\n%v\nwant:\n%v", marshalText(got), marshalText(want))
				}
			})
		}
	}
}

func TestEncodeRequiredFieldChecks(t *testing.T) {
	for _, test := range testProtos {
		if !test.partial {
			continue
		}
		for _, m := range test.decodeTo {
			t.Run(fmt.Sprintf("%s (%T)", test.desc, m), func(t *testing.T) {
				_, err := proto.Marshal(m)
				if err == nil {
					t.Fatalf("Marshal succeeded (want error)\nMessage:\n%v", marshalText(m))
				}
			})
		}
	}
}

func TestMarshalAppend(t *testing.T) {
	want := []byte("prefix")
	got := append([]byte(nil), want...)
	got, err := proto.MarshalOptions{}.MarshalAppend(got, &test3pb.TestAllTypes{
		OptionalString: "value",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, want) {
		t.Fatalf("MarshalAppend modified prefix: got %v, want prefix %v", got, want)
	}
}

// newMessage returns a new message with the same type and extension fields as m.
func newMessage(m proto.Message) proto.Message {
	n := reflect.New(reflect.TypeOf(m).Elem()).Interface().(proto.Message)
	m.ProtoReflect().KnownFields().ExtensionTypes().Range(func(xt pref.ExtensionType) bool {
		n.ProtoReflect().KnownFields().ExtensionTypes().Register(xt)
		return true
	})
	return n
}
