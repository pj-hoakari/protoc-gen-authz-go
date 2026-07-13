package generator

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func TestGenerateUsesAuthPolicyAndFailsClosed(t *testing.T) {
	publicOptions := &descriptorpb.MethodOptions{}
	publicOptions.ProtoReflect().SetUnknown(authPolicyUnknown(public))
	unspecifiedOptions := &descriptorpb.MethodOptions{}
	unspecifiedOptions.ProtoReflect().SetUnknown(authPolicyUnknown(unspecified))
	internalOptions := &descriptorpb.MethodOptions{}
	internalOptions.ProtoReflect().SetUnknown(authPolicyUnknown(internal, "greeting.read", "greeting.write"))

	file := &descriptorpb.FileDescriptorProto{
		Name:    new("example/v1/example.proto"),
		Package: new("example.v1"),
		Options: &descriptorpb.FileOptions{GoPackage: new("example.com/example/gen/examplev1;examplev1")},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: new("Request")},
			{Name: new("Response")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: new("ExampleService"),
			Method: []*descriptorpb.MethodDescriptorProto{
				{Name: new("Public"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response"), Options: publicOptions},
				{Name: new("Default"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response")},
				{Name: new("Unspecified"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response"), Options: unspecifiedOptions},
				{Name: new("Internal"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response"), Options: internalOptions},
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
		"i.verifier.Verify(ctx, AuthPolicy{Level: AuthLevelAuthenticated})",
		"i.verifier.Verify(ctx, AuthPolicy{Level: AuthLevelInternal, RequiredScopes: []string{\"greeting.read\", \"greeting.write\"}})",
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

func TestGenerateResolvesAuthPolicyMethodThenServiceThenFailsClosed(t *testing.T) {
	serviceOptions := &descriptorpb.ServiceOptions{}
	serviceOptions.ProtoReflect().SetUnknown(authPolicyUnknown(internal, "service.scope"))
	methodOptions := &descriptorpb.MethodOptions{}
	methodOptions.ProtoReflect().SetUnknown(authPolicyUnknown(public))
	unspecifiedOptions := &descriptorpb.MethodOptions{}
	unspecifiedOptions.ProtoReflect().SetUnknown(authPolicyUnknown(unspecified))

	file := &descriptorpb.FileDescriptorProto{
		Name:        new("example/v1/example.proto"),
		Package:     new("example.v1"),
		Options:     &descriptorpb.FileOptions{GoPackage: new("example.com/example/gen/examplev1;examplev1")},
		MessageType: []*descriptorpb.DescriptorProto{{Name: new("Request")}, {Name: new("Response")}},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: new("ExampleService"), Options: serviceOptions,
			Method: []*descriptorpb.MethodDescriptorProto{
				{Name: new("Inherited"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response")},
				{Name: new("Overridden"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response"), Options: methodOptions},
				{Name: new("ExplicitUnspecified"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response"), Options: unspecifiedOptions},
			},
		}},
	}
	plugin, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{ProtoFile: []*descriptorpb.FileDescriptorProto{file}, FileToGenerate: []string{file.GetName()}})
	if err != nil {
		t.Fatal(err)
	}
	if err := Generate(plugin); err != nil {
		t.Fatal(err)
	}
	content := plugin.Response().File[0].GetContent()
	for _, want := range []string{
		"case \"/example.v1.ExampleService/Inherited\":\n\t\tif err := i.verifier.Verify(ctx, AuthPolicy{Level: AuthLevelInternal, RequiredScopes: []string{\"service.scope\"}})",
		"case \"/example.v1.ExampleService/Overridden\":\n\t\treturn nil",
		"case \"/example.v1.ExampleService/ExplicitUnspecified\":\n\t\tif err := i.verifier.Verify(ctx, AuthPolicy{Level: AuthLevelAuthenticated})",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("generated code does not contain %q:\\n%s", want, content)
		}
	}
}

func TestAuthzProtoVersion(t *testing.T) {
	version := protowire.AppendTag(nil, authzProtoVersionField, protowire.VarintType)
	version = protowire.AppendVarint(version, supportedProtoVersion)
	got, ok := authzProtoVersion(version)
	if !ok || got != supportedProtoVersion {
		t.Fatalf("authzProtoVersion() = (%d, %t), want (%d, true)", got, ok, supportedProtoVersion)
	}
	if _, ok := authzProtoVersion(nil); ok {
		t.Fatal("authzProtoVersion() accepted a missing version")
	}
}

func TestSupportedProtoVersion(t *testing.T) {
	if got := SupportedProtoVersion(); got != supportedProtoVersion {
		t.Fatalf("SupportedProtoVersion() = %d, want %d", got, supportedProtoVersion)
	}
}

func TestGenerateRejectsIncompatibleAuthzProtoVersion(t *testing.T) {
	options := &descriptorpb.FileOptions{}
	version := protowire.AppendTag(nil, authzProtoVersionField, protowire.VarintType)
	options.ProtoReflect().SetUnknown(protowire.AppendVarint(version, supportedProtoVersion+1))
	options.GoPackage = new("example.com/authz/authzv1")
	file := &descriptorpb.FileDescriptorProto{Name: new(authzProtoFilename), Options: options}
	plugin, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{ProtoFile: []*descriptorpb.FileDescriptorProto{file}})
	if err != nil {
		t.Fatal(err)
	}
	err = Generate(plugin)
	if err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("Generate() error = %v, want incompatible version error", err)
	}
}

func authPolicyUnknown(level int, scopes ...string) []byte {
	policy := protowire.AppendTag(nil, 1, protowire.VarintType)
	policy = protowire.AppendVarint(policy, uint64(level))
	for _, scope := range scopes {
		policy = protowire.AppendTag(policy, 2, protowire.BytesType)
		policy = protowire.AppendString(policy, scope)
	}
	unknown := protowire.AppendTag(nil, authPolicyFieldNumber, protowire.BytesType)
	return protowire.AppendBytes(unknown, policy)
}
