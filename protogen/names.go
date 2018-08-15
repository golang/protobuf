package protogen

import (
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

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
