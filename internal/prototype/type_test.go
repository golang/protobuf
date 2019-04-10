// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype_test

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	detrand "github.com/golang/protobuf/v2/internal/detrand"
	ptype "github.com/golang/protobuf/v2/internal/prototype"
	scalar "github.com/golang/protobuf/v2/internal/scalar"
	pdesc "github.com/golang/protobuf/v2/reflect/protodesc"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"

	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

func init() {
	// Disable detrand to enable direct comparisons on outputs.
	detrand.Disable()
}

// TODO: Test protodesc.NewFile with imported files.

func TestFile(t *testing.T) {
	f1 := &ptype.File{
		Syntax:  pref.Proto2,
		Path:    "path/to/file.proto",
		Package: "test",
		Options: &descriptorpb.FileOptions{Deprecated: scalar.Bool(true)},
		Messages: []ptype.Message{{
			Name: "A", // "test.A"
			Options: &descriptorpb.MessageOptions{
				MapEntry:   scalar.Bool(true),
				Deprecated: scalar.Bool(true),
			},
			IsMapEntry: true,
			Fields: []ptype.Field{{
				Name:        "key", // "test.A.key"
				Number:      1,
				Options:     &descriptorpb.FieldOptions{Deprecated: scalar.Bool(true)},
				Cardinality: pref.Optional,
				Kind:        pref.StringKind,
			}, {
				Name:        "value", // "test.A.value"
				Number:      2,
				Cardinality: pref.Optional,
				Kind:        pref.MessageKind,
				MessageType: ptype.PlaceholderMessage("test.B"),
			}},
		}, {
			Name: "B", // "test.B"
			Fields: []ptype.Field{{
				Name:        "field_one", // "test.B.field_one"
				Number:      1,
				Cardinality: pref.Optional,
				Kind:        pref.StringKind,
				Default:     pref.ValueOf("hello, \"world!\"\n"),
				OneofName:   "O1",
			}, {
				Name:        "field_two", // "test.B.field_two"
				JSONName:    "Field2",
				Number:      2,
				Cardinality: pref.Optional,
				Kind:        pref.EnumKind,
				Default:     pref.ValueOf(pref.EnumNumber(1)),
				EnumType:    ptype.PlaceholderEnum("test.E1"),
				OneofName:   "O2",
			}, {
				Name:        "field_three", // "test.B.field_three"
				Number:      3,
				Cardinality: pref.Optional,
				Kind:        pref.MessageKind,
				MessageType: ptype.PlaceholderMessage("test.C"),
				OneofName:   "O2",
			}, {
				Name:        "field_four", // "test.B.field_four"
				JSONName:    "Field4",
				Number:      4,
				Cardinality: pref.Repeated,
				Kind:        pref.MessageKind,
				MessageType: ptype.PlaceholderMessage("test.A"),
			}, {
				Name:        "field_five", // "test.B.field_five"
				Number:      5,
				Cardinality: pref.Repeated,
				Kind:        pref.Int32Kind,
				Options:     &descriptorpb.FieldOptions{Packed: scalar.Bool(true)},
				IsPacked:    ptype.True,
			}, {
				Name:        "field_six", // "test.B.field_six"
				Number:      6,
				Cardinality: pref.Required,
				Kind:        pref.BytesKind,
			}},
			Oneofs: []ptype.Oneof{
				{
					Name: "O1", // "test.B.O1"
					Options: &descriptorpb.OneofOptions{
						UninterpretedOption: []*descriptorpb.UninterpretedOption{
							{StringValue: []byte("option")},
						},
					},
				},
				{Name: "O2"}, // "test.B.O2"
			},
			ReservedNames:   []pref.Name{"fizz", "buzz"},
			ReservedRanges:  [][2]pref.FieldNumber{{100, 200}, {300, 301}},
			ExtensionRanges: [][2]pref.FieldNumber{{1000, 2000}, {3000, 3001}},
			ExtensionRangeOptions: []pref.ProtoMessage{
				0: (*descriptorpb.ExtensionRangeOptions)(nil),
				1: new(descriptorpb.ExtensionRangeOptions),
			},
		}, {
			Name: "C", // "test.C"
			Messages: []ptype.Message{{
				Name:   "A", // "test.C.A"
				Fields: []ptype.Field{{Name: "F", Number: 1, Cardinality: pref.Required, Kind: pref.BytesKind, Default: pref.ValueOf([]byte("dead\xbe\xef"))}},
			}},
			Enums: []ptype.Enum{{
				Name:   "E1", // "test.C.E1"
				Values: []ptype.EnumValue{{Name: "FOO", Number: 0}, {Name: "BAR", Number: 1}},
			}},
			Extensions: []ptype.Extension{{
				Name:         "X", // "test.C.X"
				Number:       1000,
				Cardinality:  pref.Repeated,
				Kind:         pref.MessageKind,
				Options:      &descriptorpb.FieldOptions{Packed: scalar.Bool(false)},
				IsPacked:     ptype.False,
				MessageType:  ptype.PlaceholderMessage("test.C"),
				ExtendedType: ptype.PlaceholderMessage("test.B"),
			}},
		}},
		Enums: []ptype.Enum{{
			Name:    "E1", // "test.E1"
			Options: &descriptorpb.EnumOptions{Deprecated: scalar.Bool(true)},
			Values: []ptype.EnumValue{
				{
					Name:    "FOO",
					Number:  0,
					Options: &descriptorpb.EnumValueOptions{Deprecated: scalar.Bool(true)},
				},
				{Name: "BAR", Number: 1},
			},
			ReservedNames:  []pref.Name{"FIZZ", "BUZZ"},
			ReservedRanges: [][2]pref.EnumNumber{{10, 19}, {30, 30}},
		}},
		Extensions: []ptype.Extension{{
			Name:         "X", // "test.X"
			Number:       1000,
			Cardinality:  pref.Repeated,
			Kind:         pref.MessageKind,
			Options:      &descriptorpb.FieldOptions{Packed: scalar.Bool(true)},
			IsPacked:     ptype.True,
			MessageType:  ptype.PlaceholderMessage("test.C"),
			ExtendedType: ptype.PlaceholderMessage("test.B"),
		}},
		Services: []ptype.Service{{
			Name:    "S", // "test.S"
			Options: &descriptorpb.ServiceOptions{Deprecated: scalar.Bool(true)},
			Methods: []ptype.Method{{
				Name:              "M", // "test.S.M"
				InputType:         ptype.PlaceholderMessage("test.A"),
				OutputType:        ptype.PlaceholderMessage("test.C.A"),
				IsStreamingClient: true,
				IsStreamingServer: true,
				Options:           &descriptorpb.MethodOptions{Deprecated: scalar.Bool(true)},
			}},
		}},
	}
	fd1, err := ptype.NewFile(f1)
	if err != nil {
		t.Fatalf("prototype.NewFile() error: %v", err)
	}

	f2 := &descriptorpb.FileDescriptorProto{
		Syntax:  scalar.String("proto2"),
		Name:    scalar.String("path/to/file.proto"),
		Package: scalar.String("test"),
		Options: &descriptorpb.FileOptions{Deprecated: scalar.Bool(true)},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: scalar.String("A"),
			Options: &descriptorpb.MessageOptions{
				MapEntry:   scalar.Bool(true),
				Deprecated: scalar.Bool(true),
			},
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:    scalar.String("key"),
				Number:  scalar.Int32(1),
				Options: &descriptorpb.FieldOptions{Deprecated: scalar.Bool(true)},
				Label:   descriptorpb.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:    descriptorpb.FieldDescriptorProto_Type(pref.StringKind).Enum(),
			}, {
				Name:     scalar.String("value"),
				Number:   scalar.Int32(2),
				Label:    descriptorpb.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:     descriptorpb.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName: scalar.String(".test.B"),
			}},
		}, {
			Name: scalar.String("B"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:         scalar.String("field_one"),
				Number:       scalar.Int32(1),
				Label:        descriptorpb.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:         descriptorpb.FieldDescriptorProto_Type(pref.StringKind).Enum(),
				DefaultValue: scalar.String("hello, \"world!\"\n"),
				OneofIndex:   scalar.Int32(0),
			}, {
				Name:         scalar.String("field_two"),
				JsonName:     scalar.String("Field2"),
				Number:       scalar.Int32(2),
				Label:        descriptorpb.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:         descriptorpb.FieldDescriptorProto_Type(pref.EnumKind).Enum(),
				DefaultValue: scalar.String("BAR"),
				TypeName:     scalar.String(".test.E1"),
				OneofIndex:   scalar.Int32(1),
			}, {
				Name:       scalar.String("field_three"),
				Number:     scalar.Int32(3),
				Label:      descriptorpb.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:       descriptorpb.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName:   scalar.String(".test.C"),
				OneofIndex: scalar.Int32(1),
			}, {
				Name:     scalar.String("field_four"),
				JsonName: scalar.String("Field4"),
				Number:   scalar.Int32(4),
				Label:    descriptorpb.FieldDescriptorProto_Label(pref.Repeated).Enum(),
				Type:     descriptorpb.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName: scalar.String(".test.A"),
			}, {
				Name:    scalar.String("field_five"),
				Number:  scalar.Int32(5),
				Label:   descriptorpb.FieldDescriptorProto_Label(pref.Repeated).Enum(),
				Type:    descriptorpb.FieldDescriptorProto_Type(pref.Int32Kind).Enum(),
				Options: &descriptorpb.FieldOptions{Packed: scalar.Bool(true)},
			}, {
				Name:   scalar.String("field_six"),
				Number: scalar.Int32(6),
				Label:  descriptorpb.FieldDescriptorProto_Label(pref.Required).Enum(),
				Type:   descriptorpb.FieldDescriptorProto_Type(pref.BytesKind).Enum(),
			}},
			OneofDecl: []*descriptorpb.OneofDescriptorProto{
				{
					Name: scalar.String("O1"),
					Options: &descriptorpb.OneofOptions{
						UninterpretedOption: []*descriptorpb.UninterpretedOption{
							{StringValue: []byte("option")},
						},
					},
				},
				{Name: scalar.String("O2")},
			},
			ReservedName: []string{"fizz", "buzz"},
			ReservedRange: []*descriptorpb.DescriptorProto_ReservedRange{
				{Start: scalar.Int32(100), End: scalar.Int32(200)},
				{Start: scalar.Int32(300), End: scalar.Int32(301)},
			},
			ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{
				{Start: scalar.Int32(1000), End: scalar.Int32(2000)},
				{Start: scalar.Int32(3000), End: scalar.Int32(3001), Options: new(descriptorpb.ExtensionRangeOptions)},
			},
		}, {
			Name: scalar.String("C"),
			NestedType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("A"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:         scalar.String("F"),
					Number:       scalar.Int32(1),
					Label:        descriptorpb.FieldDescriptorProto_Label(pref.Required).Enum(),
					Type:         descriptorpb.FieldDescriptorProto_Type(pref.BytesKind).Enum(),
					DefaultValue: scalar.String(`dead\276\357`),
				}},
			}},
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("E1"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: scalar.String("FOO"), Number: scalar.Int32(0)},
					{Name: scalar.String("BAR"), Number: scalar.Int32(1)},
				},
			}},
			Extension: []*descriptorpb.FieldDescriptorProto{{
				Name:     scalar.String("X"),
				Number:   scalar.Int32(1000),
				Label:    descriptorpb.FieldDescriptorProto_Label(pref.Repeated).Enum(),
				Type:     descriptorpb.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName: scalar.String(".test.C"),
				Extendee: scalar.String(".test.B"),
			}},
		}},
		EnumType: []*descriptorpb.EnumDescriptorProto{{
			Name:    scalar.String("E1"),
			Options: &descriptorpb.EnumOptions{Deprecated: scalar.Bool(true)},
			Value: []*descriptorpb.EnumValueDescriptorProto{
				{
					Name:    scalar.String("FOO"),
					Number:  scalar.Int32(0),
					Options: &descriptorpb.EnumValueOptions{Deprecated: scalar.Bool(true)},
				},
				{Name: scalar.String("BAR"), Number: scalar.Int32(1)},
			},
			ReservedName: []string{"FIZZ", "BUZZ"},
			ReservedRange: []*descriptorpb.EnumDescriptorProto_EnumReservedRange{
				{Start: scalar.Int32(10), End: scalar.Int32(19)},
				{Start: scalar.Int32(30), End: scalar.Int32(30)},
			},
		}},
		Extension: []*descriptorpb.FieldDescriptorProto{{
			Name:     scalar.String("X"),
			Number:   scalar.Int32(1000),
			Label:    descriptorpb.FieldDescriptorProto_Label(pref.Repeated).Enum(),
			Type:     descriptorpb.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
			Options:  &descriptorpb.FieldOptions{Packed: scalar.Bool(true)},
			TypeName: scalar.String(".test.C"),
			Extendee: scalar.String(".test.B"),
		}},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name:    scalar.String("S"),
			Options: &descriptorpb.ServiceOptions{Deprecated: scalar.Bool(true)},
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:            scalar.String("M"),
				InputType:       scalar.String(".test.A"),
				OutputType:      scalar.String(".test.C.A"),
				ClientStreaming: scalar.Bool(true),
				ServerStreaming: scalar.Bool(true),
				Options:         &descriptorpb.MethodOptions{Deprecated: scalar.Bool(true)},
			}},
		}},
	}
	fd2, err := pdesc.NewFile(f2, nil)
	if err != nil {
		t.Fatalf("protodesc.NewFile() error: %v", err)
	}

	tests := []struct {
		name string
		desc pref.FileDescriptor
	}{
		{"prototype.NewFile", fd1},
		{"protodesc.NewFile", fd2},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Run sub-tests in parallel to induce potential races.
			for i := 0; i < 2; i++ {
				t.Run("Accessors", func(t *testing.T) { t.Parallel(); testFileAccessors(t, tt.desc) })
				t.Run("Format", func(t *testing.T) { t.Parallel(); testFileFormat(t, tt.desc) })
			}
		})
	}
}

