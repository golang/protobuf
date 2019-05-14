// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"bytes"

	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// Equal returns true of two messages are equal.
//
// Two messages are equal if they have identical types and registered extension fields,
// marshal to the same bytes under deterministic serialization,
// and contain no floating point NaNs.
func Equal(a, b Message) bool {
	return equalMessage(a.ProtoReflect(), b.ProtoReflect())
}

// equalMessage compares two messages.
func equalMessage(a, b pref.Message) bool {
	mda, mdb := a.Descriptor(), b.Descriptor()
	if mda != mdb && mda.FullName() != mdb.FullName() {
		return false
	}

	// TODO: The v1 says that a nil message is not equal to an empty one.
	// Decide what to do about this when v1 wraps v2.

	knowna, knownb := a.KnownFields(), b.KnownFields()

	fields := mda.Fields()
	for i, flen := 0, fields.Len(); i < flen; i++ {
		fd := fields.Get(i)
		num := fd.Number()
		hasa, hasb := knowna.Has(num), knownb.Has(num)
		if !hasa && !hasb {
			continue
		}
		if hasa != hasb || !equalFields(fd, knowna.Get(num), knownb.Get(num)) {
			return false
		}
	}
	equal := true

	unknowna, unknownb := a.UnknownFields(), b.UnknownFields()
	ulen := unknowna.Len()
	if ulen != unknownb.Len() {
		return false
	}
	unknowna.Range(func(num pref.FieldNumber, ra pref.RawFields) bool {
		rb := unknownb.Get(num)
		if !bytes.Equal([]byte(ra), []byte(rb)) {
			equal = false
			return false
		}
		return true
	})
	if !equal {
		return false
	}

	// If the set of extension types is not identical for both messages, we report
	// a inequality.
	//
	// This requirement is stringent. Registering an extension type for a message
	// without setting a value for the extension will cause that message to compare
	// as inequal to the same message without the registration.
	//
	// TODO: Revisit this behavior after eager decoding of extensions is implemented.
	xtypesa, xtypesb := knowna.ExtensionTypes(), knownb.ExtensionTypes()
	if la, lb := xtypesa.Len(), xtypesb.Len(); la != lb {
		return false
	} else if la == 0 {
		return true
	}
	xtypesa.Range(func(xt pref.ExtensionType) bool {
		num := xt.Descriptor().Number()
		if xtypesb.ByNumber(num) != xt {
			equal = false
			return false
		}
		hasa, hasb := knowna.Has(num), knownb.Has(num)
		if !hasa && !hasb {
			return true
		}
		if hasa != hasb || !equalFields(xt.Descriptor(), knowna.Get(num), knownb.Get(num)) {
			equal = false
			return false
		}
		return true
	})
	return equal
}

// equalFields compares two fields.
func equalFields(fd pref.FieldDescriptor, a, b pref.Value) bool {
	switch {
	case fd.IsList():
		return equalList(fd, a.List(), b.List())
	case fd.IsMap():
		return equalMap(fd, a.Map(), b.Map())
	default:
		return equalValue(fd, a, b)
	}
}

// equalMap compares a map field.
func equalMap(fd pref.FieldDescriptor, a, b pref.Map) bool {
	alen := a.Len()
	if alen != b.Len() {
		return false
	}
	equal := true
	a.Range(func(k pref.MapKey, va pref.Value) bool {
		vb := b.Get(k)
		if !vb.IsValid() || !equalValue(fd.MapValue(), va, vb) {
			equal = false
			return false
		}
		return true
	})
	return equal
}

// equalList compares a non-map repeated field.
func equalList(fd pref.FieldDescriptor, a, b pref.List) bool {
	alen := a.Len()
	if alen != b.Len() {
		return false
	}
	for i := 0; i < alen; i++ {
		if !equalValue(fd, a.Get(i), b.Get(i)) {
			return false
		}
	}
	return true
}

// equalValue compares the scalar value type of a field.
func equalValue(fd pref.FieldDescriptor, a, b pref.Value) bool {
	switch {
	case fd.Message() != nil:
		return equalMessage(a.Message(), b.Message())
	case fd.Kind() == pref.BytesKind:
		return bytes.Equal(a.Bytes(), b.Bytes())
	default:
		return a.Interface() == b.Interface()
	}
}
