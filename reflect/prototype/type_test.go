// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	pref "google.golang.org/proto/reflect/protoreflect"
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

func TestFile(t *testing.T) {
	f := &File{
		Syntax:  pref.Proto2,
		Path:    "path/to/file.proto",
		Package: "test",
		Messages: []Message{{
			Name:       "A", // "test.A"
			IsMapEntry: true,
			Fields: []Field{{
				Name:        "key", // "test.A.key"
				Number:      1,
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
				Default:     pref.ValueOf("hello"),
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
				IsPacked:    true,
			}, {
				Name:        "field_six", // "test.B.field_six"
				Number:      6,
				Cardinality: pref.Required,
				Kind:        pref.StringKind,
			}},
			Oneofs: []Oneof{
				{Name: "O1"}, // "test.B.O1"
				{Name: "O2"}, // "test.B.O2"
			},
			ExtensionRanges: [][2]pref.FieldNumber{{1000, 2000}},
		}, {
			Name: "C", // "test.C"
			Messages: []Message{{
				Name:   "A", // "test.C.A"
				Fields: []Field{{Name: "F", Number: 1, Cardinality: pref.Required, Kind: pref.BytesKind}},
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
				IsPacked:     false,
				MessageType:  PlaceholderMessage("test.C"),
				ExtendedType: PlaceholderMessage("test.B"),
			}},
		}},
		Enums: []Enum{{
			Name:   "E1", // "test.E1"
			Values: []EnumValue{{Name: "FOO", Number: 0}, {Name: "BAR", Number: 1}},
		}},
		Extensions: []Extension{{
			Name:         "X", // "test.X"
			Number:       1000,
			Cardinality:  pref.Repeated,
			Kind:         pref.MessageKind,
			IsPacked:     false,
			MessageType:  PlaceholderMessage("test.C"),
			ExtendedType: PlaceholderMessage("test.B"),
		}},
		Services: []Service{{
			Name: "S", // "test.S"
			Methods: []Method{{
				Name:              "M", // "test.S.M"
				InputType:         PlaceholderMessage("test.A"),
				OutputType:        PlaceholderMessage("test.C.A"),
				IsStreamingClient: true,
				IsStreamingServer: true,
			}},
		}},
	}

	fd, err := NewFile(f)
	if err != nil {
		t.Fatalf("NewFile() error: %v", err)
	}

	// Represent the descriptor as a map where each key is an accessor method
	// and the value is either the wanted tail value or another accessor map.
	type M = map[string]interface{}
	want := M{
		"Parent":        nil,
		"Syntax":        pref.Proto2,
		"Name":          pref.Name("test"),
		"FullName":      pref.FullName("test"),
		"Path":          "path/to/file.proto",
		"Package":       pref.FullName("test"),
		"IsPlaceholder": false,
		"Messages": M{
			"Len": 3,
			"Get:0": M{
				"Parent":        M{"FullName": pref.FullName("test")},
				"Syntax":        pref.Proto2,
				"Name":          pref.Name("A"),
				"FullName":      pref.FullName("test.A"),
				"IsPlaceholder": false,
				"IsMapEntry":    true,
				"Fields": M{
					"Len": 2,
					"ByNumber:1": M{
						"Parent":       M{"FullName": pref.FullName("test.A")},
						"Name":         pref.Name("key"),
						"FullName":     pref.FullName("test.A.key"),
						"Number":       pref.FieldNumber(1),
						"Cardinality":  pref.Optional,
						"Kind":         pref.StringKind,
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
				"Name": pref.Name("B"),
				"Fields": M{
					"Len":                  6,
					"ByJSONName:field_one": nil,
					"ByJSONName:fieldOne": M{
						"Name":      pref.Name("field_one"),
						"JSONName":  "fieldOne",
						"Default":   "hello",
						"OneofType": M{"Name": pref.Name("O1"), "IsPlaceholder": false},
					},
					"ByJSONName:fieldTwo": nil,
					"ByJSONName:Field2": M{
						"Name":      pref.Name("field_two"),
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
						"Default":     "",
						"OneofType":   nil,
					},
				},
				"Oneofs": M{
					"Len":       2,
					"ByName:O0": nil,
					"ByName:O1": M{
						"FullName": pref.FullName("test.B.O1"),
						"Fields": M{
							"Len":   1,
							"Get:0": M{"FullName": pref.FullName("test.B.field_one")},
						},
					},
					"Get:1": M{
						"FullName": pref.FullName("test.B.O2"),
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
					"Len":      1,
					"Get:0":    [2]pref.FieldNumber{1000, 2000},
					"Has:999":  false,
					"Has:1000": true,
					"Has:1500": true,
					"Has:1999": true,
					"Has:2000": false,
				},
			},
			"Get:2": M{
				"Name": pref.Name("C"),
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
				"Name": pref.Name("E1"),
				"Values": M{
					"Len":        2,
					"ByName:Foo": nil,
					"ByName:FOO": M{"FullName": pref.FullName("test.FOO")},
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
				"IsPacked":     false,
				"MessageType":  M{"FullName": pref.FullName("test.C"), "IsPlaceholder": false},
				"ExtendedType": M{"FullName": pref.FullName("test.B"), "IsPlaceholder": false},
			},
		},
		"Services": M{
			"Len":      1,
			"ByName:s": nil,
			"ByName:S": M{
				"Parent":   M{"FullName": pref.FullName("test")},
				"Name":     pref.Name("S"),
				"FullName": pref.FullName("test.S"),
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

	// Concurrently explore the file tree to induce races.
	const numGoRoutines = 2
	var wg sync.WaitGroup
	defer wg.Wait()
	for i := 0; i < numGoRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkAccessors(t, "", reflect.ValueOf(fd), want)
		}()
	}
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
		} else {
			got := rets[0].Interface()
			if pv, ok := got.(pref.Value); ok {
				got = pv.Interface()
			}
			if want := v; !reflect.DeepEqual(got, want) {
				t.Errorf("%v = %v, want %v", p, got, want)
			}
		}
	}
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
