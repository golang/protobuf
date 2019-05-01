// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protoregistry provides data structures to register and lookup
// protobuf descriptor types.
//
// The Files registry contains file descriptors and provides the ability
// to iterate over the files or lookup a specific descriptor within the files.
// Files only contains protobuf descriptors and has no understanding of Go
// type information that may be associated with each descriptor.
//
// The Types registry contains descriptor types for which there is a known
// Go type associated with that descriptor. It provides the ability to iterate
// over the registered types or lookup a type by name.
package protoregistry

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/v2/internal/errors"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
)

// TODO: Perhaps Register should record the frame of where the function was
// called and surface that in the error? That would help users debug duplicate
// registration issues. This presumes that we provide a way to disable automatic
// registration in generated code.

// GlobalFiles is a global registry of file descriptors.
var GlobalFiles *Files = new(Files)

// GlobalTypes is the registry used by default for type lookups
// unless a local registry is provided by the user.
var GlobalTypes *Types = new(Types)

// NotFound is a sentinel error value to indicate that the type was not found.
var NotFound = errors.New("not found")

// Files is a registry for looking up or iterating over files and the
// descriptors contained within them.
// The Find and Range methods are safe for concurrent use.
type Files struct {
	// The map of descs contains:
	//	EnumDescriptor
	//	MessageDescriptor
	//	ExtensionDescriptor
	//	ServiceDescriptor
	//	*packageDescriptor
	//
	// Note that files are stored as a slice, since a package may contain
	// multiple files.
	descs       map[protoreflect.FullName]interface{}
	filesByPath map[string][]protoreflect.FileDescriptor
}

type packageDescriptor struct {
	files []protoreflect.FileDescriptor
}

// NewFiles returns a registry initialized with the provided set of files.
// If there are duplicates, the first one takes precedence.
func NewFiles(files ...protoreflect.FileDescriptor) *Files {
	// TODO: Should last take precedence? This allows a user to intentionally
	// overwrite an existing registration.
	//
	// The use case is for implementing the existing v1 proto.RegisterFile
	// function where the behavior is last wins. However, it could be argued
	// that the v1 behavior is broken, and we can switch to first wins
	// without violating compatibility.
	r := new(Files)
	r.Register(files...) // ignore errors; first takes precedence
	return r
}

// Register registers the provided list of file descriptors.
// Placeholder files are ignored.
//
// If any descriptor within a file conflicts with the descriptor of any
// previously registered file (e.g., two enums with the same full name),
// then that file is not registered and an error is returned.
//
// It is permitted for multiple files to have the same file path.
func (r *Files) Register(files ...protoreflect.FileDescriptor) error {
	if r.descs == nil {
		r.descs = map[protoreflect.FullName]interface{}{
			"": &packageDescriptor{},
		}
		r.filesByPath = make(map[string][]protoreflect.FileDescriptor)
	}
	var firstErr error
	for _, file := range files {
		if err := r.registerFile(file); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
func (r *Files) registerFile(file protoreflect.FileDescriptor) error {
	for name := file.Package(); name != ""; name = name.Parent() {
		switch r.descs[name].(type) {
		case nil, *packageDescriptor:
		default:
			return errors.New("file %q has a name conflict over %v", file.Path(), name)
		}
	}
	var err error
	rangeRegisteredDescriptors(file, func(desc protoreflect.Descriptor) {
		if r.descs[desc.FullName()] != nil {
			err = errors.New("file %q has a name conflict over %v", file.Path(), desc.FullName())
		}
	})
	if err != nil {
		return err
	}

	path := file.Path()
	r.filesByPath[path] = append(r.filesByPath[path], file)

	for name := file.Package(); name != ""; name = name.Parent() {
		if r.descs[name] == nil {
			r.descs[name] = &packageDescriptor{}
		}
	}
	p := r.descs[file.Package()].(*packageDescriptor)
	p.files = append(p.files, file)
	rangeRegisteredDescriptors(file, func(desc protoreflect.Descriptor) {
		r.descs[desc.FullName()] = desc
	})
	return nil
}

// FindEnumByName looks up an enum by the enum's full name.
//
// This returns (nil, NotFound) if not found.
func (r *Files) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	if r == nil {
		return nil, NotFound
	}
	if d, ok := r.descs[name].(protoreflect.EnumDescriptor); ok {
		return d, nil
	}
	return nil, NotFound
}

// FindMessageByName looks up a message by the message's full name.
//
// This returns (nil, NotFound) if not found.
func (r *Files) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	if r == nil {
		return nil, NotFound
	}
	if d, ok := r.descs[name].(protoreflect.MessageDescriptor); ok {
		return d, nil
	}
	return nil, NotFound
}

