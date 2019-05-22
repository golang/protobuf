// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"

	descfmt "google.golang.org/protobuf/internal/descfmt"
	"google.golang.org/protobuf/internal/descopts"
	"google.golang.org/protobuf/internal/pragma"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

var (
	emptyFiles       fileImports
	emptyMessages    messages
	emptyFields      fields
	emptyOneofs      oneofs
	emptyNames       names
	emptyNumbers     numbers
	emptyFieldRanges fieldRanges
	emptyEnums       enums
	emptyEnumValues  enumValues
	emptyEnumRanges  enumRanges
	emptyExtensions  extensions
	emptyServices    services
)

type placeholderName pref.FullName

func (t placeholderName) ParentFile() pref.FileDescriptor     { return nil }
func (t placeholderName) Parent() (pref.Descriptor, bool)     { return nil, false }
func (t placeholderName) Index() int                          { return 0 }
func (t placeholderName) Syntax() pref.Syntax                 { return 0 }
func (t placeholderName) Name() pref.Name                     { return pref.FullName(t).Name() }
func (t placeholderName) FullName() pref.FullName             { return pref.FullName(t) }
func (t placeholderName) IsPlaceholder() bool                 { return true }
func (t placeholderName) ProtoInternal(pragma.DoNotImplement) {}

type placeholderFile struct {
	path string
	placeholderName
}

func (t placeholderFile) ParentFile() pref.FileDescriptor       { return t }
func (t placeholderFile) Options() pref.ProtoMessage            { return descopts.File }
func (t placeholderFile) Path() string                          { return t.path }
func (t placeholderFile) Package() pref.FullName                { return t.FullName() }
func (t placeholderFile) Imports() pref.FileImports             { return &emptyFiles }
func (t placeholderFile) Enums() pref.EnumDescriptors           { return &emptyEnums }
func (t placeholderFile) Messages() pref.MessageDescriptors     { return &emptyMessages }
func (t placeholderFile) Extensions() pref.ExtensionDescriptors { return &emptyExtensions }
func (t placeholderFile) Services() pref.ServiceDescriptors     { return &emptyServices }
func (t placeholderFile) Format(s fmt.State, r rune)            { descfmt.FormatDesc(s, r, t) }
func (t placeholderFile) ProtoType(pref.FileDescriptor)         {}

type placeholderMessage struct {
	placeholderName
}

func (t placeholderMessage) Options() pref.ProtoMessage                  { return descopts.Message }
func (t placeholderMessage) IsMapEntry() bool                            { return false }
func (t placeholderMessage) Fields() pref.FieldDescriptors               { return &emptyFields }
func (t placeholderMessage) Oneofs() pref.OneofDescriptors               { return &emptyOneofs }
func (t placeholderMessage) ReservedNames() pref.Names                   { return &emptyNames }
func (t placeholderMessage) ReservedRanges() pref.FieldRanges            { return &emptyFieldRanges }
func (t placeholderMessage) RequiredNumbers() pref.FieldNumbers          { return &emptyNumbers }
func (t placeholderMessage) ExtensionRanges() pref.FieldRanges           { return &emptyFieldRanges }
func (t placeholderMessage) ExtensionRangeOptions(int) pref.ProtoMessage { panic("out of bounds") }
func (t placeholderMessage) Enums() pref.EnumDescriptors                 { return &emptyEnums }
func (t placeholderMessage) Messages() pref.MessageDescriptors           { return &emptyMessages }
func (t placeholderMessage) Extensions() pref.ExtensionDescriptors       { return &emptyExtensions }
func (t placeholderMessage) Format(s fmt.State, r rune)                  { descfmt.FormatDesc(s, r, t) }
func (t placeholderMessage) ProtoType(pref.MessageDescriptor)            {}

type placeholderEnum struct {
	placeholderName
}

func (t placeholderEnum) Options() pref.ProtoMessage        { return descopts.Enum }
func (t placeholderEnum) Values() pref.EnumValueDescriptors { return &emptyEnumValues }
func (t placeholderEnum) ReservedNames() pref.Names         { return &emptyNames }
func (t placeholderEnum) ReservedRanges() pref.EnumRanges   { return &emptyEnumRanges }
func (t placeholderEnum) Format(s fmt.State, r rune)        { descfmt.FormatDesc(s, r, t) }
func (t placeholderEnum) ProtoType(pref.EnumDescriptor)     {}
