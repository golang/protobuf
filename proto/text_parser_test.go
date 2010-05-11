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
	. "goprotobuf.googlecode.com/hg/proto"
	. "./testdata/_obj/test_proto"
	"reflect"
	"testing"
)

type UnmarshalTextTest struct {
	in    string
	error string // if "", no error expected
	out   *MyMessage
}

var unMarshalTextTests = []UnmarshalTextTest{
	// Basic
	UnmarshalTextTest{
		in: " count:42\n  name:\"Dave\" ",
		out: &MyMessage{
			Count: Int32(42),
			Name:  String("Dave"),
		},
	},

	// Empty quoted string
	UnmarshalTextTest{
		in: `count:42 name:""`,
		out: &MyMessage{
			Count: Int32(42),
			Name:  String(""),
		},
	},

	// Quoted string concatenation
	UnmarshalTextTest{
		in: `count:42 name: "My name is "` + "\n" + `"elsewhere"`,
		out: &MyMessage{
			Count: Int32(42),
			Name:  String("My name is elsewhere"),
		},
	},

	// Bad quoted string
	UnmarshalTextTest{
		in:    `inner: < host: "\0" >` + "\n",
		error: `line 1.15: invalid quoted string "\0"`,
	},

	// Number too large for int64
	UnmarshalTextTest{
		in:    "count: 123456789012345678901",
		error: "line 1.7: invalid int32: 123456789012345678901",
	},

	// Number too large for int32
	UnmarshalTextTest{
		in:    "count: 1234567890123",
		error: "line 1.7: invalid int32: 1234567890123",
	},

	// Number too large for float32
	UnmarshalTextTest{
		in:    "others:< weight: 12345678901234567890123456789012345678901234567890 >",
		error: "line 1.17: invalid float32: 12345678901234567890123456789012345678901234567890",
	},

	// Number posing as a quoted string
	UnmarshalTextTest{
		in:    `inner: < host: 12 >` + "\n",
		error: `line 1.15: invalid string: 12`,
	},

	// Quoted string posing as int32
	UnmarshalTextTest{
		in:    `count: "12"`,
		error: `line 1.7: invalid int32: "12"`,
	},

	// Quoted string posing a float32
	UnmarshalTextTest{
		in:    `others:< weight: "17.4" >`,
		error: `line 1.17: invalid float32: "17.4"`,
	},

	// Enum
	UnmarshalTextTest{
		in: `count:42 bikeshed: BLUE`,
		out: &MyMessage{
			Count:    Int32(42),
			Bikeshed: NewMyMessage_Color(MyMessage_BLUE),
		},
	},

	// Repeated field
	UnmarshalTextTest{
		in: `count:42 pet: "horsey" pet:"bunny"`,
		out: &MyMessage{
			Count: Int32(42),
			Pet:   []string{"horsey", "bunny"},
		},
	},

	// Missing colon for inner message
	UnmarshalTextTest{
		in: `count:42 inner < host: "cauchy.syd" >`,
		out: &MyMessage{
			Count: Int32(42),
			Inner: &InnerMessage{
				Host: String("cauchy.syd"),
			},
		},
	},

	// Missing colon for string field
	UnmarshalTextTest{
		in:    `name "Dave"`,
		error: `line 1.5: expected ':', found "\"Dave\""`,
	},

	// Missing colon for int32 field
	UnmarshalTextTest{
		in:    `count 42`,
		error: `line 1.6: expected ':', found "42"`,
	},

	// Missing required field
	UnmarshalTextTest{
		in:    ``,
		error: `line 1.0: message test_proto.MyMessage missing required field "count"`,
	},

	// Repeated non-repeated field
	UnmarshalTextTest{
		in:    `name: "Rob" name: "Russ"`,
		error: `line 1.12: non-repeated field "name" was repeated`,
	},

	// Big all-in-one
	UnmarshalTextTest{
		in: "count:42  # Meaning\n" +
			`name:"Dave" ` +
			`quote:"\"I didn't want to go.\"" ` +
			`pet:"bunny" ` +
			`pet:"kitty" ` +
			`pet:"horsey" ` +
			`inner:<` +
			`  host:"footrest.syd" ` +
			`  port:7001 ` +
			`  connected:true ` +
			`> ` +
			`others:<` +
			`  key:3735928559 ` +
			`  value:"\x01A\a\f" ` +
			`> ` +
			`others:<` +
			"  weight:58.9  # Atomic weight of Co\n" +
			`  inner:<` +
			`    host:"lesha.mtv" ` +
			`    port:8002 ` +
			`  >` +
			`>`,
		out: &MyMessage{
			Count: Int32(42),
			Name:  String("Dave"),
			Quote: String(`"I didn't want to go."`),
			Pet:   []string{"bunny", "kitty", "horsey"},
			Inner: &InnerMessage{
				Host:      String("footrest.syd"),
				Port:      Int32(7001),
				Connected: Bool(true),
			},
			Others: []*OtherMessage{
				&OtherMessage{
					Key:   Int64(3735928559),
					Value: []byte{0x1, 'A', '\a', '\f'},
				},
				&OtherMessage{
					Weight: Float32(58.9),
					Inner: &InnerMessage{
						Host: String("lesha.mtv"),
						Port: Int32(8002),
					},
				},
			},
		},
	},
}

func TestUnmarshalText(t *testing.T) {
	for i, test := range unMarshalTextTests {
		pb := new(MyMessage)
		err := UnmarshalText(test.in, pb)
		if test.error == "" {
			// We don't expect failure.
			if err != nil {
				t.Errorf("Test %d: Unexpected error: %v", i, err)
			} else if !reflect.DeepEqual(pb, test.out) {
				t.Errorf("Test %d: Incorrect populated \n"+
					"Have: %v\nWant: %v",
					i, CompactTextString(pb), CompactTextString(test.out))
			}
		} else {
			// We do expect failure.
			if err == nil {
				t.Errorf("Test %d: Didn't get expected error: %v", i, test.error)
			} else if err.String() != test.error {
				t.Errorf("Test %d: Incorrect error.\nHave: %v\nWant: %v",
					i, err.String(), test.error)
			}
		}
	}
}
