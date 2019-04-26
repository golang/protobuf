// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	pvalue "google.golang.org/protobuf/internal/value"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

// MessageInfo provides protobuf related functionality for a given Go type
// that represents a message. A given instance of MessageInfo is tied to
// exactly one Go type, which must be a pointer to a struct type.
type MessageInfo struct {
	// GoType is the underlying message Go type and must be populated.
	// Once set, this field must never be mutated.
	GoType reflect.Type // pointer to struct

	// PBType is the underlying message descriptor type and must be populated.
	// Once set, this field must never be mutated.
	PBType pref.MessageType

	initMu   sync.Mutex // protects all unexported fields
	initDone uint32

	// Keep a separate slice of fields for efficient field encoding in tag order
	// and because iterating over a slice is substantially faster than a map.
	fields        map[pref.FieldNumber]*fieldInfo
	fieldsOrdered []*fieldInfo

	oneofs map[pref.Name]*oneofInfo

	getUnknown func(pointer) pref.RawFields
	setUnknown func(pointer, pref.RawFields)

	extensionMap func(pointer) *extensionMap

	unknownFields   func(*messageDataType) pref.UnknownFields
	extensionFields func(*messageDataType) pref.KnownFields
	methods         piface.Methods

	sizecacheOffset       offset
	extensionOffset       offset
	unknownOffset         offset
	extensionFieldInfosMu sync.RWMutex
	extensionFieldInfos   map[pref.ExtensionType]*extensionFieldInfo
}

var prefMessageType = reflect.TypeOf((*pref.Message)(nil)).Elem()

// getMessageInfo returns the MessageInfo (if any) for a type.
//
// We find the MessageInfo by calling the ProtoReflect method on the type's
// zero value and looking at the returned type to see if it is a
// messageReflectWrapper. Note that the MessageInfo may still be uninitialized
// at this point.
func getMessageInfo(mt reflect.Type) (mi *MessageInfo, ok bool) {
	method, ok := mt.MethodByName("ProtoReflect")
	if !ok {
		return nil, false
	}
	if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 || method.Type.Out(0) != prefMessageType {
		return nil, false
	}
	ret := reflect.Zero(mt).Method(method.Index).Call(nil)
	m, ok := ret[0].Elem().Interface().(*messageReflectWrapper)
	if !ok {
		return nil, ok
	}
	return m.mi, true
}

func (mi *MessageInfo) init() {
	// This function is called in the hot path. Inline the sync.Once
	// logic, since allocating a closure for Once.Do is expensive.
	// Keep init small to ensure that it can be inlined.
	if atomic.LoadUint32(&mi.initDone) == 1 {
		return
	}
	mi.initOnce()
}

func (mi *MessageInfo) initOnce() {
	mi.initMu.Lock()
	defer mi.initMu.Unlock()
	if mi.initDone == 1 {
		return
	}

	t := mi.GoType
	if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("got %v, want *struct kind", t))
	}

	si := mi.makeStructInfo(t.Elem())
	mi.makeKnownFieldsFunc(si)
	mi.makeUnknownFieldsFunc(t.Elem())
	mi.makeExtensionFieldsFunc(t.Elem())
	mi.makeMethods(t.Elem())

	atomic.StoreUint32(&mi.initDone, 1)
}

type (
	SizeCache       = int32
	UnknownFields   = []byte
	ExtensionFields = map[int32]ExtensionField
)

var (
	sizecacheType       = reflect.TypeOf(SizeCache(0))
	unknownFieldsType   = reflect.TypeOf(UnknownFields(nil))
	extensionFieldsType = reflect.TypeOf(ExtensionFields(nil))
)

func (mi *MessageInfo) makeMethods(t reflect.Type) {
	mi.sizecacheOffset = invalidOffset
	if fx, _ := t.FieldByName("XXX_sizecache"); fx.Type == sizecacheType {
		mi.sizecacheOffset = offsetOf(fx)
	}
	mi.unknownOffset = invalidOffset
	if fx, _ := t.FieldByName("XXX_unrecognized"); fx.Type == unknownFieldsType {
		mi.unknownOffset = offsetOf(fx)
	}
	mi.extensionOffset = invalidOffset
	if fx, _ := t.FieldByName("XXX_InternalExtensions"); fx.Type == extensionFieldsType {
		mi.extensionOffset = offsetOf(fx)
	} else if fx, _ = t.FieldByName("XXX_extensions"); fx.Type == extensionFieldsType {
		mi.extensionOffset = offsetOf(fx)
	}
	mi.methods.Flags = piface.MethodFlagDeterministicMarshal
	mi.methods.MarshalAppend = mi.marshalAppend
	mi.methods.Size = mi.size
}