func testFileAccessors(t *testing.T, fd pref.FileDescriptor) {
	// Represent the descriptor as a map where each key is an accessor method
	// and the value is either the wanted tail value or another accessor map.
	type M = map[string]interface{}
	want := M{
		"Parent":        nil,
		"Index":         0,
		"Syntax":        pref.Proto2,
		"Name":          pref.Name("test"),
		"FullName":      pref.FullName("test"),
		"Path":          "path/to/file.proto",
		"Package":       pref.FullName("test"),
		"IsPlaceholder": false,
		"Options":       &descriptorpb.FileOptions{Deprecated: scalar.Bool(true)},
		"Messages": M{
			"Len": 3,
			"Get:0": M{
				"Parent":        M{"FullName": pref.FullName("test")},
				"Index":         0,
				"Syntax":        pref.Proto2,
				"Name":          pref.Name("A"),
				"FullName":      pref.FullName("test.A"),
				"IsPlaceholder": false,
				"IsMapEntry":    true,
				"Options": &descriptorpb.MessageOptions{
					MapEntry:   scalar.Bool(true),
					Deprecated: scalar.Bool(true),
				},
				"Fields": M{
					"Len": 2,
					"ByNumber:1": M{
						"Parent":       M{"FullName": pref.FullName("test.A")},
						"Index":        0,
						"Name":         pref.Name("key"),
						"FullName":     pref.FullName("test.A.key"),
						"Number":       pref.FieldNumber(1),
						"Cardinality":  pref.Optional,
						"Kind":         pref.StringKind,
						"Options":      &descriptorpb.FieldOptions{Deprecated: scalar.Bool(true)},
						"HasJSONName":  false,
						"JSONName":     "key",
						"IsPacked":     false,
						"IsMap":        false,
						"IsWeak":       false,
						"Default":      "",
						"OneofType":    nil,
						"ExtendedType": nil,
						"MessageType":  nil,
						"EnumType":     nil,
					},
					"ByNumber:2": M{
						"Parent":       M{"FullName": pref.FullName("test.A")},
						"Index":        1,
						"Name":         pref.Name("value"),
						"FullName":     pref.FullName("test.A.value"),
						"Number":       pref.FieldNumber(2),
						"Cardinality":  pref.Optional,
						"Kind":         pref.MessageKind,
						"JSONName":     "value",
						"IsPacked":     false,
						"IsMap":        false,
						"IsWeak":       false,
						"Default":      nil,
						"OneofType":    nil,
						"ExtendedType": nil,
						"MessageType":  M{"FullName": pref.FullName("test.B"), "IsPlaceholder": false},
						"EnumType":     nil,
					},
					"ByNumber:3": nil,
				},
				"Oneofs":          M{"Len": 0},
				"RequiredNumbers": M{"Len": 0},
				"ExtensionRanges": M{"Len": 0},
				"Messages":        M{"Len": 0},
				"Enums":           M{"Len": 0},
				"Extensions":      M{"Len": 0},
			},
			"ByName:B": M{
				"Name":  pref.Name("B"),
				"Index": 1,
				"Fields": M{
					"Len":                  6,
					"ByJSONName:field_one": nil,
					"ByJSONName:fieldOne": M{
						"Name":      pref.Name("field_one"),
						"Index":     0,
						"JSONName":  "fieldOne",
						"Default":   "hello, \"world!\"\n",
						"OneofType": M{"Name": pref.Name("O1"), "IsPlaceholder": false},
					},
					"ByJSONName:fieldTwo": nil,
					"ByJSONName:Field2": M{
						"Name":        pref.Name("field_two"),
						"Index":       1,
						"HasJSONName": true,
						"JSONName":    "Field2",
						"Default":     pref.EnumNumber(1),
						"OneofType":   M{"Name": pref.Name("O2"), "IsPlaceholder": false},
					},
					"ByName:fieldThree": nil,
					"ByName:field_three": M{
						"IsMap":       false,
						"MessageType": M{"FullName": pref.FullName("test.C"), "IsPlaceholder": false},
						"OneofType":   M{"Name": pref.Name("O2"), "IsPlaceholder": false},
					},
					"ByNumber:12": nil,
					"ByNumber:4": M{
						"Cardinality": pref.Repeated,
						"IsMap":       true,
						"Default":     nil,
						"MessageType": M{"FullName": pref.FullName("test.A"), "IsPlaceholder": false},
					},
					"ByNumber:5": M{
						"Cardinality": pref.Repeated,
						"Kind":        pref.Int32Kind,
						"IsPacked":    true,
						"Default":     int32(0),
					},
					"ByNumber:6": M{
						"Cardinality": pref.Required,
						"Default":     []byte(nil),
						"OneofType":   nil,
					},
				},
				"Oneofs": M{
					"Len":       2,
					"ByName:O0": nil,
					"ByName:O1": M{
						"FullName": pref.FullName("test.B.O1"),
						"Index":    0,
						"Options": &descriptorpb.OneofOptions{
							UninterpretedOption: []*descriptorpb.UninterpretedOption{
								{StringValue: []byte("option")},
							},
						},
						"Fields": M{
							"Len":   1,
							"Get:0": M{"FullName": pref.FullName("test.B.field_one")},
						},
					},
					"Get:1": M{
						"FullName": pref.FullName("test.B.O2"),
						"Index":    1,
						"Fields": M{
							"Len":              2,
							"ByName:field_two": M{"Name": pref.Name("field_two")},
							"Get:1":            M{"Name": pref.Name("field_three")},
						},
					},
				},
				"ReservedNames": M{
					"Len":         2,
					"Get:0":       pref.Name("fizz"),
					"Has:buzz":    true,
					"Has:noexist": false,
				},
				"ReservedRanges": M{
					"Len":     2,
					"Get:0":   [2]pref.FieldNumber{100, 200},
					"Has:99":  false,
					"Has:100": true,
					"Has:150": true,
					"Has:199": true,
					"Has:200": false,
					"Has:300": true,
					"Has:301": false,
				},
				"RequiredNumbers": M{
					"Len":   1,
					"Get:0": pref.FieldNumber(6),
					"Has:1": false,
					"Has:6": true,
				},
				"ExtensionRanges": M{
					"Len":      2,
					"Get:0":    [2]pref.FieldNumber{1000, 2000},
					"Has:999":  false,
					"Has:1000": true,
					"Has:1500": true,
					"Has:1999": true,
					"Has:2000": false,
					"Has:3000": true,
					"Has:3001": false,
				},
				"ExtensionRangeOptions:0": (*descriptorpb.ExtensionRangeOptions)(nil),
				"ExtensionRangeOptions:1": new(descriptorpb.ExtensionRangeOptions),
			},
			"Get:2": M{
				"Name":  pref.Name("C"),
				"Index": 2,
				"Messages": M{
					"Len":   1,
					"Get:0": M{"FullName": pref.FullName("test.C.A")},
				},
				"Enums": M{
					"Len":   1,
					"Get:0": M{"FullName": pref.FullName("test.C.E1")},
				},
				"Extensions": M{
					"Len":   1,
					"Get:0": M{"FullName": pref.FullName("test.C.X")},
				},
			},
		},
		"Enums": M{
			"Len": 1,
			"Get:0": M{
				"Name":    pref.Name("E1"),
				"Options": &descriptorpb.EnumOptions{Deprecated: scalar.Bool(true)},
				"Values": M{
					"Len":        2,
					"ByName:Foo": nil,
					"ByName:FOO": M{
						"FullName": pref.FullName("test.FOO"),
						"Options":  &descriptorpb.EnumValueOptions{Deprecated: scalar.Bool(true)},
					},
					"ByNumber:2": nil,
					"ByNumber:1": M{"FullName": pref.FullName("test.BAR")},
				},
				"ReservedNames": M{
					"Len":         2,
					"Get:0":       pref.Name("FIZZ"),
					"Has:BUZZ":    true,
					"Has:NOEXIST": false,
				},
				"ReservedRanges": M{
					"Len":    2,
					"Get:0":  [2]pref.EnumNumber{10, 19},
					"Has:9":  false,
					"Has:10": true,
					"Has:15": true,
					"Has:19": true,
					"Has:20": false,
					"Has:30": true,
					"Has:31": false,
				},
			},
		},
		"Extensions": M{
			"Len": 1,
			"ByName:X": M{
				"Name":         pref.Name("X"),
				"Number":       pref.FieldNumber(1000),
				"Cardinality":  pref.Repeated,
				"Kind":         pref.MessageKind,
				"IsPacked":     false,
				"MessageType":  M{"FullName": pref.FullName("test.C"), "IsPlaceholder": false},
				"ExtendedType": M{"FullName": pref.FullName("test.B"), "IsPlaceholder": false},
				"Options":      &descriptorpb.FieldOptions{Packed: scalar.Bool(true)},
			},
		},
		"Services": M{
			"Len":      1,
			"ByName:s": nil,
			"ByName:S": M{
				"Parent":   M{"FullName": pref.FullName("test")},
				"Name":     pref.Name("S"),
				"FullName": pref.FullName("test.S"),
				"Options":  &descriptorpb.ServiceOptions{Deprecated: scalar.Bool(true)},
				"Methods": M{
					"Len": 1,
					"Get:0": M{
						"Parent":            M{"FullName": pref.FullName("test.S")},
						"Name":              pref.Name("M"),
						"FullName":          pref.FullName("test.S.M"),
						"InputType":         M{"FullName": pref.FullName("test.A"), "IsPlaceholder": false},
						"OutputType":        M{"FullName": pref.FullName("test.C.A"), "IsPlaceholder": false},
						"IsStreamingClient": true,
						"IsStreamingServer": true,
						"Options":           &descriptorpb.MethodOptions{Deprecated: scalar.Bool(true)},
					},
				},
			},
		},
		"DescriptorByName:":                 nil,
		"DescriptorByName:A":                nil,
		"DescriptorByName:test":             nil,
		"DescriptorByName:test.":            nil,
		"DescriptorByName:test.A":           M{"FullName": pref.FullName("test.A")},
		"DescriptorByName:test.A.key":       M{"FullName": pref.FullName("test.A.key")},
		"DescriptorByName:test.A.A":         nil,
		"DescriptorByName:test.A.field_one": nil,
		"DescriptorByName:test.B.field_one": M{"FullName": pref.FullName("test.B.field_one")},
		"DescriptorByName:test.B.O1":        M{"FullName": pref.FullName("test.B.O1")},
		"DescriptorByName:test.B.O3":        nil,
		"DescriptorByName:test.C.E1":        M{"FullName": pref.FullName("test.C.E1")},
		"DescriptorByName:test.C.E1.FOO":    nil,
		"DescriptorByName:test.C.FOO":       M{"FullName": pref.FullName("test.C.FOO")},
		"DescriptorByName:test.C.Foo":       nil,
		"DescriptorByName:test.C.BAZ":       nil,
		"DescriptorByName:test.E1":          M{"FullName": pref.FullName("test.E1")},
		"DescriptorByName:test.E1.FOO":      nil,
		"DescriptorByName:test.FOO":         M{"FullName": pref.FullName("test.FOO")},
		"DescriptorByName:test.Foo":         nil,
		"DescriptorByName:test.BAZ":         nil,
		"DescriptorByName:test.C.X":         M{"FullName": pref.FullName("test.C.X")},
		"DescriptorByName:test.X":           M{"FullName": pref.FullName("test.X")},
		"DescriptorByName:test.X.":          nil,
		"DescriptorByName:test.S":           M{"FullName": pref.FullName("test.S")},
		"DescriptorByName:test.S.M":         M{"FullName": pref.FullName("test.S.M")},
		"DescriptorByName:test.M":           nil,
	}
	checkAccessors(t, "", reflect.ValueOf(fd), want)
}
func checkAccessors(t *testing.T, p string, rv reflect.Value, want map[string]interface{}) {
	p0 := p
	defer func() {
		if ex := recover(); ex != nil {
			t.Errorf("panic at %v: %v", p, ex)
		}
	}()

	if rv.Interface() == nil {
		t.Errorf("%v is nil, want non-nil", p)
		return
	}
	for s, v := range want {
		// Call the accessor method.
		p = p0 + "." + s
		var rets []reflect.Value
		if i := strings.IndexByte(s, ':'); i >= 0 {
			// Accessor method takes in a single argument, which is encoded
			// after the accessor name, separated by a ':' delimiter.
			fnc := rv.MethodByName(s[:i])
			arg := reflect.New(fnc.Type().In(0)).Elem()
			s = s[i+len(":"):]
			switch arg.Kind() {
			case reflect.String:
				arg.SetString(s)
			case reflect.Int32, reflect.Int:
				n, _ := strconv.ParseInt(s, 0, 64)
				arg.SetInt(n)
			}
			rets = fnc.Call([]reflect.Value{arg})
		} else {
			rets = rv.MethodByName(s).Call(nil)
		}

		// Check that (val, ok) pattern is internally consistent.
		if len(rets) == 2 {
			if rets[0].IsNil() && rets[1].Bool() {
				t.Errorf("%v = (nil, true), want (nil, false)", p)
			}
			if !rets[0].IsNil() && !rets[1].Bool() {
				t.Errorf("%v = (non-nil, false), want (non-nil, true)", p)
			}
		}

		// Check that the accessor output matches.
		if want, ok := v.(map[string]interface{}); ok {
			checkAccessors(t, p, rets[0], want)
			continue
		}

		got := rets[0].Interface()
		if pv, ok := got.(pref.Value); ok {
			got = pv.Interface()
		}

		// Compare with proto.Equal if possible.
		gotMsg, gotMsgOK := got.(protoV1.Message)
		wantMsg, wantMsgOK := v.(protoV1.Message)
		if gotMsgOK && wantMsgOK {
			if !protoV1.Equal(gotMsg, wantMsg) {
				t.Errorf("%v = %v, want %v", p, got, want)
			}
			continue
		}

		if want := v; !reflect.DeepEqual(got, want) {
			t.Errorf("%v = %T(%v), want %T(%v)", p, got, got, want, want)
		}
	}
}

