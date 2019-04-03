package proto_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/proto"
	"github.com/google/go-cmp/cmp"
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

				got := reflect.New(reflect.TypeOf(want).Elem()).Interface().(proto.Message)
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
				if !protoV1.Equal(got.(protoV1.Message), want.(protoV1.Message)) {
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

				got := reflect.New(reflect.TypeOf(want).Elem()).Interface().(proto.Message)
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
				if !protoV1.Equal(got.(protoV1.Message), want.(protoV1.Message)) {
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