type structInfo struct {
	fieldsByNumber        map[pref.FieldNumber]reflect.StructField
	oneofsByName          map[pref.Name]reflect.StructField
	oneofWrappersByType   map[reflect.Type]pref.FieldNumber
	oneofWrappersByNumber map[pref.FieldNumber]reflect.Type
}

func (mi *MessageInfo) makeStructInfo(t reflect.Type) structInfo {
	// Generate a mapping of field numbers and names to Go struct field or type.
	si := structInfo{
		fieldsByNumber:        map[pref.FieldNumber]reflect.StructField{},
		oneofsByName:          map[pref.Name]reflect.StructField{},
		oneofWrappersByType:   map[reflect.Type]pref.FieldNumber{},
		oneofWrappersByNumber: map[pref.FieldNumber]reflect.Type{},
	}
fieldLoop:
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
			if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
				n, _ := strconv.ParseUint(s, 10, 64)
				si.fieldsByNumber[pref.FieldNumber(n)] = f
				continue fieldLoop
			}
		}
		if s := f.Tag.Get("protobuf_oneof"); len(s) > 0 {
			si.oneofsByName[pref.Name(s)] = f
			continue fieldLoop
		}
	}
	var oneofWrappers []interface{}
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofFuncs"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[3].Interface().([]interface{})
	}
	if fn, ok := reflect.PtrTo(t).MethodByName("XXX_OneofWrappers"); ok {
		oneofWrappers = fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0].Interface().([]interface{})
	}
	for _, v := range oneofWrappers {
		tf := reflect.TypeOf(v).Elem()
		f := tf.Field(0)
		for _, s := range strings.Split(f.Tag.Get("protobuf"), ",") {
			if len(s) > 0 && strings.Trim(s, "0123456789") == "" {
				n, _ := strconv.ParseUint(s, 10, 64)
				si.oneofWrappersByType[tf] = pref.FieldNumber(n)
				si.oneofWrappersByNumber[pref.FieldNumber(n)] = tf
				break
			}
		}
	}
	return si
}

// makeKnownFieldsFunc generates functions for operations that can be performed
// on each protobuf message field. It takes in a reflect.Type representing the
// Go struct and matches message fields with struct fields.
//
// This code assumes that the struct is well-formed and panics if there are
// any discrepancies.
func (mi *MessageInfo) makeKnownFieldsFunc(si structInfo) {
	mi.fields = map[pref.FieldNumber]*fieldInfo{}
	mi.fieldsOrdered = make([]*fieldInfo, 0, mi.PBType.Fields().Len())
	for i := 0; i < mi.PBType.Descriptor().Fields().Len(); i++ {
		fd := mi.PBType.Descriptor().Fields().Get(i)
		fs := si.fieldsByNumber[fd.Number()]
		var fi fieldInfo
		switch {
		case fd.ContainingOneof() != nil:
			fi = fieldInfoForOneof(fd, si.oneofsByName[fd.ContainingOneof().Name()], si.oneofWrappersByNumber[fd.Number()])
			// There is one fieldInfo for each proto message field, but only one struct
			// field for all message fields in a oneof. We install the encoder functions
			// on the fieldInfo for the first field in the oneof.
			//
			// A slightly simpler approach would be to have each fieldInfo's encoder
			// handle the case where that field is set, but this would require more
			// checks  against the current oneof type than a single map lookup.
			if fd.ContainingOneof().Fields().Get(0).Name() == fd.Name() {
				fi.funcs = makeOneofFieldCoder(si.oneofsByName[fd.ContainingOneof().Name()], fd.ContainingOneof(), si.fieldsByNumber, si.oneofWrappersByNumber)
			}
		case fd.IsMap():
			fi = fieldInfoForMap(fd, fs)
		case fd.IsList():
			fi = fieldInfoForList(fd, fs)
		case fd.Kind() == pref.MessageKind || fd.Kind() == pref.GroupKind:
			fi = fieldInfoForMessage(fd, fs)
		default:
			fi = fieldInfoForScalar(fd, fs)
		}
		fi.num = fd.Number()
		mi.fields[fd.Number()] = &fi
		mi.fieldsOrdered = append(mi.fieldsOrdered, &fi)
	}
	sort.Slice(mi.fieldsOrdered, func(i, j int) bool {
		return mi.fieldsOrdered[i].num < mi.fieldsOrdered[j].num
	})

	mi.oneofs = map[pref.Name]*oneofInfo{}
	for i := 0; i < mi.PBType.Descriptor().Oneofs().Len(); i++ {
		od := mi.PBType.Descriptor().Oneofs().Get(i)
		mi.oneofs[od.Name()] = makeOneofInfo(od, si.oneofsByName[od.Name()], si.oneofWrappersByType)
	}
}

