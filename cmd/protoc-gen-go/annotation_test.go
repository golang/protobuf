// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/v2/internal/scalar"

	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

func TestAnnotations(t *testing.T) {
	sourceFile, err := ioutil.ReadFile("testdata/annotations/annotations.pb.go")
	if err != nil {
		t.Fatal(err)
	}
	metaFile, err := ioutil.ReadFile("testdata/annotations/annotations.pb.go.meta")
	if err != nil {
		t.Fatal(err)
	}
	gotInfo := &descriptorpb.GeneratedCodeInfo{}
	if err := proto.UnmarshalText(string(metaFile), gotInfo); err != nil {
		t.Fatalf("can't parse meta file: %v", err)
	}

	wantInfo := &descriptorpb.GeneratedCodeInfo{}
	for _, want := range []struct {
		prefix, text, suffix string
		path                 []int32
	}{{
		"type ", "AnnotationsTestEnum", " int32",
		[]int32{5 /* enum_type */, 0},
	}, {
		"\t", "AnnotationsTestEnum_ANNOTATIONS_TEST_ENUM_VALUE", " AnnotationsTestEnum = 0",
		[]int32{5 /* enum_type */, 0, 2 /* value */, 0},
	}, {
		"type ", "AnnotationsTestMessage", " struct {",
		[]int32{4 /* message_type */, 0},
	}, {
		"\t", "AnnotationsTestField", " ",
		[]int32{4 /* message_type */, 0, 2 /* field */, 0},
	}, {
		"func (m *AnnotationsTestMessage) ", "GetAnnotationsTestField", "() string {",
		[]int32{4 /* message_type */, 0, 2 /* field */, 0},
	}} {
		s := want.prefix + want.text + want.suffix
		pos := bytes.Index(sourceFile, []byte(s))
		if pos < 0 {
			t.Errorf("source file does not contain: %v", s)
			continue
		}
		begin := pos + len(want.prefix)
		end := begin + len(want.text)
		wantInfo.Annotation = append(wantInfo.Annotation, &descriptorpb.GeneratedCodeInfo_Annotation{
			Path:       want.path,
			Begin:      scalar.Int32(int32(begin)),
			End:        scalar.Int32(int32(end)),
			SourceFile: scalar.String("annotations/annotations.proto"),
		})
	}
	if !proto.Equal(gotInfo, wantInfo) {
		t.Errorf("unexpected annotations for annotations.proto; got:\n%v\nwant:\n%v",
			proto.MarshalTextString(gotInfo), proto.MarshalTextString(wantInfo))
	}
}
