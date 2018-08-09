// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package set

import "math/bits"

// Int32s represents a set of integers within the range of 0..31.
type Int32s uint32

func (bs *Int32s) Len() int {
	return bits.OnesCount32(uint32(*bs))
}
func (bs *Int32s) Has(n uint64) bool {
	return uint32(*bs)&(uint32(1)<<n) > 0
}
func (bs *Int32s) Set(n uint64) {
	*(*uint32)(bs) |= uint32(1) << n
}
func (bs *Int32s) Clear(n uint64) {
	*(*uint32)(bs) &^= uint32(1) << n
}

// Int64s represents a set of integers within the range of 0..63.
type Int64s uint64

func (bs *Int64s) Len() int {
	return bits.OnesCount64(uint64(*bs))
}
func (bs *Int64s) Has(n uint64) bool {
	return uint64(*bs)&(uint64(1)<<n) > 0
}
func (bs *Int64s) Set(n uint64) {
	*(*uint64)(bs) |= uint64(1) << n
}
func (bs *Int64s) Clear(n uint64) {
	*(*uint64)(bs) &^= uint64(1) << n
}

// Ints represents a set of integers within the range of 0..math.MaxUint64.
type Ints struct {
	lo Int64s
	hi map[uint64]struct{}
}

func (bs *Ints) Len() int {
	return bs.lo.Len() + len(bs.hi)
}
func (bs *Ints) Has(n uint64) bool {
	if n < 64 {
		return bs.lo.Has(n)
	}
	_, ok := bs.hi[n]
	return ok
}
func (bs *Ints) Set(n uint64) {
	if n < 64 {
		bs.lo.Set(n)
		return
	}
	if bs.hi == nil {
		bs.hi = make(map[uint64]struct{})
	}
	bs.hi[n] = struct{}{}
}
func (bs *Ints) Clear(n uint64) {
	if n < 64 {
		bs.lo.Clear(n)
		return
	}
	delete(bs.hi, n)
}
