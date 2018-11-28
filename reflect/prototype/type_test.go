// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	protoV1 "github.com/golang/protobuf/proto"
	descriptorV1 "github.com/golang/protobuf/protoc-gen-go/descriptor"

	scalar "github.com/golang/protobuf/v2/internal/scalar"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// TestDescriptors tests that the implementations do not declare additional
// methods that do not exist on the interface types.
func TestDescriptors(t *testing.T) {
	tests := []interface{}{
		[]pref.FileDescriptor{placeholderFile{}, fileDesc{}},
		[]pref.MessageDescriptor{placeholderMessage{}, standaloneMessage{}, messageDesc{}},
		[]pref.FieldDescriptor{standaloneExtension{}, fieldDesc{}, extensionDesc{}},
		[]pref.OneofDescriptor{oneofDesc{}},
		[]pref.EnumDescriptor{placeholderEnum{}, standaloneEnum{}, enumDesc{}},
		[]pref.EnumValueDescriptor{enumValueDesc{}},
		[]pref.ServiceDescriptor{serviceDesc{}},
		[]pref.MethodDescriptor{methodDesc{}},

		[]pref.FileImports{(*fileImports)(nil)},
		[]pref.MessageDescriptors{(*messages)(nil)},
		[]pref.FieldNumbers{(*numbers)(nil)},
		[]pref.FieldRanges{(*ranges)(nil)},
		[]pref.FieldDescriptors{(*fields)(nil), (*oneofFields)(nil)},
		[]pref.OneofDescriptors{(*oneofs)(nil)},
		[]pref.ExtensionDescriptors{(*extensions)(nil)},
		[]pref.EnumDescriptors{(*enums)(nil)},
		[]pref.EnumValueDescriptors{(*enumValues)(nil)},
		[]pref.ServiceDescriptors{(*services)(nil)},
		[]pref.MethodDescriptors{(*methods)(nil)},
	}

	for _, tt := range tests {
		v := reflect.ValueOf(tt) // []T where T is an interface
		ifaceType := v.Type().Elem()
		for i := 0; i < v.Len(); i++ {
			implType := v.Index(i).Elem().Type()

			var hasName bool
			for j := 0; j < implType.NumMethod(); j++ {
				if name := implType.Method(j).Name; name == "Format" {
					hasName = true
				} else if _, ok := ifaceType.MethodByName(name); !ok {
					t.Errorf("spurious method: %v.%v", implType, name)
				}
			}
			if !hasName {
				t.Errorf("missing method: %v.Format", implType)
			}
		}
	}
}

// TestDescriptorAccessors tests that descriptorAccessors is up-to-date.
func TestDescriptorAccessors(t *testing.T) {
	ignore := map[string]bool{
		"DefaultEnumValue": true,
		"DescriptorByName": true,
		"ProtoType":        true,
	}
	rt := reflect.TypeOf((*pref.Descriptor)(nil)).Elem()
	for i := 0; i < rt.NumMethod(); i++ {
		ignore[rt.Method(i).Name] = true
	}

	for rt, m := range descriptorAccessors {
		got := map[string]bool{}
		for _, s := range m {
			got[s] = true
		}
		want := map[string]bool{}
		for i := 0; i < rt.NumMethod(); i++ {
			want[rt.Method(i).Name] = true
		}

		// Check if descriptorAccessors contains a non-existent accessor.
		// If this test fails, remove the accessor from descriptorAccessors.
		for s := range got {
			if !want[s] && !ignore[s] {
				t.Errorf("%v.%v does not exist", rt, s)
			}
		}

		// Check if there are new protoreflect interface methods that are not
		// handled by the formatter. If this fails, either add the method to
		// ignore or add them to descriptorAccessors.
		for s := range want {
			if !got[s] && !ignore[s] {
				t.Errorf("%v.%v is not called by formatter", rt, s)
			}
		}
	}
}

