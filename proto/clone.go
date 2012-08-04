// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2011 Google Inc.  All rights reserved.
// http://code.google.com/p/goprotobuf/
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Protocol buffer deep copy.
// TODO: MessageSet and RawMessage.

package proto

import (
	"log"
	"reflect"
	"strings"
)

// Clone returns a deep copy of a protocol buffer.
func Clone(pb Message) Message {
	in := reflect.ValueOf(pb)
	if in.IsNil() {
		return pb
	}

	out := reflect.New(in.Type().Elem())
	copyStruct(out.Elem(), in.Elem())
	return out.Interface().(Message)
}

func copyStruct(out, in reflect.Value) {
	for i := 0; i < in.NumField(); i++ {
		f := in.Type().Field(i)
		if strings.HasPrefix(f.Name, "XXX_") {
			continue
		}
		copyAny(out.Field(i), in.Field(i))
	}

	if emIn, ok := in.Addr().Interface().(extendableProto); ok {
		emOut := out.Addr().Interface().(extendableProto)
		copyExtension(emOut.ExtensionMap(), emIn.ExtensionMap())
	}

	uin := in.FieldByName("XXX_unrecognized").Bytes()
	if len(uin) > 0 {
		out.FieldByName("XXX_unrecognized").SetBytes(append([]byte(nil), uin...))
	}
}

func copyAny(out, in reflect.Value) {
	if in.Type() == protoMessageType {
		if !in.IsNil() {
			out.Set(reflect.ValueOf(Clone(in.Interface().(Message))))
		}
		return
	}
	switch in.Kind() {
	case reflect.Bool, reflect.Float32, reflect.Float64, reflect.Int32, reflect.Int64,
		reflect.String, reflect.Uint32, reflect.Uint64:
		out.Set(in)
	case reflect.Ptr:
		if in.IsNil() {
			return
		}
		out.Set(reflect.New(in.Type().Elem()))
		copyAny(out.Elem(), in.Elem())
	case reflect.Slice:
		if in.IsNil() {
			return
		}
		n := in.Len()
		out.Set(reflect.MakeSlice(in.Type(), n, n))
		switch in.Type().Elem().Kind() {
		case reflect.Bool, reflect.Float32, reflect.Float64, reflect.Int32, reflect.Int64,
			reflect.String, reflect.Uint32, reflect.Uint64, reflect.Uint8:
			reflect.Copy(out, in)
		default:
			for i := 0; i < n; i++ {
				copyAny(out.Index(i), in.Index(i))
			}
		}
	case reflect.Struct:
		copyStruct(out, in)
	default:
		// unknown type, so not a protocol buffer
		log.Printf("proto: don't know how to copy %v", in)
	}
}

func copyExtension(out, in map[int32]Extension) {
	for extNum, eIn := range in {
		eOut := Extension{desc: eIn.desc}
		if eIn.value != nil {
			v := reflect.New(reflect.TypeOf(eIn.value)).Elem()
			copyAny(v, reflect.ValueOf(eIn.value))
			eOut.value = v.Interface()
		}
		if eIn.enc != nil {
			eOut.enc = make([]byte, len(eIn.enc))
			copy(eOut.enc, eIn.enc)
		}

		out[extNum] = eOut
	}
}