func (mi *MessageInfo) makeUnknownFieldsFunc(t reflect.Type) {
	mi.unknownFields = makeLegacyUnknownFieldsFunc(t)

	mi.getUnknown = func(pointer) pref.RawFields { return nil }
	mi.setUnknown = func(pointer, pref.RawFields) { return }
	fu, _ := t.FieldByName("XXX_unrecognized")
	if fu.Type == unknownFieldsType {
		fieldOffset := offsetOf(fu)
		mi.getUnknown = func(p pointer) pref.RawFields {
			if p.IsNil() {
				return nil
			}
			rv := p.Apply(fieldOffset).AsValueOf(unknownFieldsType)
			return pref.RawFields(*rv.Interface().(*[]byte))
		}
		mi.setUnknown = func(p pointer, b pref.RawFields) {
			if p.IsNil() {
				panic("invalid SetUnknown on nil Message")
			}
			rv := p.Apply(fieldOffset).AsValueOf(unknownFieldsType)
			*rv.Interface().(*[]byte) = []byte(b)
		}
	} else {
		mi.getUnknown = func(pointer) pref.RawFields {
			return nil
		}
		mi.setUnknown = func(p pointer, _ pref.RawFields) {
			if p.IsNil() {
				panic("invalid SetUnknown on nil Message")
			}
		}
	}
}

func (mi *MessageInfo) makeExtensionFieldsFunc(t reflect.Type) {
	mi.extensionFields = makeLegacyExtensionFieldsFunc(t)

	fx, _ := t.FieldByName("XXX_extensions")
	if fx.Type != extensionFieldsType {
		fx, _ = t.FieldByName("XXX_InternalExtensions")
	}
	if fx.Type == extensionFieldsType {
		fieldOffset := offsetOf(fx)
		mi.extensionMap = func(p pointer) *extensionMap {
			v := p.Apply(fieldOffset).AsValueOf(extensionFieldsType)
			return (*extensionMap)(v.Interface().(*map[int32]ExtensionField))
		}
	} else {
		mi.extensionMap = func(pointer) *extensionMap {
			return (*extensionMap)(nil)
		}
	}
}

func (mi *MessageInfo) MessageOf(p interface{}) pref.Message {
	return (*messageReflectWrapper)(mi.dataTypeOf(p))
}

func (mi *MessageInfo) Methods() *piface.Methods {
	mi.init()
	return &mi.methods
}

func (mi *MessageInfo) dataTypeOf(p interface{}) *messageDataType {
	// TODO: Remove this check? This API is primarily used by generated code,
	// and should not violate this assumption. Leave this check in for now to
	// provide some sanity checks during development. This can be removed if
	// it proves to be detrimental to performance.
	if reflect.TypeOf(p) != mi.GoType {
		panic(fmt.Sprintf("type mismatch: got %T, want %v", p, mi.GoType))
	}
	return &messageDataType{pointerOfIface(p), mi}
}

