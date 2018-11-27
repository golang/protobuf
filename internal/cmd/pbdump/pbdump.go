// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// pbdump is a tool for decoding the wire format for protocol buffer messages.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/protobuf/v2/internal/encoding/pack"
	"github.com/golang/protobuf/v2/internal/encoding/wire"
	"github.com/golang/protobuf/v2/internal/scalar"
	"github.com/golang/protobuf/v2/reflect/protoreflect"
	"github.com/golang/protobuf/v2/reflect/prototype"

	descriptorpb "github.com/golang/protobuf/v2/types/descriptor"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	var fs fields
	var flagUsages []string
	flagVar := func(value flag.Value, name, usage string) {
		flagUsages = append(flagUsages, fmt.Sprintf("  -%-16v  %v", name+" "+value.String(), usage))
		flag.Var(value, name, usage)
	}
	flagBool := func(name, usage string) *bool {
		flagUsages = append(flagUsages, fmt.Sprintf("  -%-16v  %v", name, usage))
		return flag.Bool(name, false, usage)
	}
	flagVar(fieldsFlag{&fs, protoreflect.BoolKind}, "bools", "List of bool fields")
	flagVar(fieldsFlag{&fs, protoreflect.Int64Kind}, "ints", "List of int32 or int64 fields")
	flagVar(fieldsFlag{&fs, protoreflect.Sint64Kind}, "sints", "List of sint32 or sint64 fields")
	flagVar(fieldsFlag{&fs, protoreflect.Uint64Kind}, "uints", "List of enum, uint32, or uint64 fields")
	flagVar(fieldsFlag{&fs, protoreflect.Fixed32Kind}, "uint32s", "List of fixed32 fields")
	flagVar(fieldsFlag{&fs, protoreflect.Sfixed32Kind}, "int32s", "List of sfixed32 fields")
	flagVar(fieldsFlag{&fs, protoreflect.FloatKind}, "float32s", "List of float fields")
	flagVar(fieldsFlag{&fs, protoreflect.Fixed64Kind}, "uint64s", "List of fixed64 fields")
	flagVar(fieldsFlag{&fs, protoreflect.Sfixed64Kind}, "int64s", "List of sfixed64 fields")
	flagVar(fieldsFlag{&fs, protoreflect.DoubleKind}, "float64s", "List of double fields")
	flagVar(fieldsFlag{&fs, protoreflect.StringKind}, "strings", "List of string fields")
	flagVar(fieldsFlag{&fs, protoreflect.BytesKind}, "bytes", "List of bytes fields")
	flagVar(fieldsFlag{&fs, protoreflect.MessageKind}, "messages", "List of message fields")
	flagVar(fieldsFlag{&fs, protoreflect.GroupKind}, "groups", "List of group fields")
	printDesc := flagBool("print_descriptor", "Print the message descriptor")
	printSource := flagBool("print_source", "Print the output in valid Go syntax")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS]... [INPUTS]...\n\n%s\n", filepath.Base(os.Args[0]), strings.Join(append([]string{
			"Print structured representations of encoded protocol buffer messages.",
			"Since the protobuf wire format is not fully self-describing, type information",
			"about the proto message can be provided using flags (e.g., -messages).",
			"Each field list is a comma-separated list of field identifiers,",
			"where each field identifier is a dot-separated list of field numbers,",
			"identifying each field relative to the root message.",
			"",
			"For example, \"-messages 1,3,3.1 -float32s 1.2 -bools 3.1.2\" represents:",
			"",
			"    message M {",
			"        optional M1 f1 = 1;           // -messages 1",
			"        message M1 {",
			"            repeated float f2 = 2;    // -float32s 1.2",
			"        }",
			"        optional M3 f3 = 3;           // -messages 3",
			"        message M3 {",
			"            optional M1 f1 = 1;       // -messages 3.1",
			"            message M1 {",
			"                repeated bool f2 = 2; // -bools 3.1.2",
			"            }",
			"        }",
			"    }",
			"",
			"Arbitrarily complex message schemas can be represented using these flags.",
			"Scalar field types are marked as repeated so that pbdump can decode",
			"the packed representations of such field types.",
			"",
			"If no inputs are specified, the wire data is read in from stdin, otherwise",
			"the contents of each specified input file is concatenated and",
			"treated as one large message.",
			"",
			"Options:",
		}, flagUsages...), "\n"))
	}
	flag.Parse()

	// Create message types.
	var desc protoreflect.MessageDescriptor
	if len(fs) > 0 {
		var err error
		desc, err = fs.Descriptor()
		if err != nil {
			log.Fatalf("Descriptor error: %v", err)
		}
		if *printDesc {
			fmt.Printf("%#v\n", desc)
		}
	}

	// Read message input.
	var buf []byte
	if flag.NArg() == 0 {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("ReadAll error: %v", err)
		}
		buf = b
	}
	for _, f := range flag.Args() {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatalf("ReadFile error: %v", err)
		}
		buf = append(buf, b...)
	}

	// Parse and print message structure.
	defer log.Printf("fatal input: %q", buf) // debug printout if panic occurs
	var m pack.Message
	m.UnmarshalDescriptor(buf, desc)
	if *printSource {
		fmt.Printf("%#v\n", m)
	} else {
		fmt.Printf("%+v\n", m)
	}
	if !bytes.Equal(buf, m.Marshal()) || len(buf) != m.Size() {
		log.Fatalf("roundtrip mismatch:\n\tgot:  %d %x\n\twant: %d %x", m.Size(), m, len(buf), buf)
	}
	os.Exit(0) // exit cleanly, avoid debug printout
}

