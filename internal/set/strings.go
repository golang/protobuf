// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set

// Strings represents a set of strings.
type Strings map[string]struct{}

func (ss *Strings) Len() int {
	return len(*ss)
}
func (ss *Strings) Has(s string) bool {
	_, ok := (*ss)[s]
	return ok
}
func (ss *Strings) Set(s string) {
	if *ss == nil {
		*ss = make(map[string]struct{})
	}
	(*ss)[s] = struct{}{}
}
func (ss *Strings) Clear(s string) {
	delete(*ss, s)
}
