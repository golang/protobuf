// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test(t *testing.T) {
	const space = " \n\r\t"
	var V = ValueOf
	type Arr = []Value
	type Obj = [][2]Value

	tests := []struct {
		in            string
		wantVal       Value
		wantOut       string
		wantOutIndent string
		wantErr       string
	}{{
		in:      ``,
		wantErr: `unexpected EOF`,
	}, {
		in:      space,
		wantErr: `unexpected EOF`,
	}, {
		in:            space + `null` + space,
		wantVal:       V(nil),
		wantOut:       `null`,
		wantOutIndent: `null`,
	}, {
		in:            space + `true` + space,
		wantVal:       V(true),
		wantOut:       `true`,
		wantOutIndent: `true`,
	}, {
		in:            space + `false` + space,
		wantVal:       V(false),
		wantOut:       `false`,
		wantOutIndent: `false`,
	}, {
		in:            space + `0` + space,
		wantVal:       V(0.0),
		wantOut:       `0`,
		wantOutIndent: `0`,
	}, {
		in:            space + `"hello"` + space,
		wantVal:       V("hello"),
		wantOut:       `"hello"`,
		wantOutIndent: `"hello"`,
	}, {
		in:            space + `[]` + space,
		wantVal:       V(Arr{}),
		wantOut:       `[]`,
		wantOutIndent: `[]`,
	}, {
		in:            space + `{}` + space,
		wantVal:       V(Obj{}),
		wantOut:       `{}`,
		wantOutIndent: `{}`,
	}, {
		in:      `null#invalid`,
		wantErr: `8 bytes of unconsumed input`,
	}, {
		in:      `0#invalid`,
		wantErr: `8 bytes of unconsumed input`,
	}, {
		in:      `"hello"#invalid`,
		wantErr: `8 bytes of unconsumed input`,
	}, {
		in:      `[]#invalid`,
		wantErr: `8 bytes of unconsumed input`,
	}, {
		in:      `{}#invalid`,
		wantErr: `8 bytes of unconsumed input`,
	}, {
		in:      `[truee,true]`,
		wantErr: `invalid "truee" as literal`,
	}, {
		in:      `[falsee,false]`,
		wantErr: `invalid "falsee" as literal`,
	}, {
		in:      `[`,
		wantErr: `unexpected EOF`,
	}, {
		in:            `[{}]`,
		wantVal:       V(Arr{V(Obj{})}),
		wantOut:       "[{}]",
		wantOutIndent: "[\n\t{}\n]",
	}, {
		in:      `[{]}`,
		wantErr: `invalid character ']' at start of string`,
	}, {
		in:      `[,]`,
		wantErr: `invalid "," as value`,
	}, {
		in:      `{,}`,
		wantErr: `invalid character ',' at start of string`,
	}, {
		in:      `{"key""val"}`,
		wantErr: `invalid character '"', expected ':' in object`,
	}, {
		in:      `["elem0""elem1"]`,
		wantErr: `invalid character '"', expected ']' at end of array`,
	}, {
		in:      `{"hello"`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `{"hello"}`,
		wantErr: `invalid character '}', expected ':' in object`,
	}, {
		in:      `{"hello":`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `{"hello":}`,
		wantErr: `invalid "}" as value`,
	}, {
		in:      `{"hello":"goodbye"`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `{"hello":"goodbye"]`,
		wantErr: `invalid character ']', expected '}' at end of object`,
	}, {
		in:            `{"hello":"goodbye"}`,
		wantVal:       V(Obj{{V("hello"), V("goodbye")}}),
		wantOut:       `{"hello":"goodbye"}`,
		wantOutIndent: "{\n\t\"hello\": \"goodbye\"\n}",
	}, {
		in:      `{"hello":"goodbye",}`,
		wantErr: `invalid character '}' at start of string`,
	}, {
		in: `{"k":"v1","k":"v2"}`,
		wantVal: V(Obj{
			{V("k"), V("v1")}, {V("k"), V("v2")},
		}),
		wantOut:       `{"k":"v1","k":"v2"}`,
		wantOutIndent: "{\n\t\"k\": \"v1\",\n\t\"k\": \"v2\"\n}",
	}, {
		in: `{"k":{"k":{"k":"v"}}}`,
		wantVal: V(Obj{
			{V("k"), V(Obj{
				{V("k"), V(Obj{
					{V("k"), V("v")},
				})},
			})},
		}),
		wantOut:       `{"k":{"k":{"k":"v"}}}`,
		wantOutIndent: "{\n\t\"k\": {\n\t\t\"k\": {\n\t\t\t\"k\": \"v\"\n\t\t}\n\t}\n}",
	}, {
		in: `{"k":{"k":{"k":"v1","k":"v2"}}}`,
		wantVal: V(Obj{
			{V("k"), V(Obj{
				{V("k"), V(Obj{
					{V("k"), V("v1")},
					{V("k"), V("v2")},
				})},
			})},
		}),
		wantOut:       `{"k":{"k":{"k":"v1","k":"v2"}}}`,
		wantOutIndent: "{\n\t\"k\": {\n\t\t\"k\": {\n\t\t\t\"k\": \"v1\",\n\t\t\t\"k\": \"v2\"\n\t\t}\n\t}\n}",
	}, {
		in:      "  x",
		wantErr: `syntax error (line 1:3)`,
	}, {
		in:      `["💩"x`,
		wantErr: `syntax error (line 1:5)`,
	}, {
		in:      "\n\n[\"🔥🔥🔥\"x",
		wantErr: `syntax error (line 3:7)`,
	}, {
		in:      `["👍🏻👍🏿"x`,
		wantErr: `syntax error (line 1:8)`, // multi-rune emojis; could be column:6
	}, {
		in:      "\"\x00\"",
		wantErr: `invalid character '\x00' in string`,
	}, {
		in:      "\"\xff\"",
		wantErr: `invalid UTF-8 detected`,
		wantVal: V(string("\xff")),
	}, {
		in:      `"` + string(utf8.RuneError) + `"`,
		wantVal: V(string(utf8.RuneError)),
		wantOut: `"` + string(utf8.RuneError) + `"`,
	}, {
		in:      `"\uFFFD"`,
		wantVal: V(string(utf8.RuneError)),
		wantOut: `"` + string(utf8.RuneError) + `"`,
	}, {
		in:      `"\x"`,
		wantErr: `invalid escape code "\\x" in string`,
	}, {
		in:      `"\uXXXX"`,
		wantErr: `invalid escape code "\\uXXXX" in string`,
	}, {
		in:      `"\uDEAD"`, // unmatched surrogate pair
		wantErr: `unexpected EOF`,
	}, {
		in:      `"\uDEAD\uBEEF"`, // invalid surrogate half
		wantErr: `invalid escape code "\\uBEEF" in string`,
	}, {
		in:      `"\uD800\udead"`, // valid surrogate pair
		wantVal: V("𐊭"),
		wantOut: `"𐊭"`,
	}, {
		in:      `"\u0000\"\\\/\b\f\n\r\t"`,
		wantVal: V("\u0000\"\\/\b\f\n\r\t"),
		wantOut: `"\u0000\"\\/\b\f\n\r\t"`,
	}, {
		in:      `-`,
		wantErr: `invalid "-" as number`,
	}, {
		in:      `-0`,
		wantVal: V(math.Copysign(0, -1)),
		wantOut: `-0`,
	}, {
		in:      `+0`,
		wantErr: `invalid "+0" as value`,
	}, {
		in:      `-+`,
		wantErr: `invalid "-+" as number`,
	}, {
		in:      `0.`,
		wantErr: `invalid "0." as number`,
	}, {
		in:      `.1`,
		wantErr: `invalid ".1" as value`,
	}, {
		in:      `0.e1`,
		wantErr: `invalid "0.e1" as number`,
	}, {
		in:      `0.0`,
		wantVal: V(0.0),
		wantOut: "0",
	}, {
		in:      `01`,
		wantErr: `invalid "01" as number`,
	}, {
		in:      `0e`,
		wantErr: `invalid "0e" as number`,
	}, {
		in:      `0e0`,
		wantVal: V(0.0),
		wantOut: "0",
	}, {
		in:      `0E0`,
		wantVal: V(0.0),
		wantOut: "0",
	}, {
		in:      `0Ee`,
		wantErr: `invalid "0Ee" as number`,
	}, {
		in:      `-1.0E+1`,
		wantVal: V(-10.0),
		wantOut: "-10",
	}, {
		in: `
		{
		  "firstName" : "John",
		  "lastName" : "Smith" ,
		  "isAlive" : true,
		  "age" : 27,
		  "address" : {
		    "streetAddress" : "21 2nd Street" ,
		    "city" : "New York" ,
		    "state" : "NY" ,
		    "postalCode" : "10021-3100"
		  },
		  "phoneNumbers" : [
		    {
		      "type" : "home" ,
		      "number" : "212 555-1234"
		    } ,
		    {
		      "type" : "office" ,
		      "number" : "646 555-4567"
		    } ,
		    {
		      "type" : "mobile" ,
		      "number" : "123 456-7890"
		    }
		  ],
		  "children" : [] ,
		  "spouse" : null
		}
		`,
		wantVal: V(Obj{
			{V("firstName"), V("John")},
			{V("lastName"), V("Smith")},
			{V("isAlive"), V(true)},
			{V("age"), V(27.0)},
			{V("address"), V(Obj{
				{V("streetAddress"), V("21 2nd Street")},
				{V("city"), V("New York")},
				{V("state"), V("NY")},
				{V("postalCode"), V("10021-3100")},
			})},
			{V("phoneNumbers"), V(Arr{
				V(Obj{
					{V("type"), V("home")},
					{V("number"), V("212 555-1234")},
				}),
				V(Obj{
					{V("type"), V("office")},
					{V("number"), V("646 555-4567")},
				}),
				V(Obj{
					{V("type"), V("mobile")},
					{V("number"), V("123 456-7890")},
				}),
			})},
			{V("children"), V(Arr{})},
			{V("spouse"), V(nil)},
		}),
		wantOut: `{"firstName":"John","lastName":"Smith","isAlive":true,"age":27,"address":{"streetAddress":"21 2nd Street","city":"New York","state":"NY","postalCode":"10021-3100"},"phoneNumbers":[{"type":"home","number":"212 555-1234"},{"type":"office","number":"646 555-4567"},{"type":"mobile","number":"123 456-7890"}],"children":[],"spouse":null}`,
		wantOutIndent: `{
	"firstName": "John",
	"lastName": "Smith",
	"isAlive": true,
	"age": 27,
	"address": {
		"streetAddress": "21 2nd Street",
		"city": "New York",
		"state": "NY",
		"postalCode": "10021-3100"
	},
	"phoneNumbers": [
		{
			"type": "home",
			"number": "212 555-1234"
		},
		{
			"type": "office",
			"number": "646 555-4567"
		},
		{
			"type": "mobile",
			"number": "123 456-7890"
		}
	],
	"children": [],
	"spouse": null
}`,
	}}

	opts := cmp.Options{
		cmpopts.EquateEmpty(),
		cmp.Transformer("", func(v Value) interface{} {
			switch v.typ {
			case 0:
				return nil // special case so Value{} == Value{}
			case Null:
				return nil
			case Bool:
				return v.Bool()
			case Number:
				return v.Number()
			case String:
				return v.String()
			case Array:
				return v.Array()
			case Object:
				return v.Object()
			default:
				panic("invalid type")
			}
		}),
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if tt.in != "" || tt.wantVal.Type() != 0 || tt.wantErr != "" {
				gotVal, err := Unmarshal([]byte(tt.in))
				if err == nil {
					if tt.wantErr != "" {
						t.Errorf("Unmarshal(): got nil error, want %v", tt.wantErr)
					}
				} else {
					if tt.wantErr == "" {
						t.Errorf("Unmarshal(): got %v, want nil error", err)
					} else if !strings.Contains(err.Error(), tt.wantErr) {
						t.Errorf("Unmarshal(): got %v, want %v", err, tt.wantErr)
					}
				}
				if diff := cmp.Diff(gotVal, tt.wantVal, opts); diff != "" {
					t.Errorf("Unmarshal(): output mismatch (-got +want):\n%s", diff)
				}
			}
			if tt.wantOut != "" {
				gotOut, err := Marshal(tt.wantVal, "")
				if err != nil {
					t.Errorf("Marshal(): got %v, want nil error", err)
				}
				if string(gotOut) != tt.wantOut {
					t.Errorf("Marshal():\ngot:  %s\nwant: %s", gotOut, tt.wantOut)
				}
			}
			if tt.wantOutIndent != "" {
				gotOut, err := Marshal(tt.wantVal, "\t")
				if err != nil {
					t.Errorf("Marshal(Indent): got %v, want nil error", err)
				}
				if string(gotOut) != tt.wantOutIndent {
					t.Errorf("Marshal(Indent):\ngot:  %s\nwant: %s", gotOut, tt.wantOutIndent)
				}
			}
		})
	}
}
