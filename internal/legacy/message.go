// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package legacy

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"

	ptag "github.com/golang/protobuf/v2/internal/encoding/tag"
	pimpl "github.com/golang/protobuf/v2/internal/impl"
	scalar "github.com/golang/protobuf/v2/internal/scalar"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"

	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

// wrapMessage wraps v as a protoreflect.ProtoMessage,
// where v must be a *struct kind and not implement the v2 API already.
func wrapMessage(v reflect.Value) pref.ProtoMessage {
	mt := loadMessageType(v.Type())
	return mt.MessageOf(v.Interface()).Interface()
}

var messageTypeCache sync.Map // map[reflect.Type]*MessageType

// loadMessageType dynamically loads a *MessageType for t,
// where t must be a *struct kind and not implement the v2 API already.
func loadMessageType(t reflect.Type) *pimpl.MessageType {
	// Fast-path: check if a MessageType is cached for this concrete type.
	if mt, ok := messageTypeCache.Load(t); ok {
		return mt.(*pimpl.MessageType)
	}

	// Slow-path: derive message descriptor and initialize MessageType.
	md := loadMessageDesc(t)
	mt := new(pimpl.MessageType)
	mt.GoType = t
	mt.PBType = ptype.GoMessage(md, func(pref.MessageType) pref.Message {
		p := reflect.New(t.Elem()).Interface()
		return mt.MessageOf(p)
	})
	messageTypeCache.Store(t, mt)
	return mt
}

var messageDescCache sync.Map // map[reflect.Type]protoreflect.MessageDescriptor

// loadMessageDesc returns an MessageDescriptor derived from the Go type,
// which must be a *struct kind and not implement the v2 API already.
func loadMessageDesc(t reflect.Type) pref.MessageDescriptor {
	return messageDescSet{}.Load(t)
}

type messageDescSet struct {
	visited map[reflect.Type]*ptype.StandaloneMessage
	descs   []*ptype.StandaloneMessage
	types   []reflect.Type
}

func (ms messageDescSet) Load(t reflect.Type) pref.MessageDescriptor {
	// Fast-path: check if a MessageDescriptor is cached for this concrete type.
	if mi, ok := messageDescCache.Load(t); ok {
		return mi.(pref.MessageDescriptor)
	}

	// Slow-path: initialize MessageDescriptor from the Go type.

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
			messageDescCache.Store(t, md)
		}
	}
	return mds[0]
}

func (ms *messageDescSet) processMessage(t reflect.Type) pref.MessageDescriptor {
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
		fd := loadFileDesc(b)

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
		m.FullName = deriveFullName(t.Elem())
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

func (ms *messageDescSet) parseField(tag, tagKey, tagVal string, goType reflect.Type, parent *ptype.StandaloneMessage) ptype.Field {
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
			f.EnumType = ev.Type()
		} else {
			f.EnumType = loadEnumDesc(t)
		}
	}
	if f.MessageType == nil && (f.Kind == pref.MessageKind || f.Kind == pref.GroupKind) {
		if mv, ok := reflect.Zero(t).Interface().(pref.ProtoMessage); ok {
			f.MessageType = mv.ProtoReflect().Type()
		} else if t.Kind() == reflect.Map {
			m := &ptype.StandaloneMessage{
				Syntax:     parent.Syntax,
				FullName:   parent.FullName.Append(mapEntryName(f.Name)),
				Options:    &descriptorpb.MessageOptions{MapEntry: scalar.Bool(true)},
				IsMapEntry: true,
				Fields: []ptype.Field{
					ms.parseField(tagKey, "", "", t.Key(), nil),
					ms.parseField(tagVal, "", "", t.Elem(), nil),
				},
			}
			ms.visit(m, t)
			f.MessageType = ptype.PlaceholderMessage(m.FullName)
		} else if mv, ok := messageDescCache.Load(t); ok {
			f.MessageType = mv.(pref.MessageDescriptor)
		} else {
			f.MessageType = ms.processMessage(t)
		}
	}
	return f
}

func (ms *messageDescSet) visit(m *ptype.StandaloneMessage, t reflect.Type) {
	if ms.visited == nil {
		ms.visited = make(map[reflect.Type]*ptype.StandaloneMessage)
	}
	if t.Kind() != reflect.Map {
		ms.visited[t] = m
	}
	ms.descs = append(ms.descs, m)
	ms.types = append(ms.types, t)
}

// deriveFullName derives a fully qualified protobuf name for the given Go type
// The provided name is not guaranteed to be stable nor universally unique.
// It should be sufficiently unique within a program.
func deriveFullName(t reflect.Type) pref.FullName {
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

// mapEntryName derives the message name for a map field of a given name.
// This is identical to MapEntryName from parser.cc in the protoc source.
func mapEntryName(s pref.Name) pref.Name {
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
