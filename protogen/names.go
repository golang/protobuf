// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protogen

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// A GoIdent is a Go identifier, consisting of a name and import path.
// The name is a single identifier and may not be a dot-qualified selector.
type GoIdent struct {
	GoName       string
	GoImportPath GoImportPath
}

func (id GoIdent) String() string { return fmt.Sprintf("%q.%v", id.GoImportPath, id.GoName) }

// newGoIdent returns the Go identifier for a descriptor.
func newGoIdent(f *File, d protoreflect.Descriptor) GoIdent {
	name := strings.TrimPrefix(string(d.FullName()), string(f.Desc.Package())+".")
	return GoIdent{
		GoName:       camelCase(name),
		GoImportPath: f.GoImportPath,
	}
}

// A GoImportPath is the import path of a Go package. e.g., "google.golang.org/genproto/protobuf".
type GoImportPath string

func (p GoImportPath) String() string { return strconv.Quote(string(p)) }

// Ident returns a GoIdent with s as the GoName and p as the GoImportPath.
func (p GoImportPath) Ident(s string) GoIdent {
	return GoIdent{GoName: s, GoImportPath: p}
}

// A GoPackageName is the name of a Go package. e.g., "protobuf".
type GoPackageName string

// cleanPackageName converts a string to a valid Go package name.
func cleanPackageName(name string) GoPackageName {
	return GoPackageName(cleanGoName(name))
}

// cleanGoName converts a string to a valid Go identifier.
func cleanGoName(s string) string {
	// Sanitize the input to the set of valid characters,
	// which must be '_' or be in the Unicode L or N categories.
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, s)

	// Prepend '_' in the event of a Go keyword conflict or if
	// the identifier is invalid (does not start in the Unicode L category).
	r, _ := utf8.DecodeRuneInString(s)
	if token.Lookup(s).IsKeyword() || !unicode.IsLetter(r) {
		return "_" + s
	}
	return s
}

// baseName returns the last path element of the name, with the last dotted suffix removed.
func baseName(name string) string {
	// First, find the last element
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	// Now drop the suffix
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[:i]
	}
	return name
}

// camelCase converts a name to CamelCase.
//
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
// There is a remote possibility of this rewrite causing a name collision,
// but it's so remote we're prepared to pretend it's nonexistent - since the
// C++ generator lowercases names, it's extremely unlikely to have two fields
// with different capitalizations.
func camelCase(s string) string {
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '.' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip over '.' in ".{{lowercase}}".
		case c == '.':
			b = append(b, '_') // convert '.' to '_'
		case c == '_' && (i == 0 || s[i-1] == '.'):
			// Convert initial '_' to ensure we start with a capital letter.
			// Do the same for '_' after '.' to match historic behavior.
			b = append(b, 'X') // convert '_' to 'X'
		case c == '_' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip over '_' in "_{{lowercase}}".
		case isASCIIDigit(c):
			b = append(b, c)
		default:
			// Assume we have a letter now - if not, it's a bogus identifier.
			// The next word is a sequence of characters that must start upper case.
			if isASCIILower(c) {
				c -= 'a' - 'A' // convert lowercase to uppercase
			}
			b = append(b, c)

			// Accept lower case sequence that follows.
			for ; i+1 < len(s) && isASCIILower(s[i+1]); i++ {
				b = append(b, s[i+1])
			}
		}
	}
	return string(b)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
