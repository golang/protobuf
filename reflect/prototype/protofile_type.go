// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	descriptorV1 "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/v2/internal/pragma"
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
	ds descriptorsMeta
}
type fileDesc struct{ f *File }

func newFile(f *File) fileDesc {
	if f.fileMeta != nil {
		panic("already initialized")
	}
	f.fileMeta = new(fileMeta)
	return fileDesc{f}
}
func (t fileDesc) Parent() (pref.Descriptor, bool)                  { return nil, false }
func (t fileDesc) Index() int                                       { return 0 }
func (t fileDesc) Syntax() pref.Syntax                              { return t.f.Syntax }
func (t fileDesc) Name() pref.Name                                  { return t.f.Package.Name() }
func (t fileDesc) FullName() pref.FullName                          { return t.f.Package }
func (t fileDesc) IsPlaceholder() bool                              { return false }
func (t fileDesc) DescriptorProto() (pref.Message, bool)            { return nil, false }
func (t fileDesc) Options() interface{}                             { return t.f.Options }
func (t fileDesc) Path() string                                     { return t.f.Path }
func (t fileDesc) Package() pref.FullName                           { return t.f.Package }
func (t fileDesc) Imports() pref.FileImports                        { return (*fileImports)(&t.f.Imports) }
func (t fileDesc) Messages() pref.MessageDescriptors                { return t.f.ms.lazyInit(t, t.f.Messages) }
func (t fileDesc) Enums() pref.EnumDescriptors                      { return t.f.es.lazyInit(t, t.f.Enums) }
func (t fileDesc) Extensions() pref.ExtensionDescriptors            { return t.f.xs.lazyInit(t, t.f.Extensions) }
func (t fileDesc) Services() pref.ServiceDescriptors                { return t.f.ss.lazyInit(t, t.f.Services) }
func (t fileDesc) DescriptorByName(s pref.FullName) pref.Descriptor { return t.f.ds.lookup(t, s) }
func (t fileDesc) Format(s fmt.State, r rune)                       { formatDesc(s, r, t) }
func (t fileDesc) ProtoType(pref.FileDescriptor)                    {}
func (t fileDesc) ProtoInternal(pragma.DoNotImplement)              {}

// descriptorsMeta is a lazily initialized map of all descriptors declared in
// the file by full name.
type descriptorsMeta struct {
	once sync.Once
	m    map[pref.FullName]pref.Descriptor
}

func (m *descriptorsMeta) lookup(fd pref.FileDescriptor, s pref.FullName) pref.Descriptor {
	m.once.Do(func() {
		m.m = make(map[pref.FullName]pref.Descriptor)
		m.initMap(fd)
		delete(m.m, fd.Package()) // avoid registering the file descriptor itself
	})
	return m.m[s]
}
func (m *descriptorsMeta) initMap(d pref.Descriptor) {
	m.m[d.FullName()] = d
	if ds, ok := d.(interface {
		Enums() pref.EnumDescriptors
	}); ok {
		for i := 0; i < ds.Enums().Len(); i++ {
			m.initMap(ds.Enums().Get(i))
		}
	}
	if ds, ok := d.(interface {
		Values() pref.EnumValueDescriptors
	}); ok {
		for i := 0; i < ds.Values().Len(); i++ {
			m.initMap(ds.Values().Get(i))
		}
	}
	if ds, ok := d.(interface {
		Messages() pref.MessageDescriptors
	}); ok {
		for i := 0; i < ds.Messages().Len(); i++ {
			m.initMap(ds.Messages().Get(i))
		}
	}
	if ds, ok := d.(interface {
		Fields() pref.FieldDescriptors
	}); ok {
		for i := 0; i < ds.Fields().Len(); i++ {
			m.initMap(ds.Fields().Get(i))
		}
	}
	if ds, ok := d.(interface {
		Oneofs() pref.OneofDescriptors
	}); ok {
		for i := 0; i < ds.Oneofs().Len(); i++ {
			m.initMap(ds.Oneofs().Get(i))
		}
	}
	if ds, ok := d.(interface {
		Extensions() pref.ExtensionDescriptors
	}); ok {
		for i := 0; i < ds.Extensions().Len(); i++ {
			m.initMap(ds.Extensions().Get(i))
		}
	}
	if ds, ok := d.(interface {
		Services() pref.ServiceDescriptors
	}); ok {
		for i := 0; i < ds.Services().Len(); i++ {
			m.initMap(ds.Services().Get(i))
		}
	}
	if ds, ok := d.(interface {
		Methods() pref.MethodDescriptors
	}); ok {
		for i := 0; i < ds.Methods().Len(); i++ {
			m.initMap(ds.Methods().Get(i))
		}
	}
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

func (t messageDesc) Parent() (pref.Descriptor, bool)       { return t.m.parent, true }
func (t messageDesc) Index() int                            { return t.m.index }
func (t messageDesc) Syntax() pref.Syntax                   { return t.m.syntax }
func (t messageDesc) Name() pref.Name                       { return t.m.Name }
func (t messageDesc) FullName() pref.FullName               { return t.m.fullName }
func (t messageDesc) IsPlaceholder() bool                   { return false }
func (t messageDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t messageDesc) Options() interface{}                  { return t.m.Options }
func (t messageDesc) IsMapEntry() bool                      { return t.m.Options.GetMapEntry() }
func (t messageDesc) Fields() pref.FieldDescriptors         { return t.m.fs.lazyInit(t, t.m.Fields) }
func (t messageDesc) Oneofs() pref.OneofDescriptors         { return t.m.os.lazyInit(t, t.m.Oneofs) }
func (t messageDesc) RequiredNumbers() pref.FieldNumbers    { return t.m.ns.lazyInit(t.m.Fields) }
func (t messageDesc) ExtensionRanges() pref.FieldRanges     { return (*ranges)(&t.m.ExtensionRanges) }
func (t messageDesc) Messages() pref.MessageDescriptors     { return t.m.ms.lazyInit(t, t.m.Messages) }
func (t messageDesc) Enums() pref.EnumDescriptors           { return t.m.es.lazyInit(t, t.m.Enums) }
func (t messageDesc) Extensions() pref.ExtensionDescriptors { return t.m.xs.lazyInit(t, t.m.Extensions) }
func (t messageDesc) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t messageDesc) ProtoType(pref.MessageDescriptor)      {}
func (t messageDesc) ProtoInternal(pragma.DoNotImplement)   {}

