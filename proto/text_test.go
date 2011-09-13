// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2010 Google Inc.  All rights reserved.
// http://code.google.com/p/goprotobuf/
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package proto_test

import (
	"bytes"
	"strings"
	"testing"

	"goprotobuf.googlecode.com/hg/proto"

	pb "./testdata/_obj/test_proto"
)

func newTestMessage() *pb.MyMessage {
	msg := &pb.MyMessage{
		Count: proto.Int32(42),
		Name:  proto.String("Dave"),
		Quote: proto.String(`"I didn't want to go."`),
		Pet:   []string{"bunny", "kitty", "horsey"},
		Inner: &pb.InnerMessage{
			Host:      proto.String("footrest.syd"),
			Port:      proto.Int32(7001),
			Connected: proto.Bool(true),
		},
		Others: []*pb.OtherMessage{
			&pb.OtherMessage{
				Key:   proto.Int64(0xdeadbeef),
				Value: []byte{1, 65, 7, 12},
			},
			&pb.OtherMessage{
				Weight: proto.Float32(6.022),
				Inner: &pb.InnerMessage{
					Host: proto.String("lesha.mtv"),
					Port: proto.Int32(8002),
				},
			},
		},
		Bikeshed: pb.NewMyMessage_Color(pb.MyMessage_BLUE),
		Somegroup: &pb.MyMessage_SomeGroup{
			GroupField: proto.Int32(8),
		},
		// One normally wouldn't do this.
		// This is an undeclared tag 13, as a varint (wire type 0) with value 4.
		XXX_unrecognized: []byte{13<<3 | 0, 4},
	}
	ext := &pb.Ext{
		Data: proto.String("Big gobs for big rats"),
	}
	if err := proto.SetExtension(msg, pb.E_Ext_More, ext); err != nil {
		panic(err)
	}

	// Add an unknown extension. We marshal a pb.Ext, and fake the ID.
	b, err := proto.Marshal(&pb.Ext{Data: proto.String("3G skiing")})
	if err != nil {
		panic(err)
	}
	b = append(proto.EncodeVarint(104<<3|proto.WireBytes), b...)
	proto.SetRawExtension(msg, 104, b)

	// Extensions can be plain fields, too, so let's test that.
	b = append(proto.EncodeVarint(105<<3|proto.WireVarint), 19)
	proto.SetRawExtension(msg, 105, b)

	return msg
}

const text = `count: 42
name: "Dave"
quote: "\"I didn't want to go.\""
pet: "bunny"
pet: "kitty"
pet: "horsey"
inner: <
  host: "footrest.syd"
  port: 7001
  connected: true
>
others: <
  key: 3735928559
  value: "\001A\007\014"
>
others: <
  weight: 6.022
  inner: <
    host: "lesha.mtv"
    port: 8002
  >
>
bikeshed: BLUE
SomeGroup {
  group_field: 8
}
/* 2 unknown bytes */
tag13: 4
[test_proto.Ext.more]: <
  data: "Big gobs for big rats"
>
/* 13 unknown bytes */
tag104: "\t3G skiing"
/* 3 unknown bytes */
tag105: 19
`

func TestMarshalTextFull(t *testing.T) {
	buf := new(bytes.Buffer)
	proto.MarshalText(buf, newTestMessage())
	s := buf.String()
	if s != text {
		t.Errorf("Got:\n===\n%v===\nExpected:\n===\n%v===\n", s, text)
	}
}

func compact(src string) string {
	// s/[ \n]+/ /g; s/ $//;
	dst := make([]byte, len(src))
	space, comment := false, false
	j := 0
	for i := 0; i < len(src); i++ {
		if strings.HasPrefix(src[i:], "/*") {
			comment = true
			i++
			continue
		}
		if comment && strings.HasPrefix(src[i:], "*/") {
			comment = false
			i++
			continue
		}
		if comment {
			continue
		}
		c := src[i]
		if c == ' ' || c == '\n' {
			space = true
			continue
		}
		if j > 0 && (dst[j-1] == ':' || dst[j-1] == '<' || dst[j-1] == '{') {
			space = false
		}
		if c == '{' {
			space = false
		}
		if space {
			dst[j] = ' '
			j++
			space = false
		}
		dst[j] = c
		j++
	}
	if space {
		dst[j] = ' '
		j++
	}
	return string(dst[0:j])
}

var compactText = compact(text)

func TestCompactText(t *testing.T) {
	s := proto.CompactTextString(newTestMessage())
	if s != compactText {
		t.Errorf("Got:\n===\n%v===\nExpected:\n===\n%v\n===\n", s, compactText)
	}
}

func TestStringEscaping(t *testing.T) {
	testCases := []struct {
		in  *pb.Strings
		out string
	}{
		{
			// Test data from C++ test (TextFormatTest.StringEscape).
			// Single divergence: we don't escape apostrophes.
			&pb.Strings{StringField: proto.String("\"A string with ' characters \n and \r newlines and \t tabs and \001 slashes \\ and  multiple   spaces")},
			"string_field: \"\\\"A string with ' characters \\n and \\r newlines and \\t tabs and \\001 slashes \\\\ and  multiple   spaces\"\n",
		},
		{
			// Test data from the same C++ test.
			&pb.Strings{StringField: proto.String("\350\260\267\346\255\214")},
			"string_field: \"\\350\\260\\267\\346\\255\\214\"\n",
		},
	}

	for i, tc := range testCases {
		var buf bytes.Buffer
		proto.MarshalText(&buf, tc.in)
		if s := buf.String(); s != tc.out {
			t.Errorf("#%d: Got:\n%s\nExpected:\n%s\n", i, s, tc.out)
		}
	}
}


