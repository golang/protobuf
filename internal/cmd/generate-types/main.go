// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run . -execute

package main

import (
	"flag"
	"go/format"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

var run = flag.Bool("execute", false, "Write generated files to destination.")

func main() {
	flag.Parse()
	chdirRoot()
	writeSource("reflect/prototype/protofile_list_gen.go", generateListTypes())
}

// chdirRoot changes the working directory to the repository root.
func chdirRoot() {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		panic(err)
	}
	if err := os.Chdir(strings.TrimSuffix(string(out), "\n")); err != nil {
		panic(err)
	}
}

// Expr is a single line Go expression.
type Expr string

type DescriptorType string

const (
	MessageDesc   DescriptorType = "Message"
	FieldDesc     DescriptorType = "Field"
	OneofDesc     DescriptorType = "Oneof"
	ExtensionDesc DescriptorType = "Extension"
	EnumDesc      DescriptorType = "Enum"
	EnumValueDesc DescriptorType = "EnumValue"
	ServiceDesc   DescriptorType = "Service"
	MethodDesc    DescriptorType = "Method"
)

func (d DescriptorType) Expr() Expr {
	return "protoreflect." + Expr(d) + "Descriptor"
}
func (d DescriptorType) NumberExpr() Expr {
	switch d {
	case FieldDesc:
		return "protoreflect.FieldNumber"
	case EnumValueDesc:
		return "protoreflect.EnumNumber"
	default:
		return ""
	}
}

func generateListTypes() string {
	// TODO: If Go2 has generics, replace this with a single container type.
	return mustExecute(listTypesTemplate, []DescriptorType{
		MessageDesc, FieldDesc, OneofDesc, ExtensionDesc, EnumDesc, EnumValueDesc, ServiceDesc, MethodDesc,
	})
}

var listTypesTemplate = template.Must(template.New("").Funcs(template.FuncMap{
	"unexport": func(t DescriptorType) Expr {
		return Expr(string(unicode.ToLower(rune(t[0]))) + string(t[1:]))
	},
}).Parse(`
	{{- range .}}
	{{$nameList     := (printf "%ss"     (unexport .))}} {{/* e.g., "messages" */}}
	{{$nameListMeta := (printf "%ssMeta" (unexport .))}} {{/* e.g., "messagesMeta" */}}
	{{$nameMeta     := (printf "%sMeta"  (unexport .))}} {{/* e.g., "messageMeta" */}}
	{{$nameDesc     := (printf "%sDesc"  (unexport .))}} {{/* e.g., "messageDesc" */}}

	type {{$nameListMeta}} struct {
		once     sync.Once
		typs     []{{.}}
		nameOnce sync.Once
		byName   map[protoreflect.Name]*{{.}}
		{{- if (eq . "Field")}}
		jsonOnce sync.Once
		byJSON   map[string]*{{.}}
		{{- end}}
		{{- if .NumberExpr}}
		numOnce  sync.Once
		byNum    map[{{.NumberExpr}}]*{{.}}
		{{- end}}
	}
	type {{$nameList}} {{$nameListMeta}}

	func (p *{{$nameListMeta}}) lazyInit(parent protoreflect.Descriptor, ts []{{.}}) *{{$nameList}} {
		p.once.Do(func() {
			nb := nameBuilderPool.Get().(*nameBuilder)
			defer nameBuilderPool.Put(nb)
			metas := make([]{{$nameMeta}}, len(ts))
			for i := range ts {
				t := &ts[i]
				if t.{{$nameMeta}} != nil {
					panic("already initialized")
				}
				t.{{$nameMeta}} = &metas[i]
				t.inheritedMeta.init(nb, parent, t.Name, {{printf "%v" (eq . "EnumValue")}})
			}
			p.typs = ts
		})
		return (*{{$nameList}})(p)
	}
	func (p *{{$nameList}}) Len() int            { return len(p.typs) }
	func (p *{{$nameList}}) Get(i int) {{.Expr}} { return {{$nameDesc}}{&p.typs[i]} }
	func (p *{{$nameList}}) ByName(s protoreflect.Name) {{.Expr}} {
		p.nameOnce.Do(func() {
			if len(p.typs) > 0 {
				p.byName = make(map[protoreflect.Name]*{{.}}, len(p.typs))
				for i := range p.typs {
					t := &p.typs[i]
					p.byName[t.Name] = t
				}
			}
		})
		t := p.byName[s]
		if t == nil {
			return nil
		}
		return {{$nameDesc}}{t}
	}
	{{- if (eq . "Field")}}
	func (p *{{$nameList}}) ByJSONName(s string) {{.Expr}} {
		p.jsonOnce.Do(func() {
			if len(p.typs) > 0 {
				p.byJSON = make(map[string]*{{.}}, len(p.typs))
				for i := range p.typs {
					t := &p.typs[i]
					s := {{$nameDesc}}{t}.JSONName()
					if _, ok := p.byJSON[s]; !ok {
						p.byJSON[s] = t
					}
				}
			}
		})
		t := p.byJSON[s]
		if t == nil {
			return nil
		}
		return {{$nameDesc}}{t}
	}
	{{- end}}
	{{- if .NumberExpr}}
	func (p *{{$nameList}}) ByNumber(n {{.NumberExpr}}) {{.Expr}} {
		p.numOnce.Do(func() {
			if len(p.typs) > 0 {
				p.byNum = make(map[{{.NumberExpr}}]*{{.}}, len(p.typs))
				for i := range p.typs {
					t := &p.typs[i]
					if _, ok := p.byNum[t.Number]; !ok {
						p.byNum[t.Number] = t
					}
				}
			}
		})
		t := p.byNum[n]
		if t == nil {
			return nil
		}
		return {{$nameDesc}}{t}
	}
	{{- end}}
	func (p *{{$nameList}}) Format(s fmt.State, r rune)          { formatList(s, r, p) }
	func (p *{{$nameList}}) ProtoInternal(pragma.DoNotImplement) {}
	{{- end}}
`))

func mustExecute(t *template.Template, data interface{}) string {
	var sb strings.Builder
	if err := t.Execute(&sb, data); err != nil {
		panic(err)
	}
	return sb.String()
}

func writeSource(file, src string) {
	// Crude but effective way to detect used imports.
	var imports []string
	for _, pkg := range []string{
		"fmt",
		"sync",
		"",
		"google.golang.org/proto/internal/pragma",
		"google.golang.org/proto/reflect/protoreflect",
	} {
		if pkg == "" {
			imports = append(imports, "") // blank line between stdlib and proto packages
		} else if regexp.MustCompile(`[^\pL_0-9]` + path.Base(pkg) + `\.`).MatchString(src) {
			imports = append(imports, strconv.Quote(pkg))
		}
	}

	s := strings.Join([]string{
		"// Copyright 2018 The Go Authors. All rights reserved.",
		"// Use of this source code is governed by a BSD-style.",
		"// license that can be found in the LICENSE file.",
		"",
		"// Code generated by generate-types. DO NOT EDIT.",
		"",
		"package " + path.Base(path.Dir(path.Join("proto", file))),
		"",
		"import (" + strings.Join(imports, "\n") + ")",
		"",
		src,
	}, "\n")
	b, err := format.Source([]byte(s))
	if err != nil {
		panic(err)
	}

	if *run {
		if err := ioutil.WriteFile(file, b, 0664); err != nil {
			panic(err)
		}
	} else {
		if err := ioutil.WriteFile(file+".tmp", b, 0664); err != nil {
			panic(err)
		}
		defer os.Remove(file + ".tmp")

		cmd := exec.Command("diff", file, file+".tmp", "-u")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