// fields is a tree of fields, keyed by a field number.
// Fields representing messages or groups have sub-fields.
type fields map[wire.Number]*field
type field struct {
	kind protoreflect.Kind
	sub  fields // only for MessageKind or GroupKind
}

// Set parses s as a comma-separated list (see the help above for the format)
// and treats each field identifier as the specified kind.
func (fs *fields) Set(s string, k protoreflect.Kind) error {
	if *fs == nil {
		*fs = make(fields)
	}
	for _, s := range strings.Split(s, ",") {
		if err := fs.set("", strings.TrimSpace(s), k); err != nil {
			return err
		}
	}
	return nil
}
func (fs fields) set(prefix, s string, k protoreflect.Kind) error {
	if s == "" {
		return nil
	}

	// Parse next field number.
	i := strings.IndexByte(s, '.')
	if i < 0 {
		i = len(s)
	}
	prefix = strings.TrimPrefix(prefix+"."+s[:i], ".")
	n, _ := strconv.ParseInt(s[:i], 10, 32)
	num := wire.Number(n)
	if num < wire.MinValidNumber || wire.MaxValidNumber < num {
		return fmt.Errorf("invalid field: %v", prefix)
	}
	s = strings.TrimPrefix(s[i:], ".")

	// Handle the current field.
	if fs[num] == nil {
		fs[num] = &field{0, make(fields)}
	}
	if len(s) == 0 {
		if fs[num].kind.IsValid() {
			return fmt.Errorf("field %v already set as %v type", prefix, fs[num].kind)
		}
		fs[num].kind = k
	}
	if err := fs[num].sub.set(prefix, s, k); err != nil {
		return err
	}

	// Verify that only messages or groups can have sub-fields.
	k2 := fs[num].kind
	if k2 > 0 && k2 != protoreflect.MessageKind && k2 != protoreflect.GroupKind && len(fs[num].sub) > 0 {
		return fmt.Errorf("field %v of %v type cannot have sub-fields", prefix, k2)
	}
	return nil
}

// Descriptor returns the field tree as a message descriptor.
func (fs fields) Descriptor() (protoreflect.MessageDescriptor, error) {
	ftyp, err := prototype.NewFile(&prototype.File{
		Syntax:   protoreflect.Proto2,
		Messages: []prototype.Message{fs.messageDescriptor("M")},
	})
	if err != nil {
		return nil, err
	}
	return ftyp.Messages().Get(0), nil
}
func (fs fields) messageDescriptor(name protoreflect.FullName) prototype.Message {
	m := prototype.Message{Name: name.Name()}
	for _, n := range fs.sortedNums() {
		f := prototype.Field{
			Name:        protoreflect.Name(fmt.Sprintf("f%d", n)),
			Number:      n,
			Cardinality: protoreflect.Optional,
			Kind:        fs[n].kind,
		}
		if !f.Kind.IsValid() {
			f.Kind = protoreflect.MessageKind
		}
		switch f.Kind {
		case protoreflect.BoolKind, protoreflect.EnumKind,
			protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
			protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind,
			protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind,
			protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
			f.Cardinality = protoreflect.Repeated
			f.Options = &descriptorpb.FieldOptions{Packed: scalar.Bool(true)}
		case protoreflect.MessageKind, protoreflect.GroupKind:
			s := name.Append(protoreflect.Name(fmt.Sprintf("M%d", n)))
			f.MessageType = prototype.PlaceholderMessage(s)
			m.Messages = append(m.Messages, fs[n].sub.messageDescriptor(s))
		}
		m.Fields = append(m.Fields, f)
	}
	return m
}

func (fs fields) sortedNums() (ns []wire.Number) {
	for n := range fs {
		ns = append(ns, n)
	}
	sort.Slice(ns, func(i, j int) bool { return ns[i] < ns[j] })
	return ns
}

// fieldsFlag is an implementation of flag.Value that is keyed a specific kind.
type fieldsFlag struct {
	f *fields
	k protoreflect.Kind
}

func (fs fieldsFlag) String() string     { return "FIELDS" }
func (fs fieldsFlag) Set(s string) error { return fs.f.Set(s, fs.k) }
