// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoregistry_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	pimpl "google.golang.org/protobuf/internal/impl"
	ptype "google.golang.org/protobuf/internal/prototype"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"

	test2pb "google.golang.org/protobuf/internal/testprotos/test"
	testpb "google.golang.org/protobuf/reflect/protoregistry/testprotos"
)

func TestFiles(t *testing.T) {
	type (
		file struct {
			Path string
			Pkg  pref.FullName
		}
		testFile struct {
			inFile  *ptype.File
			wantErr string
		}
		testRangePkg struct {
			inPkg     pref.FullName
			wantFiles []file
		}
		testRangePath struct {
			inPath    string
			wantFiles []file
		}
	)

	tests := []struct {
		files      []testFile
		rangePkgs  []testRangePkg
		rangePaths []testRangePath
	}{{
		// Test that overlapping packages and files are permitted.
		files: []testFile{
			{inFile: &ptype.File{Syntax: pref.Proto2, Package: "foo.bar"}},
			{inFile: &ptype.File{Syntax: pref.Proto2, Path: "foo/bar/test.proto", Package: "my.test"}},
			{inFile: &ptype.File{Syntax: pref.Proto2, Path: "foo/bar/test.proto", Package: "foo.bar.baz"}},
			{inFile: &ptype.File{Syntax: pref.Proto2, Package: "my.test.package"}},
			{inFile: &ptype.File{Syntax: pref.Proto2, Package: "foo.bar"}},
			{inFile: &ptype.File{Syntax: pref.Proto2, Path: "foo/bar/baz/../test.proto", Package: "my.test"}},
		},

		rangePkgs: []testRangePkg{{
			inPkg: "nothing",
		}, {
			inPkg: "",
		}, {
			inPkg: ".",
		}, {
			inPkg: "foo",
		}, {
			inPkg: "foo.",
		}, {
			inPkg: "foo..",
		}, {
			inPkg: "foo.bar",
			wantFiles: []file{
				{"", "foo.bar"},
				{"", "foo.bar"},
			},
		}, {
			inPkg: "foo.bar.baz",
			wantFiles: []file{
				{"foo/bar/test.proto", "foo.bar.baz"},
			},
		}, {
			inPkg: "fo",
		}},

		rangePaths: []testRangePath{{
			inPath: "nothing",
		}, {
			inPath: "",
			wantFiles: []file{
				{"", "foo.bar"},
				{"", "foo.bar"},
				{"", "my.test.package"},
			},
		}, {
			inPath: "foo/bar/test.proto",
			wantFiles: []file{
				{"foo/bar/test.proto", "foo.bar.baz"},
				{"foo/bar/test.proto", "my.test"},
			},
		}},
	}, {
		// Test when new enum conflicts with existing package.
		files: []testFile{{
			inFile: &ptype.File{Syntax: pref.Proto2, Path: "test1a.proto", Package: "foo.bar.baz"},
		}, {
			inFile:  &ptype.File{Syntax: pref.Proto2, Path: "test1b.proto", Enums: []ptype.Enum{{Name: "foo"}}},
			wantErr: `file "test1b.proto" has a name conflict over foo`,
		}},
	}, {
		// Test when new package conflicts with existing enum.
		files: []testFile{{
			inFile: &ptype.File{Syntax: pref.Proto2, Path: "test2a.proto", Enums: []ptype.Enum{{Name: "foo"}}},
		}, {
			inFile:  &ptype.File{Syntax: pref.Proto2, Path: "test2b.proto", Package: "foo.bar.baz"},
			wantErr: `file "test2b.proto" has a name conflict over foo`,
		}},
	}, {
		// Test when new enum conflicts with existing enum in same package.
		files: []testFile{{
			inFile: &ptype.File{Syntax: pref.Proto2, Path: "test3a.proto", Package: "foo", Enums: []ptype.Enum{{Name: "BAR"}}},
		}, {
			inFile:  &ptype.File{Syntax: pref.Proto2, Path: "test3b.proto", Package: "foo", Enums: []ptype.Enum{{Name: "BAR"}}},
			wantErr: `file "test3b.proto" has a name conflict over foo.BAR`,
		}},
	}, {
		files: []testFile{{
			inFile: &ptype.File{
				Syntax:  pref.Proto2,
				Package: "fizz.buzz",
				Messages: []ptype.Message{{
					Name: "Message",
					Fields: []ptype.Field{{
						Name:        "Field",
						Number:      1,
						Cardinality: pref.Optional,
						Kind:        pref.StringKind,
						OneofName:   "Oneof",
					}},
					Oneofs:          []ptype.Oneof{{Name: "Oneof"}},
					ExtensionRanges: [][2]pref.FieldNumber{{1000, 2000}},
				}},
				Enums: []ptype.Enum{{
					Name:   "Enum",
					Values: []ptype.EnumValue{{Name: "EnumValue", Number: 0}},
				}},
				Extensions: []ptype.Extension{{
					Name:         "Extension",
					Number:       1000,
					Cardinality:  pref.Optional,
					Kind:         pref.StringKind,
					ExtendedType: ptype.PlaceholderMessage("fizz.buzz.Message"),
				}},
				Services: []ptype.Service{{
					Name: "Service",
					Methods: []ptype.Method{{
						Name:              "Method",
						InputType:         ptype.PlaceholderMessage("fizz.buzz.Message"),
						OutputType:        ptype.PlaceholderMessage("fizz.buzz.Message"),
						IsStreamingClient: true,
						IsStreamingServer: true,
					}},
				}},
			},
		}, {
			inFile: &ptype.File{
				Syntax:  pref.Proto2,
				Package: "fizz.buzz.gazz",
				Enums: []ptype.Enum{{
					Name:   "Enum",
					Values: []ptype.EnumValue{{Name: "EnumValue", Number: 0}},
				}},
			},
		}, {
			// Previously failed registration should not pollute the namespace.
			inFile: &ptype.File{
				Syntax:  pref.Proto2,
				Package: "fizz.buzz",
				Enums: []ptype.Enum{{
					Name:   "Enum1",
					Values: []ptype.EnumValue{{Name: "EnumValue1", Number: 0}},
				}, {
					Name:   "Enum2",
					Values: []ptype.EnumValue{{Name: "EnumValue2", Number: 0}},
				}},
			},
		}, {
			// Make sure we can register without package name.
			inFile: &ptype.File{
				Syntax: pref.Proto2,
				Messages: []ptype.Message{{
					Name: "Message",
					Messages: []ptype.Message{{
						Name: "Message",
						Messages: []ptype.Message{{
							Name: "Message",
						}},
					}},
				}},
			},
		}},
	}}

	sortFiles := cmpopts.SortSlices(func(x, y file) bool {
		return x.Path < y.Path || (x.Path == y.Path && x.Pkg < y.Pkg)
	})
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			var files preg.Files
			for i, tc := range tt.files {
				fd, err := ptype.NewFile(tc.inFile)
				if err != nil {
					t.Fatalf("file %d, prototype.NewFile() error: %v", i, err)
				}
				gotErr := files.Register(fd)
				if (gotErr == nil && tc.wantErr != "") || !strings.Contains(fmt.Sprint(gotErr), tc.wantErr) {
					t.Errorf("file %d, Register() = %v, want %v", i, gotErr, tc.wantErr)
				}
			}

			for _, tc := range tt.rangePkgs {
				var gotFiles []file
				files.RangeFilesByPackage(tc.inPkg, func(fd pref.FileDescriptor) bool {
					gotFiles = append(gotFiles, file{fd.Path(), fd.Package()})
					return true
				})
				if diff := cmp.Diff(tc.wantFiles, gotFiles, sortFiles); diff != "" {
					t.Errorf("RangeFilesByPackage(%v) mismatch (-want +got):\n%v", tc.inPkg, diff)
				}
			}

			for _, tc := range tt.rangePaths {
				var gotFiles []file
				files.RangeFilesByPath(tc.inPath, func(fd pref.FileDescriptor) bool {
					gotFiles = append(gotFiles, file{fd.Path(), fd.Package()})
					return true
				})
				if diff := cmp.Diff(tc.wantFiles, gotFiles, sortFiles); diff != "" {
					t.Errorf("RangeFilesByPath(%v) mismatch (-want +got):\n%v", tc.inPath, diff)
				}
			}
		})
	}
}

