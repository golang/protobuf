package micro

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

// Paths for packages used by code generated in this file,
// relative to the import_prefix of the generator.Generator.
const (
	contextPkgPath = "golang.org/x/net/context"
	clientPkgPath  = "github.com/micro/go-micro/client"
	serverPkgPath  = "github.com/micro/go-micro/server"
)

func init() {
	generator.RegisterPlugin(new(micro))
}

// micro is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for go-micro support.
type micro struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "micro".
func (g *micro) Name() string {
	return "micro"
}

// The names for packages imported in the generated code.
// They may vary from the final path component of the import path
// if the name is used by other packages.
var (
	contextPkg string
	clientPkg  string
	serverPkg  string
)

// Init initializes the plugin.
func (g *micro) Init(gen *generator.Generator) {
	g.gen = gen
	contextPkg = generator.RegisterUniquePackageName("context", nil)
	clientPkg = generator.RegisterUniquePackageName("client", nil)
	serverPkg = generator.RegisterUniquePackageName("server", nil)
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *micro) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *micro) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *micro) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *micro) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P("var _ ", contextPkg, ".Context")
	g.P("var _ ", clientPkg, ".Option")
	g.P("var _ ", serverPkg, ".Option")
	g.P()
	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}
}

// GenerateImports generates the import declaration for this file.
func (g *micro) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	g.P("import (")
	g.P(contextPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, contextPkgPath)))
	g.P(clientPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, clientPkgPath)))
	g.P(serverPkg, " ", strconv.Quote(path.Join(g.gen.ImportPrefix, serverPkgPath)))
	g.P(")")
	g.P()
}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in go-micro?
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (g *micro) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	path := fmt.Sprintf("6,%d", index) // 6 means service.

	origServName := service.GetName()
	serviceName := strings.ToLower(service.GetName())
	if pkg := file.GetPackage(); pkg != "" {
		serviceName = pkg
	}
	servName := generator.CamelCase(origServName)

	g.P()
	g.P("// Client API for ", servName, " service")
	g.P()

	// Client interface.
	g.P("type ", servName, "Client interface {")
	for i, method := range service.Method {
		g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		g.P(g.generateClientSignature(servName, method))
	}
	g.P("}")
	g.P()

	// Client structure.
	g.P("type ", unexport(servName), "Client struct {")
	g.P("c ", clientPkg, ".Client")
	g.P("}")
	g.P()

	// NewClient factory.
	g.P("func New", servName, "Client (c ", clientPkg, ".Client) ", servName, "Client {")
	g.P("if c == nil {")
	g.P("c = ", clientPkg, ".NewClient()")
	g.P("}")
	g.P("return &", unexport(servName), "Client{")
	g.P("c: c,")
	g.P("}")
	g.P("}")
	g.P()
	var methodIndex, streamIndex int
	serviceDescVar := "_" + servName + "_serviceDesc"
	// Client method implementations.
	for _, method := range service.Method {
		var descExpr string
		if !method.GetServerStreaming() {
			// Unary RPC method
			descExpr = fmt.Sprintf("&%s.Methods[%d]", serviceDescVar, methodIndex)
			methodIndex++
		} else {
			// Streaming RPC method
			descExpr = fmt.Sprintf("&%s.Streams[%d]", serviceDescVar, streamIndex)
			streamIndex++
		}
		g.generateClientMethod(serviceName, servName, serviceDescVar, method, descExpr)
	}

	g.P("// Server API for ", servName, " service")
	g.P()

	// Server interface.
	serverType := servName + "Server"
	g.P("type ", serverType, " interface {")
	for i, method := range service.Method {
		g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		g.P(g.generateServerSignature(servName, method))
	}
	g.P("}")
	g.P()
	// Server registration.
	g.P("func Register", servName, "Server(s ", serverPkg, ".Server, srv ", serverType, ") {")
	g.P("s.Handle(s.NewHandler(srv))")
	g.P("}")
	g.P()

/*
		// Server handler implementations.
		var handlerNames []string
		for _, method := range service.Method {
			hname := g.generateServerMethod(servName, method)
			handlerNames = append(handlerNames, hname)
		}

		// Service descriptor.
		g.P("var ", serviceDescVar, " = ", serverPkg, ".ServiceDesc {")
		g.P("ServiceName: ", strconv.Quote(fullServName), ",")
		g.P("HandlerType: (*", serverType, ")(nil),")
		g.P("Methods: []", serverPkg, ".MethodDesc{")
		for i, method := range service.Method {
			if method.GetServerStreaming() || method.GetClientStreaming() {
				continue
			}
			g.P("{")
			g.P("MethodName: ", strconv.Quote(method.GetName()), ",")
			g.P("Handler: ", handlerNames[i], ",")
			g.P("},")
		}
		g.P("},")
		g.P("Streams: []", serverPkg, ".StreamDesc{")
		for i, method := range service.Method {
			if !method.GetServerStreaming() && !method.GetClientStreaming() {
				continue
			}
			g.P("{")
			g.P("StreamName: ", strconv.Quote(method.GetName()), ",")
			g.P("Handler: ", handlerNames[i], ",")
			if method.GetServerStreaming() {
				g.P("ServerStreams: true,")
			}
			if method.GetClientStreaming() {
				g.P("ClientStreams: true,")
			}
			g.P("},")
		}
		g.P("},")
		g.P("}")
		g.P()
	*/
}

// generateClientSignature returns the client-side signature for a method.
func (g *micro) generateClientSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	reqArg := ", in *" + g.typeName(method.GetInputType())
	respName := "*" + g.typeName(method.GetOutputType())
	if method.GetServerStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
	}
	return fmt.Sprintf("%s(ctx %s.Context%s) (%s, error)", methName, contextPkg, reqArg, respName)
}

