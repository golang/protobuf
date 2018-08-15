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

	"google.golang.org/proto/internal/encoding/pack"
	"google.golang.org/proto/internal/encoding/wire"
	"google.golang.org/proto/reflect/protoreflect"
	"google.golang.org/proto/reflect/prototype"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	var fs fields
	flag.Var((*boolFields)(&fs), "bools", "List of bool fields")
	flag.Var((*intFields)(&fs), "ints", "List of int32 or int64 fields")
	flag.Var((*sintFields)(&fs), "sints", "List of sint32 or sint64 fields")
	flag.Var((*uintFields)(&fs), "uints", "List of enum, uint32, or uint64 fields")
	flag.Var((*uint32Fields)(&fs), "uint32s", "List of fixed32 fields")
	flag.Var((*int32Fields)(&fs), "int32s", "List of sfixed32 fields")
	flag.Var((*float32Fields)(&fs), "float32s", "List of float fields")
	flag.Var((*uint64Fields)(&fs), "uint64s", "List of fixed64 fields")
	flag.Var((*int64Fields)(&fs), "int64s", "List of sfixed64 fields")
	flag.Var((*float64Fields)(&fs), "float64s", "List of double fields")
	flag.Var((*stringFields)(&fs), "strings", "List of string fields")
	flag.Var((*bytesFields)(&fs), "bytes", "List of bytes fields")
	flag.Var((*messageFields)(&fs), "messages", "List of message fields")
	flag.Var((*groupFields)(&fs), "groups", "List of group fields")
	printDesc := flag.Bool("print_descriptor", false, "Print the message descriptor")
	printSource := flag.Bool("print_source", false, "Print the output in valid Go syntax")
	flag.Usage = func() {
		log.Printf("Usage: %s [OPTIONS]... [INPUTS]...\n\n%s\n", filepath.Base(os.Args[0]), strings.Join([]string{
			"Print structured representations of encoded protocol buffer messages.",
			"Since the protobuf wire format is not fully self-describing, type information",
			"about the proto message can be provided using flags (e.g., -messages).",
			"Each field list is a comma-separated list of field identifiers,",
			"where each field identifier is a dot-separated list of field numbers,",
			"identifying each field relative to the root message.",
			"",
			"For example, \"-messages 1,3,3.1 -float32s 1.2 -bools 3.1.2\" represents:",
			"",
			"	message M {",
			"		optional M1 f1 = 1;           // -messages 1",
			"		message M1 {",
			"			repeated float f2 = 2;    // -float32s 1.2",
			"		}",
			"		optional M3 f3 = 3;           // -messages 3",
			"		message M3 {",
			"			optional M1 f1 = 1;       // -messages 3.1",
			"			message M1 {",
			"				repeated bool f2 = 2; // -bools 3.1.2",
			"			}",
			"		}",
			"	}",
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
			"  -bools fields      " + flag.Lookup("bools").Usage,
			"  -ints fields       " + flag.Lookup("ints").Usage,
			"  -sints fields      " + flag.Lookup("sints").Usage,
			"  -uints fields      " + flag.Lookup("uints").Usage,
			"  -int32s fields     " + flag.Lookup("int32s").Usage,
			"  -int64s fields     " + flag.Lookup("int64s").Usage,
			"  -uint32s fields    " + flag.Lookup("uint32s").Usage,
			"  -uint64s fields    " + flag.Lookup("uint64s").Usage,
			"  -float32s fields   " + flag.Lookup("float32s").Usage,
			"  -float64s fields   " + flag.Lookup("float64s").Usage,
			"  -strings fields    " + flag.Lookup("strings").Usage,
			"  -bytes fields      " + flag.Lookup("bytes").Usage,
			"  -messages fields   " + flag.Lookup("messages").Usage,
			"  -groups fields     " + flag.Lookup("groups").Usage,
			"  -print_descriptor  " + flag.Lookup("print_descriptor").Usage,
			"  -print_source      " + flag.Lookup("print_source").Usage,
		}, "\n"))
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
			log.Printf("%#v\n", desc)
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
		fmt.Fprintf(os.Stdout, "%#v\n", m)
	} else {
		fmt.Fprintf(os.Stdout, "%+v\n", m)
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
			f.IsPacked = true
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

type (
	boolFields    fields
	intFields     fields
	sintFields    fields
	uintFields    fields
	uint32Fields  fields
	int32Fields   fields
	float32Fields fields
	uint64Fields  fields
	int64Fields   fields
	float64Fields fields
	stringFields  fields
	bytesFields   fields
	messageFields fields
	groupFields   fields
)

// String and Set implement flag.Value.
// The String method is not implemented since the flag helper never prints it.
func (p *boolFields) String() string        { return "not implemented" }
func (p *intFields) String() string         { return "not implemented" }
func (p *sintFields) String() string        { return "not implemented" }
func (p *uintFields) String() string        { return "not implemented" }
func (p *uint32Fields) String() string      { return "not implemented" }
func (p *int32Fields) String() string       { return "not implemented" }
func (p *float32Fields) String() string     { return "not implemented" }
func (p *uint64Fields) String() string      { return "not implemented" }
func (p *int64Fields) String() string       { return "not implemented" }
func (p *float64Fields) String() string     { return "not implemented" }
func (p *stringFields) String() string      { return "not implemented" }
func (p *bytesFields) String() string       { return "not implemented" }
func (p *messageFields) String() string     { return "not implemented" }
func (p *groupFields) String() string       { return "not implemented" }
func (p *boolFields) Set(s string) error    { return (*fields)(p).Set(s, protoreflect.BoolKind) }
func (p *intFields) Set(s string) error     { return (*fields)(p).Set(s, protoreflect.Int64Kind) }
func (p *sintFields) Set(s string) error    { return (*fields)(p).Set(s, protoreflect.Sint64Kind) }
func (p *uintFields) Set(s string) error    { return (*fields)(p).Set(s, protoreflect.Uint64Kind) }
func (p *uint32Fields) Set(s string) error  { return (*fields)(p).Set(s, protoreflect.Fixed32Kind) }
func (p *int32Fields) Set(s string) error   { return (*fields)(p).Set(s, protoreflect.Sfixed32Kind) }
func (p *float32Fields) Set(s string) error { return (*fields)(p).Set(s, protoreflect.FloatKind) }
func (p *uint64Fields) Set(s string) error  { return (*fields)(p).Set(s, protoreflect.Fixed64Kind) }
func (p *int64Fields) Set(s string) error   { return (*fields)(p).Set(s, protoreflect.Sfixed64Kind) }
func (p *float64Fields) Set(s string) error { return (*fields)(p).Set(s, protoreflect.DoubleKind) }
func (p *stringFields) Set(s string) error  { return (*fields)(p).Set(s, protoreflect.StringKind) }
func (p *bytesFields) Set(s string) error   { return (*fields)(p).Set(s, protoreflect.BytesKind) }
func (p *messageFields) Set(s string) error { return (*fields)(p).Set(s, protoreflect.MessageKind) }
func (p *groupFields) Set(s string) error   { return (*fields)(p).Set(s, protoreflect.GroupKind) }
