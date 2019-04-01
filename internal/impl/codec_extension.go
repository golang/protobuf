// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"google.golang.org/protobuf/internal/encoding/wire"
	piface "google.golang.org/protobuf/runtime/protoiface"
)

type extensionFieldInfo struct {
	wiretag uint64
	tagsize int
	funcs   ifaceCoderFuncs
}

func (mi *MessageType) extensionFieldInfo(desc *piface.ExtensionDescV1) *extensionFieldInfo {
	// As of this time (Go 1.12, linux/amd64), an RWMutex benchmarks as faster
	// than a sync.Map.
	mi.extensionFieldInfosMu.RLock()
	e, ok := mi.extensionFieldInfos[desc]
	mi.extensionFieldInfosMu.RUnlock()
	if ok {
		return e
	}

	etype := extensionTypeFromDesc(desc)
	wiretag := wire.EncodeTag(etype.Number(), wireTypes[etype.Kind()])
	e = &extensionFieldInfo{
		wiretag: wiretag,
		tagsize: wire.SizeVarint(wiretag),
		funcs:   encoderFuncsForValue(etype, etype.GoType()),
	}

	mi.extensionFieldInfosMu.Lock()
	if mi.extensionFieldInfos == nil {
		mi.extensionFieldInfos = make(map[*piface.ExtensionDescV1]*extensionFieldInfo)
	}
	mi.extensionFieldInfos[desc] = e
	mi.extensionFieldInfosMu.Unlock()
	return e
}
