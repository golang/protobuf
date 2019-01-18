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
	"sort"
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
	filesByPackage filesByPackage
	filesByPath    filesByPath
}

type (
	filesByPackage struct {
		// files is a list of files all in the same package.
		files []protoreflect.FileDescriptor
		// subs is a tree of files all in a sub-package scope.
		// It also maps all top-level identifiers declared in files
		// as the notProtoPackage sentinel value.
		subs map[protoreflect.Name]*filesByPackage // invariant: len(Name) > 0
	}
	filesByPath map[string][]protoreflect.FileDescriptor
)

// notProtoPackage is a sentinel value to indicate that some identifier maps
// to an actual protobuf declaration and is not a sub-package.
var notProtoPackage = new(filesByPackage)

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
	var firstErr error
fileLoop:
	for _, file := range files {
		if file.IsPlaceholder() {
			continue // TODO: Should this be an error instead?
		}

		// Register the file into the filesByPackage tree.
		//
		// The prototype package validates that a FileDescriptor is internally
		// consistent such it does not have conflicts within itself.
		// However, we need to ensure that the inserted file does not conflict
		// with other previously inserted files.
		pkg := file.Package()
		root := &r.filesByPackage
		for len(pkg) > 0 {
			var prefix protoreflect.Name
			prefix, pkg = splitPrefix(pkg)

			// Add a new sub-package segment.
			switch nextRoot := root.subs[prefix]; nextRoot {
			case nil:
				nextRoot = new(filesByPackage)
				if root.subs == nil {
					root.subs = make(map[protoreflect.Name]*filesByPackage)
				}
				root.subs[prefix] = nextRoot
				root = nextRoot
			case notProtoPackage:
				if firstErr == nil {
					name := strings.TrimSuffix(strings.TrimSuffix(string(file.Package()), string(pkg)), ".")
					firstErr = errors.New("file %q has a name conflict over %v", file.Path(), name)
				}
				continue fileLoop
			default:
				root = nextRoot
			}
		}

		// Check for top-level conflicts within the same package.
		// The current file cannot add any top-level declaration that conflict
		// with another top-level declaration or sub-package name.
		var conflicts []protoreflect.Name
		rangeTopLevelDeclarations(file, func(s protoreflect.Name) {
			if root.subs[s] == nil {
				if root.subs == nil {
					root.subs = make(map[protoreflect.Name]*filesByPackage)
				}
				root.subs[s] = notProtoPackage
			} else {
				conflicts = append(conflicts, s)
			}
		})
		if len(conflicts) > 0 {
			// Remove inserted identifiers to make registration failure atomic.
			sort.Slice(conflicts, func(i, j int) bool { return conflicts[i] < conflicts[j] })
			rangeTopLevelDeclarations(file, func(s protoreflect.Name) {
				i := sort.Search(len(conflicts), func(i int) bool { return conflicts[i] >= s })
				if has := i < len(conflicts) && conflicts[i] == s; !has {
					delete(root.subs, s) // remove everything not in conflicts
				}
			})

			if firstErr == nil {
				name := file.Package().Append(conflicts[0])
				firstErr = errors.New("file %q has a name conflict over %v", file.Path(), name)
			}
			continue fileLoop
		}
		root.files = append(root.files, file)

		// Register the file into the filesByPath map.
		//
		// There is no check for conflicts in file path since the path is
		// heavily dependent on how protoc is invoked. When protoc is being
		// invoked by different parties in a distributed manner, it is
		// unreasonable to assume nor ensure that the path is unique.
		if r.filesByPath == nil {
			r.filesByPath = make(filesByPath)
		}
		r.filesByPath[file.Path()] = append(r.filesByPath[file.Path()], file)
	}
	return firstErr
}

// FindDescriptorByName looks up any descriptor (except files) by its full name.
// Files are not handled since multiple file descriptors may belong in
// the same package and have the same full name (see RangeFilesByPackage).
//
// This return (nil, NotFound) if not found.
func (r *Files) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if r == nil {
		return nil, NotFound
	}
	pkg := name
	root := &r.filesByPackage
	for len(pkg) > 0 {
		var prefix protoreflect.Name
		prefix, pkg = splitPrefix(pkg)
		switch nextRoot := root.subs[prefix]; nextRoot {
		case nil:
			return nil, NotFound
		case notProtoPackage:
			// Search current root's package for the descriptor.
			for _, fd := range root.files {
				if d := fd.DescriptorByName(name); d != nil {
					return d, nil
				}
			}
			return nil, NotFound
		default:
			root = nextRoot
		}
	}
	return nil, NotFound
}

