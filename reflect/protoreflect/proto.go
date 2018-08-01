// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protoreflect

import (
	"regexp"
)

// TODO: This is a stub while the full implementation is under review.
// See https://golang.org/cl/127823.

type Name string

var (
	regexName = regexp.MustCompile(`^[_a-zA-Z][_a-zA-Z0-9]*$`)
)

func (n Name) IsValid() bool {
	return regexName.MatchString(string(n))
}
