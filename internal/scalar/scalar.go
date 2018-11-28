// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scalar provides wrappers for populating optional scalar fields.
package scalar

// TODO: Should this be public in the v2 API? Where should it live?
// Would we want to do something different if Go gets generics?

func Bool(v bool) *bool          { return &v }
func Int32(v int32) *int32       { return &v }
func Int64(v int64) *int64       { return &v }
func Uint32(v uint32) *uint32    { return &v }
func Uint64(v uint64) *uint64    { return &v }
func Float32(v float32) *float32 { return &v }
func Float64(v float64) *float64 { return &v }
func String(v string) *string    { return &v }