type fieldMeta struct {
	inheritedMeta

	js jsonName
	dv defaultValue
	ot oneofReference
	mt messageReference
	et enumReference
}
type fieldDesc struct{ f *Field }

func (t fieldDesc) Parent() (pref.Descriptor, bool)       { return t.f.parent, true }
func (t fieldDesc) Index() int                            { return t.f.index }
func (t fieldDesc) Syntax() pref.Syntax                   { return t.f.syntax }
func (t fieldDesc) Name() pref.Name                       { return t.f.Name }
func (t fieldDesc) FullName() pref.FullName               { return t.f.fullName }
func (t fieldDesc) IsPlaceholder() bool                   { return false }
func (t fieldDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t fieldDesc) Options() interface{}                  { return t.f.Options }
func (t fieldDesc) Number() pref.FieldNumber              { return t.f.Number }
func (t fieldDesc) Cardinality() pref.Cardinality         { return t.f.Cardinality }
func (t fieldDesc) Kind() pref.Kind                       { return t.f.Kind }
func (t fieldDesc) JSONName() string                      { return t.f.js.lazyInit(t.f) }
func (t fieldDesc) IsPacked() bool                        { return fieldIsPacked(t) }
func (t fieldDesc) IsMap() bool                           { return isMap(t) }
func (t fieldDesc) IsWeak() bool                          { return t.f.Options.GetWeak() }
func (t fieldDesc) Default() pref.Value                   { return t.f.dv.lazyInit(t, t.f.Default) }
func (t fieldDesc) HasDefault() bool                      { return t.f.Default.IsValid() }
func (t fieldDesc) OneofType() pref.OneofDescriptor       { return t.f.ot.lazyInit(t, t.f.OneofName) }
func (t fieldDesc) ExtendedType() pref.MessageDescriptor  { return nil }
func (t fieldDesc) MessageType() pref.MessageDescriptor   { return t.f.mt.lazyInit(t, &t.f.MessageType) }
func (t fieldDesc) EnumType() pref.EnumDescriptor         { return t.f.et.lazyInit(t, &t.f.EnumType) }
func (t fieldDesc) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t fieldDesc) ProtoType(pref.FieldDescriptor)        {}
func (t fieldDesc) ProtoInternal(pragma.DoNotImplement)   {}

func fieldIsPacked(t fieldDesc) bool {
	if t.f.Options != nil && t.f.Options.Packed != nil {
		return *t.f.Options.Packed
	}
	// https://developers.google.com/protocol-buffers/docs/proto3:
	// "In proto3, repeated fields of scalar numeric types use packed
	// encoding by default."
	return (t.f.syntax == pref.Proto3 &&
		t.f.Cardinality == pref.Repeated &&
		isScalarNumeric[t.f.Kind])
}

var isScalarNumeric = map[pref.Kind]bool{
	pref.BoolKind:     true,
	pref.EnumKind:     true,
	pref.Int32Kind:    true,
	pref.Sint32Kind:   true,
	pref.Uint32Kind:   true,
	pref.Int64Kind:    true,
	pref.Sint64Kind:   true,
	pref.Uint64Kind:   true,
	pref.Sfixed32Kind: true,
	pref.Fixed32Kind:  true,
	pref.FloatKind:    true,
	pref.Sfixed64Kind: true,
	pref.Fixed64Kind:  true,
	pref.DoubleKind:   true,
}