// messageDataType is a tuple of a pointer to the message data and
// a pointer to the message type.
//
// TODO: Unfortunately, we need to close over a pointer and MessageInfo,
// which incurs an an allocation. This pair is similar to a Go interface,
// which is essentially a tuple of the same thing. We can make this efficient
// with reflect.NamedOf (see https://golang.org/issues/16522).
//
// With that hypothetical API, we could dynamically create a new named type
// that has the same underlying type as MessageInfo.GoType, and
// dynamically create methods that close over MessageInfo.
// Since the new type would have the same underlying type, we could directly
// convert between pointers of those types, giving us an efficient way to swap
// out the method set.
//
// Barring the ability to dynamically create named types, the workaround is
//	1. either to accept the cost of an allocation for this wrapper struct or
//	2. generate more types and methods, at the expense of binary size increase.
type messageDataType struct {
	p  pointer
	mi *MessageInfo
}

type messageReflectWrapper messageDataType

func (m *messageReflectWrapper) Descriptor() pref.MessageDescriptor {
	return m.mi.PBType.Descriptor()
}
func (m *messageReflectWrapper) New() pref.Message {
	return m.mi.PBType.New()
}
func (m *messageReflectWrapper) Interface() pref.ProtoMessage {
	if m, ok := m.ProtoUnwrap().(pref.ProtoMessage); ok {
		return m
	}
	return (*messageIfaceWrapper)(m)
}
func (m *messageReflectWrapper) ProtoUnwrap() interface{} {
	return m.p.AsIfaceOf(m.mi.GoType.Elem())
}

func (m *messageReflectWrapper) Len() (cnt int) {
	m.mi.init()
	for _, fi := range m.mi.fields {
		if fi.has(m.p) {
			cnt++
		}
	}
	return cnt + m.mi.extensionMap(m.p).Len()
}
func (m *messageReflectWrapper) Range(f func(pref.FieldDescriptor, pref.Value) bool) {
	m.mi.init()
	for _, fi := range m.mi.fields {
		if fi.has(m.p) {
			if !f(fi.fieldDesc, fi.get(m.p)) {
				return
			}
		}
	}
	m.mi.extensionMap(m.p).Range(f)
}
func (m *messageReflectWrapper) Has(fd pref.FieldDescriptor) bool {
	if fi, xt := m.checkField(fd); fi != nil {
		return fi.has(m.p)
	} else {
		return m.mi.extensionMap(m.p).Has(xt)
	}
}
func (m *messageReflectWrapper) Clear(fd pref.FieldDescriptor) {
	if fi, xt := m.checkField(fd); fi != nil {
		fi.clear(m.p)
	} else {
		m.mi.extensionMap(m.p).Clear(xt)
	}
}
func (m *messageReflectWrapper) Get(fd pref.FieldDescriptor) pref.Value {
	if fi, xt := m.checkField(fd); fi != nil {
		return fi.get(m.p)
	} else {
		return m.mi.extensionMap(m.p).Get(xt)
	}
}
func (m *messageReflectWrapper) Set(fd pref.FieldDescriptor, v pref.Value) {
	if fi, xt := m.checkField(fd); fi != nil {
		fi.set(m.p, v)
	} else {
		m.mi.extensionMap(m.p).Set(xt, v)
	}
}
func (m *messageReflectWrapper) Mutable(fd pref.FieldDescriptor) pref.Value {
	if fi, xt := m.checkField(fd); fi != nil {
		return fi.mutable(m.p)
	} else {
		return m.mi.extensionMap(m.p).Mutable(xt)
	}
}
func (m *messageReflectWrapper) NewMessage(fd pref.FieldDescriptor) pref.Message {
	if fi, xt := m.checkField(fd); fi != nil {
		return fi.newMessage()
	} else {
		return xt.New().Message()
	}
}
func (m *messageReflectWrapper) WhichOneof(od pref.OneofDescriptor) pref.FieldDescriptor {
	m.mi.init()
	if oi := m.mi.oneofs[od.Name()]; oi != nil && oi.oneofDesc == od {
		return od.Fields().ByNumber(oi.which(m.p))
	}
	panic("invalid oneof descriptor")
}
func (m *messageReflectWrapper) GetUnknown() pref.RawFields {
	m.mi.init()
	return m.mi.getUnknown(m.p)
}
func (m *messageReflectWrapper) SetUnknown(b pref.RawFields) {
	m.mi.init()
	m.mi.setUnknown(m.p, b)
}

