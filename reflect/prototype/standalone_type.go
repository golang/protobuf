// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"

	"github.com/golang/protobuf/v2/internal/pragma"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

type standaloneMessage struct{ m *StandaloneMessage }

func (t standaloneMessage) Parent() (pref.Descriptor, bool)       { return nil, false }
func (t standaloneMessage) Index() int                            { return 0 }
func (t standaloneMessage) Syntax() pref.Syntax                   { return t.m.Syntax }
func (t standaloneMessage) Name() pref.Name                       { return t.m.FullName.Name() }
func (t standaloneMessage) FullName() pref.FullName               { return t.m.FullName }
func (t standaloneMessage) IsPlaceholder() bool                   { return false }
func (t standaloneMessage) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t standaloneMessage) Options() interface{}                  { return t.m.Options }
func (t standaloneMessage) IsMapEntry() bool                      { return t.m.Options.GetMapEntry() }
func (t standaloneMessage) Fields() pref.FieldDescriptors         { return t.m.fields.lazyInit(t, t.m.Fields) }
func (t standaloneMessage) Oneofs() pref.OneofDescriptors         { return t.m.oneofs.lazyInit(t, t.m.Oneofs) }
func (t standaloneMessage) RequiredNumbers() pref.FieldNumbers    { return t.m.nums.lazyInit(t.m.Fields) }
func (t standaloneMessage) ExtensionRanges() pref.FieldRanges     { return (*ranges)(&t.m.ExtensionRanges) }
func (t standaloneMessage) Messages() pref.MessageDescriptors     { return &emptyMessages }
func (t standaloneMessage) Enums() pref.EnumDescriptors           { return &emptyEnums }
func (t standaloneMessage) Extensions() pref.ExtensionDescriptors { return &emptyExtensions }
func (t standaloneMessage) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t standaloneMessage) ProtoType(pref.MessageDescriptor)      {}
func (t standaloneMessage) ProtoInternal(pragma.DoNotImplement)   {}

type standaloneEnum struct{ e *StandaloneEnum }

func (t standaloneEnum) Parent() (pref.Descriptor, bool)       { return nil, false }
func (t standaloneEnum) Index() int                            { return 0 }
func (t standaloneEnum) Syntax() pref.Syntax                   { return t.e.Syntax }
func (t standaloneEnum) Name() pref.Name                       { return t.e.FullName.Name() }
func (t standaloneEnum) FullName() pref.FullName               { return t.e.FullName }
func (t standaloneEnum) IsPlaceholder() bool                   { return false }
func (t standaloneEnum) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t standaloneEnum) Options() interface{}                  { return t.e.Options }
func (t standaloneEnum) Values() pref.EnumValueDescriptors     { return t.e.vals.lazyInit(t, t.e.Values) }
func (t standaloneEnum) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t standaloneEnum) ProtoType(pref.EnumDescriptor)         {}
func (t standaloneEnum) ProtoInternal(pragma.DoNotImplement)   {}

type standaloneExtension struct{ x *StandaloneExtension }

func (t standaloneExtension) Parent() (pref.Descriptor, bool)       { return nil, false }
func (t standaloneExtension) Index() int                            { return 0 }
func (t standaloneExtension) Syntax() pref.Syntax                   { return pref.Proto2 }
func (t standaloneExtension) Name() pref.Name                       { return t.x.FullName.Name() }
func (t standaloneExtension) FullName() pref.FullName               { return t.x.FullName }
func (t standaloneExtension) IsPlaceholder() bool                   { return false }
func (t standaloneExtension) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t standaloneExtension) Options() interface{}                  { return t.x.Options }
func (t standaloneExtension) Number() pref.FieldNumber              { return t.x.Number }
func (t standaloneExtension) Cardinality() pref.Cardinality         { return t.x.Cardinality }
func (t standaloneExtension) Kind() pref.Kind                       { return t.x.Kind }
func (t standaloneExtension) JSONName() string                      { return "" }
func (t standaloneExtension) IsPacked() bool                        { return t.x.Options.GetPacked() }
func (t standaloneExtension) IsMap() bool                           { return false }
func (t standaloneExtension) IsWeak() bool                          { return false }
func (t standaloneExtension) Default() pref.Value                   { return t.x.dv.value(t, t.x.Default) }
func (t standaloneExtension) DefaultEnumValue() pref.EnumValueDescriptor {
	return t.x.dv.enum(t, t.x.Default)
}
func (t standaloneExtension) HasDefault() bool                     { return t.x.Default.IsValid() }
func (t standaloneExtension) OneofType() pref.OneofDescriptor      { return nil }
func (t standaloneExtension) MessageType() pref.MessageDescriptor  { return t.x.MessageType }
func (t standaloneExtension) EnumType() pref.EnumDescriptor        { return t.x.EnumType }
func (t standaloneExtension) ExtendedType() pref.MessageDescriptor { return t.x.ExtendedType }
func (t standaloneExtension) Format(s fmt.State, r rune)           { formatDesc(s, r, t) }
func (t standaloneExtension) ProtoType(pref.FieldDescriptor)       {}
func (t standaloneExtension) ProtoInternal(pragma.DoNotImplement)  {}
