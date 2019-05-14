// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"testing"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

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