// RangeFiles iterates over all registered files.
// The iteration order is undefined.
func (r *Files) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	r.RangeFilesByPackage("", f) // empty package is a prefix for all packages
}

// RangeFilesByPackage iterates over all registered files filtered by
// the given proto package prefix. It iterates over files with an exact package
// match before iterating over files with general prefix match.
// The iteration order is undefined within exact matches or prefix matches.
func (r *Files) RangeFilesByPackage(pkg protoreflect.FullName, f func(protoreflect.FileDescriptor) bool) {
	if r == nil {
		return
	}
	if strings.HasSuffix(string(pkg), ".") {
		return // avoid edge case where splitPrefix allows trailing dot
	}
	root := &r.filesByPackage
	for len(pkg) > 0 && root != nil {
		var prefix protoreflect.Name
		prefix, pkg = splitPrefix(pkg)
		root = root.subs[prefix]
	}
	rangeFiles(root, f)
}
func rangeFiles(fs *filesByPackage, f func(protoreflect.FileDescriptor) bool) bool {
	if fs == nil {
		return true
	}
	// Iterate over exact matches.
	for _, fd := range fs.files { // TODO: iterate non-deterministically
		if !f(fd) {
			return false
		}
	}
	// Iterate over prefix matches.
	for _, fs := range fs.subs {
		if !rangeFiles(fs, f) {
			return false
		}
	}
	return true
}

// RangeFilesByPath iterates over all registered files filtered by
// the given proto path. The iteration order is undefined.
func (r *Files) RangeFilesByPath(path string, f func(protoreflect.FileDescriptor) bool) {
	if r == nil {
		return
	}
	for _, fd := range r.filesByPath[path] { // TODO: iterate non-deterministically
		if !f(fd) {
			return
		}
	}
}

func splitPrefix(name protoreflect.FullName) (protoreflect.Name, protoreflect.FullName) {
	if i := strings.IndexByte(string(name), '.'); i >= 0 {
		return protoreflect.Name(name[:i]), name[i+len("."):]
	}
	return protoreflect.Name(name), ""
}

// rangeTopLevelDeclarations iterates over the name of all top-level
// declarations in the proto file.
func rangeTopLevelDeclarations(fd protoreflect.FileDescriptor, f func(protoreflect.Name)) {
	for i := 0; i < fd.Enums().Len(); i++ {
		e := fd.Enums().Get(i)
		f(e.Name())

		// TODO: Drop ranging over top-level enum values. The current
		// implementation of fileinit.FileBuilder does not initialize the names
		// for enum values in enums. Doing so reduces init time considerably.
		// If we drop this, it means that conflict checks in the registry
		// is not complete. However, this may be okay since the most common
		// reason for a conflict is due to vendored proto files, which are
		// most certainly going to have a name conflict on the parent enum.
		for i := 0; i < e.Values().Len(); i++ {
			f(e.Values().Get(i).Name())
		}
	}
	for i := 0; i < fd.Messages().Len(); i++ {
		f(fd.Messages().Get(i).Name())
	}
	for i := 0; i < fd.Extensions().Len(); i++ {
		f(fd.Extensions().Get(i).Name())
	}
	for i := 0; i < fd.Services().Len(); i++ {
		f(fd.Services().Get(i).Name())
	}
}

// Type is an interface satisfied by protoreflect.EnumType,
// protoreflect.MessageType, or protoreflect.ExtensionType.
type Type interface {
	protoreflect.Descriptor
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
			name := typ.FullName()
			if r.typesByName[name] != nil {
				if firstErr == nil {
					firstErr = errors.New("%v %v is already registered", typeName(typ), name)
				}
				continue typeLoop
			}

			// Check for conflicts in extensionsByMessage.
			if xt, _ := typ.(protoreflect.ExtensionType); xt != nil {
				field := xt.Number()
				message := xt.ExtendedType().FullName()
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
