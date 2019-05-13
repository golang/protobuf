// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	descopts "github.com/golang/protobuf/v2/internal/descopts"
	pragma "github.com/golang/protobuf/v2/internal/pragma"
	pfmt "github.com/golang/protobuf/v2/internal/typefmt"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

// inheritedMeta is information inherited from the parent.
type inheritedMeta struct {
	parent   pref.Descriptor
	index    int
	syntax   pref.Syntax
	fullName pref.FullName
}

func (m *inheritedMeta) init(nb *nameBuilder, parent pref.Descriptor, index int, name pref.Name, child bool) {
	// Most descriptors are namespaced as a child of their parent.
	// However, EnumValues are the exception in that they are namespaced
	// as a sibling of the parent Enum type.
	prefix := parent.FullName()
	if child {
		prefix = prefix.Parent()
	}

	m.parent = parent
	m.index = index
	m.syntax = parent.Syntax()
	m.fullName = nb.Append(prefix, name)
}

type fileMeta struct {
	ms messagesMeta
	es enumsMeta
	xs extensionsMeta
	ss servicesMeta
}
type fileDesc struct{ f *File }

// altOptions returns m as is if it is non-nil. Otherwise, it returns alt.
func altOptions(m, alt pref.ProtoMessage) pref.ProtoMessage {
	if m != nil {
		return m
	}
	return alt
}

func newFile(f *File) fileDesc {
	if f.fileMeta != nil {
		panic("already initialized")
	}
	f.fileMeta = new(fileMeta)
	return fileDesc{f}
}
func (t fileDesc) ParentFile() pref.FileDescriptor       { return t }
func (t fileDesc) Parent() (pref.Descriptor, bool)       { return nil, false }
func (t fileDesc) Index() int                            { return 0 }
func (t fileDesc) Syntax() pref.Syntax                   { return t.f.Syntax }
func (t fileDesc) Name() pref.Name                       { return t.f.Package.Name() }
func (t fileDesc) FullName() pref.FullName               { return t.f.Package }
func (t fileDesc) IsPlaceholder() bool                   { return false }
func (t fileDesc) Options() pref.ProtoMessage            { return altOptions(t.f.Options, descopts.File) }
func (t fileDesc) Path() string                          { return t.f.Path }
func (t fileDesc) Package() pref.FullName                { return t.f.Package }
func (t fileDesc) Imports() pref.FileImports             { return (*fileImports)(&t.f.Imports) }
func (t fileDesc) Enums() pref.EnumDescriptors           { return t.f.es.lazyInit(t, t.f.Enums) }
func (t fileDesc) Messages() pref.MessageDescriptors     { return t.f.ms.lazyInit(t, t.f.Messages) }
func (t fileDesc) Extensions() pref.ExtensionDescriptors { return t.f.xs.lazyInit(t, t.f.Extensions) }
func (t fileDesc) Services() pref.ServiceDescriptors     { return t.f.ss.lazyInit(t, t.f.Services) }
func (t fileDesc) Format(s fmt.State, r rune)            { pfmt.FormatDesc(s, r, t) }
func (t fileDesc) ProtoType(pref.FileDescriptor)         {}
func (t fileDesc) ProtoInternal(pragma.DoNotImplement)   {}

func parentFile(d protoreflect.Descriptor) protoreflect.FileDescriptor {
	for ; d != nil; d, _ = d.Parent() {
		if fd, ok := d.(protoreflect.FileDescriptor); ok {
			return fd
		}
	}
	return nil
}

type messageMeta struct {
	inheritedMeta

	fs fieldsMeta
	os oneofsMeta
	ns numbersMeta
	ms messagesMeta
	es enumsMeta
	xs extensionsMeta
}
type messageDesc struct{ m *Message }