func isMap(t pref.FieldDescriptor) bool {
	if t.Cardinality() == pref.Repeated && t.Kind() == pref.MessageKind {
		if mt := t.MessageType(); mt != nil {
			return mt.Options().(*descriptorV1.MessageOptions).GetMapEntry()
		}
	}
	return false
}

type jsonName struct{ once sync.Once }

func (p *jsonName) lazyInit(f *Field) string {
	p.once.Do(func() {
		// TODO: We may need to share this logic with jsonpb for implementation
		// of the FieldMask well-known type.
		if f.JSONName != "" {
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
		f.JSONName = string(b)
	})
	return f.JSONName
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

func (t oneofDesc) Parent() (pref.Descriptor, bool)       { return t.o.parent, true }
func (t oneofDesc) Index() int                            { return t.o.index }
func (t oneofDesc) Syntax() pref.Syntax                   { return t.o.syntax }
func (t oneofDesc) Name() pref.Name                       { return t.o.Name }
func (t oneofDesc) FullName() pref.FullName               { return t.o.fullName }
func (t oneofDesc) IsPlaceholder() bool                   { return false }
func (t oneofDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t oneofDesc) Options() interface{}                  { return t.o.Options }
func (t oneofDesc) Fields() pref.FieldDescriptors         { return t.o.fs.lazyInit(t) }
func (t oneofDesc) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t oneofDesc) ProtoType(pref.OneofDescriptor)        {}
func (t oneofDesc) ProtoInternal(pragma.DoNotImplement)   {}

type extensionMeta struct {
	inheritedMeta

	dv defaultValue
	xt messageReference
	mt messageReference
	et enumReference
}
type extensionDesc struct{ x *Extension }

func (t extensionDesc) Parent() (pref.Descriptor, bool)       { return t.x.parent, true }
func (t extensionDesc) Syntax() pref.Syntax                   { return t.x.syntax }
func (t extensionDesc) Index() int                            { return t.x.index }
func (t extensionDesc) Name() pref.Name                       { return t.x.Name }
func (t extensionDesc) FullName() pref.FullName               { return t.x.fullName }
func (t extensionDesc) IsPlaceholder() bool                   { return false }
func (t extensionDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t extensionDesc) Options() interface{}                  { return t.x.Options }
func (t extensionDesc) Number() pref.FieldNumber              { return t.x.Number }
func (t extensionDesc) Cardinality() pref.Cardinality         { return t.x.Cardinality }
func (t extensionDesc) Kind() pref.Kind                       { return t.x.Kind }
func (t extensionDesc) JSONName() string                      { return "" }
func (t extensionDesc) IsPacked() bool                        { return extIsPacked(t) }
func (t extensionDesc) IsMap() bool                           { return false }
func (t extensionDesc) IsWeak() bool                          { return false }
func (t extensionDesc) Default() pref.Value                   { return t.x.dv.lazyInit(t, t.x.Default) }
func (t extensionDesc) HasDefault() bool                      { return t.x.Default.IsValid() }
func (t extensionDesc) OneofType() pref.OneofDescriptor       { return nil }
func (t extensionDesc) ExtendedType() pref.MessageDescriptor {
	return t.x.xt.lazyInit(t, &t.x.ExtendedType)
}
func (t extensionDesc) MessageType() pref.MessageDescriptor {
	return t.x.mt.lazyInit(t, &t.x.MessageType)
}
func (t extensionDesc) EnumType() pref.EnumDescriptor       { return t.x.et.lazyInit(t, &t.x.EnumType) }
func (t extensionDesc) Format(s fmt.State, r rune)          { formatDesc(s, r, t) }
func (t extensionDesc) ProtoType(pref.FieldDescriptor)      {}
func (t extensionDesc) ProtoInternal(pragma.DoNotImplement) {}

func extIsPacked(t extensionDesc) bool {
	// TODO: When a proto3 file extends a proto2 message (permitted only for
	// extending descriptor options), does the extension have proto2 or proto3
	// semantics? If the latter, repeated, scalar, numeric, proto3 extension
	// fields should default to packed. If the former, perhaps the extension syntax
	// should be protoreflect.Proto2.
	return t.x.Options.GetPacked()
}

type enumMeta struct {
	inheritedMeta

	vs enumValuesMeta
}
type enumDesc struct{ e *Enum }

func (t enumDesc) Parent() (pref.Descriptor, bool)       { return t.e.parent, true }
func (t enumDesc) Index() int                            { return t.e.index }
func (t enumDesc) Syntax() pref.Syntax                   { return t.e.syntax }
func (t enumDesc) Name() pref.Name                       { return t.e.Name }
func (t enumDesc) FullName() pref.FullName               { return t.e.fullName }
func (t enumDesc) IsPlaceholder() bool                   { return false }
func (t enumDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t enumDesc) Options() interface{}                  { return t.e.Options }
func (t enumDesc) Values() pref.EnumValueDescriptors     { return t.e.vs.lazyInit(t, t.e.Values) }
func (t enumDesc) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t enumDesc) ProtoType(pref.EnumDescriptor)         {}
func (t enumDesc) ProtoInternal(pragma.DoNotImplement)   {}