func TestFilesLookup(t *testing.T) {
	files := []pref.FileDescriptor{
		test2pb.File_test_test_proto,
		test2pb.File_test_test_import_proto,
	}
	r := preg.NewFiles(files...)
	checkEnums := func(enums pref.EnumDescriptors) {
		for i := 0; i < enums.Len(); i++ {
			want := enums.Get(i)
			if got, err := r.FindEnumByName(want.FullName()); err != nil {
				t.Errorf("FindEnumByName(%q): unexpected error: %v", want.FullName(), err)
			} else if got != want {
				t.Errorf("FindEnumByName(%q): found descriptor %v (%p), %p", want.FullName(), got.FullName(), got, want)
			}
		}
	}
	checkExtensions := func(exts pref.ExtensionDescriptors) {
		for i := 0; i < exts.Len(); i++ {
			want := exts.Get(i)
			if got, err := r.FindExtensionByName(want.FullName()); err != nil {
				t.Errorf("FindExtensionByName(%q): unexpected error: %v", want.FullName(), err)
			} else if got != want {
				t.Errorf("FindExtensionByName(%q): found descriptor %v (%p), %p", want.FullName(), got.FullName(), got, want)
			}
		}
	}
	var checkMessages func(pref.MessageDescriptors)
	checkMessages = func(messages pref.MessageDescriptors) {
		for i := 0; i < messages.Len(); i++ {
			want := messages.Get(i)
			if got, err := r.FindMessageByName(want.FullName()); err != nil {
				t.Errorf("FindMessageByName(%q): unexpected error: %v", want.FullName(), err)
			} else if got != want {
				t.Errorf("FindMessageByName(%q): found descriptor %v (%p), %p", want.FullName(), got.FullName(), got, want)
			}
			checkEnums(want.Enums())
			checkExtensions(want.Extensions())
			checkMessages(want.Messages())
		}
	}
	checkServices := func(services pref.ServiceDescriptors) {
		for i := 0; i < services.Len(); i++ {
			want := services.Get(i)
			if got, err := r.FindServiceByName(want.FullName()); err != nil {
				t.Errorf("FindServiceByName(%q): unexpected error: %v", want.FullName(), err)
			} else if got != want {
				t.Errorf("FindServiceByName(%q): found descriptor %v (%p), %p", want.FullName(), got.FullName(), got, want)
			}
		}
	}
	for _, fd := range files {
		checkEnums(fd.Enums())
		checkExtensions(fd.Extensions())
		checkMessages(fd.Messages())
		checkServices(fd.Services())
	}
}