func (t messageDesc) ParentFile() pref.FileDescriptor { return parentFile(t) }
func (t messageDesc) Parent() (pref.Descriptor, bool) { return t.m.parent, true }
func (t messageDesc) Index() int                      { return t.m.index }
func (t messageDesc) Syntax() pref.Syntax             { return t.m.syntax }
func (t messageDesc) Name() pref.Name                 { return t.m.Name }
func (t messageDesc) FullName() pref.FullName         { return t.m.fullName }
func (t messageDesc) IsPlaceholder() bool             { return false }
func (t messageDesc) Options() pref.ProtoMessage {
	return altOptions(t.m.Options, descopts.Message)
}
func (t messageDesc) IsMapEntry() bool                   { return t.m.IsMapEntry }
func (t messageDesc) Fields() pref.FieldDescriptors      { return t.m.fs.lazyInit(t, t.m.Fields) }
func (t messageDesc) Oneofs() pref.OneofDescriptors      { return t.m.os.lazyInit(t, t.m.Oneofs) }
func (t messageDesc) ReservedNames() pref.Names          { return (*names)(&t.m.ReservedNames) }
func (t messageDesc) ReservedRanges() pref.FieldRanges   { return (*fieldRanges)(&t.m.ReservedRanges) }
func (t messageDesc) RequiredNumbers() pref.FieldNumbers { return t.m.ns.lazyInit(t.m.Fields) }
func (t messageDesc) ExtensionRanges() pref.FieldRanges  { return (*fieldRanges)(&t.m.ExtensionRanges) }
func (t messageDesc) ExtensionRangeOptions(i int) pref.ProtoMessage {
	return extensionRangeOptions(i, len(t.m.ExtensionRanges), t.m.ExtensionRangeOptions)
}
func (t messageDesc) Enums() pref.EnumDescriptors           { return t.m.es.lazyInit(t, t.m.Enums) }
func (t messageDesc) Messages() pref.MessageDescriptors     { return t.m.ms.lazyInit(t, t.m.Messages) }
func (t messageDesc) Extensions() pref.ExtensionDescriptors { return t.m.xs.lazyInit(t, t.m.Extensions) }
func (t messageDesc) Format(s fmt.State, r rune)            { pfmt.FormatDesc(s, r, t) }
func (t messageDesc) ProtoType(pref.MessageDescriptor)      {}
func (t messageDesc) ProtoInternal(pragma.DoNotImplement)   {}

func extensionRangeOptions(i, n int, ms []pref.ProtoMessage) pref.ProtoMessage {
	if i < 0 || i >= n {
		panic("out of bounds")
	}
	var m pref.ProtoMessage
	if i < len(ms) {
		m = ms[i]
	}
	if m == nil {
		m = descopts.ExtensionRange
	}
	return m
}

type fieldMeta struct {
	inheritedMeta

	js jsonName
	dv defaultValue
	ot oneofReference
	mt messageReference
	et enumReference
}
type fieldDesc struct{ f *Field }

