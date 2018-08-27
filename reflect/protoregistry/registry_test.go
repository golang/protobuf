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

	pref "google.golang.org/proto/reflect/protoreflect"
	preg "google.golang.org/proto/reflect/protoregistry"
	ptype "google.golang.org/proto/reflect/prototype"
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
		testFindDesc struct {
			inName pref.FullName
			wantOk bool
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
		findDescs  []testFindDesc
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
			wantFiles: []file{
				{"", "foo.bar"},
				{"", "foo.bar"},
				{"foo/bar/test.proto", "foo.bar.baz"},
				{"foo/bar/test.proto", "my.test"},
				{"", "my.test.package"},
				{"foo/bar/baz/../test.proto", "my.test"},
			},
		}, {
			inPkg: ".",
		}, {
			inPkg: "foo",
			wantFiles: []file{
				{"", "foo.bar"},
				{"", "foo.bar"},
				{"foo/bar/test.proto", "foo.bar.baz"},
			},
		}, {
			inPkg: "foo.",
		}, {
			inPkg: "foo..",
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
			// Conflict over a single declaration.
			inFile: &ptype.File{
				Syntax:  pref.Proto2,
				Package: "fizz.buzz",
				Enums: []ptype.Enum{{
					Name:   "Enum1",
					Values: []ptype.EnumValue{{Name: "EnumValue1", Number: 0}},
				}, {
					Name:   "Enum2",
					Values: []ptype.EnumValue{{Name: "EnumValue2", Number: 0}},
				}, {
					Name:   "Enum3",
					Values: []ptype.EnumValue{{Name: "Enum", Number: 0}}, // conflict
				}},
			},
			wantErr: "name conflict over fizz.buzz.Enum",
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

		findDescs: []testFindDesc{
			{"", false},
			{"Enum", false},
			{"Message", true},
			{"Message.", false},
			{"Message.Message", true},
			{"Message.Message.Message", true},
			{"Message.Message.Message.Message", false},
			{"fizz.buzz", false},
			{"fizz.buzz.Enum", true},
			{"fizz.buzz.Enum1", true},
			{"fizz.buzz.Enum1.EnumValue", false},
			{"fizz.buzz.EnumValue", true},
			{"fizz.buzz.Message", true},
			{"fizz.buzz.Message.Field", true},
			{"fizz.buzz.Message.Oneof", true},
			{"fizz.buzz.Extension", true},
			{"fizz.buzz.Service", true},
			{"fizz.buzz.Service.Method", true},
			{"fizz.buzz.Method", false},
		},
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

			for _, tc := range tt.findDescs {
				got, err := files.FindDescriptorByName(tc.inName)
				if (got == nil) == (err == nil) {
					if tc.wantOk {
						t.Errorf("FindDescriptorByName(%v) = (%v, %v), want (non-nil, nil)", tc.inName, got, err)
					} else {
						t.Errorf("FindDescriptorByName(%v) = (%v, %v), want (nil, NotFound)", tc.inName, got, err)
					}
				}

				gotName := pref.FullName("<nil>")
				if got != nil {
					gotName = got.FullName()
				}
				wantName := pref.FullName("<nil>")
				if tc.wantOk {
					wantName = tc.inName
				}
				if gotName != wantName {
					t.Errorf("FindDescriptorByName(%v) = %v, want %v", tc.inName, gotName, wantName)
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