func TestTypes(t *testing.T) {
	// Suffix 1 in registry, 2 in parent, 3 in resolver.
	mt1 := pimpl.Export{}.MessageTypeOf(&testpb.Message1{})
	mt2 := pimpl.Export{}.MessageTypeOf(&testpb.Message2{})
	mt3 := pimpl.Export{}.MessageTypeOf(&testpb.Message3{})
	et1 := pimpl.Export{}.EnumTypeOf(testpb.Enum1_ONE)
	et2 := pimpl.Export{}.EnumTypeOf(testpb.Enum2_UNO)
	et3 := pimpl.Export{}.EnumTypeOf(testpb.Enum3_YI)
	// Suffix indicates field number.
	xt11 := testpb.E_StringField.Type
	xt12 := testpb.E_EnumField.Type
	xt13 := testpb.E_MessageField.Type
	xt21 := testpb.E_Message4_MessageField.Type
	xt22 := testpb.E_Message4_EnumField.Type
	xt23 := testpb.E_Message4_StringField.Type
	parent := &preg.Types{}
	if err := parent.Register(mt2, et2, xt12, xt22); err != nil {
		t.Fatalf("parent.Register() returns unexpected error: %v", err)
	}
	registry := &preg.Types{
		Parent: parent,
		Resolver: func(url string) (preg.Type, error) {
			switch {
			case strings.HasSuffix(url, "testprotos.Message3"):
				return mt3, nil
			case strings.HasSuffix(url, "testprotos.Enum3"):
				return et3, nil
			case strings.HasSuffix(url, "testprotos.message_field"):
				return xt13, nil
			case strings.HasSuffix(url, "testprotos.Message4.string_field"):
				return xt23, nil
			}
			return nil, preg.NotFound
		},
	}
	if err := registry.Register(mt1, et1, xt11, xt21); err != nil {
		t.Fatalf("registry.Register() returns unexpected error: %v", err)
	}

	t.Run("FindMessageByName", func(t *testing.T) {
		tests := []struct {
			name         string
			messageType  pref.MessageType
			wantErr      bool
			wantNotFound bool
		}{{
			name:        "testprotos.Message1",
			messageType: mt1,
		}, {
			name:        "testprotos.Message2",
			messageType: mt2,
		}, {
			name:        "testprotos.Message3",
			messageType: mt3,
		}, {
			name:         "testprotos.NoSuchMessage",
			wantErr:      true,
			wantNotFound: true,
		}, {
			name:    "testprotos.Enum1",
			wantErr: true,
		}, {
			name:    "testprotos.Enum2",
			wantErr: true,
		}, {
			name:    "testprotos.Enum3",
			wantErr: true,
		}}
		for _, tc := range tests {
			got, err := registry.FindMessageByName(pref.FullName(tc.name))
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("FindMessageByName(%v) = (_, %v), want error? %t", tc.name, err, tc.wantErr)
				continue
			}
			if tc.wantNotFound && err != preg.NotFound {
				t.Errorf("FindMessageByName(%v) got error: %v, want NotFound error", tc.name, err)
				continue
			}
			if got != tc.messageType {
				t.Errorf("FindMessageByName(%v) got wrong value: %v", tc.name, got)
			}
		}
	})

	t.Run("FindMessageByURL", func(t *testing.T) {
		tests := []struct {
			name         string
			messageType  pref.MessageType
			wantErr      bool
			wantNotFound bool
		}{{
			name:        "testprotos.Message1",
			messageType: mt1,
		}, {
			name:        "foo.com/testprotos.Message2",
			messageType: mt2,
		}, {
			name:        "/testprotos.Message3",
			messageType: mt3,
		}, {
			name:         "type.googleapis.com/testprotos.Nada",
			wantErr:      true,
			wantNotFound: true,
		}, {
			name:    "testprotos.Enum1",
			wantErr: true,
		}}
		for _, tc := range tests {
			got, err := registry.FindMessageByURL(tc.name)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("FindMessageByURL(%v) = (_, %v), want error? %t", tc.name, err, tc.wantErr)
				continue
			}
			if tc.wantNotFound && err != preg.NotFound {
				t.Errorf("FindMessageByURL(%v) got error: %v, want NotFound error", tc.name, err)
				continue
			}
			if got != tc.messageType {
				t.Errorf("FindMessageByURL(%v) got wrong value: %v", tc.name, got)
			}
		}
	})

	t.Run("FindEnumByName", func(t *testing.T) {
		tests := []struct {
			name         string
			enumType     pref.EnumType
			wantErr      bool
			wantNotFound bool
		}{{
			name:     "testprotos.Enum1",
			enumType: et1,
		}, {
			name:     "testprotos.Enum2",
			enumType: et2,
		}, {
			name:     "testprotos.Enum3",
			enumType: et3,
		}, {
			name:         "testprotos.None",
			wantErr:      true,
			wantNotFound: true,
		}, {
			name:    "testprotos.Message1",
			wantErr: true,
		}}
		for _, tc := range tests {
			got, err := registry.FindEnumByName(pref.FullName(tc.name))
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("FindEnumByName(%v) = (_, %v), want error? %t", tc.name, err, tc.wantErr)
				continue
			}
			if tc.wantNotFound && err != preg.NotFound {
				t.Errorf("FindEnumByName(%v) got error: %v, want NotFound error", tc.name, err)
				continue
			}
			if got != tc.enumType {
				t.Errorf("FindEnumByName(%v) got wrong value: %v", tc.name, got)
			}
		}
	})

	t.Run("FindExtensionByName", func(t *testing.T) {
		tests := []struct {
			name          string
			extensionType pref.ExtensionType
			wantErr       bool
			wantNotFound  bool
		}{{
			name:          "testprotos.string_field",
			extensionType: xt11,
		}, {
			name:          "testprotos.enum_field",
			extensionType: xt12,
		}, {
			name:          "testprotos.message_field",
			extensionType: xt13,
		}, {
			name:          "testprotos.Message4.message_field",
			extensionType: xt21,
		}, {
			name:          "testprotos.Message4.enum_field",
			extensionType: xt22,
		}, {
			name:          "testprotos.Message4.string_field",
			extensionType: xt23,
		}, {
			name:         "testprotos.None",
			wantErr:      true,
			wantNotFound: true,
		}, {
			name:    "testprotos.Message1",
			wantErr: true,
		}}
		for _, tc := range tests {
			got, err := registry.FindExtensionByName(pref.FullName(tc.name))
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("FindExtensionByName(%v) = (_, %v), want error? %t", tc.name, err, tc.wantErr)
				continue
			}
			if tc.wantNotFound && err != preg.NotFound {
				t.Errorf("FindExtensionByName(%v) got error: %v, want NotFound error", tc.name, err)
				continue
			}
			if got != tc.extensionType {
				t.Errorf("FindExtensionByName(%v) got wrong value: %v", tc.name, got)
			}
		}
	})

	t.Run("FindExtensionByNumber", func(t *testing.T) {
		tests := []struct {
			parent        string
			number        int32
			extensionType pref.ExtensionType
			wantErr       bool
			wantNotFound  bool
		}{{
			parent:        "testprotos.Message1",
			number:        11,
			extensionType: xt11,
		}, {
			parent:        "testprotos.Message1",
			number:        12,
			extensionType: xt12,
		}, {
			// FindExtensionByNumber does not use Resolver.
			parent:       "testprotos.Message1",
			number:       13,
			wantErr:      true,
			wantNotFound: true,
		}, {
			parent:        "testprotos.Message1",
			number:        21,
			extensionType: xt21,
		}, {
			parent:        "testprotos.Message1",
			number:        22,
			extensionType: xt22,
		}, {
			// FindExtensionByNumber does not use Resolver.
			parent:       "testprotos.Message1",
			number:       23,
			wantErr:      true,
			wantNotFound: true,
		}, {
			parent:       "testprotos.NoSuchMessage",
			number:       11,
			wantErr:      true,
			wantNotFound: true,
		}, {
			parent:       "testprotos.Message1",
			number:       30,
			wantErr:      true,
			wantNotFound: true,
		}, {
			parent:       "testprotos.Message1",
			number:       99,
			wantErr:      true,
			wantNotFound: true,
		}}
		for _, tc := range tests {
			got, err := registry.FindExtensionByNumber(pref.FullName(tc.parent), pref.FieldNumber(tc.number))
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("FindExtensionByNumber(%v, %d) = (_, %v), want error? %t", tc.parent, tc.number, err, tc.wantErr)
				continue
			}
			if tc.wantNotFound && err != preg.NotFound {
				t.Errorf("FindExtensionByNumber(%v, %d) got error %v, want NotFound error", tc.parent, tc.number, err)
				continue
			}
			if got != tc.extensionType {
				t.Errorf("FindExtensionByNumber(%v, %d) got wrong value: %v", tc.parent, tc.number, got)
			}
		}
	})

	fullName := func(t preg.Type) pref.FullName {
		switch t := t.(type) {
		case pref.EnumType:
			return t.Descriptor().FullName()
		case pref.MessageType:
			return t.Descriptor().FullName()
		case pref.ExtensionType:
			return t.Descriptor().FullName()
		default:
			panic("invalid type")
		}
	}
	sortTypes := cmpopts.SortSlices(func(x, y preg.Type) bool {
		return fullName(x) < fullName(y)
	})
	compare := cmp.Comparer(func(x, y preg.Type) bool {
		return x == y
	})

	t.Run("RangeMessages", func(t *testing.T) {
		// RangeMessages do not include messages from Resolver.
		want := []preg.Type{mt1, mt2}
		var got []preg.Type
		registry.RangeMessages(func(mt pref.MessageType) bool {
			got = append(got, mt)
			return true
		})

		diff := cmp.Diff(want, got, sortTypes, compare)
		if diff != "" {
			t.Errorf("RangeMessages() mismatch (-want +got):\n%v", diff)
		}
	})

	t.Run("RangeEnums", func(t *testing.T) {
		// RangeEnums do not include enums from Resolver.
		want := []preg.Type{et1, et2}
		var got []preg.Type
		registry.RangeEnums(func(et pref.EnumType) bool {
			got = append(got, et)
			return true
		})

		diff := cmp.Diff(want, got, sortTypes, compare)
		if diff != "" {
			t.Errorf("RangeEnums() mismatch (-want +got):\n%v", diff)
		}
	})

	t.Run("RangeExtensions", func(t *testing.T) {
		// RangeExtensions do not include messages from Resolver.
		want := []preg.Type{xt11, xt12, xt21, xt22}
		var got []preg.Type
		registry.RangeExtensions(func(xt pref.ExtensionType) bool {
			got = append(got, xt)
			return true
		})

		diff := cmp.Diff(want, got, sortTypes, compare)
		if diff != "" {
			t.Errorf("RangeExtensions() mismatch (-want +got):\n%v", diff)
		}
	})

	t.Run("RangeExtensionsByMessage", func(t *testing.T) {
		// RangeExtensions do not include messages from Resolver.
		want := []preg.Type{xt11, xt12, xt21, xt22}
		var got []preg.Type
		registry.RangeExtensionsByMessage(pref.FullName("testprotos.Message1"), func(xt pref.ExtensionType) bool {
			got = append(got, xt)
			return true
		})

		diff := cmp.Diff(want, got, sortTypes, compare)
		if diff != "" {
			t.Errorf("RangeExtensionsByMessage() mismatch (-want +got):\n%v", diff)
		}
	})
}