func (t fieldDesc) ParentFile() pref.FileDescriptor { return parentFile(t) }
func (t fieldDesc) Parent() (pref.Descriptor, bool) { return t.f.parent, true }
func (t fieldDesc) Index() int                      { return t.f.index }
func (t fieldDesc) Syntax() pref.Syntax             { return t.f.syntax }
func (t fieldDesc) Name() pref.Name                 { return t.f.Name }
func (t fieldDesc) FullName() pref.FullName         { return t.f.fullName }
func (t fieldDesc) IsPlaceholder() bool             { return false }
func (t fieldDesc) Options() pref.ProtoMessage      { return altOptions(t.f.Options, descopts.Field) }
func (t fieldDesc) Number() pref.FieldNumber        { return t.f.Number }
func (t fieldDesc) Cardinality() pref.Cardinality   { return t.f.Cardinality }
func (t fieldDesc) Kind() pref.Kind                 { return t.f.Kind }
func (t fieldDesc) HasJSONName() bool               { return t.f.JSONName != "" }
func (t fieldDesc) JSONName() string                { return t.f.js.lazyInit(t.f) }
func (t fieldDesc) IsPacked() bool {
	return isPacked(t.f.IsPacked, t.f.syntax, t.f.Cardinality, t.f.Kind)
}
func (t fieldDesc) IsExtension() bool { return false }
func (t fieldDesc) IsWeak() bool      { return t.f.IsWeak }
func (t fieldDesc) IsList() bool {
	return t.f.Cardinality == pref.Repeated && !t.IsMap()
}
func (t fieldDesc) IsMap() bool {
	mt := t.Message()
	return mt != nil && mt.IsMapEntry()
}
func (t fieldDesc) MapKey() pref.FieldDescriptor {
	if !t.IsMap() {
		return nil
	}
	return t.Message().Fields().ByNumber(1)
}
func (t fieldDesc) MapValue() pref.FieldDescriptor {
	if !t.IsMap() {
		return nil
	}
	return t.Message().Fields().ByNumber(2)
}
func (t fieldDesc) HasDefault() bool                           { return t.f.Default.IsValid() }
func (t fieldDesc) Default() pref.Value                        { return t.f.dv.value(t, t.f.Default) }
func (t fieldDesc) DefaultEnumValue() pref.EnumValueDescriptor { return t.f.dv.enum(t, t.f.Default) }
func (t fieldDesc) ContainingOneof() pref.OneofDescriptor      { return t.f.ot.lazyInit(t, t.f.OneofName) }
func (t fieldDesc) ContainingMessage() pref.MessageDescriptor {
	return t.f.parent.(pref.MessageDescriptor)
}
func (t fieldDesc) Enum() pref.EnumDescriptor           { return t.f.et.lazyInit(t, &t.f.EnumType) }
func (t fieldDesc) Message() pref.MessageDescriptor     { return t.f.mt.lazyInit(t, &t.f.MessageType) }
func (t fieldDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, t) }
func (t fieldDesc) ProtoType(pref.FieldDescriptor)      {}
func (t fieldDesc) ProtoInternal(pragma.DoNotImplement) {}

// TODO: Remove this.
func (t fieldDesc) Oneof() pref.OneofDescriptor      { return t.f.ot.lazyInit(t, t.f.OneofName) }
func (t fieldDesc) Extendee() pref.MessageDescriptor { return nil }

func isPacked(packed OptionalBool, s pref.Syntax, c pref.Cardinality, k pref.Kind) bool {
	if packed == False || (packed == DefaultBool && s == pref.Proto2) {
		return false
	}
	if c != pref.Repeated {
		return false
	}
	switch k {
	case pref.StringKind, pref.BytesKind, pref.MessageKind, pref.GroupKind:
		return false
	}
	return true
}

type jsonName struct {
	once sync.Once
	name string
}

func (p *jsonName) lazyInit(f *Field) string {
	p.once.Do(func() {
		// TODO: We may need to share this logic with jsonpb for implementation
		// of the FieldMask well-known type.
		if f.JSONName != "" {
			p.name = f.JSONName
			return
		}
		var b []byte
		var wasUnderscore bool
		for i := 0; i < len(f.Name); i++ { // proto identifiers are always ASCII
			c := f.Name[i]
			if c != '_' {
				isLower := 'a' <= c && c <= 'z'
				if wasUnderscore && isLower {
					c -= 'a' - 'A'
				}
				b = append(b, c)
			}
			wasUnderscore = c == '_'
		}
		p.name = string(b)
	})
	return p.name
}

// oneofReference resolves the name of a oneof by searching the parent
// message for the matching OneofDescriptor declaration.
type oneofReference struct {
	once sync.Once
	otyp pref.OneofDescriptor
}

func (p *oneofReference) lazyInit(parent pref.Descriptor, name pref.Name) pref.OneofDescriptor {
	p.once.Do(func() {
		if name != "" {
			mtyp, _ := parent.Parent()
			p.otyp = mtyp.(pref.MessageDescriptor).Oneofs().ByName(name)
			// TODO: We need validate to detect this mismatch.
		}
	})
	return p.otyp
}

