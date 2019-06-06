// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protodesc

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/internal/scalar"
	"google.golang.org/protobuf/reflect/protoregistry"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Tests validation logic for malformed descriptors.
func TestNewFile_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name    string
		deps    []*descriptorpb.FileDescriptorProto
		fd      *descriptorpb.FileDescriptorProto
		wantErr string
	}{{
		name: "field number reserved",
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("field-number-reserved.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("BadMessage"),
				ReservedRange: []*descriptorpb.DescriptorProto_ReservedRange{{
					Start: scalar.Int32(3),
					End:   scalar.Int32(4),
				}},
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("good_field"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				}, {
					Name:   scalar.String("bad_field"),
					Number: scalar.Int32(3),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				}},
			}},
		},
		wantErr: "reserved number 3",
	}, {
		name: "field name reserved",
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("field-name-reserved.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name:         scalar.String("BadMessage"),
				ReservedName: []string{"bad_field", "baz"},
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("good_field"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				}, {
					Name:   scalar.String("bad_field"),
					Number: scalar.Int32(3),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				}},
			}},
		},
		wantErr: `reserved name "bad_field"`,
	}, {
		name: "normal field with extendee",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("extensible.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("ExtensibleMessage"),
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{{
					Start: scalar.Int32(1000),
					End:   scalar.Int32(2000),
				}},
			}},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:       scalar.String("field-with-extendee.proto"),
			Syntax:     scalar.String("proto2"),
			Package:    scalar.String("foo"),
			Dependency: []string{"extensible.proto"},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("BadMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("good_field"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				}, {
					Name:     scalar.String("bad_field"),
					Number:   scalar.Int32(3),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					Extendee: scalar.String(".foo.ExtensibleMessage"),
				}},
			}},
		},
		wantErr: "may not have extendee",
	}, {
		name: "type_name on int32 field",
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("int32-with-type-name.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("AnEnum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   scalar.String("UNKNOWN"),
					Number: scalar.Int32(0),
				}},
			}},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("BadMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("good_field"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				}, {
					Name:     scalar.String("bad_field"),
					Number:   scalar.Int32(3),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					TypeName: scalar.String("AnEnum"),
				}},
			}},
		},
		wantErr: "type_name",
	}, {
		name: "type_name on string extension",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("extensible.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("ExtensibleMessage"),
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{{
					Start: scalar.Int32(1000),
					End:   scalar.Int32(2000),
				}},
			}},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:       scalar.String("string-ext-with-type-name.proto"),
			Syntax:     scalar.String("proto2"),
			Package:    scalar.String("bar"),
			Dependency: []string{"extensible.proto"},
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("AnEnum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   scalar.String("UNKNOWN"),
					Number: scalar.Int32(0),
				}},
			}},
			Extension: []*descriptorpb.FieldDescriptorProto{{
				Name:     scalar.String("my_ext"),
				Number:   scalar.Int32(1000),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Extendee: scalar.String(".foo.ExtensibleMessage"),
				TypeName: scalar.String("AnEnum"),
			}},
		},
		wantErr: "type_name",
	}, {
		name: "oneof_index on extension",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("extensible.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("ExtensibleMessage"),
				ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{{
					Start: scalar.Int32(1000),
					End:   scalar.Int32(2000),
				}},
			}},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:       scalar.String("ext-with-oneof-index.proto"),
			Syntax:     scalar.String("proto2"),
			Package:    scalar.String("bar"),
			Dependency: []string{"extensible.proto"},
			Extension: []*descriptorpb.FieldDescriptorProto{{
				Name:       scalar.String("my_ext"),
				Number:     scalar.Int32(1000),
				Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Extendee:   scalar.String(".foo.ExtensibleMessage"),
				OneofIndex: scalar.Int32(0),
			}},
		},
		wantErr: "oneof_index",
	}, {
		name: "enum with reserved number",
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("enum-with-reserved-number.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("AnEnum"),
				ReservedRange: []*descriptorpb.EnumDescriptorProto_EnumReservedRange{{
					Start: scalar.Int32(5),
					End:   scalar.Int32(6),
				}, {
					Start: scalar.Int32(10),
					End:   scalar.Int32(12),
				}},
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   scalar.String("UNKNOWN"),
					Number: scalar.Int32(0),
				}, {
					Name:   scalar.String("FOO"),
					Number: scalar.Int32(1),
				}, {
					Name:   scalar.String("BAR"),
					Number: scalar.Int32(2),
				}, {
					Name:   scalar.String("BAD"),
					Number: scalar.Int32(11),
				}},
			}},
		},
		wantErr: "reserved number 11",
	}, {
		name: "enum with reserved number",
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("enum-with-reserved-name.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("ParentMessage"),
				EnumType: []*descriptorpb.EnumDescriptorProto{{
					Name:         scalar.String("AnEnum"),
					ReservedName: []string{"ABC", "XYZ"},
					Value: []*descriptorpb.EnumValueDescriptorProto{{
						Name:   scalar.String("UNKNOWN"),
						Number: scalar.Int32(0),
					}, {
						Name:   scalar.String("FOO"),
						Number: scalar.Int32(1),
					}, {
						Name:   scalar.String("BAR"),
						Number: scalar.Int32(2),
					}, {
						Name:   scalar.String("XYZ"),
						Number: scalar.Int32(3),
					}},
				}},
			}},
		},
		wantErr: `reserved name "XYZ"`,
	}, {
		name: "message dependency without import",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("foo.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Foo"),
			}},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("message-dependency-without-import.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("bar"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("id"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}, {
					Name:     scalar.String("foo"),
					Number:   scalar.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: scalar.String(".foo.Foo"),
				}},
			}},
		},
		wantErr: "foo.Foo without import of foo.proto",
	}, {
		name: "enum dependency without import",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("foo.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("Foo"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   scalar.String("UNKNOWN"),
					Number: scalar.Int32(0),
				}},
			}},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("enum-dependency-without-import.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("bar"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("id"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}, {
					Name:     scalar.String("foo"),
					Number:   scalar.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
					TypeName: scalar.String(".foo.Foo"),
				}},
			}},
		},
		wantErr: "foo.Foo without import of foo.proto",
	}, {
		name: "message dependency on without import on file imported by a public import",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("foo.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Foo"),
			}},
		}, {
			Name:       scalar.String("baz.proto"),
			Syntax:     scalar.String("proto2"),
			Package:    scalar.String("foo"),
			Dependency: []string{"foo.proto"},
		}, {
			Name:             scalar.String("old-baz.proto"),
			Syntax:           scalar.String("proto2"),
			Package:          scalar.String("foo"),
			Dependency:       []string{"baz.proto"},
			PublicDependency: []int32{0},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:       scalar.String("message-dependency-without-import.proto"),
			Syntax:     scalar.String("proto2"),
			Package:    scalar.String("bar"),
			Dependency: []string{"old-baz.proto"},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("id"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}, {
					Name:     scalar.String("foo"),
					Number:   scalar.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: scalar.String(".foo.Foo"),
				}},
			}},
		},
		wantErr: "foo.Foo without import of foo.proto",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := new(protoregistry.Files)
			for _, dep := range tc.deps {
				f, err := NewFile(dep, r)
				if err != nil {
					t.Fatalf("Error creating dependency: %v", err)
				}
				if err := r.Register(f); err != nil {
					t.Fatalf("Error adding dependency: %v", err)
				}
			}
			if _, err := NewFile(tc.fd, r); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("NewFile: got err = %v; want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// Sanity checks for well-formed descriptors. Most behavior with well-formed descriptors is covered
// by other tests that rely on generated descriptors.
func TestNewFile_ValidationOK(t *testing.T) {
	testCases := []struct {
		name string
		deps []*descriptorpb.FileDescriptorProto
		fd   *descriptorpb.FileDescriptorProto
	}{{
		name: "self contained file",
		fd: &descriptorpb.FileDescriptorProto{
			Name:    scalar.String("self-contained.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("TopLevelEnum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   scalar.String("UNKNOWN"),
					Number: scalar.Int32(0),
				}},
			}},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("TopLevelMessage"),
				EnumType: []*descriptorpb.EnumDescriptorProto{{
					Name: scalar.String("NestedEnum"),
					Value: []*descriptorpb.EnumValueDescriptorProto{{
						Name:   scalar.String("UNKNOWN"),
						Number: scalar.Int32(0),
					}},
				}},
				NestedType: []*descriptorpb.DescriptorProto{{
					Name: scalar.String("NestedMessage"),
				}},
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("id"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}, {
					Name:     scalar.String("top_level_enum"),
					Number:   scalar.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
					TypeName: scalar.String(".foo.TopLevelEnum"),
				}, {
					Name:     scalar.String("nested_enum"),
					Number:   scalar.Int32(3),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
					TypeName: scalar.String(".foo.TopLevelMessage.NestedEnum"),
				}, {
					Name:     scalar.String("nested_message"),
					Number:   scalar.Int32(4),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: scalar.String(".foo.TopLevelMessage.NestedMessage"),
				}},
			}},
		},
	}, {
		name: "external types with explicit import",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("foo.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("FooMessage"),
			}},
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("BarEnum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   scalar.String("UNKNOWN"),
					Number: scalar.Int32(0),
				}},
			}},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:       scalar.String("external-types-with-explicit-import.proto"),
			Syntax:     scalar.String("proto2"),
			Package:    scalar.String("bar"),
			Dependency: []string{"foo.proto"},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("id"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}, {
					Name:     scalar.String("foo"),
					Number:   scalar.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: scalar.String(".foo.FooMessage"),
				}, {
					Name:     scalar.String("bar"),
					Number:   scalar.Int32(3),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
					TypeName: scalar.String(".foo.BarEnum"),
				}},
			}},
		},
	}, {
		name: "external types with transitive public imports",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("quux.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("QuuxMessage"),
			}},
		}, {
			Name:             scalar.String("foo.proto"),
			Syntax:           scalar.String("proto2"),
			Package:          scalar.String("foo"),
			Dependency:       []string{"quux.proto"},
			PublicDependency: []int32{0},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("FooMessage"),
			}},
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: scalar.String("BarEnum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   scalar.String("UNKNOWN"),
					Number: scalar.Int32(0),
				}},
			}},
		}, {
			Name:             scalar.String("old-name.proto"),
			Syntax:           scalar.String("proto2"),
			Package:          scalar.String("foo"),
			Dependency:       []string{"foo.proto"},
			PublicDependency: []int32{0},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:       scalar.String("external-types-with-public-import.proto"),
			Syntax:     scalar.String("proto2"),
			Package:    scalar.String("bar"),
			Dependency: []string{"old-name.proto"},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("id"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}, {
					Name:     scalar.String("foo"),
					Number:   scalar.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: scalar.String(".foo.FooMessage"),
				}, {
					Name:     scalar.String("bar"),
					Number:   scalar.Int32(3),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
					TypeName: scalar.String(".foo.BarEnum"),
				}, {
					Name:     scalar.String("quux"),
					Number:   scalar.Int32(4),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: scalar.String(".foo.QuuxMessage"),
				}},
			}},
		},
	}, {
		name: "external type from weak import",
		deps: []*descriptorpb.FileDescriptorProto{{
			Name:    scalar.String("weak.proto"),
			Syntax:  scalar.String("proto2"),
			Package: scalar.String("foo"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("WeakMessage"),
			}},
		}},
		fd: &descriptorpb.FileDescriptorProto{
			Name:           scalar.String("external-type-from-weak-import.proto"),
			Syntax:         scalar.String("proto2"),
			Package:        scalar.String("bar"),
			Dependency:     []string{"weak.proto"},
			WeakDependency: []int32{0},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: scalar.String("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   scalar.String("id"),
					Number: scalar.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}, {
					Name:     scalar.String("weak_message"),
					Number:   scalar.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: scalar.String(".foo.WeakMessage"),
					Options: &descriptorpb.FieldOptions{
						Weak: scalar.Bool(true),
					},
				}},
			}},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := new(protoregistry.Files)
			for _, dep := range tc.deps {
				f, err := NewFile(dep, r)
				if err != nil {
					t.Fatalf("error creating dependency: %v", err)
				}
				if err := r.Register(f); err != nil {
					t.Fatalf("error adding dependency: %v", err)
				}
			}
			if _, err := NewFile(tc.fd, r); err != nil {
				t.Errorf("unexpected NewFile error: %v", err)
			}
		})
	}
}