type enumValueMeta struct {
	inheritedMeta
}
type enumValueDesc struct{ v *EnumValue }

func (t enumValueDesc) Parent() (pref.Descriptor, bool)       { return t.v.parent, true }
func (t enumValueDesc) Index() int                            { return t.v.index }
func (t enumValueDesc) Syntax() pref.Syntax                   { return t.v.syntax }
func (t enumValueDesc) Name() pref.Name                       { return t.v.Name }
func (t enumValueDesc) FullName() pref.FullName               { return t.v.fullName }
func (t enumValueDesc) IsPlaceholder() bool                   { return false }
func (t enumValueDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t enumValueDesc) Options() interface{}                  { return t.v.Options }
func (t enumValueDesc) Number() pref.EnumNumber               { return t.v.Number }
func (t enumValueDesc) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t enumValueDesc) ProtoType(pref.EnumValueDescriptor)    {}
func (t enumValueDesc) ProtoInternal(pragma.DoNotImplement)   {}

type serviceMeta struct {
	inheritedMeta

	ms methodsMeta
}
type serviceDesc struct{ s *Service }

func (t serviceDesc) Parent() (pref.Descriptor, bool)       { return t.s.parent, true }
func (t serviceDesc) Index() int                            { return t.s.index }
func (t serviceDesc) Syntax() pref.Syntax                   { return t.s.syntax }
func (t serviceDesc) Name() pref.Name                       { return t.s.Name }
func (t serviceDesc) FullName() pref.FullName               { return t.s.fullName }
func (t serviceDesc) IsPlaceholder() bool                   { return false }
func (t serviceDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t serviceDesc) Options() interface{}                  { return t.s.Options }
func (t serviceDesc) Methods() pref.MethodDescriptors       { return t.s.ms.lazyInit(t, t.s.Methods) }
func (t serviceDesc) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t serviceDesc) ProtoType(pref.ServiceDescriptor)      {}
func (t serviceDesc) ProtoInternal(pragma.DoNotImplement)   {}

type methodMeta struct {
	inheritedMeta

	mit messageReference
	mot messageReference
}
type methodDesc struct{ m *Method }

func (t methodDesc) Parent() (pref.Descriptor, bool)       { return t.m.parent, true }
func (t methodDesc) Index() int                            { return t.m.index }
func (t methodDesc) Syntax() pref.Syntax                   { return t.m.syntax }
func (t methodDesc) Name() pref.Name                       { return t.m.Name }
func (t methodDesc) FullName() pref.FullName               { return t.m.fullName }
func (t methodDesc) IsPlaceholder() bool                   { return false }
func (t methodDesc) DescriptorProto() (pref.Message, bool) { return nil, false }
func (t methodDesc) Options() interface{}                  { return t.m.Options }
func (t methodDesc) InputType() pref.MessageDescriptor     { return t.m.mit.lazyInit(t, &t.m.InputType) }
func (t methodDesc) OutputType() pref.MessageDescriptor    { return t.m.mot.lazyInit(t, &t.m.OutputType) }
func (t methodDesc) IsStreamingClient() bool               { return t.m.IsStreamingClient }
func (t methodDesc) IsStreamingServer() bool               { return t.m.IsStreamingServer }
func (t methodDesc) Format(s fmt.State, r rune)            { formatDesc(s, r, t) }
func (t methodDesc) ProtoType(pref.MethodDescriptor)       {}
func (t methodDesc) ProtoInternal(pragma.DoNotImplement)   {}

type defaultValue struct {
	once sync.Once
	val  pref.Value
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

func (p *defaultValue) lazyInit(t pref.FieldDescriptor, v pref.Value) pref.Value {
	p.once.Do(func() {
		p.val = v
		if v.IsValid() {
			switch t.Kind() {
			case pref.EnumKind:
				// Treat a string value as an identifier referencing some enum
				// value by name and extract the enum number.
				// If this fails, validateMessage will later detect that the
				// default value for an enum value is the wrong type.
				if s, ok := v.Interface().(string); ok {
					v := t.EnumType().Values().ByName(pref.Name(s))
					if v != nil {
						p.val = pref.ValueOf(v.Number())
					}
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
				if et := t.EnumType(); et != nil {
					if vs := et.Values(); vs.Len() > 0 {
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
	return p.val
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