type oneofMeta struct {
	inheritedMeta

	fs oneofFieldsMeta
}
type oneofDesc struct{ o *Oneof }

func (t oneofDesc) ParentFile() pref.FileDescriptor     { return parentFile(t) }
func (t oneofDesc) Parent() (pref.Descriptor, bool)     { return t.o.parent, true }
func (t oneofDesc) Index() int                          { return t.o.index }
func (t oneofDesc) Syntax() pref.Syntax                 { return t.o.syntax }
func (t oneofDesc) Name() pref.Name                     { return t.o.Name }
func (t oneofDesc) FullName() pref.FullName             { return t.o.fullName }
func (t oneofDesc) IsPlaceholder() bool                 { return false }
func (t oneofDesc) Options() pref.ProtoMessage          { return altOptions(t.o.Options, descopts.Oneof) }
func (t oneofDesc) Fields() pref.FieldDescriptors       { return t.o.fs.lazyInit(t) }
func (t oneofDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, t) }
func (t oneofDesc) ProtoType(pref.OneofDescriptor)      {}
func (t oneofDesc) ProtoInternal(pragma.DoNotImplement) {}

type extensionMeta struct {
	inheritedMeta

	dv defaultValue
	xt messageReference
	mt messageReference
	et enumReference
}
type extensionDesc struct{ x *Extension }

func (t extensionDesc) ParentFile() pref.FileDescriptor { return parentFile(t) }
func (t extensionDesc) Parent() (pref.Descriptor, bool) { return t.x.parent, true }
func (t extensionDesc) Syntax() pref.Syntax             { return t.x.syntax }
func (t extensionDesc) Index() int                      { return t.x.index }
func (t extensionDesc) Name() pref.Name                 { return t.x.Name }
func (t extensionDesc) FullName() pref.FullName         { return t.x.fullName }
func (t extensionDesc) IsPlaceholder() bool             { return false }
func (t extensionDesc) Options() pref.ProtoMessage {
	return altOptions(t.x.Options, descopts.Field)
}
func (t extensionDesc) Number() pref.FieldNumber      { return t.x.Number }
func (t extensionDesc) Cardinality() pref.Cardinality { return t.x.Cardinality }
func (t extensionDesc) Kind() pref.Kind               { return t.x.Kind }
func (t extensionDesc) HasJSONName() bool             { return false }
func (t extensionDesc) JSONName() string              { return "" }
func (t extensionDesc) IsPacked() bool {
	// Extensions always use proto2 defaults for packing.
	return isPacked(t.x.IsPacked, pref.Proto2, t.x.Cardinality, t.x.Kind)
}
func (t extensionDesc) IsExtension() bool                          { return true }
func (t extensionDesc) IsWeak() bool                               { return false }
func (t extensionDesc) IsList() bool                               { return t.x.Cardinality == pref.Repeated }
func (t extensionDesc) IsMap() bool                                { return false }
func (t extensionDesc) MapKey() pref.FieldDescriptor               { return nil }
func (t extensionDesc) MapValue() pref.FieldDescriptor             { return nil }
func (t extensionDesc) HasDefault() bool                           { return t.x.Default.IsValid() }
func (t extensionDesc) Default() pref.Value                        { return t.x.dv.value(t, t.x.Default) }
func (t extensionDesc) DefaultEnumValue() pref.EnumValueDescriptor { return t.x.dv.enum(t, t.x.Default) }
func (t extensionDesc) ContainingOneof() pref.OneofDescriptor      { return nil }
func (t extensionDesc) ContainingMessage() pref.MessageDescriptor {
	return t.x.xt.lazyInit(t, &t.x.ExtendedType)
}
func (t extensionDesc) Enum() pref.EnumDescriptor           { return t.x.et.lazyInit(t, &t.x.EnumType) }
func (t extensionDesc) Message() pref.MessageDescriptor     { return t.x.mt.lazyInit(t, &t.x.MessageType) }
func (t extensionDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, t) }
func (t extensionDesc) ProtoType(pref.FieldDescriptor)      {}
func (t extensionDesc) ProtoInternal(pragma.DoNotImplement) {}

