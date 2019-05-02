// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	protoV2 "github.com/golang/protobuf/v2/proto"
	"github.com/golang/protobuf/v2/reflect/protodesc"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/protoregistry"
	"github.com/golang/protobuf/v2/runtime/protoimpl"
	"github.com/golang/protobuf/v2/runtime/protolegacy"

	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

// filePath is the path to the proto source file.
type filePath = string // e.g., "google/protobuf/descriptor.proto"

// fileDescGZIP is the compressed contents of the encoded FileDescriptorProto.
type fileDescGZIP = []byte

var fileCache sync.Map // map[filePath]fileDescGZIP

// RegisterFile is called from generated code and registers the compressed
// FileDescriptorProto with the file path for a proto source file.
//
// Deprecated: Use protoregistry.GlobalFiles.Register instead.
func RegisterFile(s filePath, d fileDescGZIP) {
	// Decompress the descriptor.
	zr, err := gzip.NewReader(bytes.NewReader(d))
	if err != nil {
		panic(fmt.Sprintf("proto: invalid compressed file descriptor: %v", err))
	}
	b, err := ioutil.ReadAll(zr)
	if err != nil {
		panic(fmt.Sprintf("proto: invalid compressed file descriptor: %v", err))
	}

	// Parse the raw descriptor proto.
	var pb descriptorpb.FileDescriptorProto
	if err := protoV2.Unmarshal(b, &pb); err != nil {
		panic(fmt.Sprintf("proto: unmarshal failure: %v", err))
	}

	// Convert the raw descriptor to a structured file descriptor.
	fd, err := protodesc.NewFile(&pb, nil)
	if err != nil {
		// TODO: Ignore errors due to placeholders.
		panic(fmt.Sprintf("proto: descriptor parsing failure: %v", err))
	}

	// Register the descriptor in the v2 registry and cache the result locally.
	if err := protoregistry.GlobalFiles.Register(fd); err != nil {
		printWarning(err)
		return
	}
	fileCache.Store(s, b)
}

// FileDescriptor returns the compressed FileDescriptorProto given the file path
// for a proto source file. It returns nil if not found.
//
// Deprecated: Use protoregistry.GlobalFiles.RangeFilesByPath instead.
func FileDescriptor(s filePath) (d fileDescGZIP) {
	if d, ok := fileCache.Load(s); ok {
		return d.(fileDescGZIP)
	}

	// Find the descriptor in the v2 registry.
	var n int
	protoregistry.GlobalFiles.RangeFilesByPath(s, func(fd pref.FileDescriptor) bool {
		n++

		// Convert the structured file descriptor to the raw descriptor proto.
		pb := protodesc.ToFileDescriptorProto(fd)
		b, err := protoV2.Marshal(pb)
		if err != nil {
			panic(fmt.Sprintf("proto: marshal failure: %v", err))
		}
		bb := new(bytes.Buffer)
		zw := gzip.NewWriter(bb)
		if _, err := zw.Write(b); err != nil {
			panic(fmt.Sprintf("proto: compression failure: %v", err))
		}
		if err := zw.Close(); err != nil {
			panic(fmt.Sprintf("proto: compression failure: %v", err))
		}
		d = bb.Bytes()
		return true
	})
	if n > 1 {
		return d // best-effort; may be non-deterministic
	}

	// Locally cache the raw descriptor form for the file.
	if len(d) > 0 {
		fileCache.Store(s, d)
	}
	return d
}

// enumName is the name of an enum. For historical reasons, the enum name is
// neither the full Go name nor the full protobuf name of the enum.
// The name is the dot-separated combination of just the proto package that the
// enum is declared within followed by the Go type name of the generated enum.
type enumName = string // e.g., "my.proto.package.GoMessage_GoEnum"

// enumsByName maps enum values by name to their numeric counterpart.
type enumsByName = map[string]int32

// enumsByNumber maps enum values by number to their name counterpart.
type enumsByNumber = map[int32]string

var enumCache sync.Map // map[enumName]enumsByName