// FindExtensionByName looks up an extension field by the field's full name.
// Note that this is the full name of the field as determined by
// where the extension is declared and is unrelated to the full name of the
// message being extended.
//
// This returns (nil, NotFound) if not found.
func (r *Files) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionDescriptor, error) {
	if r == nil {
		return nil, NotFound
	}
	if d, ok := r.descs[name].(protoreflect.ExtensionDescriptor); ok {
		return d, nil
	}
	return nil, NotFound
}

// FindServiceByName looks up a service by the service's full name.
//
// This returns (nil, NotFound) if not found.
func (r *Files) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	if r == nil {
		return nil, NotFound
	}
	if d, ok := r.descs[name].(protoreflect.ServiceDescriptor); ok {
		return d, nil
	}
	return nil, NotFound
}

// RangeFiles iterates over all registered files.
// The iteration order is undefined.
func (r *Files) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	if r == nil {
		return
	}
	for _, d := range r.descs {
		if p, ok := d.(*packageDescriptor); ok {
			for _, file := range p.files {
				if !f(file) {
					return
				}
			}
		}
	}
}

// RangeFilesByPath iterates over all registered files filtered by
// the given proto path. The iteration order is undefined.
func (r *Files) RangeFilesByPath(path string, f func(protoreflect.FileDescriptor) bool) {
	if r == nil {
		return
	}
	for _, file := range r.filesByPath[path] {
		if !f(file) {
			return
		}
	}
}

// RangeFilesByPackage iterates over all registered files in a give proto package.
// The iteration order is undefined.
func (r *Files) RangeFilesByPackage(name protoreflect.FullName, f func(protoreflect.FileDescriptor) bool) {
	if r == nil {
		return
	}
	p, ok := r.descs[name].(*packageDescriptor)
	if !ok {
		return
	}
	for _, file := range p.files {
		if !f(file) {
			return
		}
	}
}

// rangeRegisteredDescriptors iterates over all descriptors in a file which
// will be entered into the registry: enums, messages, extensions, and services.
func rangeRegisteredDescriptors(fd protoreflect.FileDescriptor, f func(protoreflect.Descriptor)) {
	rangeRegisteredMessageDescriptors(fd.Messages(), f)
	for i := 0; i < fd.Enums().Len(); i++ {
		e := fd.Enums().Get(i)
		f(e)
	}
	for i := 0; i < fd.Extensions().Len(); i++ {
		f(fd.Extensions().Get(i))
	}
	for i := 0; i < fd.Services().Len(); i++ {
		f(fd.Services().Get(i))
	}
}
func rangeRegisteredMessageDescriptors(messages protoreflect.MessageDescriptors, f func(protoreflect.Descriptor)) {
	for i := 0; i < messages.Len(); i++ {
		md := messages.Get(i)
		f(md)
		rangeRegisteredMessageDescriptors(md.Messages(), f)
		for i := 0; i < md.Enums().Len(); i++ {
			e := md.Enums().Get(i)
			f(e)
		}
		for i := 0; i < md.Extensions().Len(); i++ {
			f(md.Extensions().Get(i))
		}
	}
}

// Type is an interface satisfied by protoreflect.EnumType,
// protoreflect.MessageType, or protoreflect.ExtensionType.
type Type interface {
	GoType() reflect.Type
}

var (
	_ Type = protoreflect.EnumType(nil)
	_ Type = protoreflect.MessageType(nil)
	_ Type = protoreflect.ExtensionType(nil)
)