// TODO: Remove this.
func (t extensionDesc) Oneof() pref.OneofDescriptor      { return nil }
func (t extensionDesc) Extendee() pref.MessageDescriptor { return t.x.xt.lazyInit(t, &t.x.ExtendedType) }

type enumMeta struct {
	inheritedMeta

	vs enumValuesMeta
}
type enumDesc struct{ e *Enum }

func (t enumDesc) ParentFile() pref.FileDescriptor     { return parentFile(t) }
func (t enumDesc) Parent() (pref.Descriptor, bool)     { return t.e.parent, true }
func (t enumDesc) Index() int                          { return t.e.index }
func (t enumDesc) Syntax() pref.Syntax                 { return t.e.syntax }
func (t enumDesc) Name() pref.Name                     { return t.e.Name }
func (t enumDesc) FullName() pref.FullName             { return t.e.fullName }
func (t enumDesc) IsPlaceholder() bool                 { return false }
func (t enumDesc) Options() pref.ProtoMessage          { return altOptions(t.e.Options, descopts.Enum) }
func (t enumDesc) Values() pref.EnumValueDescriptors   { return t.e.vs.lazyInit(t, t.e.Values) }
func (t enumDesc) ReservedNames() pref.Names           { return (*names)(&t.e.ReservedNames) }
func (t enumDesc) ReservedRanges() pref.EnumRanges     { return (*enumRanges)(&t.e.ReservedRanges) }
func (t enumDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, t) }
func (t enumDesc) ProtoType(pref.EnumDescriptor)       {}
func (t enumDesc) ProtoInternal(pragma.DoNotImplement) {}

type enumValueMeta struct {
	inheritedMeta
}
type enumValueDesc struct{ v *EnumValue }

func (t enumValueDesc) ParentFile() pref.FileDescriptor { return parentFile(t) }
func (t enumValueDesc) Parent() (pref.Descriptor, bool) { return t.v.parent, true }
func (t enumValueDesc) Index() int                      { return t.v.index }
func (t enumValueDesc) Syntax() pref.Syntax             { return t.v.syntax }
func (t enumValueDesc) Name() pref.Name                 { return t.v.Name }
func (t enumValueDesc) FullName() pref.FullName         { return t.v.fullName }
func (t enumValueDesc) IsPlaceholder() bool             { return false }
func (t enumValueDesc) Options() pref.ProtoMessage {
	return altOptions(t.v.Options, descopts.EnumValue)
}
func (t enumValueDesc) Number() pref.EnumNumber             { return t.v.Number }
func (t enumValueDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, t) }
func (t enumValueDesc) ProtoType(pref.EnumValueDescriptor)  {}
func (t enumValueDesc) ProtoInternal(pragma.DoNotImplement) {}

type serviceMeta struct {
	inheritedMeta

	ms methodsMeta
}
type serviceDesc struct{ s *Service }

func (t serviceDesc) ParentFile() pref.FileDescriptor { return parentFile(t) }
func (t serviceDesc) Parent() (pref.Descriptor, bool) { return t.s.parent, true }
func (t serviceDesc) Index() int                      { return t.s.index }
func (t serviceDesc) Syntax() pref.Syntax             { return t.s.syntax }
func (t serviceDesc) Name() pref.Name                 { return t.s.Name }
func (t serviceDesc) FullName() pref.FullName         { return t.s.fullName }
func (t serviceDesc) IsPlaceholder() bool             { return false }
func (t serviceDesc) Options() pref.ProtoMessage {
	return altOptions(t.s.Options, descopts.Service)
}
func (t serviceDesc) Methods() pref.MethodDescriptors     { return t.s.ms.lazyInit(t, t.s.Methods) }
func (t serviceDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, t) }
func (t serviceDesc) ProtoType(pref.ServiceDescriptor)    {}
func (t serviceDesc) ProtoInternal(pragma.DoNotImplement) {}