// TODO: Test NewFileFromDescriptorProto with imported files.

func TestFile(t *testing.T) {
	f1 := &File{
		Syntax:  pref.Proto2,
		Path:    "path/to/file.proto",
		Package: "test",
		Options: &descriptorV1.FileOptions{Deprecated: scalar.Bool(true)},
		Messages: []Message{{
			Name: "A", // "test.A"
			Options: &descriptorV1.MessageOptions{
				MapEntry:   scalar.Bool(true),
				Deprecated: scalar.Bool(true),
			},
			Fields: []Field{{
				Name:        "key", // "test.A.key"
				Number:      1,
				Options:     &descriptorV1.FieldOptions{Deprecated: scalar.Bool(true)},
				Cardinality: pref.Optional,
				Kind:        pref.StringKind,
			}, {
				Name:        "value", // "test.A.value"
				Number:      2,
				Cardinality: pref.Optional,
				Kind:        pref.MessageKind,
				MessageType: PlaceholderMessage("test.B"),
			}},
		}, {
			Name: "B", // "test.B"
			Fields: []Field{{
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
				EnumType:    PlaceholderEnum("test.E1"),
				OneofName:   "O2",
			}, {
				Name:        "field_three", // "test.B.field_three"
				Number:      3,
				Cardinality: pref.Optional,
				Kind:        pref.MessageKind,
				MessageType: PlaceholderMessage("test.C"),
				OneofName:   "O2",
			}, {
				Name:        "field_four", // "test.B.field_four"
				JSONName:    "Field4",
				Number:      4,
				Cardinality: pref.Repeated,
				Kind:        pref.MessageKind,
				MessageType: PlaceholderMessage("test.A"),
			}, {
				Name:        "field_five", // "test.B.field_five"
				Number:      5,
				Cardinality: pref.Repeated,
				Kind:        pref.Int32Kind,
				Options:     &descriptorV1.FieldOptions{Packed: scalar.Bool(true)},
			}, {
				Name:        "field_six", // "test.B.field_six"
				Number:      6,
				Cardinality: pref.Required,
				Kind:        pref.BytesKind,
			}},
			Oneofs: []Oneof{
				{
					Name: "O1", // "test.B.O1"
					Options: &descriptorV1.OneofOptions{
						UninterpretedOption: []*descriptorV1.UninterpretedOption{
							{StringValue: []byte("option")},
						},
					},
				},
				{Name: "O2"}, // "test.B.O2"
			},
			ExtensionRanges: [][2]pref.FieldNumber{{1000, 2000}, {3000, 3001}},
		}, {
			Name: "C", // "test.C"
			Messages: []Message{{
				Name:   "A", // "test.C.A"
				Fields: []Field{{Name: "F", Number: 1, Cardinality: pref.Required, Kind: pref.BytesKind, Default: pref.ValueOf([]byte("dead\xbe\xef"))}},
			}},
			Enums: []Enum{{
				Name:   "E1", // "test.C.E1"
				Values: []EnumValue{{Name: "FOO", Number: 0}, {Name: "BAR", Number: 1}},
			}},
			Extensions: []Extension{{
				Name:         "X", // "test.C.X"
				Number:       1000,
				Cardinality:  pref.Repeated,
				Kind:         pref.MessageKind,
				Options:      &descriptorV1.FieldOptions{Packed: scalar.Bool(false)},
				MessageType:  PlaceholderMessage("test.C"),
				ExtendedType: PlaceholderMessage("test.B"),
			}},
		}},
		Enums: []Enum{{
			Name:    "E1", // "test.E1"
			Options: &descriptorV1.EnumOptions{Deprecated: scalar.Bool(true)},
			Values: []EnumValue{
				{
					Name:    "FOO",
					Number:  0,
					Options: &descriptorV1.EnumValueOptions{Deprecated: scalar.Bool(true)},
				},
				{Name: "BAR", Number: 1},
			},
		}},
		Extensions: []Extension{{
			Name:         "X", // "test.X"
			Number:       1000,
			Cardinality:  pref.Repeated,
			Kind:         pref.MessageKind,
			Options:      &descriptorV1.FieldOptions{Packed: scalar.Bool(true)},
			MessageType:  PlaceholderMessage("test.C"),
			ExtendedType: PlaceholderMessage("test.B"),
		}},
		Services: []Service{{
			Name:    "S", // "test.S"
			Options: &descriptorV1.ServiceOptions{Deprecated: scalar.Bool(true)},
			Methods: []Method{{
				Name:              "M", // "test.S.M"
				InputType:         PlaceholderMessage("test.A"),
				OutputType:        PlaceholderMessage("test.C.A"),
				IsStreamingClient: true,
				IsStreamingServer: true,
				Options:           &descriptorV1.MethodOptions{Deprecated: scalar.Bool(true)},
			}},
		}},
	}
	fd1, err := NewFile(f1)
	if err != nil {
		t.Fatalf("NewFile() error: %v", err)
	}

	f2 := &descriptorV1.FileDescriptorProto{
		Syntax:  scalar.String("proto2"),
		Name:    scalar.String("path/to/file.proto"),
		Package: scalar.String("test"),
		Options: &descriptorV1.FileOptions{Deprecated: scalar.Bool(true)},
		MessageType: []*descriptorV1.DescriptorProto{{
			Name: scalar.String("A"),
			Options: &descriptorV1.MessageOptions{
				MapEntry:   scalar.Bool(true),
				Deprecated: scalar.Bool(true),
			},
			Field: []*descriptorV1.FieldDescriptorProto{{
				Name:    scalar.String("key"),
				Number:  scalar.Int32(1),
				Options: &descriptorV1.FieldOptions{Deprecated: scalar.Bool(true)},
				Label:   descriptorV1.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:    descriptorV1.FieldDescriptorProto_Type(pref.StringKind).Enum(),
			}, {
				Name:     scalar.String("value"),
				Number:   scalar.Int32(2),
				Label:    descriptorV1.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:     descriptorV1.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName: scalar.String(".test.B"),
			}},
		}, {
			Name: scalar.String("B"),
			Field: []*descriptorV1.FieldDescriptorProto{{
				Name:         scalar.String("field_one"),
				Number:       scalar.Int32(1),
				Label:        descriptorV1.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:         descriptorV1.FieldDescriptorProto_Type(pref.StringKind).Enum(),
				DefaultValue: scalar.String("hello, \"world!\"\n"),
				OneofIndex:   scalar.Int32(0),
			}, {
				Name:         scalar.String("field_two"),
				JsonName:     scalar.String("Field2"),
				Number:       scalar.Int32(2),
				Label:        descriptorV1.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:         descriptorV1.FieldDescriptorProto_Type(pref.EnumKind).Enum(),
				DefaultValue: scalar.String("BAR"),
				TypeName:     scalar.String(".test.E1"),
				OneofIndex:   scalar.Int32(1),
			}, {
				Name:       scalar.String("field_three"),
				Number:     scalar.Int32(3),
				Label:      descriptorV1.FieldDescriptorProto_Label(pref.Optional).Enum(),
				Type:       descriptorV1.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName:   scalar.String(".test.C"),
				OneofIndex: scalar.Int32(1),
			}, {
				Name:     scalar.String("field_four"),
				JsonName: scalar.String("Field4"),
				Number:   scalar.Int32(4),
				Label:    descriptorV1.FieldDescriptorProto_Label(pref.Repeated).Enum(),
				Type:     descriptorV1.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName: scalar.String(".test.A"),
			}, {
				Name:    scalar.String("field_five"),
				Number:  scalar.Int32(5),
				Label:   descriptorV1.FieldDescriptorProto_Label(pref.Repeated).Enum(),
				Type:    descriptorV1.FieldDescriptorProto_Type(pref.Int32Kind).Enum(),
				Options: &descriptorV1.FieldOptions{Packed: scalar.Bool(true)},
			}, {
				Name:   scalar.String("field_six"),
				Number: scalar.Int32(6),
				Label:  descriptorV1.FieldDescriptorProto_Label(pref.Required).Enum(),
				Type:   descriptorV1.FieldDescriptorProto_Type(pref.BytesKind).Enum(),
			}},
			OneofDecl: []*descriptorV1.OneofDescriptorProto{
				{
					Name: scalar.String("O1"),
					Options: &descriptorV1.OneofOptions{
						UninterpretedOption: []*descriptorV1.UninterpretedOption{
							{StringValue: []byte("option")},
						},
					},
				},
				{Name: scalar.String("O2")},
			},
			ExtensionRange: []*descriptorV1.DescriptorProto_ExtensionRange{
				{Start: scalar.Int32(1000), End: scalar.Int32(2000)},
				{Start: scalar.Int32(3000), End: scalar.Int32(3001)},
			},
		}, {
			Name: scalar.String("C"),
			NestedType: []*descriptorV1.DescriptorProto{{
				Name: scalar.String("A"),
				Field: []*descriptorV1.FieldDescriptorProto{{
					Name:         scalar.String("F"),
					Number:       scalar.Int32(1),
					Label:        descriptorV1.FieldDescriptorProto_Label(pref.Required).Enum(),
					Type:         descriptorV1.FieldDescriptorProto_Type(pref.BytesKind).Enum(),
					DefaultValue: scalar.String(`dead\276\357`),
				}},
			}},
			EnumType: []*descriptorV1.EnumDescriptorProto{{
				Name: scalar.String("E1"),
				Value: []*descriptorV1.EnumValueDescriptorProto{
					{Name: scalar.String("FOO"), Number: scalar.Int32(0)},
					{Name: scalar.String("BAR"), Number: scalar.Int32(1)},
				},
			}},
			Extension: []*descriptorV1.FieldDescriptorProto{{
				Name:     scalar.String("X"),
				Number:   scalar.Int32(1000),
				Label:    descriptorV1.FieldDescriptorProto_Label(pref.Repeated).Enum(),
				Type:     descriptorV1.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
				TypeName: scalar.String(".test.C"),
				Extendee: scalar.String(".test.B"),
			}},
		}},
		EnumType: []*descriptorV1.EnumDescriptorProto{{
			Name:    scalar.String("E1"),
			Options: &descriptorV1.EnumOptions{Deprecated: scalar.Bool(true)},
			Value: []*descriptorV1.EnumValueDescriptorProto{
				{
					Name:    scalar.String("FOO"),
					Number:  scalar.Int32(0),
					Options: &descriptorV1.EnumValueOptions{Deprecated: scalar.Bool(true)},
				},
				{Name: scalar.String("BAR"), Number: scalar.Int32(1)},
			},
		}},
		Extension: []*descriptorV1.FieldDescriptorProto{{
			Name:     scalar.String("X"),
			Number:   scalar.Int32(1000),
			Label:    descriptorV1.FieldDescriptorProto_Label(pref.Repeated).Enum(),
			Type:     descriptorV1.FieldDescriptorProto_Type(pref.MessageKind).Enum(),
			Options:  &descriptorV1.FieldOptions{Packed: scalar.Bool(true)},
			TypeName: scalar.String(".test.C"),
			Extendee: scalar.String(".test.B"),
		}},
		Service: []*descriptorV1.ServiceDescriptorProto{{
			Name:    scalar.String("S"),
			Options: &descriptorV1.ServiceOptions{Deprecated: scalar.Bool(true)},
			Method: []*descriptorV1.MethodDescriptorProto{{
				Name:            scalar.String("M"),
				InputType:       scalar.String(".test.A"),
				OutputType:      scalar.String(".test.C.A"),
				ClientStreaming: scalar.Bool(true),
				ServerStreaming: scalar.Bool(true),
				Options:         &descriptorV1.MethodOptions{Deprecated: scalar.Bool(true)},
			}},
		}},
	}
	fd2, err := NewFileFromDescriptorProto(f2, nil)
	if err != nil {
		t.Fatalf("NewFileFromDescriptorProto() error: %v", err)
	}

	tests := []struct {
		name string
		desc pref.FileDescriptor
	}{
		{"NewFile", fd1},
		{"NewFileFromDescriptorProto", fd2},
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
		"Options":       &descriptorV1.FileOptions{Deprecated: scalar.Bool(true)},
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
				"Options": &descriptorV1.MessageOptions{
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
						"Options":      &descriptorV1.FieldOptions{Deprecated: scalar.Bool(true)},
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
						"Name":      pref.Name("field_two"),
						"Index":     1,
						"JSONName":  "Field2",
						"Default":   pref.EnumNumber(1),
						"OneofType": M{"Name": pref.Name("O2"), "IsPlaceholder": false},
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
						"Options": &descriptorV1.OneofOptions{
							UninterpretedOption: []*descriptorV1.UninterpretedOption{
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
				},
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
				"Options": &descriptorV1.EnumOptions{Deprecated: scalar.Bool(true)},
				"Values": M{
					"Len":        2,
					"ByName:Foo": nil,
					"ByName:FOO": M{
						"FullName": pref.FullName("test.FOO"),
						"Options":  &descriptorV1.EnumValueOptions{Deprecated: scalar.Bool(true)},
					},
					"ByNumber:2": nil,
					"ByNumber:1": M{"FullName": pref.FullName("test.BAR")},
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
				"IsPacked":     true,
				"MessageType":  M{"FullName": pref.FullName("test.C"), "IsPlaceholder": false},
				"ExtendedType": M{"FullName": pref.FullName("test.B"), "IsPlaceholder": false},
				"Options":      &descriptorV1.FieldOptions{Packed: scalar.Bool(true)},
			},
		},
		"Services": M{
			"Len":      1,
			"ByName:s": nil,
			"ByName:S": M{
				"Parent":   M{"FullName": pref.FullName("test")},
				"Name":     pref.Name("S"),
				"FullName": pref.FullName("test.S"),
				"Options":  &descriptorV1.ServiceOptions{Deprecated: scalar.Bool(true)},
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
						"Options":           &descriptorV1.MethodOptions{Deprecated: scalar.Bool(true)},
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
	if rv.Interface() == nil {
		t.Errorf("%v is nil, want non-nil", p)
		return
	}
	for s, v := range want {
		// Call the accessor method.
		p := p + "." + s
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
			t.Errorf("%v = %v, want %v", p, got, want)
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
	}]
	Extensions: [{
		Name:         X
		Number:       1000
		Cardinality:  repeated
		Kind:         message
		IsPacked:     true
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
		got = strings.Replace(got, "FileDescriptor ", "FileDescriptor", 1) // cleanup randomizer
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

func TestResolve(t *testing.T) {
	f := &File{
		Syntax:  pref.Proto2,
		Package: "test",
		Messages: []Message{{
			Name:   "FooMessage",
			Fields: []Field{{Name: "F", Number: 1, Cardinality: pref.Optional, Kind: pref.BytesKind}},
			Messages: []Message{{
				Name:   "FooMessage",
				Fields: []Field{{Name: "F", Number: 1, Cardinality: pref.Optional, Kind: pref.BytesKind}},
			}, {
				Name:   "BarMessage",
				Fields: []Field{{Name: "F", Number: 1, Cardinality: pref.Optional, Kind: pref.BytesKind}},
			}},
			Enums: []Enum{{
				Name:   "FooEnum",
				Values: []EnumValue{{Name: "E", Number: 0}},
			}, {
				Name:   "BarEnum",
				Values: []EnumValue{{Name: "E", Number: 0}},
			}},
		}, {
			Name:   "BarMessage",
			Fields: []Field{{Name: "F", Number: 1, Cardinality: pref.Optional, Kind: pref.BytesKind}},
		}},
		Enums: []Enum{{
			Name:   "FooEnum",
			Values: []EnumValue{{Name: "E", Number: 0}},
		}, {
			Name:   "BarEnum",
			Values: []EnumValue{{Name: "E", Number: 0}},
		}},
	}

	fd, err := NewFile(f)
	if err != nil {
		t.Fatalf("NewFile() error: %v", err)
	}

	tests := []struct {
		parent pref.Descriptor
		name   pref.FullName
		want   pref.Descriptor
	}{{
		parent: fd.Enums().Get(0),
		name:   "test.Foo",
		want:   nil,
	}, {
		parent: fd.Enums().Get(0),
		name:   "test.FooEnum",
		want:   fd.Enums().Get(0),
	}, {
		parent: fd.Enums().Get(0),
		name:   "test.BarEnum",
		want:   fd.Enums().Get(1),
	}, {
		parent: fd.Enums().Get(0),
		name:   "test.BarMessage",
		want:   fd.Messages().Get(1),
	}, {
		parent: fd.Enums().Get(0),
		name:   "test.FooMessage.BarMessage",
		want:   fd.Messages().Get(0).Messages().Get(1),
	}, {
		parent: fd.Enums().Get(0),
		name:   "test.FooMessage.Bar",
		want:   nil,
	}, {
		parent: fd.Messages().Get(1),
		name:   "test.FooMessage.BarEnum",
		want:   fd.Messages().Get(0).Enums().Get(1),
	}, {
		parent: fd.Messages().Get(1),
		name:   "test.FooEnum",
		want:   fd.Enums().Get(0),
	}, {
		parent: fd.Messages().Get(0),
		name:   "test.FooEnum",
		want:   fd.Enums().Get(0),
	}, {
		parent: fd.Messages().Get(0),
		name:   "test.FooEnum.NonExistent",
		want:   nil,
	}, {
		parent: fd.Messages().Get(0),
		name:   "test.FooMessage.FooEnum",
		want:   fd.Messages().Get(0).Enums().Get(0),
	}, {
		parent: fd.Messages().Get(0),
		name:   "test.FooMessage",
		want:   fd.Messages().Get(0),
	}, {
		parent: fd.Messages().Get(0),
		name:   "test.FooMessage.Fizz",
		want:   nil,
	}, {
		parent: fd.Messages().Get(0).Messages().Get(0),
		name:   "test.FooMessage.FooMessage",
		want:   fd.Messages().Get(0).Messages().Get(0),
	}, {
		parent: fd.Messages().Get(0).Messages().Get(0),
		name:   "test.FooMessage.BarMessage",
		want:   fd.Messages().Get(0).Messages().Get(1),
	}, {
		parent: fd.Messages().Get(0).Messages().Get(0),
		name:   "test.BarMessage.FooMessage",
		want:   nil,
	}, {
		parent: fd.Messages().Get(0).Messages().Get(0),
		name:   "test.BarMessage",
		want:   fd.Messages().Get(1),
	}, {
		parent: fd.Messages().Get(0).Messages().Get(0),
		name:   "test.BarMessageExtra",
		want:   nil,
	}, {
		parent: fd.Messages().Get(0).Messages().Get(0),
		name:   "taste.BarMessage",
		want:   nil,
	}}

	for _, tt := range tests {
		got := resolveReference(tt.parent, tt.name)
		if got != tt.want {
			fullName := func(d pref.Descriptor) string {
				if d == nil {
					return "<nil>"
				}
				return string(d.FullName())
			}
			t.Errorf("resolveReference(%v, %v) = %v, want %v", fullName(tt.parent), tt.name, fullName(got), fullName(tt.want))
		}
	}
}
