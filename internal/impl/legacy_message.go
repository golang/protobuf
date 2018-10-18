// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/golang/protobuf/v2/internal/encoding/text"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	ptype "github.com/golang/protobuf/v2/reflect/prototype"
)

var messageDescCache sync.Map // map[reflect.Type]protoreflect.MessageDescriptor

// loadMessageDesc returns an MessageDescriptor derived from the Go type,
// which must be an *struct kind and not implement the v2 API already.
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
	if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("got %v, want *struct kind", t))
	}

	// Derive name and syntax from the raw descriptor.
	m := new(ptype.StandaloneMessage)
	mv := reflect.New(t.Elem()).Interface()
	if _, ok := mv.(pref.ProtoMessage); ok {
		panic(fmt.Sprintf("%v already implements proto.Message", t))
	}
	if md, ok := mv.(legacyMessage); ok {
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
		// If the type does not implement legacyMessage, then the only way to
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

func (ms *messageDescSet) parseField(tag, tagKey, tagVal string, t reflect.Type, parent *ptype.StandaloneMessage) (f ptype.Field) {
	isOptional := t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct
	isRepeated := t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
	if isOptional || isRepeated {
		t = t.Elem()
	}

	for len(tag) > 0 {
		i := strings.IndexByte(tag, ',')
		if i < 0 {
			i = len(tag)
		}
		switch s := tag[:i]; {
		case strings.HasPrefix(s, "name="):
			f.Name = pref.Name(s[len("name="):])
		case strings.Trim(s, "0123456789") == "":
			n, _ := strconv.ParseUint(s, 10, 32)
			f.Number = pref.FieldNumber(n)
		case s == "opt":
			f.Cardinality = pref.Optional
		case s == "req":
			f.Cardinality = pref.Required
		case s == "rep":
			f.Cardinality = pref.Repeated
		case s == "varint":
			switch t.Kind() {
			case reflect.Bool:
				f.Kind = pref.BoolKind
			case reflect.Int32:
				f.Kind = pref.Int32Kind
			case reflect.Int64:
				f.Kind = pref.Int64Kind
			case reflect.Uint32:
				f.Kind = pref.Uint32Kind
			case reflect.Uint64:
				f.Kind = pref.Uint64Kind
			}
		case s == "zigzag32":
			if t.Kind() == reflect.Int32 {
				f.Kind = pref.Sint32Kind
			}
		case s == "zigzag64":
			if t.Kind() == reflect.Int64 {
				f.Kind = pref.Sint64Kind
			}
		case s == "fixed32":
			switch t.Kind() {
			case reflect.Int32:
				f.Kind = pref.Sfixed32Kind
			case reflect.Uint32:
				f.Kind = pref.Fixed32Kind
			case reflect.Float32:
				f.Kind = pref.FloatKind
			}
		case s == "fixed64":
			switch t.Kind() {
			case reflect.Int64:
				f.Kind = pref.Sfixed64Kind
			case reflect.Uint64:
				f.Kind = pref.Fixed64Kind
			case reflect.Float64:
				f.Kind = pref.DoubleKind
			}
		case s == "bytes":
			switch {
			case t.Kind() == reflect.String:
				f.Kind = pref.StringKind
			case t.Kind() == reflect.Slice && t.Elem() == byteType:
				f.Kind = pref.BytesKind
			default:
				f.Kind = pref.MessageKind
			}
		case s == "group":
			f.Kind = pref.GroupKind
		case strings.HasPrefix(s, "enum="):
			f.Kind = pref.EnumKind
		case strings.HasPrefix(s, "json="):
			f.JSONName = s[len("json="):]
		case s == "packed":
			f.IsPacked = true
		case strings.HasPrefix(s, "weak="):
			f.IsWeak = true
			f.MessageType = ptype.PlaceholderMessage(pref.FullName(s[len("weak="):]))
		case strings.HasPrefix(s, "def="):
			// The default tag is special in that everything afterwards is the
			// default regardless of the presence of commas.
			s, i = tag[len("def="):], len(tag)

			// Defaults are parsed last, so Kind is populated.
			switch f.Kind {
			case pref.BoolKind:
				switch s {
				case "1":
					f.Default = pref.ValueOf(true)
				case "0":
					f.Default = pref.ValueOf(false)
				}
			case pref.EnumKind:
				n, _ := strconv.ParseInt(s, 10, 32)
				f.Default = pref.ValueOf(pref.EnumNumber(n))
			case pref.Int32Kind, pref.Sint32Kind, pref.Sfixed32Kind:
				n, _ := strconv.ParseInt(s, 10, 32)
				f.Default = pref.ValueOf(int32(n))
			case pref.Int64Kind, pref.Sint64Kind, pref.Sfixed64Kind:
				n, _ := strconv.ParseInt(s, 10, 64)
				f.Default = pref.ValueOf(int64(n))
			case pref.Uint32Kind, pref.Fixed32Kind:
				n, _ := strconv.ParseUint(s, 10, 32)
				f.Default = pref.ValueOf(uint32(n))
			case pref.Uint64Kind, pref.Fixed64Kind:
				n, _ := strconv.ParseUint(s, 10, 64)
				f.Default = pref.ValueOf(uint64(n))
			case pref.FloatKind, pref.DoubleKind:
				n, _ := strconv.ParseFloat(s, 64)
				switch s {
				case "nan":
					n = math.NaN()
				case "inf":
					n = math.Inf(+1)
				case "-inf":
					n = math.Inf(-1)
				}
				if f.Kind == pref.FloatKind {
					f.Default = pref.ValueOf(float32(n))
				} else {
					f.Default = pref.ValueOf(float64(n))
				}
			case pref.StringKind:
				f.Default = pref.ValueOf(string(s))
			case pref.BytesKind:
				// The default value is in escaped form (C-style).
				// TODO: Export unmarshalString in the text package to avoid this hack.
				v, err := text.Unmarshal([]byte(`["` + s + `"]:0`))
				if err == nil && len(v.Message()) == 1 {
					s := v.Message()[0][0].String()
					f.Default = pref.ValueOf([]byte(s))
				}
			}
		}
		tag = strings.TrimPrefix(tag[i:], ",")
	}

	// The generator uses the group message name instead of the field name.
	// We obtain the real field name by lowercasing the group name.
	if f.Kind == pref.GroupKind {
		f.Name = pref.Name(strings.ToLower(string(f.Name)))
	}

	// Populate EnumType and MessageType.
	if f.EnumType == nil && f.Kind == pref.EnumKind {
		if ev, ok := reflect.Zero(t).Interface().(pref.ProtoEnum); ok {
			f.EnumType = ev.ProtoReflect().Type()
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
