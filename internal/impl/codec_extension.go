// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/internal/encoding/wire"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type extensionFieldInfo struct {
	wiretag uint64
	tagsize int
	funcs   ifaceCoderFuncs
}

func (mi *MessageInfo) extensionFieldInfo(xt pref.ExtensionType) *extensionFieldInfo {
	// As of this time (Go 1.12, linux/amd64), an RWMutex benchmarks as faster
	// than a sync.Map.
	mi.extensionFieldInfosMu.RLock()
	e, ok := mi.extensionFieldInfos[xt]
	mi.extensionFieldInfosMu.RUnlock()
	if ok {
		return e
	}

	wiretag := wire.EncodeTag(xt.Number(), wireTypes[xt.Kind()])
	e = &extensionFieldInfo{
		wiretag: wiretag,
		tagsize: wire.SizeVarint(wiretag),
		funcs:   encoderFuncsForValue(xt, xt.GoType()),
	}

	mi.extensionFieldInfosMu.Lock()
	if mi.extensionFieldInfos == nil {
		mi.extensionFieldInfos = make(map[pref.ExtensionType]*extensionFieldInfo)
	}
	mi.extensionFieldInfos[xt] = e
	mi.extensionFieldInfosMu.Unlock()
	return e
}

type ExtensionField struct {
	typ pref.ExtensionType

	// value is either the value of GetValue,
	// or a *lazyExtensionValue that then returns the value of GetValue.
	value interface{} // TODO: switch to protoreflect.Value
}

func (f ExtensionField) HasType() bool {
	return f.typ != nil
}
func (f ExtensionField) GetType() pref.ExtensionType {
	return f.typ
}
func (f *ExtensionField) SetType(t pref.ExtensionType) {
	f.typ = t
}

// HasValue reports whether a value is set for the extension field.
// This may be called concurrently.
func (f ExtensionField) HasValue() bool {
	return f.value != nil
}

// GetValue returns the concrete value for the extension field.
// Let the type of Desc.ExtensionType be the "API type" and
// the type of GetValue be the "storage type".
// The API type and storage type are the same except:
//	* for scalars (except []byte), where the API type uses *T,
//	while the storage type uses T.
//	* for repeated fields, where the API type uses []T,
//	while the storage type uses *[]T.
//
// The reason for the divergence is so that the storage type more naturally
// matches what is expected of when retrieving the values through the
// protobuf reflection APIs.
//
// GetValue is only populated if Desc is also populated.
// This may be called concurrently.
//
// TODO: switch interface{} to protoreflect.Value
func (f ExtensionField) GetValue() interface{} {
	if f, ok := f.value.(*lazyExtensionValue); ok {
		return f.GetValue()
	}
	return f.value
}

// SetEagerValue sets the current value of the extension.
// This must not be called concurrently.
func (f *ExtensionField) SetEagerValue(v interface{}) {
	f.value = v
}

// SetLazyValue sets a value that is to be lazily evaluated upon first use.
// The returned value must not be nil.
// This must not be called concurrently.
func (f *ExtensionField) SetLazyValue(v func() interface{}) {
	f.value = &lazyExtensionValue{value: v}
}

type lazyExtensionValue struct {
	once  uint32      // atomically set if value is valid
	mu    sync.Mutex  // protects value
	value interface{} // either the value itself or a func() interface{}
}

func (v *lazyExtensionValue) GetValue() interface{} {
	if atomic.LoadUint32(&v.once) == 0 {
		v.mu.Lock()
		if f, ok := v.value.(func() interface{}); ok {
			v.value = f()
		}
		atomic.StoreUint32(&v.once, 1)
		v.mu.Unlock()
	}
	return v.value
}
