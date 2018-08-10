// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	pref "google.golang.org/proto/reflect/protoreflect"
)

// TODO: This cannot be implemented without proto.Unmarshal.

type descriptorFileMeta struct{}

func (p *descriptorFileMeta) lazyInit(t fileDesc) (pref.Message, bool) {
	return nil, false
}

type descriptorSubMeta struct{}

func (p *descriptorSubMeta) lazyInit(t pref.Descriptor) (pref.Message, bool) {
	return nil, false
}

type descriptorOptionsMeta struct{}

func (p *descriptorOptionsMeta) lazyInit(t pref.Descriptor) (pref.DescriptorOptions, bool) {
	return nil, false
}
