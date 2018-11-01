// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"reflect"
	"testing"

	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test(t *testing.T) {
	m := &ptype.StandaloneMessage{
		Syntax:   pref.Proto3,
		FullName: "golang.org.example.FooMessage",
		Fields: []ptype.Field{{
			Name:        "foo_field",
			Number:      1337,
			Cardinality: pref.Repeated,
			Kind:        pref.BytesKind,
			JSONName:    "fooField",
			Default:     pref.ValueOf([]byte("hello, \xde\xad\xbe\xef\n")),
		}},
	}
	md, err := ptype.NewMessage(m)
	if err != nil {
		t.Fatalf("unexpected NewMessage error: %v", err)
	}

	// Marshal test.
	gotTag := Marshal(md.Fields().Get(0), "")
	wantTag := `bytes,1337,rep,name=foo_field,json=fooField,proto3,def=hello, \336\255\276\357\n`
	if gotTag != wantTag {
		t.Errorf("Marshal() = `%v`, want `%v`", gotTag, wantTag)
	}

	// Unmarshal test.
	gotField := Unmarshal(wantTag, reflect.TypeOf([]byte{}))
	wantField := m.Fields[0]
	opts := cmp.Options{
		cmp.Transformer("UnwrapValue", func(x pref.Value) interface{} {
			return x.Interface()
		}),
		cmpopts.IgnoreUnexported(ptype.Field{}),
		cmpopts.IgnoreFields(ptype.Field{}, "Options"),
	}
	if diff := cmp.Diff(wantField, gotField, opts); diff != "" {
		t.Errorf("Unmarshal() mismatch (-want +got):\n%v", diff)
	}
}
