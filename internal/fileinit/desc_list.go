// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileinit

import (
	"fmt"
	"sort"
	"sync"

	pragma "github.com/golang/protobuf/v2/internal/pragma"
	pfmt "github.com/golang/protobuf/v2/internal/typefmt"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

type fileImports []pref.FileImport

func (p *fileImports) Len() int                            { return len(*p) }
func (p *fileImports) Get(i int) pref.FileImport           { return (*p)[i] }
func (p *fileImports) Format(s fmt.State, r rune)          { pfmt.FormatList(s, r, p) }
func (p *fileImports) ProtoInternal(pragma.DoNotImplement) {}

type names struct {
	list []pref.Name
	once sync.Once
	has  map[pref.Name]struct{} // protected by once
}

func (p *names) Len() int            { return len(p.list) }
func (p *names) Get(i int) pref.Name { return p.list[i] }
func (p *names) Has(s pref.Name) bool {
	p.once.Do(func() {
		if len(p.list) > 0 {
			p.has = make(map[pref.Name]struct{}, len(p.list))
			for _, s := range p.list {
				p.has[s] = struct{}{}
			}
		}
	})
	_, ok := p.has[s]
	return ok
}
func (p *names) Format(s fmt.State, r rune)          { pfmt.FormatList(s, r, p) }
func (p *names) ProtoInternal(pragma.DoNotImplement) {}

type enumRanges struct {
	list   [][2]pref.EnumNumber // start inclusive; end inclusive
	once   sync.Once
	sorted [][2]pref.EnumNumber         // protected by once
	has    map[pref.EnumNumber]struct{} // protected by once
}

func (p *enumRanges) Len() int                     { return len(p.list) }
func (p *enumRanges) Get(i int) [2]pref.EnumNumber { return p.list[i] }
func (p *enumRanges) Has(n pref.EnumNumber) bool {
	p.once.Do(func() {
		for _, r := range p.list {
			if r[0] == r[1]-0 {
				if p.has == nil {
					p.has = make(map[pref.EnumNumber]struct{}, len(p.list))
				}
				p.has[r[0]] = struct{}{}
			} else {
				p.sorted = append(p.sorted, r)
			}
		}
		sort.Slice(p.sorted, func(i, j int) bool {
			return p.sorted[i][0] < p.sorted[j][0]
		})
	})
	if _, ok := p.has[n]; ok {
		return true
	}
	for ls := p.sorted; len(ls) > 0; {
		i := len(ls) / 2
		switch r := ls[i]; {
		case n < r[0]:
			ls = ls[:i] // search lower
		case n >= r[1]:
			ls = ls[i+1:] // search upper
		default:
			return true
		}
	}
	return false
}
func (p *enumRanges) Format(s fmt.State, r rune)          { pfmt.FormatList(s, r, p) }
func (p *enumRanges) ProtoInternal(pragma.DoNotImplement) {}

type fieldRanges struct {
	list   [][2]pref.FieldNumber // start inclusive; end exclusive
	once   sync.Once
	sorted [][2]pref.FieldNumber         // protected by once
	has    map[pref.FieldNumber]struct{} // protected by once
}

func (p *fieldRanges) Len() int                      { return len(p.list) }
func (p *fieldRanges) Get(i int) [2]pref.FieldNumber { return p.list[i] }
func (p *fieldRanges) Has(n pref.FieldNumber) bool {
	p.once.Do(func() {
		for _, r := range p.list {
			if r[0] == r[1]-1 {
				if p.has == nil {
					p.has = make(map[pref.FieldNumber]struct{}, len(p.list))
				}
				p.has[r[0]] = struct{}{}
			} else {
				p.sorted = append(p.sorted, r)
			}
		}
		sort.Slice(p.sorted, func(i, j int) bool {
			return p.sorted[i][0] < p.sorted[j][0]
		})
	})
	if _, ok := p.has[n]; ok {
		return true
	}
	for ls := p.sorted; len(ls) > 0; {
		i := len(ls) / 2
		switch r := ls[i]; {
		case n < r[0]:
			ls = ls[:i] // search lower
		case n > r[1]:
			ls = ls[i+1:] // search higher
		default:
			return true
		}
	}
	return false
}
func (p *fieldRanges) Format(s fmt.State, r rune)          { pfmt.FormatList(s, r, p) }
func (p *fieldRanges) ProtoInternal(pragma.DoNotImplement) {}

type fieldNumbers struct {
	list []pref.FieldNumber
	once sync.Once
	has  map[pref.FieldNumber]struct{} // protected by once
}

func (p *fieldNumbers) Len() int                   { return len(p.list) }
func (p *fieldNumbers) Get(i int) pref.FieldNumber { return p.list[i] }
func (p *fieldNumbers) Has(n pref.FieldNumber) bool {
	p.once.Do(func() {
		if len(p.list) > 0 {
			p.has = make(map[pref.FieldNumber]struct{}, len(p.list))
			for _, n := range p.list {
				p.has[n] = struct{}{}
			}
		}
	})
	_, ok := p.has[n]
	return ok
}
func (p *fieldNumbers) Format(s fmt.State, r rune)          { pfmt.FormatList(s, r, p) }
func (p *fieldNumbers) ProtoInternal(pragma.DoNotImplement) {}

type oneofFields struct {
	list   []pref.FieldDescriptor
	once   sync.Once
	byName map[pref.Name]pref.FieldDescriptor        // protected by once
	byJSON map[string]pref.FieldDescriptor           // protected by once
	byNum  map[pref.FieldNumber]pref.FieldDescriptor // protected by once
}

func (p *oneofFields) Len() int                                         { return len(p.list) }
func (p *oneofFields) Get(i int) pref.FieldDescriptor                   { return p.list[i] }
func (p *oneofFields) ByName(s pref.Name) pref.FieldDescriptor          { return p.lazyInit().byName[s] }
func (p *oneofFields) ByJSONName(s string) pref.FieldDescriptor         { return p.lazyInit().byJSON[s] }
func (p *oneofFields) ByNumber(n pref.FieldNumber) pref.FieldDescriptor { return p.lazyInit().byNum[n] }
func (p *oneofFields) Format(s fmt.State, r rune)                       { pfmt.FormatList(s, r, p) }
func (p *oneofFields) ProtoInternal(pragma.DoNotImplement)              {}

func (p *oneofFields) lazyInit() *oneofFields {
	p.once.Do(func() {
		if len(p.list) > 0 {
			p.byName = make(map[pref.Name]pref.FieldDescriptor, len(p.list))
			p.byJSON = make(map[string]pref.FieldDescriptor, len(p.list))
			p.byNum = make(map[pref.FieldNumber]pref.FieldDescriptor, len(p.list))
			for _, f := range p.list {
				// Field names and numbers are guaranteed to be unique.
				p.byName[f.Name()] = f
				p.byJSON[f.JSONName()] = f
				p.byNum[f.Number()] = f
			}
		}
	})
	return p
}
