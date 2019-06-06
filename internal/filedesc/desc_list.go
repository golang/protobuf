// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filedesc

import (
	"fmt"
	"sort"
	"sync"

	"google.golang.org/protobuf/internal/descfmt"
	"google.golang.org/protobuf/internal/pragma"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

type FileImports []pref.FileImport

func (p *FileImports) Len() int                            { return len(*p) }
func (p *FileImports) Get(i int) pref.FileImport           { return (*p)[i] }
func (p *FileImports) Format(s fmt.State, r rune)          { descfmt.FormatList(s, r, p) }
func (p *FileImports) ProtoInternal(pragma.DoNotImplement) {}

type Names struct {
	List []pref.Name
	once sync.Once
	has  map[pref.Name]struct{} // protected by once
}

func (p *Names) Len() int            { return len(p.List) }
func (p *Names) Get(i int) pref.Name { return p.List[i] }
func (p *Names) Has(s pref.Name) bool {
	p.once.Do(func() {
		if len(p.List) > 0 {
			p.has = make(map[pref.Name]struct{}, len(p.List))
			for _, s := range p.List {
				p.has[s] = struct{}{}
			}
		}
	})
	_, ok := p.has[s]
	return ok
}
func (p *Names) Format(s fmt.State, r rune)          { descfmt.FormatList(s, r, p) }
func (p *Names) ProtoInternal(pragma.DoNotImplement) {}

type EnumRanges struct {
	List   [][2]pref.EnumNumber // start inclusive; end inclusive
	once   sync.Once
	sorted [][2]pref.EnumNumber         // protected by once
	has    map[pref.EnumNumber]struct{} // protected by once
}

func (p *EnumRanges) Len() int                     { return len(p.List) }
func (p *EnumRanges) Get(i int) [2]pref.EnumNumber { return p.List[i] }
func (p *EnumRanges) Has(n pref.EnumNumber) bool {
	p.once.Do(func() {
		for _, r := range p.List {
			if r[0] == r[1]-0 {
				if p.has == nil {
					p.has = make(map[pref.EnumNumber]struct{}, len(p.List))
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
			ls = ls[i+1:] // search upper
		default:
			return true
		}
	}
	return false
}
func (p *EnumRanges) Format(s fmt.State, r rune)          { descfmt.FormatList(s, r, p) }
func (p *EnumRanges) ProtoInternal(pragma.DoNotImplement) {}

type FieldRanges struct {
	List   [][2]pref.FieldNumber // start inclusive; end exclusive
	once   sync.Once
	sorted [][2]pref.FieldNumber         // protected by once
	has    map[pref.FieldNumber]struct{} // protected by once
}

func (p *FieldRanges) Len() int                      { return len(p.List) }
func (p *FieldRanges) Get(i int) [2]pref.FieldNumber { return p.List[i] }
func (p *FieldRanges) Has(n pref.FieldNumber) bool {
	p.once.Do(func() {
		for _, r := range p.List {
			if r[0] == r[1]-1 {
				if p.has == nil {
					p.has = make(map[pref.FieldNumber]struct{}, len(p.List))
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
			ls = ls[i+1:] // search higher
		default:
			return true
		}
	}
	return false
}
func (p *FieldRanges) Format(s fmt.State, r rune)          { descfmt.FormatList(s, r, p) }
func (p *FieldRanges) ProtoInternal(pragma.DoNotImplement) {}

type FieldNumbers struct {
	List []pref.FieldNumber
	once sync.Once
	has  map[pref.FieldNumber]struct{} // protected by once
}

func (p *FieldNumbers) Len() int                   { return len(p.List) }
func (p *FieldNumbers) Get(i int) pref.FieldNumber { return p.List[i] }
func (p *FieldNumbers) Has(n pref.FieldNumber) bool {
	p.once.Do(func() {
		if len(p.List) > 0 {
			p.has = make(map[pref.FieldNumber]struct{}, len(p.List))
			for _, n := range p.List {
				p.has[n] = struct{}{}
			}
		}
	})
	_, ok := p.has[n]
	return ok
}
func (p *FieldNumbers) Format(s fmt.State, r rune)          { descfmt.FormatList(s, r, p) }
func (p *FieldNumbers) ProtoInternal(pragma.DoNotImplement) {}

type OneofFields struct {
	List   []pref.FieldDescriptor
	once   sync.Once
	byName map[pref.Name]pref.FieldDescriptor        // protected by once
	byJSON map[string]pref.FieldDescriptor           // protected by once
	byNum  map[pref.FieldNumber]pref.FieldDescriptor // protected by once
}

func (p *OneofFields) Len() int                                         { return len(p.List) }
func (p *OneofFields) Get(i int) pref.FieldDescriptor                   { return p.List[i] }
func (p *OneofFields) ByName(s pref.Name) pref.FieldDescriptor          { return p.lazyInit().byName[s] }
func (p *OneofFields) ByJSONName(s string) pref.FieldDescriptor         { return p.lazyInit().byJSON[s] }
func (p *OneofFields) ByNumber(n pref.FieldNumber) pref.FieldDescriptor { return p.lazyInit().byNum[n] }
func (p *OneofFields) Format(s fmt.State, r rune)                       { descfmt.FormatList(s, r, p) }
func (p *OneofFields) ProtoInternal(pragma.DoNotImplement)              {}

func (p *OneofFields) lazyInit() *OneofFields {
	p.once.Do(func() {
		if len(p.List) > 0 {
			p.byName = make(map[pref.Name]pref.FieldDescriptor, len(p.List))
			p.byJSON = make(map[string]pref.FieldDescriptor, len(p.List))
			p.byNum = make(map[pref.FieldNumber]pref.FieldDescriptor, len(p.List))
			for _, f := range p.List {
				// Field names and numbers are guaranteed to be unique.
				p.byName[f.Name()] = f
				p.byJSON[f.JSONName()] = f
				p.byNum[f.Number()] = f
			}
		}
	})
	return p
}
