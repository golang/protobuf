// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import pvalue "github.com/golang/protobuf/v2/internal/value"

// TODO: Add a default LegacyWrapper that panics with a more helpful message?
var legacyWrapper pvalue.LegacyWrapper

// RegisterLegacyWrapper registers a set of constructor functions that are
// called when a legacy enum or message is encountered that does not natively
// support the protobuf reflection APIs.
func RegisterLegacyWrapper(w pvalue.LegacyWrapper) {
	legacyWrapper = w
}
