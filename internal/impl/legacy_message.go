// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"

	ptag "google.golang.org/protobuf/internal/encoding/tag"
	ptype "google.golang.org/protobuf/internal/prototype"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/prototype"
)

// legacyWrapMessage wraps v as a protoreflect.ProtoMessage,
// where v must be a *struct kind and not implement the v2 API already.
func legacyWrapMessage(v reflect.Value) pref.ProtoMessage {
	mt := legacyLoadMessageInfo(v.Type())
	return mt.MessageOf(v.Interface()).Interface()
}

var legacyMessageTypeCache sync.Map // map[reflect.Type]*MessageInfo

// legacyLoadMessageInfo dynamically loads a *MessageInfo for t,
// where t must be a *struct kind and not implement the v2 API already.
func legacyLoadMessageInfo(t reflect.Type) *MessageInfo {
	// Fast-path: check if a MessageInfo is cached for this concrete type.
	if mt, ok := legacyMessageTypeCache.Load(t); ok {
		return mt.(*MessageInfo)
	}

	// Slow-path: derive message descriptor and initialize MessageInfo.
	md := LegacyLoadMessageDesc(t)
	mt := new(MessageInfo)
	mt.GoType = t
	mt.PBType = &prototype.Message{
		MessageDescriptor: md,
		NewMessage: func() pref.Message {
			return mt.MessageOf(reflect.New(t.Elem()).Interface())
		},
	}
	if mt, ok := legacyMessageTypeCache.LoadOrStore(t, mt); ok {
		return mt.(*MessageInfo)
	}
	return mt
}

var (
	legacyMessageDescLock  sync.Mutex
	legacyMessageDescCache sync.Map // map[reflect.Type]protoreflect.MessageDescriptor
)

// LegacyLoadMessageDesc returns an MessageDescriptor derived from the Go type,
// which must be a *struct kind and not implement the v2 API already.
//
// This is exported for testing purposes.
func LegacyLoadMessageDesc(t reflect.Type) pref.MessageDescriptor {
	return legacyMessageDescSet{}.Load(t)
}

type legacyMessageDescSet struct {
	visited map[reflect.Type]*ptype.StandaloneMessage
	descs   []*ptype.StandaloneMessage
	types   []reflect.Type
}

func (ms legacyMessageDescSet) Load(t reflect.Type) pref.MessageDescriptor {
	// Fast-path: check if a MessageDescriptor is cached for this concrete type.
	if mi, ok := legacyMessageDescCache.Load(t); ok {
		return mi.(pref.MessageDescriptor)
	}

	// Slow-path: initialize MessageDescriptor from the Go type.
	//
	// Hold a global lock during message creation to ensure that each Go type
	// maps to exactly one MessageDescriptor. After obtaining the lock, we must
	// check again whether the message has already been handled.
	legacyMessageDescLock.Lock()
	defer legacyMessageDescLock.Unlock()
	if mi, ok := legacyMessageDescCache.Load(t); ok {
		return mi.(pref.MessageDescriptor)
	}

	// Processing t recursively populates descs and types with all sub-messages.
	// The descriptor for the first type is guaranteed to be at the front.
	ms.processMessage(t)

	// Within a proto file it is possible for cyclic dependencies to exist
	// between multiple message types. When these cases arise, the set of
	// message descriptors must be created together.
	mds, err := ptype.NewMessages(ms.descs)
	if err != nil {
		panic(err)
	}
	for i, md := range mds {
		// Protobuf semantics represents map entries under-the-hood as
		// pseudo-messages (has a descriptor, but no generated Go type).
		// Avoid caching these fake messages.
		if t := ms.types[i]; t.Kind() != reflect.Map {
			legacyMessageDescCache.Store(t, md)
		}
	}
	return mds[0]
}

