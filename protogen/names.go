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

// A GoPackageName is the name of a Go package. e.g., "protobuf".
type GoPackageName string

// cleanPacakgeName converts a string to a valid Go package name.
func cleanPackageName(name string) GoPackageName {
	name = strings.Map(badToUnderscore, name)
	// Identifier must not be keyword: insert _.
	if token.Lookup(name).IsKeyword() {
		name = "_" + name
	}
	// Identifier must not begin with digit: insert _.
	if r, _ := utf8.DecodeRuneInString(name); unicode.IsDigit(r) {
		name = "_" + name
	}
	return GoPackageName(name)
}

var isGoPredeclaredIdentifier = map[string]bool{
	"append":     true,
	"bool":       true,
	"byte":       true,
	"cap":        true,
	"close":      true,
	"complex":    true,
	"complex128": true,
	"complex64":  true,
	"copy":       true,
	"delete":     true,
	"error":      true,
	"false":      true,
	"float32":    true,
	"float64":    true,
	"imag":       true,
	"int":        true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"int8":       true,
	"iota":       true,
	"len":        true,
	"make":       true,
	"new":        true,
	"nil":        true,
	"panic":      true,
	"print":      true,
	"println":    true,
	"real":       true,
	"recover":    true,
	"rune":       true,
	"string":     true,
	"true":       true,
	"uint":       true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uint8":      true,
	"uintptr":    true,
}

// badToUnderscore is the mapping function used to generate Go names from package names,
// which can be dotted in the input .proto file.  It replaces non-identifier characters such as
// dot or dash with underscore.
func badToUnderscore(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return r
	}
	return '_'
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
	if s == "" {
		return ""
	}
	var t []byte
	i := 0
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '.' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip over .<lowercase>, to match historic behavior.
		case c == '.':
			t = append(t, '_') // Convert . to _.
		case c == '_' && (i == 0 || s[i-1] == '.'):
			// Convert initial _ to X so we start with a capital letter.
			// Do the same for _ after .; not strictly necessary, but matches
			// historic behavior.
			t = append(t, 'X')
		case c == '_' && i+1 < len(s) && isASCIILower(s[i+1]):
			// Skip the underscore in s.
		case isASCIIDigit(c):
			t = append(t, c)
		default:
			// Assume we have a letter now - if not, it's a bogus identifier.
			// The next word is a sequence of characters that must start upper case.
			if isASCIILower(c) {
				c ^= ' ' // Make it a capital letter.
			}
			t = append(t, c) // Guaranteed not lower case.
			// Accept lower case sequence that follows.
			for i+1 < len(s) && isASCIILower(s[i+1]) {
				i++
				t = append(t, s[i])
			}
		}
	}
	return string(t)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
