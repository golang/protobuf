// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/golang/protobuf/v2/internal/detrand"
	"github.com/golang/protobuf/v2/internal/flags"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Disable detrand to enable direct comparisons on outputs.
func init() { detrand.Disable() }

var S = fmt.Sprintf
var V = ValueOf
var ID = func(n protoreflect.Name) Value { return V(n) }

type Lst = []Value
type Msg = [][2]Value

func Test(t *testing.T) {
	const space = " \n\r\t"

	tests := []struct {
		in             string
		wantVal        Value
		wantOut        string
		wantOutBracket string
		wantOutASCII   string
		wantOutIndent  string
		wantErr        string
	}{{
		in:            "",
		wantVal:       V(Msg{}),
		wantOutIndent: "\n",
	}, {
		in:      S("%s# hello%s", space, space),
		wantVal: V(Msg{}),
	}, {
		in:      S("%s# hello\rfoo:bar", space),
		wantVal: V(Msg{}),
	}, {
		// Comments only extend until the newline.
		in:            S("%s# hello\nfoo:bar", space),
		wantVal:       V(Msg{{ID("foo"), ID("bar")}}),
		wantOut:       "foo:bar",
		wantOutIndent: "foo: bar\n",
	}, {
		// NUL is an invalid whitespace since C++ uses C-strings.
		in:      "\x00",
		wantErr: `invalid "\x00" as identifier`,
	}, {
		in:      "foo:0",
		wantVal: V(Msg{{ID("foo"), V(uint32(0))}}),
		wantOut: "foo:0",
	}, {
		in:      S("%sfoo%s:0", space, space),
		wantVal: V(Msg{{ID("foo"), V(uint32(0))}}),
	}, {
		in:      "foo bar:0",
		wantErr: `expected ':' after message key`,
	}, {
		in:            "[foo]:0",
		wantVal:       V(Msg{{V("foo"), V(uint32(0))}}),
		wantOut:       "[foo]:0",
		wantOutIndent: "[foo]: 0\n",
	}, {
		in:      S("%s[%sfoo%s]%s:0", space, space, space, space),
		wantVal: V(Msg{{V("foo"), V(uint32(0))}}),
	}, {
		in:            "[proto.package.name]:0",
		wantVal:       V(Msg{{V("proto.package.name"), V(uint32(0))}}),
		wantOut:       "[proto.package.name]:0",
		wantOutIndent: "[proto.package.name]: 0\n",
	}, {
		in:      S("%s[%sproto.package.name%s]%s:0", space, space, space, space),
		wantVal: V(Msg{{V("proto.package.name"), V(uint32(0))}}),
	}, {
		in:            "['sub.domain.com\x2fpath\x2fto\x2fproto.package.name']:0",
		wantVal:       V(Msg{{V("sub.domain.com/path/to/proto.package.name"), V(uint32(0))}}),
		wantOut:       "[sub.domain.com/path/to/proto.package.name]:0",
		wantOutIndent: "[sub.domain.com/path/to/proto.package.name]: 0\n",
	}, {
		in:      "[\"sub.domain.com\x2fpath\x2fto\x2fproto.package.name\"]:0",
		wantVal: V(Msg{{V("sub.domain.com/path/to/proto.package.name"), V(uint32(0))}}),
	}, {
		in:      S("%s[%s'sub.domain.com\x2fpath\x2fto\x2fproto.package.name'%s]%s:0", space, space, space, space),
		wantVal: V(Msg{{V("sub.domain.com/path/to/proto.package.name"), V(uint32(0))}}),
	}, {
		in:      S("%s[%s\"sub.domain.com\x2fpath\x2fto\x2fproto.package.name\"%s]%s:0", space, space, space, space),
		wantVal: V(Msg{{V("sub.domain.com/path/to/proto.package.name"), V(uint32(0))}}),
	}, {
		in:            `['http://example.com/path/to/proto.package.name']:0`,
		wantVal:       V(Msg{{V("http://example.com/path/to/proto.package.name"), V(uint32(0))}}),
		wantOut:       `["http://example.com/path/to/proto.package.name"]:0`,
		wantOutIndent: `["http://example.com/path/to/proto.package.name"]: 0` + "\n",
	}, {
		in:      "[proto.package.name:0",
		wantErr: `invalid character ':', expected ']' at end of extension name`,
	}, {
		in:      "[proto.package name]:0",
		wantErr: `invalid character 'n', expected ']' at end of extension name`,
	}, {
		in:      `["proto.package" "name"]:0`,
		wantErr: `invalid character '"', expected ']' at end of extension name`,
	}, {
		in:      `["\z"]`,
		wantErr: `invalid escape code "\\z" in string`,
	}, {
		in:      "[$]",
		wantErr: `invalid "$" as identifier`,
	}, {
		// This parses fine, but should result in a error later since no
		// type name in proto will ever be just a number.
		in:      "[20]:0",
		wantVal: V(Msg{{V("20"), V(uint32(0))}}),
		wantOut: "[20]:0",
	}, {
		in:      "20:0",
		wantVal: V(Msg{{V(uint32(20)), V(uint32(0))}}),
		wantOut: "20:0",
	}, {
		in:      "0x20:0",
		wantVal: V(Msg{{V(uint32(0x20)), V(uint32(0))}}),
		wantOut: "32:0",
	}, {
		in:      "020:0",
		wantVal: V(Msg{{V(uint32(020)), V(uint32(0))}}),
		wantOut: "16:0",
	}, {
		in:      "-20:0",
		wantErr: `invalid "-20" as identifier`,
	}, {
		in: `foo:true bar:"s" baz:{} qux:[] wib:id`,
		wantVal: V(Msg{
			{ID("foo"), V(true)},
			{ID("bar"), V("s")},
			{ID("baz"), V(Msg{})},
			{ID("qux"), V(Lst{})},
			{ID("wib"), ID("id")},
		}),
		wantOut:       `foo:true bar:"s" baz:{} qux:[] wib:id`,
		wantOutIndent: "foo: true\nbar: \"s\"\nbaz: {}\nqux: []\nwib: id\n",
	}, {
		in: S(`%sfoo%s:%strue%s %sbar%s:%s"s"%s %sbaz%s:%s<>%s %squx%s:%s[]%s %swib%s:%sid%s`,
			space, space, space, space, space, space, space, space, space, space, space, space, space, space, space, space, space, space, space, space),
		wantVal: V(Msg{
			{ID("foo"), V(true)},
			{ID("bar"), V("s")},
			{ID("baz"), V(Msg{})},
			{ID("qux"), V(Lst{})},
			{ID("wib"), ID("id")},
		}),
	}, {
		in:            `foo:true;`,
		wantVal:       V(Msg{{ID("foo"), V(true)}}),
		wantOut:       "foo:true",
		wantOutIndent: "foo: true\n",
	}, {
		in:      `foo:true,`,
		wantVal: V(Msg{{ID("foo"), V(true)}}),
	}, {
		in:      `foo:bar;,`,
		wantErr: `invalid "," as identifier`,
	}, {
		in:      `foo:bar,;`,
		wantErr: `invalid ";" as identifier`,
	}, {
		in:      `footrue`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `foo true`,
		wantErr: `expected ':' after message key`,
	}, {
		in:      `foo"s"`,
		wantErr: `expected ':' after message key`,
	}, {
		in:      `foo "s"`,
		wantErr: `expected ':' after message key`,
	}, {
		in:             `foo{}`,
		wantVal:        V(Msg{{ID("foo"), V(Msg{})}}),
		wantOut:        "foo:{}",
		wantOutBracket: "foo:<>",
		wantOutIndent:  "foo: {}\n",
	}, {
		in:      `foo {}`,
		wantVal: V(Msg{{ID("foo"), V(Msg{})}}),
	}, {
		in:      `foo<>`,
		wantVal: V(Msg{{ID("foo"), V(Msg{})}}),
	}, {
		in:      `foo <>`,
		wantVal: V(Msg{{ID("foo"), V(Msg{})}}),
	}, {
		in:      `foo[]`,
		wantErr: `expected ':' after message key`,
	}, {
		in:      `foo []`,
		wantErr: `expected ':' after message key`,
	}, {
		in:      `foo:truebar:true`,
		wantErr: `invalid ":" as identifier`,
	}, {
		in:            `foo:"s"bar:true`,
		wantVal:       V(Msg{{ID("foo"), V("s")}, {ID("bar"), V(true)}}),
		wantOut:       `foo:"s" bar:true`,
		wantOutIndent: "foo: \"s\"\nbar: true\n",
	}, {
		in:      `foo:0bar:true`,
		wantErr: `invalid "0bar" as number or bool`,
	}, {
		in:             `foo:{}bar:true`,
		wantVal:        V(Msg{{ID("foo"), V(Msg{})}, {ID("bar"), V(true)}}),
		wantOut:        "foo:{} bar:true",
		wantOutBracket: "foo:<> bar:true",
		wantOutIndent:  "foo: {}\nbar: true\n",
	}, {
		in:      `foo:[]bar:true`,
		wantVal: V(Msg{{ID("foo"), V(Lst{})}, {ID("bar"), V(true)}}),
	}, {
		in:             `foo{bar:true}`,
		wantVal:        V(Msg{{ID("foo"), V(Msg{{ID("bar"), V(true)}})}}),
		wantOut:        "foo:{bar:true}",
		wantOutBracket: "foo:<bar:true>",
		wantOutIndent:  "foo: {\n\tbar: true\n}\n",
	}, {
		in:      `foo<bar:true>`,
		wantVal: V(Msg{{ID("foo"), V(Msg{{ID("bar"), V(true)}})}}),
	}, {
		in:      `foo{bar:true,}`,
		wantVal: V(Msg{{ID("foo"), V(Msg{{ID("bar"), V(true)}})}}),
	}, {
		in:      `foo{bar:true;}`,
		wantVal: V(Msg{{ID("foo"), V(Msg{{ID("bar"), V(true)}})}}),
	}, {
		in:      `foo{`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `foo{ `,
		wantErr: `unexpected EOF`,
	}, {
		in:      `foo{[`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `foo{[ `,
		wantErr: `unexpected EOF`,
	}, {
		in:      `foo{bar:true,;}`,
		wantErr: `invalid ";" as identifier`,
	}, {
		in:      `foo{bar:true;,}`,
		wantErr: `invalid "," as identifier`,
	}, {
		in:             `foo<bar:{}>`,
		wantVal:        V(Msg{{ID("foo"), V(Msg{{ID("bar"), V(Msg{})}})}}),
		wantOut:        "foo:{bar:{}}",
		wantOutBracket: "foo:<bar:<>>",
		wantOutIndent:  "foo: {\n\tbar: {}\n}\n",
	}, {
		in:      `foo<bar:{>`,
		wantErr: `invalid character '>', expected '}' at end of message`,
	}, {
		in:      `foo<bar:{}`,
		wantErr: `unexpected EOF`,
	}, {
		in:             `arr:[]`,
		wantVal:        V(Msg{{ID("arr"), V(Lst{})}}),
		wantOut:        "arr:[]",
		wantOutBracket: "arr:[]",
		wantOutIndent:  "arr: []\n",
	}, {
		in:      `arr:[,]`,
		wantErr: `invalid "," as number or bool`,
	}, {
		in:      `arr:[0 0]`,
		wantErr: `invalid character '0', expected ']' at end of list`,
	}, {
		in:             `arr:["foo" "bar"]`,
		wantVal:        V(Msg{{ID("arr"), V(Lst{V("foobar")})}}),
		wantOut:        `arr:["foobar"]`,
		wantOutBracket: `arr:["foobar"]`,
		wantOutIndent:  "arr: [\n\t\"foobar\"\n]\n",
	}, {
		in:      `arr:[0,]`,
		wantErr: `invalid "]" as number or bool`,
	}, {
		in: `arr:[true,0,"",id,[],{}]`,
		wantVal: V(Msg{{ID("arr"), V(Lst{
			V(true), V(uint32(0)), V(""), ID("id"), V(Lst{}), V(Msg{}),
		})}}),
		wantOut:        `arr:[true,0,"",id,[],{}]`,
		wantOutBracket: `arr:[true,0,"",id,[],<>]`,
		wantOutIndent:  "arr: [\n\ttrue,\n\t0,\n\t\"\",\n\tid,\n\t[],\n\t{}\n]\n",
	}, {
		in: S(`arr:[%strue%s,%s0%s,%s""%s,%sid%s,%s[]%s,%s{}%s]`,
			space, space, space, space, space, space, space, space, space, space, space, space),
		wantVal: V(Msg{{ID("arr"), V(Lst{
			V(true), V(uint32(0)), V(""), ID("id"), V(Lst{}), V(Msg{}),
		})}}),
	}, {
		in:      `arr:[`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `{`,
		wantErr: `invalid "{" as identifier`,
	}, {
		in:      `<`,
		wantErr: `invalid "<" as identifier`,
	}, {
		in:      `[`,
		wantErr: "unexpected EOF",
	}, {
		in:      `}`,
		wantErr: "1 bytes of unconsumed input",
	}, {
		in:      `>`,
		wantErr: "1 bytes of unconsumed input",
	}, {
		in:      `]`,
		wantErr: `invalid "]" as identifier`,
	}, {
		in:      `str: "'"`,
		wantVal: V(Msg{{ID("str"), V(`'`)}}),
		wantOut: `str:"'"`,
	}, {
		in:      `str: '"'`,
		wantVal: V(Msg{{ID("str"), V(`"`)}}),
		wantOut: `str:"\""`,
	}, {
		// String that has as few escaped characters as possible.
		in: `str: ` + func() string {
			var b []byte
			for i := 0; i < utf8.RuneSelf; i++ {
				switch i {
				case 0, '\\', '\n', '\'': // these must be escaped, so ignore them
				default:
					b = append(b, byte(i))
				}
			}
			return "'" + string(b) + "'"
		}(),
		wantVal:      V(Msg{{ID("str"), V("\x01\x02\x03\x04\x05\x06\a\b\t\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~\u007f")}}),
		wantOut:      `str:"\x01\x02\x03\x04\x05\x06\x07\x08\t\x0b\x0c\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_` + "`abcdefghijklmnopqrstuvwxyz{|}~\x7f\"",
		wantOutASCII: `str:"\x01\x02\x03\x04\x05\x06\x07\x08\t\x0b\x0c\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_` + "`abcdefghijklmnopqrstuvwxyz{|}~\x7f\"",
	}, {
		in:           "str: '\xde\xad\xbe\xef'",
		wantVal:      V(Msg{{ID("str"), V("\xde\xad\xbe\xef")}}),
		wantOut:      "str:\"\u07ad\\xbe\\xef\"",
		wantOutASCII: `str:"\u07ad\xbe\xef"`,
		wantErr:      "invalid UTF-8 detected",
	}, {
		// Valid UTF-8 wire encoding, but sub-optimal encoding.
		in:           "str: '\xc0\x80'",
		wantVal:      V(Msg{{ID("str"), V("\xc0\x80")}}),
		wantOut:      `str:"\xc0\x80"`,
		wantOutASCII: `str:"\xc0\x80"`,
		wantErr:      "invalid UTF-8 detected",
	}, {
		// Valid UTF-8 wire encoding, but invalid rune (surrogate pair).
		in:           "str: '\xed\xa0\x80'",
		wantVal:      V(Msg{{ID("str"), V("\xed\xa0\x80")}}),
		wantOut:      `str:"\xed\xa0\x80"`,
		wantOutASCII: `str:"\xed\xa0\x80"`,
		wantErr:      "invalid UTF-8 detected",
	}, {
		// Valid UTF-8 wire encoding, but invalid rune (above max rune).
		in:           "str: '\xf7\xbf\xbf\xbf'",
		wantVal:      V(Msg{{ID("str"), V("\xf7\xbf\xbf\xbf")}}),
		wantOut:      `str:"\xf7\xbf\xbf\xbf"`,
		wantOutASCII: `str:"\xf7\xbf\xbf\xbf"`,
		wantErr:      "invalid UTF-8 detected",
	}, {
		// Valid UTF-8 wire encoding of the RuneError rune.
		in:           "str: '\xef\xbf\xbd'",
		wantVal:      V(Msg{{ID("str"), V(string(utf8.RuneError))}}),
		wantOut:      `str:"` + string(utf8.RuneError) + `"`,
		wantOutASCII: `str:"\ufffd"`,
	}, {
		in:           "str: 'hello\u1234world'",
		wantVal:      V(Msg{{ID("str"), V("hello\u1234world")}}),
		wantOut:      "str:\"hello\u1234world\"",
		wantOutASCII: `str:"hello\u1234world"`,
	}, {
		in:           `str: '\"\'\\\?\a\b\n\r\t\v\f\1\12\123\xA\xaB\x12\uAb8f\U0010FFFF'`,
		wantVal:      V(Msg{{ID("str"), V("\"'\\?\a\b\n\r\t\v\f\x01\nS\n\xab\x12\uab8f\U0010ffff")}}),
		wantOut:      `str:"\"'\\?\x07\x08\n\r\t\x0b\x0c\x01\nS\n\xab\x12` + "\uab8f\U0010ffff" + `"`,
		wantOutASCII: `str:"\"'\\?\x07\x08\n\r\t\x0b\x0c\x01\nS\n\xab\x12\uab8f\U0010ffff"`,
	}, {
		in:      `str: '`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `str: '\`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `str: '\'`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `str: '\8'`,
		wantErr: `invalid escape code "\\8" in string`,
	}, {
		in:           `str: '\1x'`,
		wantVal:      V(Msg{{ID("str"), V("\001x")}}),
		wantOut:      `str:"\x01x"`,
		wantOutASCII: `str:"\x01x"`,
	}, {
		in:           `str: '\12x'`,
		wantVal:      V(Msg{{ID("str"), V("\012x")}}),
		wantOut:      `str:"\nx"`,
		wantOutASCII: `str:"\nx"`,
	}, {
		in:           `str: '\123x'`,
		wantVal:      V(Msg{{ID("str"), V("\123x")}}),
		wantOut:      `str:"Sx"`,
		wantOutASCII: `str:"Sx"`,
	}, {
		in:           `str: '\1234x'`,
		wantVal:      V(Msg{{ID("str"), V("\1234x")}}),
		wantOut:      `str:"S4x"`,
		wantOutASCII: `str:"S4x"`,
	}, {
		in:           `str: '\1'`,
		wantVal:      V(Msg{{ID("str"), V("\001")}}),
		wantOut:      `str:"\x01"`,
		wantOutASCII: `str:"\x01"`,
	}, {
		in:           `str: '\12'`,
		wantVal:      V(Msg{{ID("str"), V("\012")}}),
		wantOut:      `str:"\n"`,
		wantOutASCII: `str:"\n"`,
	}, {
		in:           `str: '\123'`,
		wantVal:      V(Msg{{ID("str"), V("\123")}}),
		wantOut:      `str:"S"`,
		wantOutASCII: `str:"S"`,
	}, {
		in:           `str: '\1234'`,
		wantVal:      V(Msg{{ID("str"), V("\1234")}}),
		wantOut:      `str:"S4"`,
		wantOutASCII: `str:"S4"`,
	}, {
		in:           `str: '\377'`,
		wantVal:      V(Msg{{ID("str"), V("\377")}}),
		wantOut:      `str:"\xff"`,
		wantOutASCII: `str:"\xff"`,
	}, {
		// Overflow octal escape.
		in:      `str: '\400'`,
		wantErr: `invalid octal escape code "\\400" in string`,
	}, {
		in:           `str: '\xfx'`,
		wantVal:      V(Msg{{ID("str"), V("\x0fx")}}),
		wantOut:      `str:"\x0fx"`,
		wantOutASCII: `str:"\x0fx"`,
	}, {
		in:           `str: '\xffx'`,
		wantVal:      V(Msg{{ID("str"), V("\xffx")}}),
		wantOut:      `str:"\xffx"`,
		wantOutASCII: `str:"\xffx"`,
	}, {
		in:           `str: '\xfffx'`,
		wantVal:      V(Msg{{ID("str"), V("\xfffx")}}),
		wantOut:      `str:"\xfffx"`,
		wantOutASCII: `str:"\xfffx"`,
	}, {
		in:           `str: '\xf'`,
		wantVal:      V(Msg{{ID("str"), V("\x0f")}}),
		wantOut:      `str:"\x0f"`,
		wantOutASCII: `str:"\x0f"`,
	}, {
		in:           `str: '\xff'`,
		wantVal:      V(Msg{{ID("str"), V("\xff")}}),
		wantOut:      `str:"\xff"`,
		wantOutASCII: `str:"\xff"`,
	}, {
		in:           `str: '\xfff'`,
		wantVal:      V(Msg{{ID("str"), V("\xfff")}}),
		wantOut:      `str:"\xfff"`,
		wantOutASCII: `str:"\xfff"`,
	}, {
		in:      `str: '\xz'`,
		wantErr: `invalid hex escape code "\\x" in string`,
	}, {
		in:      `str: '\uPo'`,
		wantErr: `unexpected EOF`,
	}, {
		in:      `str: '\uPoo'`,
		wantErr: `invalid Unicode escape code "\\uPoo'" in string`,
	}, {
		in:      `str: '\uPoop'`,
		wantErr: `invalid Unicode escape code "\\uPoop" in string`,
	}, {
		// Unmatched surrogate pair.
		in:      `str: '\uDEAD'`,
		wantErr: `unexpected EOF`, // trying to reader other half
	}, {
		// Surrogate pair with invalid other half.
		in:      `str: '\uDEAD\u0000'`,
		wantErr: `invalid Unicode escape code "\\u0000" in string`,
	}, {
		// Properly matched surrogate pair.
		in:           `str: '\uD800\uDEAD'`,
		wantVal:      V(Msg{{ID("str"), V("𐊭")}}),
		wantOut:      `str:"𐊭"`,
		wantOutASCII: `str:"\U000102ad"`,
	}, {
		// Overflow on Unicode rune.
		in:      `str: '\U00110000'`,
		wantErr: `invalid Unicode escape code "\\U00110000" in string`,
	}, {
		in:      `str: '\z'`,
		wantErr: `invalid escape code "\\z" in string`,
	}, {
		// Strings cannot have NUL literal since C-style strings forbid them.
		in:      "str: '\x00'",
		wantErr: `invalid character '\x00' in string`,
	}, {
		// Strings cannot have newline literal. The C++ permits them if an
		// option is specified to allow them. In Go, we always forbid them.
		in:      "str: '\n'",
		wantErr: `invalid character '\n' in string`,
	}, {
		in:           "name: \"My name is \"\n\"elsewhere\"",
		wantVal:      V(Msg{{ID("name"), V("My name is elsewhere")}}),
		wantOut:      `name:"My name is elsewhere"`,
		wantOutASCII: `name:"My name is elsewhere"`,
	}, {
		in:      "name: 'My name is '\n'elsewhere'",
		wantVal: V(Msg{{ID("name"), V("My name is elsewhere")}}),
	}, {
		in:      "name: 'My name is '\n\"elsewhere\"",
		wantVal: V(Msg{{ID("name"), V("My name is elsewhere")}}),
	}, {
		in:      "name: \"My name is \"\n'elsewhere'",
		wantVal: V(Msg{{ID("name"), V("My name is elsewhere")}}),
	}, {
		in:      "name: \"My \"'name '\"is \"\n'elsewhere'",
		wantVal: V(Msg{{ID("name"), V("My name is elsewhere")}}),
	}, {
		in:      `crazy:"x'"'\""\''"'z"`,
		wantVal: V(Msg{{ID("crazy"), V(`x'""''z`)}}),
	}, {
		in: `nums: [t,T,true,True,TRUE,f,F,false,False,FALSE]`,
		wantVal: V(Msg{{ID("nums"), V(Lst{
			V(true),
			ID("T"),
			V(true),
			V(true),
			ID("TRUE"),
			V(false),
			ID("F"),
			V(false),
			V(false),
			ID("FALSE"),
		})}}),
		wantOut:       "nums:[true,T,true,true,TRUE,false,F,false,false,FALSE]",
		wantOutIndent: "nums: [\n\ttrue,\n\tT,\n\ttrue,\n\ttrue,\n\tTRUE,\n\tfalse,\n\tF,\n\tfalse,\n\tfalse,\n\tFALSE\n]\n",
	}, {
		in: `nums: [nan,inf,-inf,NaN,NAN,Inf,INF]`,
		wantVal: V(Msg{{ID("nums"), V(Lst{
			V(math.NaN()),
			V(math.Inf(+1)),
			V(math.Inf(-1)),
			ID("NaN"),
			ID("NAN"),
			ID("Inf"),
			ID("INF"),
		})}}),
		wantOut:       "nums:[nan,inf,-inf,NaN,NAN,Inf,INF]",
		wantOutIndent: "nums: [\n\tnan,\n\tinf,\n\t-inf,\n\tNaN,\n\tNAN,\n\tInf,\n\tINF\n]\n",
	}, {
		// C++ permits this, but we currently reject this.
		in:      `num: -nan`,
		wantErr: `invalid "-nan" as number or bool`,
	}, {
		in: `nums: [0,-0,-9876543210,9876543210,0x0,0x0123456789abcdef,-0x0123456789abcdef,01234567,-01234567]`,
		wantVal: V(Msg{{ID("nums"), V(Lst{
			V(uint32(0)),
			V(int32(-0)),
			V(int64(-9876543210)),
			V(uint64(9876543210)),
			V(uint32(0x0)),
			V(uint64(0x0123456789abcdef)),
			V(int64(-0x0123456789abcdef)),
			V(uint64(01234567)),
			V(int64(-01234567)),
		})}}),
		wantOut:       "nums:[0,0,-9876543210,9876543210,0,81985529216486895,-81985529216486895,342391,-342391]",
		wantOutIndent: "nums: [\n\t0,\n\t0,\n\t-9876543210,\n\t9876543210,\n\t0,\n\t81985529216486895,\n\t-81985529216486895,\n\t342391,\n\t-342391\n]\n",
	}, {
		in: `nums: [0.,0f,1f,10f,-0f,-1f,-10f,1.0,0.1e-3,1.5e+5,1e10,.0]`,
		wantVal: V(Msg{{ID("nums"), V(Lst{
			V(0.0),
			V(0.0),
			V(1.0),
			V(10.0),
			V(-0.0),
			V(-1.0),
			V(-10.0),
			V(1.0),
			V(0.1e-3),
			V(1.5e+5),
			V(1.0e+10),
			V(0.0),
		})}}),
		wantOut:       "nums:[0,0,1,10,0,-1,-10,1,0.0001,150000,1e+10,0]",
		wantOutIndent: "nums: [\n\t0,\n\t0,\n\t1,\n\t10,\n\t0,\n\t-1,\n\t-10,\n\t1,\n\t0.0001,\n\t150000,\n\t1e+10,\n\t0\n]\n",
	}, {
		in: `nums: [0xbeefbeef,0xbeefbeefbeefbeef]`,
		wantVal: V(Msg{{ID("nums"), func() Value {
			if flags.Proto1Legacy {
				return V(Lst{V(int32(-1091584273)), V(int64(-4688318750159552785))})
			} else {
				return V(Lst{V(uint32(0xbeefbeef)), V(uint64(0xbeefbeefbeefbeef))})
			}
		}()}}),
	}, {
		in:      `num: +0`,
		wantErr: `invalid "+0" as number or bool`,
	}, {
		in:      `num: 01.1234`,
		wantErr: `invalid "01.1234" as number or bool`,
	}, {
		in:      `num: 0x`,
		wantErr: `invalid "0x" as number or bool`,
	}, {
		in:      `num: 0xX`,
		wantErr: `invalid "0xX" as number or bool`,
	}, {
		in:      `num: 0800`,
		wantErr: `invalid "0800" as number or bool`,
	}, {
		in:      `num: true.`,
		wantErr: `invalid "true." as number or bool`,
	}, {
		in:      `num: .`,
		wantErr: `parsing ".": invalid syntax`,
	}, {
		in:      `num: -.`,
		wantErr: `parsing "-.": invalid syntax`,
	}, {
		in:      `num: 1e10000`,
		wantErr: `parsing "1e10000": value out of range`,
	}, {
		in:      `num: 99999999999999999999`,
		wantErr: `parsing "99999999999999999999": value out of range`,
	}, {
		in:      `num: -99999999999999999999`,
		wantErr: `parsing "-99999999999999999999": value out of range`,
	}, {
		in:      "x:  -",
		wantErr: `syntax error (line 1:5)`,
	}, {
		in:      "x:[\"💩\"x",
		wantErr: `syntax error (line 1:7)`,
	}, {
		in:      "x:\n\n[\"🔥🔥🔥\"x",
		wantErr: `syntax error (line 3:7)`,
	}, {
		in:      "x:[\"👍🏻👍🏿\"x",
		wantErr: `syntax error (line 1:10)`, // multi-rune emojis; could be column:8
	}, {
		in: `
			firstName : "John",
			lastName : "Smith" ,
			isAlive : true,
			age : 27,
			address { # missing colon is okay for messages
			    streetAddress : "21 2nd Street" ,
			    city : "New York" ,
			    state : "NY" ,
			    postalCode : "10021-3100" ; # trailing semicolon is okay
			},
			phoneNumbers : [ {
			    type : "home" ,
			    number : "212 555-1234"
			} , {
			    type : "office" ,
			    number : "646 555-4567"
			} , {
			    type : "mobile" ,
			    number : "123 456-7890" , # trailing comma is okay
			} ],
			children : [] ,
			spouse : null`,
		wantVal: V(Msg{
			{ID("firstName"), V("John")},
			{ID("lastName"), V("Smith")},
			{ID("isAlive"), V(true)},
			{ID("age"), V(27.0)},
			{ID("address"), V(Msg{
				{ID("streetAddress"), V("21 2nd Street")},
				{ID("city"), V("New York")},
				{ID("state"), V("NY")},
				{ID("postalCode"), V("10021-3100")},
			})},
			{ID("phoneNumbers"), V([]Value{
				V(Msg{
					{ID("type"), V("home")},
					{ID("number"), V("212 555-1234")},
				}),
				V(Msg{
					{ID("type"), V("office")},
					{ID("number"), V("646 555-4567")},
				}),
				V(Msg{
					{ID("type"), V("mobile")},
					{ID("number"), V("123 456-7890")},
				}),
			})},
			{ID("children"), V([]Value{})},
			{ID("spouse"), V(protoreflect.Name("null"))},
		}),
		wantOut:        `firstName:"John" lastName:"Smith" isAlive:true age:27 address:{streetAddress:"21 2nd Street" city:"New York" state:"NY" postalCode:"10021-3100"} phoneNumbers:[{type:"home" number:"212 555-1234"},{type:"office" number:"646 555-4567"},{type:"mobile" number:"123 456-7890"}] children:[] spouse:null`,
		wantOutBracket: `firstName:"John" lastName:"Smith" isAlive:true age:27 address:<streetAddress:"21 2nd Street" city:"New York" state:"NY" postalCode:"10021-3100"> phoneNumbers:[<type:"home" number:"212 555-1234">,<type:"office" number:"646 555-4567">,<type:"mobile" number:"123 456-7890">] children:[] spouse:null`,
		wantOutIndent: `firstName: "John"
lastName: "Smith"
isAlive: true
age: 27
address: {
	streetAddress: "21 2nd Street"
	city: "New York"
	state: "NY"
	postalCode: "10021-3100"
}
phoneNumbers: [
	{
		type: "home"
		number: "212 555-1234"
	},
	{
		type: "office"
		number: "646 555-4567"
	},
	{
		type: "mobile"
		number: "123 456-7890"
	}
]
children: []
spouse: null
`,
	}}

	opts := cmp.Options{
		cmpopts.EquateEmpty(),

		// Transform composites (List and Message).
		cmp.FilterValues(func(x, y Value) bool {
			return (x.Type() == List && y.Type() == List) || (x.Type() == Message && y.Type() == Message)
		}, cmp.Transformer("", func(v Value) interface{} {
			if v.Type() == List {
				return v.List()
			} else {
				return v.Message()
			}
		})),

		// Compare scalars (Bool, Int, Uint, Float, String, Name).
		cmp.FilterValues(func(x, y Value) bool {
			return !(x.Type() == List && y.Type() == List) && !(x.Type() == Message && y.Type() == Message)
		}, cmp.Comparer(func(x, y Value) bool {
			if x.Type() == List || x.Type() == Message || y.Type() == List || y.Type() == Message {
				return false
			}
			// Ensure golden value is always in x variable.
			if len(x.raw) > 0 {
				x, y = y, x
			}
			switch x.Type() {
			case Bool:
				want, _ := x.Bool()
				got, ok := y.Bool()
				return got == want && ok
			case Int:
				want, _ := x.Int(true)
				got, ok := y.Int(want < math.MinInt32 || math.MaxInt32 < want)
				return got == want && ok
			case Uint:
				want, _ := x.Uint(true)
				got, ok := y.Uint(math.MaxUint32 < want)
				return got == want && ok
			case Float:
				want, _ := x.Float(true)
				got, ok := y.Float(math.MaxFloat32 < math.Abs(want))
				if math.IsNaN(got) || math.IsNaN(want) {
					return math.IsNaN(got) == math.IsNaN(want)
				}
				return got == want && ok
			case Name:
				want, _ := x.Name()
				got, ok := y.Name()
				return got == want && ok
			default:
				return x.String() == y.String()
			}
		})),
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
				gotOut, err := Marshal(tt.wantVal, "", [2]byte{0, 0}, false)
				if err != nil {
					t.Errorf("Marshal(): got %v, want nil error", err)
				}
				if string(gotOut) != tt.wantOut {
					t.Errorf("Marshal():\ngot:  %s\nwant: %s", gotOut, tt.wantOut)
				}
			}
			if tt.wantOutBracket != "" {
				gotOut, err := Marshal(tt.wantVal, "", [2]byte{'<', '>'}, false)
				if err != nil {
					t.Errorf("Marshal(Bracket): got %v, want nil error", err)
				}
				if string(gotOut) != tt.wantOutBracket {
					t.Errorf("Marshal(Bracket):\ngot:  %s\nwant: %s", gotOut, tt.wantOutBracket)
				}
			}
			if tt.wantOutASCII != "" {
				gotOut, err := Marshal(tt.wantVal, "", [2]byte{0, 0}, true)
				if err != nil {
					t.Errorf("Marshal(ASCII): got %v, want nil error", err)
				}
				if string(gotOut) != tt.wantOutASCII {
					t.Errorf("Marshal(ASCII):\ngot:  %s\nwant: %s", gotOut, tt.wantOutASCII)
				}
			}
			if tt.wantOutIndent != "" {
				gotOut, err := Marshal(tt.wantVal, "\t", [2]byte{0, 0}, false)
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