func (ms *legacyMessageDescSet) processMessage(t reflect.Type) pref.MessageDescriptor {
	// Fast-path: Obtain a placeholder if the message is already processed.
	if m, ok := ms.visited[t]; ok {
		return ptype.PlaceholderMessage(m.FullName)
	}

	// Slow-path: Walk over the struct fields to derive the message descriptor.
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct || t.Elem().PkgPath() == "" {
		panic(fmt.Sprintf("got %v, want named *struct kind", t))
	}

	// Derive name and syntax from the raw descriptor.
	m := new(ptype.StandaloneMessage)
	mv := reflect.New(t.Elem()).Interface()
	if _, ok := mv.(pref.ProtoMessage); ok {
		panic(fmt.Sprintf("%v already implements proto.Message", t))
	}
	if md, ok := mv.(messageV1); ok {
		b, idxs := md.Descriptor()
		fd := legacyLoadFileDesc(b)

		// Derive syntax.
		switch fd.GetSyntax() {
		case "proto2", "":
			m.Syntax = pref.Proto2
		case "proto3":
			m.Syntax = pref.Proto3
		}

		// Derive full name.
		md := fd.MessageType[idxs[0]]
		m.FullName = pref.FullName(fd.GetPackage()).Append(pref.Name(md.GetName()))
		for _, i := range idxs[1:] {
			md = md.NestedType[i]
			m.FullName = m.FullName.Append(pref.Name(md.GetName()))
		}
	} else {
		// If the type does not implement messageV1, then the only way to
		// obtain the full name is through the registry. However, this is
		// unreliable as some generated messages register with a fork of
		// golang/protobuf, so the registry may not have this information.
		m.FullName = legacyDeriveFullName(t.Elem())
		m.Syntax = pref.Proto2

		// Try to determine if the message is using proto3 by checking scalars.
		for i := 0; i < t.Elem().NumField(); i++ {
			f := t.Elem().Field(i)
			if tag := f.Tag.Get("protobuf"); tag != "" {
				switch f.Type.Kind() {
				case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
					m.Syntax = pref.Proto3
				}
				for _, s := range strings.Split(tag, ",") {
					if s == "proto3" {
						m.Syntax = pref.Proto3
					}
				}
			}
		}
	}
	ms.visit(m, t)

	// Obtain a list of oneof wrapper types.
	var oneofWrappers []reflect.Type
	if fn, ok := t.MethodByName("XXX_OneofFuncs"); ok {
		vs := fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[3]
		for _, v := range vs.Interface().([]interface{}) {
			oneofWrappers = append(oneofWrappers, reflect.TypeOf(v))
		}
	}
	if fn, ok := t.MethodByName("XXX_OneofWrappers"); ok {
		vs := fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0]
		for _, v := range vs.Interface().([]interface{}) {
			oneofWrappers = append(oneofWrappers, reflect.TypeOf(v))
		}
	}

	// Obtain a list of the extension ranges.
	if fn, ok := t.MethodByName("ExtensionRangeArray"); ok {
		vs := fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0]
		for i := 0; i < vs.Len(); i++ {
			v := vs.Index(i)
			m.ExtensionRanges = append(m.ExtensionRanges, [2]pref.FieldNumber{
				pref.FieldNumber(v.FieldByName("Start").Int()),
				pref.FieldNumber(v.FieldByName("End").Int() + 1),
			})
		}
	}

	// Derive the message fields by inspecting the struct fields.
	for i := 0; i < t.Elem().NumField(); i++ {
		f := t.Elem().Field(i)
		if tag := f.Tag.Get("protobuf"); tag != "" {
			tagKey := f.Tag.Get("protobuf_key")
			tagVal := f.Tag.Get("protobuf_val")
			m.Fields = append(m.Fields, ms.parseField(tag, tagKey, tagVal, f.Type, m))
		}
		if tag := f.Tag.Get("protobuf_oneof"); tag != "" {
			name := pref.Name(tag)
			m.Oneofs = append(m.Oneofs, ptype.Oneof{Name: name})
			for _, t := range oneofWrappers {
				if t.Implements(f.Type) {
					f := t.Elem().Field(0)
					if tag := f.Tag.Get("protobuf"); tag != "" {
						ft := ms.parseField(tag, "", "", f.Type, m)
						ft.OneofName = name
						m.Fields = append(m.Fields, ft)
					}
				}
			}
		}
	}

	return ptype.PlaceholderMessage(m.FullName)
}