func testFileFormat(t *testing.T, fd pref.FileDescriptor) {
	const want = `FileDescriptor{
	Syntax:  proto2
	Path:    "path/to/file.proto"
	Package: test
	Messages: [{
		Name:       A
		IsMapEntry: true
		Fields: [{
			Name:        key
			Number:      1
			Cardinality: optional
			Kind:        string
			JSONName:    "key"
		}, {
			Name:        value
			Number:      2
			Cardinality: optional
			Kind:        message
			JSONName:    "value"
			MessageType: test.B
		}]
	}, {
		Name: B
		Fields: [{
			Name:        field_one
			Number:      1
			Cardinality: optional
			Kind:        string
			JSONName:    "fieldOne"
			HasDefault:  true
			Default:     "hello, \"world!\"\n"
			OneofType:   O1
		}, {
			Name:        field_two
			Number:      2
			Cardinality: optional
			Kind:        enum
			HasJSONName: true
			JSONName:    "Field2"
			HasDefault:  true
			Default:     1
			OneofType:   O2
			EnumType:    test.E1
		}, {
			Name:        field_three
			Number:      3
			Cardinality: optional
			Kind:        message
			JSONName:    "fieldThree"
			OneofType:   O2
			MessageType: test.C
		}, {
			Name:        field_four
			Number:      4
			Cardinality: repeated
			Kind:        message
			HasJSONName: true
			JSONName:    "Field4"
			IsMap:       true
			MessageType: test.A
		}, {
			Name:        field_five
			Number:      5
			Cardinality: repeated
			Kind:        int32
			JSONName:    "fieldFive"
			IsPacked:    true
		}, {
			Name:        field_six
			Number:      6
			Cardinality: required
			Kind:        bytes
			JSONName:    "fieldSix"
		}]
		Oneofs: [{
			Name:   O1
			Fields: [field_one]
		}, {
			Name:   O2
			Fields: [field_two, field_three]
		}]
		ReservedNames:   [fizz, buzz]
		ReservedRanges:  [100:200, 300]
		RequiredNumbers: [6]
		ExtensionRanges: [1000:2000, 3000]
	}, {
		Name: C
		Messages: [{
			Name: A
			Fields: [{
				Name:        F
				Number:      1
				Cardinality: required
				Kind:        bytes
				JSONName:    "F"
				HasDefault:  true
				Default:     "dead\xbe\xef"
			}]
			RequiredNumbers: [1]
		}]
		Enums: [{
			Name: E1
			Values: [
				{Name: FOO}
				{Name: BAR, Number: 1}
			]
		}]
		Extensions: [{
			Name:         X
			Number:       1000
			Cardinality:  repeated
			Kind:         message
			ExtendedType: test.B
			MessageType:  test.C
		}]
	}]
	Enums: [{
		Name: E1
		Values: [
			{Name: FOO}
			{Name: BAR, Number: 1}
		]
		ReservedNames:  [FIZZ, BUZZ]
		ReservedRanges: [10:20, 30]
	}]
	Extensions: [{
		Name:         X
		Number:       1000
		Cardinality:  repeated
		Kind:         message
		ExtendedType: test.B
		MessageType:  test.C
	}]
	Services: [{
		Name: S
		Methods: [{
			Name:              M
			InputType:         test.A
			OutputType:        test.C.A
			IsStreamingClient: true
			IsStreamingServer: true
		}]
	}]
}`
	tests := []struct{ fmt, want string }{{"%v", compactMultiFormat(want)}, {"%+v", want}}
	for _, tt := range tests {
		got := fmt.Sprintf(tt.fmt, fd)
		if got != tt.want {
			t.Errorf("fmt.Sprintf(%q, fd):\ngot:  %s\nwant: %s", tt.fmt, got, tt.want)
		}
	}
}

// compactMultiFormat returns the single line form of a multi line output.
func compactMultiFormat(s string) string {
	var b []byte
	for _, s := range strings.Split(s, "\n") {
		s = strings.TrimSpace(s)
		s = regexp.MustCompile(": +").ReplaceAllString(s, ": ")
		prevWord := len(b) > 0 && b[len(b)-1] != '[' && b[len(b)-1] != '{'
		nextWord := len(s) > 0 && s[0] != ']' && s[0] != '}'
		if prevWord && nextWord {
			b = append(b, ", "...)
		}
		b = append(b, s...)
	}
	return string(b)
}