// Types is a registry for looking up or iterating over descriptor types.
// The Find and Range methods are safe for concurrent use.
type Types struct {
	// Parent sets the parent registry to consult if a find operation
	// could not locate the appropriate entry.
	//
	// Setting a parent results in each Range operation also iterating over the
	// entries contained within the parent. In such a case, it is possible for
	// Range to emit duplicates (since they may exist in both child and parent).
	// Range iteration is guaranteed to iterate over local entries before
	// iterating over parent entries.
	Parent *Types

	// Resolver sets the local resolver to consult if the local registry does
	// not contain an entry. The resolver takes precedence over the parent.
	//
	// The url is a URL where the full name of the type is the last segment
	// of the path (i.e. string following the last '/' character).
	// When missing a '/' character, the URL is the full name of the type.
	// See documentation on the google.protobuf.Any.type_url field for details.
	//
	// If the resolver returns a result, it is not automatically registered
	// into the local registry. Thus, a resolver function should cache results
	// such that it deterministically returns the same result given the
	// same URL assuming the error returned is nil or NotFound.
	//
	// If the resolver returns the NotFound error, the registry will consult the
	// parent registry if it is set.
	//
	// Setting a resolver has no effect on the result of each Range operation.
	Resolver func(url string) (Type, error)

	// TODO: The syntax of the URL is ill-defined and the protobuf team recently
	// changed the documented semantics in a way that breaks prior usages.
	// I do not believe they can do this and need to sync up with the
	// protobuf team again to hash out what the proper syntax of the URL is.

	// TODO: Should we separate this out as a registry for each type?
	//
	// In Java, the extension and message registry are distinct classes.
	// Their extension registry has knowledge of distinct Java types,
	// while their message registry only contains descriptor information.
	//
	// In Go, we have always registered messages, enums, and extensions.
	// Messages and extensions are registered with Go information, while enums
	// are only registered with descriptor information. We cannot drop Go type
	// information for messages otherwise we would be unable to implement
	// portions of the v1 API such as ptypes.DynamicAny.
	//
	// There is no enum registry in Java. In v1, we used the enum registry
	// because enum types provided no reflective methods. The addition of
	// ProtoReflect removes that need.

	typesByName         typesByName
	extensionsByMessage extensionsByMessage
}

type (
	typesByName         map[protoreflect.FullName]Type
	extensionsByMessage map[protoreflect.FullName]extensionsByNumber
	extensionsByNumber  map[protoreflect.FieldNumber]protoreflect.ExtensionType
)

// NewTypes returns a registry initialized with the provided set of types.
// If there are conflicts, the first one takes precedence.
func NewTypes(typs ...Type) *Types {
	// TODO: Allow setting resolver and parent via constructor?
	r := new(Types)
	r.Register(typs...) // ignore errors; first takes precedence
	return r
}

// Register registers the provided list of descriptor types.
//
// If a registration conflict occurs for enum, message, or extension types
// (e.g., two different types have the same full name),
// then the first type takes precedence and an error is returned.
func (r *Types) Register(typs ...Type) error {
	var firstErr error
typeLoop:
	for _, typ := range typs {
		switch typ.(type) {
		case protoreflect.EnumType, protoreflect.MessageType, protoreflect.ExtensionType:
			// Check for conflicts in typesByName.
			var name protoreflect.FullName
			switch t := typ.(type) {
			case protoreflect.EnumType:
				name = t.Descriptor().FullName()
			case protoreflect.MessageType:
				name = t.Descriptor().FullName()
			case protoreflect.ExtensionType:
				name = t.Descriptor().FullName()
			default:
				panic(fmt.Sprintf("invalid type: %T", t))
			}
			if r.typesByName[name] != nil {
				if firstErr == nil {
					firstErr = errors.New("%v %v is already registered", typeName(typ), name)
				}
				continue typeLoop
			}

			// Check for conflicts in extensionsByMessage.
			if xt, _ := typ.(protoreflect.ExtensionType); xt != nil {
				field := xt.Descriptor().Number()
				message := xt.Descriptor().Extendee().FullName()
				if r.extensionsByMessage[message][field] != nil {
					if firstErr == nil {
						firstErr = errors.New("extension %v is already registered on message %v", name, message)
					}
					continue typeLoop
				}

				// Update extensionsByMessage.
				if r.extensionsByMessage == nil {
					r.extensionsByMessage = make(extensionsByMessage)
				}
				if r.extensionsByMessage[message] == nil {
					r.extensionsByMessage[message] = make(extensionsByNumber)
				}
				r.extensionsByMessage[message][field] = xt
			}

			// Update typesByName.
			if r.typesByName == nil {
				r.typesByName = make(typesByName)
			}
			r.typesByName[name] = typ
		default:
			if firstErr == nil {
				firstErr = errors.New("invalid type: %v", typeName(typ))
			}
		}
	}
	return firstErr
}

// FindEnumByName looks up an enum by its full name.
// E.g., "google.protobuf.Field.Kind".
//
// This returns (nil, NotFound) if not found.
func (r *Types) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	r.globalCheck()
	if r == nil {
		return nil, NotFound
	}
	v, _ := r.typesByName[enum]
	if v == nil && r.Resolver != nil {
		var err error
		v, err = r.Resolver(string(enum))
		if err != nil && err != NotFound {
			return nil, err
		}
	}
	if v != nil {
		if et, _ := v.(protoreflect.EnumType); et != nil {
			return et, nil
		}
		return nil, errors.New("found wrong type: got %v, want enum", typeName(v))
	}
	return r.Parent.FindEnumByName(enum)
}