func (ms *legacyMessageDescSet) parseField(tag, tagKey, tagVal string, goType reflect.Type, parent *ptype.StandaloneMessage) ptype.Field {
	t := goType
	isOptional := t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct
	isRepeated := t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
	if isOptional || isRepeated {
		t = t.Elem()
	}
	f := ptag.Unmarshal(tag, t)

	// Populate EnumType and MessageType.
	if f.EnumType == nil && f.Kind == pref.EnumKind {
		if ev, ok := reflect.Zero(t).Interface().(pref.Enum); ok {
			f.EnumType = ev.Descriptor()
		} else {
			f.EnumType = LegacyLoadEnumDesc(t)
		}
	}
	if f.MessageType == nil && (f.Kind == pref.MessageKind || f.Kind == pref.GroupKind) {
		if mv, ok := reflect.Zero(t).Interface().(pref.ProtoMessage); ok {
			f.MessageType = mv.ProtoReflect().Descriptor()
		} else if t.Kind() == reflect.Map {
			m := &ptype.StandaloneMessage{
				Syntax:     parent.Syntax,
				FullName:   parent.FullName.Append(legacyMapEntryName(f.Name)),
				IsMapEntry: true,
				Fields: []ptype.Field{
					ms.parseField(tagKey, "", "", t.Key(), nil),
					ms.parseField(tagVal, "", "", t.Elem(), nil),
				},
			}
			ms.visit(m, t)
			f.MessageType = ptype.PlaceholderMessage(m.FullName)
		} else if mv, ok := legacyMessageDescCache.Load(t); ok {
			f.MessageType = mv.(pref.MessageDescriptor)
		} else {
			f.MessageType = ms.processMessage(t)
		}
	}
	return f
}

func (ms *legacyMessageDescSet) visit(m *ptype.StandaloneMessage, t reflect.Type) {
	if ms.visited == nil {
		ms.visited = make(map[reflect.Type]*ptype.StandaloneMessage)
	}
	if t.Kind() != reflect.Map {
		ms.visited[t] = m
	}
	ms.descs = append(ms.descs, m)
	ms.types = append(ms.types, t)
}

// legacyDeriveFullName derives a fully qualified protobuf name for the given Go type
// The provided name is not guaranteed to be stable nor universally unique.
// It should be sufficiently unique within a program.
func legacyDeriveFullName(t reflect.Type) pref.FullName {
	sanitize := func(r rune) rune {
		switch {
		case r == '/':
			return '.'
		case 'a' <= r && r <= 'z', 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
			return r
		default:
			return '_'
		}
	}
	prefix := strings.Map(sanitize, t.PkgPath())
	suffix := strings.Map(sanitize, t.Name())
	if suffix == "" {
		suffix = fmt.Sprintf("UnknownX%X", reflect.ValueOf(t).Pointer())
	}

	ss := append(strings.Split(prefix, "."), suffix)
	for i, s := range ss {
		if s == "" || ('0' <= s[0] && s[0] <= '9') {
			ss[i] = "x" + s
		}
	}
	return pref.FullName(strings.Join(ss, "."))
}

// legacyMapEntryName derives the message name for a map field of a given name.
// This is identical to MapEntryName from parser.cc in the protoc source.
func legacyMapEntryName(s pref.Name) pref.Name {
	var b []byte
	nextUpper := true
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '_' {
			nextUpper = true
		} else {
			if nextUpper {
				c = byte(unicode.ToUpper(rune(c)))
				nextUpper = false
			}
			b = append(b, c)
		}
	}
	return pref.Name(append(b, "Entry"...))
}
