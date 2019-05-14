// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	ptype "google.golang.org/protobuf/internal/prototype"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

func TestFields(t *testing.T) {
	type fieldsKind struct {
		kind   pref.Kind
		fields string
	}
	tests := []struct {
		inFields []fieldsKind
		wantMsg  ptype.Message
		wantErr  string
	}{{
		inFields: []fieldsKind{{pref.MessageKind, ""}},
		wantMsg:  ptype.Message{Name: "M"},
	}, {
		inFields: []fieldsKind{{pref.MessageKind, "987654321"}},
		wantErr:  "invalid field: 987654321",
	}, {
		inFields: []fieldsKind{{pref.MessageKind, "-1"}},
		wantErr:  "invalid field: -1",
	}, {
		inFields: []fieldsKind{{pref.MessageKind, "k"}},
		wantErr:  "invalid field: k",
	}, {
		inFields: []fieldsKind{{pref.MessageKind, "1.2"}, {pref.Int32Kind, "1"}},
		wantErr:  "field 1 of int32 type cannot have sub-fields",
	}, {
		inFields: []fieldsKind{{pref.Int32Kind, "1"}, {pref.MessageKind, "1.2"}},
		wantErr:  "field 1 of int32 type cannot have sub-fields",
	}, {
		inFields: []fieldsKind{{pref.Int32Kind, "30"}, {pref.Int32Kind, "30"}},
		wantErr:  "field 30 already set as int32 type",
	}, {
		inFields: []fieldsKind{
			{pref.Int32Kind, "10.20.31"},
			{pref.MessageKind, "  10.20.30, 10.21   "},
			{pref.GroupKind, "10"},
		},
		wantMsg: ptype.Message{
			Name: "M",
			Fields: []ptype.Field{
				{Name: "f10", Number: 10, Cardinality: pref.Optional, Kind: pref.GroupKind, MessageType: ptype.PlaceholderMessage("M.M10")},
			},
			Messages: []ptype.Message{{
				Name: "M10",
				Fields: []ptype.Field{
					{Name: "f20", Number: 20, Cardinality: pref.Optional, Kind: pref.MessageKind, MessageType: ptype.PlaceholderMessage("M.M10.M20")},
					{Name: "f21", Number: 21, Cardinality: pref.Optional, Kind: pref.MessageKind, MessageType: ptype.PlaceholderMessage("M.M10.M21")},
				},
				Messages: []ptype.Message{{
					Name: "M20",
					Fields: []ptype.Field{
						{Name: "f30", Number: 30, Cardinality: pref.Optional, Kind: pref.MessageKind, MessageType: ptype.PlaceholderMessage("M.M10.M20.M30")},
						{Name: "f31", Number: 31, Cardinality: pref.Repeated, Kind: pref.Int32Kind},
					},
					Messages: []ptype.Message{{
						Name: "M30",
					}},
				}, {
					Name: "M21",
				}},
			}},
		},
	}}

	opts := cmp.Options{
		cmp.Comparer(func(x, y pref.Descriptor) bool {
			if x == nil || y == nil {
				return x == nil && y == nil
			}
			return x.FullName() == y.FullName()
		}),
		cmpopts.IgnoreFields(ptype.Field{}, "Default"),
		cmpopts.IgnoreFields(ptype.Field{}, "Options"),
		cmpopts.IgnoreUnexported(ptype.Message{}, ptype.Field{}),
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			var fields fields
			for i, tc := range tt.inFields {
				gotErr := fields.Set(tc.fields, tc.kind)
				if gotErr != nil {
					if tt.wantErr == "" || !strings.Contains(fmt.Sprint(gotErr), tt.wantErr) {
						t.Fatalf("fields %d, Set(%q, %v) = %v, want %v", i, tc.fields, tc.kind, gotErr, tt.wantErr)
					}
					return
				}
			}
			if tt.wantErr != "" {
				t.Errorf("all Set calls succeeded, want %v error", tt.wantErr)
			}
			gotMsg := fields.messageDescriptor("M")
			if diff := cmp.Diff(tt.wantMsg, gotMsg, opts); diff != "" {
				t.Errorf("messageDescriptor() mismatch (-want +got):\n%v", diff)
			}
			if _, err := fields.Descriptor(); err != nil {
				t.Errorf("Descriptor() = %v, want nil error", err)
			}
		})
	}
}