// RegisterEnum is called from the generated code and registers the mapping of
// enum value names to enum numbers for the enum identified by s.
//
// Deprecated: Use protoregistry.GlobalTypes.Register instead.
func RegisterEnum(s enumName, _ enumsByNumber, m enumsByName) {
	if _, ok := enumCache.Load(s); ok {
		panic("proto: duplicate enum registered: " + s)
	}
	enumCache.Store(s, m)

	// This does not forward registration to the v2 registry since this API
	// lacks sufficient information to construct a complete v2 enum descriptor.
}

// EnumValueMap returns the mapping from enum value names to enum numbers for
// the enum of the given name. It returns nil if not found.
//
// Deprecated: Use protoregistry.GlobalTypes.FindEnumByName instead.
func EnumValueMap(s enumName) (m enumsByName) {
	v, ok := enumCache.Load(s)
	if ok {
		return v.(enumsByName)
	}

	// Construct the mapping from a v2 enum descriptor.
	var protoPkg pref.FullName
	if i := strings.LastIndexByte(s, '.'); i >= 0 {
		protoPkg = pref.FullName(s[:i])
	}
	protoregistry.GlobalFiles.RangeFilesByPackage(pref.FullName(protoPkg), func(fd pref.FileDescriptor) bool {
		return walkEnums(fd, func(ed pref.EnumDescriptor) bool {
			if s == hybridEnumName(ed) {
				m = make(enumsByName)
				evs := ed.Values()
				for i := evs.Len() - 1; i >= 0; i-- {
					ev := evs.Get(i)
					m[string(ev.Name())] = int32(ev.Number())
				}
				return false
			}
			return true
		})
	})

	if m != nil {
		enumCache.Store(s, m)
	}
	return m
}

// walkEnums recursively walks all enums declared in d.
func walkEnums(d interface {
	Enums() pref.EnumDescriptors
	Messages() pref.MessageDescriptors
}, f func(pref.EnumDescriptor) bool) bool {
	cont := true
	eds := d.Enums()
	for i := eds.Len() - 1; cont && i >= 0; i-- {
		cont = cont && f(eds.Get(i))
	}
	mds := d.Messages()
	for i := mds.Len() - 1; cont && i >= 0; i-- {
		cont = cont && walkEnums(mds.Get(i), f)
	}
	return cont
}

// hybridEnumName returns the legacy enum identifier.
func hybridEnumName(ed pref.EnumDescriptor) enumName {
	var protoPkg string
	for parent, _ := ed.Parent(); parent != nil; parent, _ = parent.Parent() {
		if fd, ok := parent.(pref.FileDescriptor); ok {
			protoPkg = string(fd.Package())
			break
		}
	}
	if protoPkg == "" {
		return camelCase(string(ed.FullName()))
	}
	return protoPkg + "." + camelCase(strings.TrimPrefix(string(ed.FullName()), protoPkg+"."))
}

// camelCase is a copy of the v2 protogen.camelCase function.
func camelCase(s string) string {
	isASCIILower := func(c byte) bool {
		return 'a' <= c && c <= 'z'
	}
	isASCIIDigit := func(c byte) bool {
		return '0' <= c && c <= '9'
	}

	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '.' && i+1 < len(s) && isASCIILower(s[i+1]):
			continue
		case c == '.':
			b = append(b, '_')
		case c == '_' && (i == 0 || s[i-1] == '.'):
			b = append(b, 'X')
		case c == '_' && i+1 < len(s) && isASCIILower(s[i+1]):
			continue
		case isASCIIDigit(c):
			b = append(b, c)
		default:
			if isASCIILower(c) {
				c -= 'a' - 'A'
			}
			b = append(b, c)
			for ; i+1 < len(s) && isASCIILower(s[i+1]); i++ {
				b = append(b, s[i+1])
			}
		}
	}
	return string(b)
}

// messageName is the full name of protobuf message.
type messageName = string

var messageTypeCache sync.Map // map[messageName]reflect.Type

