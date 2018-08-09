// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package set provides simple set data structures for uint64 and string types.
//
// The API for every set is:
//	type Set(T {}) opaque
//
//	// Len reports the number of elements in the set.
//	func (Set) Len() int
//
//	// Has reports whether an item is in the set.
//	func (Set) Has(T) bool
//
//	// Set inserts the item into the set.
//	func (Set) Set(T)
//
//	// Clear removes the item from the set.
//	func (Set) Clear(T)
package set