// checkField verifies that the provided field descriptor is valid.
// Exactly one of the returned values is populated.
func (m *messageReflectWrapper) checkField(fd pref.FieldDescriptor) (*fieldInfo, pref.ExtensionType) {
	m.mi.init()
	if fi := m.mi.fields[fd.Number()]; fi != nil {
		if fi.fieldDesc != fd {
			panic("mismatching field descriptor")
		}
		return fi, nil
	}
	if fd.IsExtension() {
		if fd.ContainingMessage().FullName() != m.mi.PBType.FullName() {
			// TODO: Should this be exact containing message descriptor match?
			panic("mismatching containing message")
		}
		if !m.mi.PBType.ExtensionRanges().Has(fd.Number()) {
			panic("invalid extension field")
		}
		return nil, fd.(pref.ExtensionType)
	}
	panic("invalid field descriptor")
}

type extensionMap map[int32]ExtensionField

func (m *extensionMap) Len() int {
	if m != nil {
		return len(*m)
	}
	return 0
}
func (m *extensionMap) Range(f func(pref.FieldDescriptor, pref.Value) bool) {
	if m != nil {
		for _, x := range *m {
			xt := x.GetType()
			if !f(xt, xt.ValueOf(x.GetValue())) {
				return
			}
		}
	}
}
func (m *extensionMap) Has(xt pref.ExtensionType) (ok bool) {
	if m != nil {
		_, ok = (*m)[int32(xt.Number())]
	}
	return ok
}
func (m *extensionMap) Clear(xt pref.ExtensionType) {
	delete(*m, int32(xt.Number()))
}
func (m *extensionMap) Get(xt pref.ExtensionType) pref.Value {
	if m != nil {
		if x, ok := (*m)[int32(xt.Number())]; ok {
			return xt.ValueOf(x.GetValue())
		}
	}
	if !isComposite(xt) {
		return defaultValueOf(xt)
	}
	return frozenValueOf(xt.New())
}
func (m *extensionMap) Set(xt pref.ExtensionType, v pref.Value) {
	if *m == nil {
		*m = make(map[int32]ExtensionField)
	}
	var x ExtensionField
	x.SetType(xt)
	x.SetEagerValue(xt.InterfaceOf(v))
	(*m)[int32(xt.Number())] = x
}
func (m *extensionMap) Mutable(xt pref.ExtensionType) pref.Value {
	if !isComposite(xt) {
		panic("invalid Mutable on field with non-composite type")
	}
	if x, ok := (*m)[int32(xt.Number())]; ok {
		return xt.ValueOf(x.GetValue())
	}
	v := xt.New()
	m.Set(xt, v)
	return v
}

func isComposite(fd pref.FieldDescriptor) bool {
	return fd.Kind() == pref.MessageKind || fd.Kind() == pref.GroupKind || fd.IsList() || fd.IsMap()
}

var _ pvalue.Unwrapper = (*messageReflectWrapper)(nil)

type messageIfaceWrapper messageDataType

func (m *messageIfaceWrapper) ProtoReflect() pref.Message {
	return (*messageReflectWrapper)(m)
}
func (m *messageIfaceWrapper) XXX_Methods() *piface.Methods {
	// TODO: Consider not recreating this on every call.
	m.mi.init()
	return &piface.Methods{
		Flags:         piface.MethodFlagDeterministicMarshal,
		MarshalAppend: m.marshalAppend,
		Size:          m.size,
	}
}
func (m *messageIfaceWrapper) ProtoUnwrap() interface{} {
	return m.p.AsIfaceOf(m.mi.GoType.Elem())
}
func (m *messageIfaceWrapper) marshalAppend(b []byte, _ pref.ProtoMessage, opts piface.MarshalOptions) ([]byte, error) {
	return m.mi.marshalAppendPointer(b, m.p, newMarshalOptions(opts))
}
func (m *messageIfaceWrapper) size(msg pref.ProtoMessage) (size int) {
	return m.mi.sizePointer(m.p, 0)
}
