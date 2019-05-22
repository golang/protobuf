// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"unicode/utf8"

	"google.golang.org/protobuf/internal/encoding/wire"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/proto"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type errInvalidUTF8 struct{}

func (errInvalidUTF8) Error() string     { return "string field contains invalid UTF-8" }
func (errInvalidUTF8) InvalidUTF8() bool { return true }

func makeOneofFieldCoder(fs reflect.StructField, od pref.OneofDescriptor, structFields map[pref.FieldNumber]reflect.StructField, otypes map[pref.FieldNumber]reflect.Type) pointerCoderFuncs {
	type oneofFieldInfo struct {
		wiretag uint64
		tagsize int
		funcs   pointerCoderFuncs
	}

	oneofFieldInfos := make(map[reflect.Type]oneofFieldInfo)
	for i, fields := 0, od.Fields(); i < fields.Len(); i++ {
		fd := fields.Get(i)
		ot := otypes[fd.Number()]
		wiretag := wire.EncodeTag(fd.Number(), wireTypes[fd.Kind()])
		oneofFieldInfos[ot] = oneofFieldInfo{
			wiretag: wiretag,
			tagsize: wire.SizeVarint(wiretag),
			funcs:   fieldCoder(fd, ot.Field(0).Type),
		}
	}
	ft := fs.Type
	return pointerCoderFuncs{
		size: func(p pointer, _ int, opts marshalOptions) int {
			v := p.AsValueOf(ft).Elem()
			if v.IsNil() {
				return 0
			}
			v = v.Elem() // interface -> *struct
			telem := v.Elem().Type()
			info, ok := oneofFieldInfos[telem]
			if !ok {
				panic(fmt.Errorf("invalid oneof type %v", telem))
			}
			return info.funcs.size(pointerOfValue(v).Apply(zeroOffset), info.tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			v := p.AsValueOf(ft).Elem()
			if v.IsNil() {
				return b, nil
			}
			v = v.Elem() // interface -> *struct
			telem := v.Elem().Type()
			info, ok := oneofFieldInfos[telem]
			if !ok {
				panic(fmt.Errorf("invalid oneof type %v", telem))
			}
			return info.funcs.marshal(b, pointerOfValue(v).Apply(zeroOffset), info.wiretag, opts)
		},
	}
}

func makeMessageFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if fi, ok := getMessageInfo(ft); ok {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeMessageInfo(p, fi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendMessageInfo(b, p, wiretag, fi, opts)
			},
		}
	} else {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				m := legacyWrapMessage(p.AsValueOf(ft).Elem())
				return sizeMessage(m, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				m := legacyWrapMessage(p.AsValueOf(ft).Elem())
				return appendMessage(b, m, wiretag, opts)
			},
		}
	}
}

func sizeMessageInfo(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	return wire.SizeBytes(mi.sizePointer(p.Elem(), opts)) + tagsize
}

func appendMessageInfo(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(mi.sizePointer(p.Elem(), opts)))
	return mi.marshalAppendPointer(b, p.Elem(), opts)
}

func sizeMessage(m proto.Message, tagsize int, _ marshalOptions) int {
	return wire.SizeBytes(proto.Size(m)) + tagsize
}

func appendMessage(b []byte, m proto.Message, wiretag uint64, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(proto.Size(m)))
	return opts.Options().MarshalAppend(b, m)
}

func sizeMessageIface(ival interface{}, tagsize int, opts marshalOptions) int {
	m := Export{}.MessageOf(ival).Interface()
	return sizeMessage(m, tagsize, opts)
}

func appendMessageIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	m := Export{}.MessageOf(ival).Interface()
	return appendMessage(b, m, wiretag, opts)
}

var coderMessageIface = ifaceCoderFuncs{
	size:    sizeMessageIface,
	marshal: appendMessageIface,
}

func makeGroupFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if fi, ok := getMessageInfo(ft); ok {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeGroupType(p, fi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendGroupType(b, p, wiretag, fi, opts)
			},
		}
	} else {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				m := legacyWrapMessage(p.AsValueOf(ft).Elem())
				return sizeGroup(m, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				m := legacyWrapMessage(p.AsValueOf(ft).Elem())
				return appendGroup(b, m, wiretag, opts)
			},
		}
	}
}

func sizeGroupType(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	return 2*tagsize + mi.sizePointer(p.Elem(), opts)
}

func appendGroupType(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag) // start group
	b, err := mi.marshalAppendPointer(b, p.Elem(), opts)
	b = wire.AppendVarint(b, wiretag+1) // end group
	return b, err
}

func sizeGroup(m proto.Message, tagsize int, _ marshalOptions) int {
	return 2*tagsize + proto.Size(m)
}

func appendGroup(b []byte, m proto.Message, wiretag uint64, opts marshalOptions) ([]byte, error) {
	b = wire.AppendVarint(b, wiretag) // start group
	b, err := opts.Options().MarshalAppend(b, m)
	b = wire.AppendVarint(b, wiretag+1) // end group
	return b, err
}

