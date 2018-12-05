// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoV1

// Unmarshaler is the interface representing objects that can
// unmarshal themselves.  The argument points to data that may be
// overwritten, so implementations should not keep references to the
// buffer.
// Unmarshal implementations should not clear the receiver.
// Any unmarshaled data should be merged into the receiver.
// Callers of Unmarshal that do not want to retain existing data
// should Reset the receiver before calling Unmarshal.
type Unmarshaler interface {
	Unmarshal([]byte) error
}

// newUnmarshaler is the interface representing objects that can
// unmarshal themselves. The semantics are identical to Unmarshaler.
//
// This exists to support protoc-gen-go generated messages.
// The proto package will stop type-asserting to this interface in the future.
//
// DO NOT DEPEND ON THIS.
type newUnmarshaler interface {
	XXX_Unmarshal([]byte) error
}

// Unmarshal parses the protocol buffer representation in buf and places the
// decoded result in pb.  If the struct underlying pb does not match
// the data in buf, the results can be unpredictable.
//
// Unmarshal resets pb before starting to unmarshal, so any
// existing data in pb is always removed. Use UnmarshalMerge
// to preserve and append to existing data.
func Unmarshal(buf []byte, pb Message) error {
	pb.Reset()
	if u, ok := pb.(newUnmarshaler); ok {
		return u.XXX_Unmarshal(buf)
	}
	if u, ok := pb.(Unmarshaler); ok {
		return u.Unmarshal(buf)
	}
	return NewBuffer(buf).Unmarshal(pb)
}

// Unmarshal parses the protocol buffer representation in the
// Buffer and places the decoded result in pb.  If the struct
// underlying pb does not match the data in the buffer, the results can be
// unpredictable.
//
// Unlike proto.Unmarshal, this does not reset pb before starting to unmarshal.
func (p *Buffer) Unmarshal(pb Message) error {
	// If the object can unmarshal itself, let it.
	if u, ok := pb.(newUnmarshaler); ok {
		err := u.XXX_Unmarshal(p.buf[p.index:])
		p.index = len(p.buf)
		return err
	}
	if u, ok := pb.(Unmarshaler); ok {
		// NOTE: The history of proto have unfortunately been inconsistent
		// whether Unmarshaler should or should not implicitly clear itself.
		// Some implementations do, most do not.
		// Thus, calling this here may or may not do what people want.
		//
		// See https://github.com/golang/protobuf/issues/424
		err := u.Unmarshal(p.buf[p.index:])
		p.index = len(p.buf)
		return err
	}

	// Slow workaround for messages that aren't Unmarshalers.
	// This includes some hand-coded .pb.go files and
	// bootstrap protos.
	// TODO: fix all of those and then add Unmarshal to
	// the Message interface. Then:
	// The cast above and code below can be deleted.
	// The old unmarshaler can be deleted.
	// Clients can call Unmarshal directly (can already do that, actually).
	var info InternalMessageInfo
	err := info.Unmarshal(pb, p.buf[p.index:])
	p.index = len(p.buf)
	return err
}
