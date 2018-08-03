// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package flags provides a set of flags controlled by build tags.
package flags

// Proto1Legacy specifies whether to enable support for legacy proto1
// functionality such as MessageSets, weak fields, and various other obscure
// behavior that is necessary to maintain backwards compatibility with proto1.
//
// This is disabled by default unless built with the "proto1_legacy" tag.
//
// WARNING: The compatibility agreement covers nothing provided by this flag.
// As such, functionality may suddenly be removed or changed at our discretion.
const Proto1Legacy = proto1Legacy