func sizeGroupIface(ival interface{}, tagsize int, opts marshalOptions) int {
	m := Export{}.MessageOf(ival).Interface()
	return sizeGroup(m, tagsize, opts)
}

func appendGroupIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	m := Export{}.MessageOf(ival).Interface()
	return appendGroup(b, m, wiretag, opts)
}

var coderGroupIface = ifaceCoderFuncs{
	size:    sizeGroupIface,
	marshal: appendGroupIface,
}

func makeMessageSliceFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if fi, ok := getMessageInfo(ft); ok {
		return pointerCoderFuncs{
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendMessageSliceInfo(b, p, wiretag, fi, opts)
			},
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeMessageSliceInfo(p, fi, tagsize, opts)
			},
		}
	}
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeMessageSlice(p, ft, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendMessageSlice(b, p, wiretag, ft, opts)
		},
	}
}

func sizeMessageSliceInfo(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		n += wire.SizeBytes(mi.sizePointer(v, opts)) + tagsize
	}
	return n
}

func appendMessageSliceInfo(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var nerr errors.NonFatal
	var err error
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		siz := mi.sizePointer(v, opts)
		b = wire.AppendVarint(b, uint64(siz))
		b, err = mi.marshalAppendPointer(b, v, opts)
		if !nerr.Merge(err) {
			return b, err
		}
	}
	return b, nerr.E
}

func sizeMessageSlice(p pointer, goType reflect.Type, tagsize int, _ marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		m := Export{}.MessageOf(v.AsValueOf(goType.Elem()).Interface()).Interface()
		n += wire.SizeBytes(proto.Size(m)) + tagsize
	}
	return n
}

func appendMessageSlice(b []byte, p pointer, wiretag uint64, goType reflect.Type, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var nerr errors.NonFatal
	var err error
	for _, v := range s {
		m := Export{}.MessageOf(v.AsValueOf(goType.Elem()).Interface()).Interface()
		b = wire.AppendVarint(b, wiretag)
		siz := proto.Size(m)
		b = wire.AppendVarint(b, uint64(siz))
		b, err = opts.Options().MarshalAppend(b, m)
		if !nerr.Merge(err) {
			return b, err
		}
	}
	return b, nerr.E
}

// Slices of messages

func sizeMessageSliceIface(ival interface{}, tagsize int, opts marshalOptions) int {
	p := pointerOfIface(ival)
	return sizeMessageSlice(p, reflect.TypeOf(ival).Elem().Elem(), tagsize, opts)
}

func appendMessageSliceIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	p := pointerOfIface(ival)
	return appendMessageSlice(b, p, wiretag, reflect.TypeOf(ival).Elem().Elem(), opts)
}

var coderMessageSliceIface = ifaceCoderFuncs{
	size:    sizeMessageSliceIface,
	marshal: appendMessageSliceIface,
}

func makeGroupSliceFieldCoder(fd pref.FieldDescriptor, ft reflect.Type) pointerCoderFuncs {
	if fi, ok := getMessageInfo(ft); ok {
		return pointerCoderFuncs{
			size: func(p pointer, tagsize int, opts marshalOptions) int {
				return sizeGroupSliceInfo(p, fi, tagsize, opts)
			},
			marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
				return appendGroupSliceInfo(b, p, wiretag, fi, opts)
			},
		}
	}
	return pointerCoderFuncs{
		size: func(p pointer, tagsize int, opts marshalOptions) int {
			return sizeGroupSlice(p, ft, tagsize, opts)
		},
		marshal: func(b []byte, p pointer, wiretag uint64, opts marshalOptions) ([]byte, error) {
			return appendGroupSlice(b, p, wiretag, ft, opts)
		},
	}
}

func sizeGroupSlice(p pointer, messageType reflect.Type, tagsize int, _ marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		m := Export{}.MessageOf(v.AsValueOf(messageType.Elem()).Interface()).Interface()
		n += 2*tagsize + proto.Size(m)
	}
	return n
}

func appendGroupSlice(b []byte, p pointer, wiretag uint64, messageType reflect.Type, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var nerr errors.NonFatal
	var err error
	for _, v := range s {
		m := Export{}.MessageOf(v.AsValueOf(messageType.Elem()).Interface()).Interface()
		b = wire.AppendVarint(b, wiretag) // start group
		b, err = opts.Options().MarshalAppend(b, m)
		if !nerr.Merge(err) {
			return b, err
		}
		b = wire.AppendVarint(b, wiretag+1) // end group
	}
	return b, nerr.E
}

func sizeGroupSliceInfo(p pointer, mi *MessageInfo, tagsize int, opts marshalOptions) int {
	s := p.PointerSlice()
	n := 0
	for _, v := range s {
		n += 2*tagsize + mi.sizePointer(v, opts)
	}
	return n
}

func appendGroupSliceInfo(b []byte, p pointer, wiretag uint64, mi *MessageInfo, opts marshalOptions) ([]byte, error) {
	s := p.PointerSlice()
	var nerr errors.NonFatal
	var err error
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag) // start group
		b, err = mi.marshalAppendPointer(b, v, opts)
		if !nerr.Merge(err) {
			return b, err
		}
		b = wire.AppendVarint(b, wiretag+1) // end group
	}
	return b, nerr.E
}