type methodMeta struct {
	inheritedMeta

	mit messageReference
	mot messageReference
}
type methodDesc struct{ m *Method }

func (t methodDesc) ParentFile() pref.FileDescriptor     { return parentFile(t) }
func (t methodDesc) Parent() (pref.Descriptor, bool)     { return t.m.parent, true }
func (t methodDesc) Index() int                          { return t.m.index }
func (t methodDesc) Syntax() pref.Syntax                 { return t.m.syntax }
func (t methodDesc) Name() pref.Name                     { return t.m.Name }
func (t methodDesc) FullName() pref.FullName             { return t.m.fullName }
func (t methodDesc) IsPlaceholder() bool                 { return false }
func (t methodDesc) Options() pref.ProtoMessage          { return altOptions(t.m.Options, descopts.Method) }
func (t methodDesc) Input() pref.MessageDescriptor       { return t.m.mit.lazyInit(t, &t.m.InputType) }
func (t methodDesc) Output() pref.MessageDescriptor      { return t.m.mot.lazyInit(t, &t.m.OutputType) }
func (t methodDesc) IsStreamingClient() bool             { return t.m.IsStreamingClient }
func (t methodDesc) IsStreamingServer() bool             { return t.m.IsStreamingServer }
func (t methodDesc) Format(s fmt.State, r rune)          { pfmt.FormatDesc(s, r, t) }
func (t methodDesc) ProtoType(pref.MethodDescriptor)     {}
func (t methodDesc) ProtoInternal(pragma.DoNotImplement) {}

type defaultValue struct {
	once sync.Once
	val  pref.Value
	eval pref.EnumValueDescriptor
	buf  []byte
}

var (
	zeroBool    = pref.ValueOf(false)
	zeroInt32   = pref.ValueOf(int32(0))
	zeroInt64   = pref.ValueOf(int64(0))
	zeroUint32  = pref.ValueOf(uint32(0))
	zeroUint64  = pref.ValueOf(uint64(0))
	zeroFloat32 = pref.ValueOf(float32(0))
	zeroFloat64 = pref.ValueOf(float64(0))
	zeroString  = pref.ValueOf(string(""))
	zeroBytes   = pref.ValueOf([]byte(nil))
	zeroEnum    = pref.ValueOf(pref.EnumNumber(0))
)

func (p *defaultValue) lazyInit(t pref.FieldDescriptor, v pref.Value) {
	p.once.Do(func() {
		p.val = v
		if v.IsValid() {
			switch t.Kind() {
			case pref.EnumKind:
				// Treat a string value as an identifier referencing some enum
				// value by name and extract the enum number.
				// If this fails, validateMessage will later detect that the
				// default value for an enum value is the wrong type.
				switch v := v.Interface().(type) {
				case string:
					if ev := t.Enum().Values().ByName(pref.Name(v)); ev != nil {
						p.eval = ev
						p.val = pref.ValueOf(p.eval.Number())
					}
				case pref.EnumNumber:
					p.eval = t.Enum().Values().ByNumber(v)
				}
			case pref.BytesKind:
				// Store a copy of the default bytes, so that we can detect
				// accidental mutations of the original value.
				if b, ok := v.Interface().([]byte); ok && len(b) > 0 {
					p.buf = append([]byte(nil), b...)
				}
			}
			return
		}
		switch t.Kind() {
		case pref.BoolKind:
			p.val = zeroBool
		case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
			p.val = zeroInt32
		case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
			p.val = zeroInt64
		case pref.Uint32Kind, pref.Fixed32Kind:
			p.val = zeroUint32
		case pref.Uint64Kind, pref.Fixed64Kind:
			p.val = zeroUint64
		case pref.FloatKind:
			p.val = zeroFloat32
		case pref.DoubleKind:
			p.val = zeroFloat64
		case pref.StringKind:
			p.val = zeroString
		case pref.BytesKind:
			p.val = zeroBytes
		case pref.EnumKind:
			p.val = zeroEnum
			if t.Syntax() == pref.Proto2 {
				if ed := t.Enum(); ed != nil {
					if vs := ed.Values(); vs.Len() > 0 {
						p.val = pref.ValueOf(vs.Get(0).Number())
					}
				}
			}
		}
	})
	if len(p.buf) > 0 && !bytes.Equal(p.buf, p.val.Bytes()) {
		// TODO: Avoid panic if we're running with the race detector and instead
		// spawn a goroutine that periodically resets this value back to the
		// original to induce a race that can be detected by the detector.
		panic(fmt.Sprintf("proto: detected mutation on the default bytes for %v", t.FullName()))
	}
}

