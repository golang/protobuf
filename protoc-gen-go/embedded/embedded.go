package embedded

import (
	"bufio"
	"bytes"
	"os"
	"regexp"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/generator"
)

var (
	// protoTypeRe is used to get the field's type.
	protoTypeRe = regexp.MustCompile(`^\s*([^\s]*)\s*[^\s]*\s*=`)
	// goTypeRe is used to get field's pointer type.
	goTypeRe = regexp.MustCompile(`(\*[^\s]+)(.*)`)
)

func init() {
	generator.RegisterPlugin(new(embedded))
}

// embedded is an implementation of the Go protocol buffer compiler's
// plugin architecture. It generates bindings for struct embedding support.
type embedded struct {
	generator *generator.Generator
	embedded  [][]byte
}

// Name returns the name of this plugin, "embedded".
func (r *embedded) Name() string {
	return "embedded"
}

// Init initializes the plugin.
func (r *embedded) Init(generator *generator.Generator) {
	r.generator = generator
}

// P forwards to g.gen.P.
func (r *embedded) P(args ...interface{}) { r.generator.P(args...) }

// Generate generates code for the services in the given file.
func (r *embedded) Generate(file *generator.FileDescriptor) {
	r.build(*file.Name)
	r.generate()
}

// GenerateImports generates the import declaration for this file.
func (r *embedded) GenerateImports(file *generator.FileDescriptor) {}

// build is used to find/build slice of embedded fields.
func (r *embedded) build(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		if field, ok := isEmbedded(line); ok {
			f := make([]byte, len(field))
			copy(f, field)
			r.embedded = append(r.embedded, f)
		}
	}
}

// isEmbedded returns true if the given line has specified to embed its struct.
func isEmbedded(line []byte) (structType []byte, ok bool) {
	if !bytes.Contains(line, []byte("[(go_embed)=true]")) {
		return nil, false
	}
	return protoTypeRe.FindAllSubmatch(line, -1)[0][1], true
}

// generate updates the given file to embed the fields.
func (r *embedded) generate() {
	if len(r.embedded) == 0 {
		return
	}

	readbuf := bytes.NewBuffer([]byte{})
	readbuf.Write(r.generator.Buffer.Bytes())
	buf := bytes.NewBuffer([]byte{})
	reader := bufio.NewReader(readbuf)

outer:
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			buf.WriteString("\n")
			break
		}

		l := bytes.TrimSpace(line)
		m := goTypeRe.FindSubmatch(l)

		if len(r.embedded) > 0 && len(m) > 0 {
			want := "*" + strings.Replace(string(r.embedded[0]), ".", "_", -1)
			have := strings.Replace(string(m[1]), ".", "_", -1)

			if strings.Compare(want, have) == 0 {
				buf.Write(m[0])
				buf.WriteString("\n")
				r.embedded = r.embedded[1:]
				continue outer
			}
		}

		buf.Write(line)
		buf.WriteString("\n")
	}

	r.generator.Buffer.Reset()
	r.generator.Buffer.Write(buf.Bytes())
}