func (g *micro) generateClientMethod(reqServ, servName, serviceDescVar string, method *pb.MethodDescriptorProto, descExpr string) {
	reqMethod := fmt.Sprintf("%s.%s", servName, method.GetName())
	methName := generator.CamelCase(method.GetName())
	//	inType := g.typeName(method.GetInputType())
	outType := g.typeName(method.GetOutputType())

	g.P("func (c *", unexport(servName), "Client) ", g.generateClientSignature(servName, method), "{")
	g.P(`req := c.c.NewRequest("`, reqServ, `", "`, reqMethod, `", in)`)
	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		g.P("out := new(", outType, ")")
		// TODO: Pass descExpr to Invoke.
		g.P("err := ", `c.c.Call(ctx, req, out)`)
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
		return
	}
	streamType := unexport(servName) + methName + "Client"
	g.P("outCh := make(chan *", outType, ")")
	g.P("stream, err := c.c.Stream(ctx, req, outCh)")
	g.P("if err != nil { return nil, err }")
	g.P("return &", streamType, "{stream, outCh}, nil")
	g.P("}")
	g.P()

	//genSend := method.GetClientStreaming()
	genRecv := method.GetServerStreaming()
	//genCloseAndRecv := !method.GetServerStreaming()

	// Stream auxiliary types and methods.
	g.P("type ", servName, "_", methName, "Client interface {")
	if genRecv {
		g.P("Next() (*", outType, ", error)")
	}
	g.P(clientPkg, ".Streamer")
	g.P("}")
	g.P()

	g.P("type ", streamType, " struct {")
	g.P(clientPkg, ".Streamer")
	g.P("next chan *", outType)
	g.P("}")
	g.P()

	if genRecv {
		g.P("func (x *", streamType, ") Next() (*", outType, ", error) {")
		g.P("out, ok := <-x.next")
		g.P("if !ok {")
		g.P("return nil, fmt.Errorf(`chan closed`)")
		g.P("}")
		g.P("return out, nil")
		g.P("}")
		g.P()
	}
}

// generateServerSignature returns the server-side signature for a method.
func (g *micro) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "error"
	reqArgs = append(reqArgs, contextPkg+".Context")

//	if !method.GetServerStreaming() && !method.GetClientStreaming() {
//		ret = "(*" + g.typeName(method.GetOutputType()) + ", error)"
//	}
	if !method.GetClientStreaming() && !method.GetServerStreaming() {
		reqArgs = append(reqArgs, "*"+g.typeName(method.GetInputType()))
		reqArgs = append(reqArgs, "*"+g.typeName(method.GetOutputType()))
	}
	if method.GetServerStreaming() || method.GetClientStreaming() {
	//	reqArgs = append(reqArgs, servName+"_"+generator.CamelCase(origMethName)+"Server")
		reqArgs = append(reqArgs, "func(*"+g.typeName(method.GetOutputType())+") error")
	}

	return methName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

func (g *micro) generateServerMethod(servName string, method *pb.MethodDescriptorProto) string {
	methName := generator.CamelCase(method.GetName())
	hname := fmt.Sprintf("_%s_%s_Handler", servName, methName)
	inType := g.typeName(method.GetInputType())
	outType := g.typeName(method.GetOutputType())

	if !method.GetServerStreaming() && !method.GetClientStreaming() {
		g.P("func ", hname, "(srv interface{}, ctx ", contextPkg, ".Context, dec func(interface{}) error) (interface{}, error) {")
		g.P("in := new(", inType, ")")
		g.P("if err := dec(in); err != nil { return nil, err }")
		g.P("out, err := srv.(", servName, "Server).", methName, "(ctx, in)")
		g.P("if err != nil { return nil, err }")
		g.P("return out, nil")
		g.P("}")
		g.P()
		return hname
	}
	streamType := unexport(servName) + methName + "Server"
	g.P("func ", hname, "(srv interface{}, stream ", serverPkg, ".ServerStream) error {")
	if !method.GetClientStreaming() {
		g.P("m := new(", inType, ")")
		g.P("if err := stream.RecvMsg(m); err != nil { return err }")
		g.P("return srv.(", servName, "Server).", methName, "(m, &", streamType, "{stream})")
	} else {
		g.P("return srv.(", servName, "Server).", methName, "(&", streamType, "{stream})")
	}
	g.P("}")
	g.P()

	genSend := method.GetServerStreaming()
	genSendAndClose := !method.GetServerStreaming()
	genRecv := method.GetClientStreaming()

	// Stream auxiliary types and methods.
	g.P("type ", servName, "_", methName, "Server interface {")
	if genSend {
		g.P("Send(*", outType, ") error")
	}
	if genSendAndClose {
		g.P("SendAndClose(*", outType, ") error")
	}
	if genRecv {
		g.P("Recv() (*", inType, ", error)")
	}
	g.P(serverPkg, ".ServerStream")
	g.P("}")
	g.P()

	g.P("type ", streamType, " struct {")
	g.P(serverPkg, ".ServerStream")
	g.P("}")
	g.P()

	if genSend {
		g.P("func (x *", streamType, ") Send(m *", outType, ") error {")
		g.P("return x.ServerStream.SendMsg(m)")
		g.P("}")
		g.P()
	}
	if genSendAndClose {
		g.P("func (x *", streamType, ") SendAndClose(m *", outType, ") error {")
		g.P("return x.ServerStream.SendMsg(m)")
		g.P("}")
		g.P()
	}
	if genRecv {
		g.P("func (x *", streamType, ") Recv() (*", inType, ", error) {")
		g.P("m := new(", inType, ")")
		g.P("if err := x.ServerStream.RecvMsg(m); err != nil { return nil, err }")
		g.P("return m, nil")
		g.P("}")
		g.P()
	}

	return hname
}