func (p *defaultValue) value(t pref.FieldDescriptor, v pref.Value) pref.Value {
	p.lazyInit(t, v)
	return p.val
}

func (p *defaultValue) enum(t pref.FieldDescriptor, v pref.Value) pref.EnumValueDescriptor {
	p.lazyInit(t, v)
	return p.eval
}

// messageReference resolves PlaceholderMessages that reference declarations
// within the FileDescriptor tree that parent is a member of.
type messageReference struct{ once sync.Once }

func (p *messageReference) lazyInit(parent pref.Descriptor, pt *pref.MessageDescriptor) pref.MessageDescriptor {
	p.once.Do(func() {
		if t := *pt; t != nil && t.IsPlaceholder() {
			if d, ok := resolveReference(parent, t.FullName()).(pref.MessageDescriptor); ok {
				*pt = d
			}
		}
	})
	return *pt
}

// enumReference resolves PlaceholderEnums that reference declarations
// within the FileDescriptor tree that parent is a member of.
type enumReference struct{ once sync.Once }

func (p *enumReference) lazyInit(parent pref.Descriptor, pt *pref.EnumDescriptor) pref.EnumDescriptor {
	p.once.Do(func() {
		if t := *pt; t != nil && t.IsPlaceholder() {
			if d, ok := resolveReference(parent, t.FullName()).(pref.EnumDescriptor); ok {
				*pt = d
			}
		}
	})
	return *pt
}

// resolveReference searches parent for the MessageDescriptor or EnumDescriptor
// declaration identified by refName. This returns nil if not found.
func resolveReference(parent pref.Descriptor, refName pref.FullName) pref.Descriptor {
	// Ascend upwards until a prefix match is found.
	cur := parent
	for cur != nil {
		curName := cur.FullName()
		if strings.HasPrefix(string(refName), string(curName)) {
			if len(refName) == len(curName) {
				refName = refName[len(curName):]
				break // e.g., refName: foo.firetruck, curName: foo.firetruck
			} else if refName[len(curName)] == '.' {
				refName = refName[len(curName)+len("."):]
				break // e.g., refName: foo.firetruck.driver, curName: foo.firetruck
			} else if len(curName) == 0 {
				break // FileDescriptor has no package name
			}
			// No match. (e.g., refName: foo.firetruck, curName: foo.fire)
		}
		cur, _ = cur.Parent() // nil after ascending above FileDescriptor
	}

	// Descend downwards to resolve all relative names.
	for cur != nil && len(refName) > 0 {
		var head pref.Name
		head, refName = pref.Name(refName), ""
		if i := strings.IndexByte(string(head), '.'); i >= 0 {
			head, refName = head[:i], pref.FullName(head[i+len("."):])
		}

		// Search the current descriptor for the nested declaration.
		var next pref.Descriptor
		if t, ok := cur.(interface {
			Messages() pref.MessageDescriptors
		}); ok && next == nil {
			if d := t.Messages().ByName(head); d != nil {
				next = d
			}
		}
		if t, ok := cur.(interface {
			Enums() pref.EnumDescriptors
		}); ok && next == nil {
			if d := t.Enums().ByName(head); d != nil {
				next = d
			}
		}
		cur = next // nil if not found
	}
	return cur
}
