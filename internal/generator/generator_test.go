package generator

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerateUsesAuthLevelAndFailsClosed(t *testing.T) {
	publicOptions := &descriptorpb.MethodOptions{}
	publicOptions.ProtoReflect().SetUnknown(authLevelUnknown(public))
	unspecifiedOptions := &descriptorpb.MethodOptions{}
	unspecifiedOptions.ProtoReflect().SetUnknown(authLevelUnknown(unspecified))

	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("example/v1/example.proto"),
		Package: proto.String("example.v1"),
		Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/example/gen/examplev1;examplev1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Request")},
			{Name: proto.String("Response")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: proto.String("ExampleService"),
			Method: []*descriptorpb.MethodDescriptorProto{
				{Name: proto.String("Public"), InputType: proto.String(".example.v1.Request"), OutputType: proto.String(".example.v1.Response"), Options: publicOptions},
				{Name: proto.String("Default"), InputType: proto.String(".example.v1.Request"), OutputType: proto.String(".example.v1.Response")},
				{Name: proto.String("Unspecified"), InputType: proto.String(".example.v1.Request"), OutputType: proto.String(".example.v1.Response"), Options: unspecifiedOptions},
			},
		}},
	}
	req := &pluginpb.CodeGeneratorRequest{ProtoFile: []*descriptorpb.FileDescriptorProto{file}, FileToGenerate: []string{file.GetName()}}
	plugin, err := (protogen.Options{}).New(req)
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(plugin); err != nil {
		t.Fatal(err)
	}
	response := plugin.Response()
	if len(response.File) != 1 {
		t.Fatalf("generated %d files, want 1", len(response.File))
	}
	content := response.File[0].GetContent()
	if _, err := parser.ParseFile(token.NewFileSet(), response.File[0].GetName(), content, parser.AllErrors); err != nil {
		t.Fatalf("generated Go is invalid: %v\n%s", err, content)
	}
	for _, want := range []string{
		"case \"/example.v1.ExampleService/Public\":",
		"case \"/example.v1.ExampleService/Default\":",
		"case \"/example.v1.ExampleService/Unspecified\":",
		"i.verifier.Verify(ctx, AuthLevelAuthenticated)",
		"func NewExampleServiceHandlerWithAuthz",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("generated code does not contain %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "i.verifier.Verify(ctx, AuthLevelPublic)") {
		t.Errorf("PUBLIC method must not invoke verifier:\n%s", content)
	}
}

func authLevelUnknown(level int) []byte {
	unknown := protowire.AppendTag(nil, authLevelFieldNumber, protowire.VarintType)
	return protowire.AppendVarint(unknown, uint64(level))
}