func sizeGroupSliceIface(ival interface{}, tagsize int, opts marshalOptions) int {
	p := pointerOfIface(ival)
	return sizeGroupSlice(p, reflect.TypeOf(ival).Elem().Elem(), tagsize, opts)
}

func appendGroupSliceIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	p := pointerOfIface(ival)
	return appendGroupSlice(b, p, wiretag, reflect.TypeOf(ival).Elem().Elem(), opts)
}

var coderGroupSliceIface = ifaceCoderFuncs{
	size:    sizeGroupSliceIface,
	marshal: appendGroupSliceIface,
}

// Enums

func sizeEnumIface(ival interface{}, tagsize int, _ marshalOptions) (n int) {
	v := reflect.ValueOf(ival).Int()
	return wire.SizeVarint(uint64(v)) + tagsize
}

func appendEnumIface(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := reflect.ValueOf(ival).Int()
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendVarint(b, uint64(v))
	return b, nil
}

var coderEnumIface = ifaceCoderFuncs{
	size:    sizeEnumIface,
	marshal: appendEnumIface,
}

func sizeEnumSliceIface(ival interface{}, tagsize int, opts marshalOptions) (size int) {
	return sizeEnumSliceReflect(reflect.ValueOf(ival).Elem(), tagsize, opts)
}

func sizeEnumSliceReflect(s reflect.Value, tagsize int, _ marshalOptions) (size int) {
	for i, llen := 0, s.Len(); i < llen; i++ {
		size += wire.SizeVarint(uint64(s.Index(i).Int())) + tagsize
	}
	return size
}

func appendEnumSliceIface(b []byte, ival interface{}, wiretag uint64, opts marshalOptions) ([]byte, error) {
	return appendEnumSliceReflect(b, reflect.ValueOf(ival).Elem(), wiretag, opts)
}

func appendEnumSliceReflect(b []byte, s reflect.Value, wiretag uint64, opts marshalOptions) ([]byte, error) {
	for i, llen := 0, s.Len(); i < llen; i++ {
		b = wire.AppendVarint(b, wiretag)
		b = wire.AppendVarint(b, uint64(s.Index(i).Int()))
	}
	return b, nil
}

var coderEnumSliceIface = ifaceCoderFuncs{
	size:    sizeEnumSliceIface,
	marshal: appendEnumSliceIface,
}

// Strings with UTF8 validation.

func appendStringValidateUTF8(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.String()
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendBytes(b, []byte(v))
	if !utf8.ValidString(v) {
		return b, errInvalidUTF8{}
	}
	return b, nil
}

var coderStringValidateUTF8 = pointerCoderFuncs{
	size:    sizeString,
	marshal: appendStringValidateUTF8,
}

func appendStringNoZeroValidateUTF8(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := *p.String()
	if len(v) == 0 {
		return b, nil
	}
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendBytes(b, []byte(v))
	if !utf8.ValidString(v) {
		return b, errInvalidUTF8{}
	}
	return b, nil
}

var coderStringNoZeroValidateUTF8 = pointerCoderFuncs{
	size:    sizeStringNoZero,
	marshal: appendStringNoZeroValidateUTF8,
}

func sizeStringSliceValidateUTF8(p pointer, tagsize int, _ marshalOptions) (size int) {
	s := *p.StringSlice()
	for _, v := range s {
		size += tagsize + wire.SizeBytes(len([]byte(v)))
	}
	return size
}

func appendStringSliceValidateUTF8(b []byte, p pointer, wiretag uint64, _ marshalOptions) ([]byte, error) {
	s := *p.StringSlice()
	var err error
	for _, v := range s {
		b = wire.AppendVarint(b, wiretag)
		b = wire.AppendBytes(b, []byte(v))
		if !utf8.ValidString(v) {
			err = errInvalidUTF8{}
		}
	}
	return b, err
}

var coderStringSliceValidateUTF8 = pointerCoderFuncs{
	size:    sizeStringSliceValidateUTF8,
	marshal: appendStringSliceValidateUTF8,
}

func sizeStringIfaceValidateUTF8(ival interface{}, tagsize int, _ marshalOptions) int {
	v := ival.(string)
	return tagsize + wire.SizeBytes(len([]byte(v)))
}

func appendStringIfaceValidateUTF8(b []byte, ival interface{}, wiretag uint64, _ marshalOptions) ([]byte, error) {
	v := ival.(string)
	b = wire.AppendVarint(b, wiretag)
	b = wire.AppendBytes(b, []byte(v))
	if !utf8.ValidString(v) {
		return b, errInvalidUTF8{}
	}
	return b, nil
}

var coderStringIfaceValidateUTF8 = ifaceCoderFuncs{
	size:    sizeStringIfaceValidateUTF8,
	marshal: appendStringIfaceValidateUTF8,
}