// RegisterType is called from generated code and register the message Go type
// for a message of the given name.
//
// Deprecated: Use protoregistry.GlobalTypes.Register instead.
func RegisterType(m Message, s messageName) {
	mt := protoimpl.X.MessageTypeOf(m)
	if s != messageName(mt.Descriptor().FullName()) {
		panic(fmt.Sprintf("proto: inconsistent message name: got %v, want %v", s, mt.Descriptor().FullName()))
	}
	if err := protoregistry.GlobalTypes.Register(mt); err != nil {
		printWarning(err)
		return
	}
	messageTypeCache.Store(s, reflect.TypeOf(m))
}

// RegisterMapType is called from generated code and registers the Go map type
// for a protobuf message representing a map entry.
//
// Deprecated: Do not use.
func RegisterMapType(m interface{}, s messageName) {
	t := reflect.TypeOf(m)
	if t.Kind() != reflect.Map {
		panic(fmt.Sprintf("invalid map kind: %v", t))
	}
	if _, ok := messageTypeCache.Load(s); ok {
		printWarning(fmt.Errorf("proto: duplicate proto message registered: %s", s))
		return
	}
	messageTypeCache.Store(s, t)
}

// MessageType returns the message type for a named message.
// It returns nil if not found.
//
// Deprecated: Use protoregistry.GlobalTypes.FindMessageByName instead.
func MessageType(s messageName) reflect.Type {
	if t, ok := messageTypeCache.Load(s); ok {
		return t.(reflect.Type)
	}

	// Derive the message type from the v2 registry.
	var t reflect.Type
	mt, _ := protoregistry.GlobalTypes.FindMessageByName(pref.FullName(s))
	if mt != nil {
		t = mt.GoType()
	}
	// TODO: Support retrieving Go map types for map entry messages?

	if t != nil {
		messageTypeCache.Store(s, t)
	}
	return t
}

// MessageName returns the full protobuf name for the given message type.
//
// Deprecated: Use pref.MessageDescriptor.FullName instead.
func MessageName(m Message) messageName {
	if m, ok := m.(interface {
		XXX_MessageName() messageName
	}); ok {
		return m.XXX_MessageName()
	}
	return messageName(protoimpl.X.MessageDescriptorOf(m).FullName())
}

// RegisterExtension is called from the generated code and registers
// the extension descriptor.
//
// Deprecated: Use protoregistry.GlobalTypes.Register instead.
func RegisterExtension(d *ExtensionDesc) {
	xt := protolegacy.X.ExtensionTypeFromDesc(d)
	if err := protoregistry.GlobalTypes.Register(xt); err != nil {
		panic(err)
	}
}

type extensionsByNumber = map[int32]*ExtensionDesc

var extensionCache sync.Map // map[messageName]extensionsByNumber

// RegisteredExtensions returns a map of the registered extensions for the
// provided protobuf message, indexed by the extension field number.
//
// Deprecated: Use protoregistry.GlobalTypes.RangeExtensionsByMessage instead.
func RegisteredExtensions(m Message) extensionsByNumber {
	s := MessageName(m)
	if xs, ok := extensionCache.Load(s); ok {
		return xs.(extensionsByNumber)
	}

	var xs extensionsByNumber
	protoregistry.GlobalTypes.RangeExtensionsByMessage(pref.FullName(s), func(xt pref.ExtensionType) bool {
		if xs == nil {
			xs = make(extensionsByNumber)
		}
		xs[int32(xt.Descriptor().Number())] = protolegacy.X.ExtensionDescFromType(xt)
		return true
	})

	if xs == nil {
		return nil
	}
	if xs, ok := extensionCache.LoadOrStore(s, xs); ok {
		return xs.(extensionsByNumber)
	}
	return xs
}

// printWarning prints a warning to os.Stderr regarding a registration conflict.
func printWarning(err error) {
	// TODO: Provide a link in the warning to a page that explains
	// what the user should do instead?
	b := make([]byte, 0, 1<<12)
	b = append(b, "==================\n"...)
	b = append(b, "WARNING: "+err.Error()+"\n"...)
	b = append(b, "A future release of proto will panic on registration conflicts.\n\n"...)
	b = b[:len(b)+runtime.Stack(b[len(b):cap(b)], false)]
	b = append(b, "==================\n"...)
	os.Stderr.Write(b)
}
