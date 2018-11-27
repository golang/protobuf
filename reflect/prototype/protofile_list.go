// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prototype

import (
	"fmt"
	"sync"

	pragma "github.com/golang/protobuf/v2/internal/pragma"
	pset "github.com/golang/protobuf/v2/internal/set"
	pref "github.com/golang/protobuf/v2/reflect/protoreflect"
)

type numbersMeta struct {
	once sync.Once
	ns   []pref.FieldNumber
	nss  pset.Ints
}
type numbers numbersMeta

func (p *numbersMeta) lazyInit(fs []Field) *numbers {
	p.once.Do(func() {
		for i := range fs {
			if f := &fs[i]; f.Cardinality == pref.Required {
				p.ns = append(p.ns, f.Number)
				p.nss.Set(uint64(f.Number))
			}
		}
	})
	return (*numbers)(p)
}
func (p *numbers) Len() int                            { return len(p.ns) }
func (p *numbers) Get(i int) pref.FieldNumber          { return p.ns[i] }
func (p *numbers) Has(n pref.FieldNumber) bool         { return p.nss.Has(uint64(n)) }
func (p *numbers) Format(s fmt.State, r rune)          { formatList(s, r, p) }
func (p *numbers) ProtoInternal(pragma.DoNotImplement) {}

type ranges [][2]pref.FieldNumber

func (p *ranges) Len() int                      { return len(*p) }
func (p *ranges) Get(i int) [2]pref.FieldNumber { return (*p)[i] }
func (p *ranges) Has(n pref.FieldNumber) bool {
	for _, r := range *p {
		if r[0] <= n && n < r[1] {
			return true
		}
	}
	return false
}
func (p *ranges) Format(s fmt.State, r rune)          { formatList(s, r, p) }
func (p *ranges) ProtoInternal(pragma.DoNotImplement) {}

type fileImports []pref.FileImport

func (p *fileImports) Len() int                            { return len(*p) }
func (p *fileImports) Get(i int) pref.FileImport           { return (*p)[i] }
func (p *fileImports) Format(s fmt.State, r rune)          { formatList(s, r, p) }
func (p *fileImports) ProtoInternal(pragma.DoNotImplement) {}

type oneofFieldsMeta struct {
	once   sync.Once
	typs   []pref.FieldDescriptor
	byName map[pref.Name]pref.FieldDescriptor
	byJSON map[string]pref.FieldDescriptor
	byNum  map[pref.FieldNumber]pref.FieldDescriptor
}
type oneofFields oneofFieldsMeta

func (p *oneofFieldsMeta) lazyInit(parent pref.Descriptor) *oneofFields {
	p.once.Do(func() {
		otyp := parent.(pref.OneofDescriptor)
		mtyp, _ := parent.Parent()
		fs := mtyp.(pref.MessageDescriptor).Fields()
		for i := 0; i < fs.Len(); i++ {
			if f := fs.Get(i); otyp == f.OneofType() {
				p.typs = append(p.typs, f)
			}
		}
		if len(p.typs) > 0 {
			p.byName = make(map[pref.Name]pref.FieldDescriptor, len(p.typs))
			p.byJSON = make(map[string]pref.FieldDescriptor, len(p.typs))
			p.byNum = make(map[pref.FieldNumber]pref.FieldDescriptor, len(p.typs))
			for _, f := range p.typs {
				p.byName[f.Name()] = f
				p.byJSON[f.JSONName()] = f
				p.byNum[f.Number()] = f
			}
		}
	})
	return (*oneofFields)(p)
}
func (p *oneofFields) Len() int                                         { return len(p.typs) }
func (p *oneofFields) Get(i int) pref.FieldDescriptor                   { return p.typs[i] }
func (p *oneofFields) ByName(s pref.Name) pref.FieldDescriptor          { return p.byName[s] }
func (p *oneofFields) ByJSONName(s string) pref.FieldDescriptor         { return p.byJSON[s] }
func (p *oneofFields) ByNumber(n pref.FieldNumber) pref.FieldDescriptor { return p.byNum[n] }
func (p *oneofFields) Format(s fmt.State, r rune)                       { formatList(s, r, p) }
func (p *oneofFields) ProtoInternal(pragma.DoNotImplement)              {}