// FindMessageByName looks up a message by its full name.
// E.g., "google.protobuf.Any"
//
// This return (nil, NotFound) if not found.
func (r *Types) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	// The full name by itself is a valid URL.
	return r.FindMessageByURL(string(message))
}

// FindMessageByURL looks up a message by a URL identifier.
// See Resolver for the format of the URL.
//
// This returns (nil, NotFound) if not found.
func (r *Types) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	r.globalCheck()
	if r == nil {
		return nil, NotFound
	}
	message := protoreflect.FullName(url)
	if i := strings.LastIndexByte(url, '/'); i >= 0 {
		message = message[i+len("/"):]
	}

	v, _ := r.typesByName[message]
	if v == nil && r.Resolver != nil {
		var err error
		v, err = r.Resolver(url)
		if err != nil && err != NotFound {
			return nil, err
		}
	}
	if v != nil {
		if mt, _ := v.(protoreflect.MessageType); mt != nil {
			return mt, nil
		}
		return nil, errors.New("found wrong type: got %v, want message", typeName(v))
	}
	return r.Parent.FindMessageByURL(url)
}

// FindExtensionByName looks up a extension field by the field's full name.
// Note that this is the full name of the field as determined by
// where the extension is declared and is unrelated to the full name of the
// message being extended.
//
// This returns (nil, NotFound) if not found.
func (r *Types) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	r.globalCheck()
	if r == nil {
		return nil, NotFound
	}
	v, _ := r.typesByName[field]
	if v == nil && r.Resolver != nil {
		var err error
		v, err = r.Resolver(string(field))
		if err != nil && err != NotFound {
			return nil, err
		}
	}
	if v != nil {
		if xt, _ := v.(protoreflect.ExtensionType); xt != nil {
			return xt, nil
		}
		return nil, errors.New("found wrong type: got %v, want extension", typeName(v))
	}
	return r.Parent.FindExtensionByName(field)
}

// FindExtensionByNumber looks up a extension field by the field number
// within some parent message, identified by full name.
//
// This returns (nil, NotFound) if not found.
func (r *Types) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	r.globalCheck()
	if r == nil {
		return nil, NotFound
	}
	if xt, ok := r.extensionsByMessage[message][field]; ok {
		return xt, nil
	}
	return r.Parent.FindExtensionByNumber(message, field)
}

// RangeEnums iterates over all registered enums.
// Iteration order is undefined.
func (r *Types) RangeEnums(f func(protoreflect.EnumType) bool) {
	r.globalCheck()
	if r == nil {
		return
	}
	for _, typ := range r.typesByName {
		if et, ok := typ.(protoreflect.EnumType); ok {
			if !f(et) {
				return
			}
		}
	}
	r.Parent.RangeEnums(f)
}

// RangeMessages iterates over all registered messages.
// Iteration order is undefined.
func (r *Types) RangeMessages(f func(protoreflect.MessageType) bool) {
	r.globalCheck()
	if r == nil {
		return
	}
	for _, typ := range r.typesByName {
		if mt, ok := typ.(protoreflect.MessageType); ok {
			if !f(mt) {
				return
			}
		}
	}
	r.Parent.RangeMessages(f)
}

// RangeExtensions iterates over all registered extensions.
// Iteration order is undefined.
func (r *Types) RangeExtensions(f func(protoreflect.ExtensionType) bool) {
	r.globalCheck()
	if r == nil {
		return
	}
	for _, typ := range r.typesByName {
		if xt, ok := typ.(protoreflect.ExtensionType); ok {
			if !f(xt) {
				return
			}
		}
	}
	r.Parent.RangeExtensions(f)
}

// RangeExtensionsByMessage iterates over all registered extensions filtered
// by a given message type. Iteration order is undefined.
func (r *Types) RangeExtensionsByMessage(message protoreflect.FullName, f func(protoreflect.ExtensionType) bool) {
	r.globalCheck()
	if r == nil {
		return
	}
	for _, xt := range r.extensionsByMessage[message] {
		if !f(xt) {
			return
		}
	}
	r.Parent.RangeExtensionsByMessage(message, f)
}

func (r *Types) globalCheck() {
	if r == GlobalTypes && (r.Parent != nil || r.Resolver != nil) {
		panic("GlobalTypes.Parent and GlobalTypes.Resolver cannot be set")
	}
}

func typeName(t Type) string {
	switch t.(type) {
	case protoreflect.EnumType:
		return "enum"
	case protoreflect.MessageType:
		return "message"
	case protoreflect.ExtensionType:
		return "extension"
	default:
		return fmt.Sprintf("%T", t)
	}
}
