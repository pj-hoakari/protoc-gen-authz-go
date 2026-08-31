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

	authzv1 "github.com/pj-hoakari/protoc-gen-authz-go/authz/v1"
)

func TestGenerateWritesPolicyTableAndFailsClosed(t *testing.T) {
	publicOptions := &descriptorpb.MethodOptions{}
	publicOptions.ProtoReflect().SetUnknown(authPolicyUnknown(public))
	unspecifiedOptions := &descriptorpb.MethodOptions{}
	unspecifiedOptions.ProtoReflect().SetUnknown(authPolicyUnknown(unspecified))
	internalOptions := &descriptorpb.MethodOptions{}
	internalOptions.ProtoReflect().SetUnknown(authPolicyUnknown(internal, scopes("greeting.read", "greeting.write")))
	scopedOptions := &descriptorpb.MethodOptions{}
	scopedOptions.ProtoReflect().SetUnknown(authPolicyUnknown(authenticated, scopes("tenant.claim"), tokenUses("registration")))

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
				{Name: new("Scoped"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response"), Options: scopedOptions},
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
	if got, want := response.File[0].GetName(), "example.com/example/gen/examplev1/examplev1connect/example.authz.connect.go"; got != want {
		t.Errorf("generated file %q, want %q", got, want)
	}
	content := response.File[0].GetContent()
	if _, err := parser.ParseFile(token.NewFileSet(), response.File[0].GetName(), content, parser.AllErrors); err != nil {
		t.Fatalf("generated Go is invalid: %v\n%s", err, content)
	}
	for _, want := range []string{
		"package examplev1connect",
		`authz "github.com/pj-hoakari/protoc-gen-authz-go/authz"`,
		"// ExampleServicePolicies is the effective policy of every procedure of example.v1.ExampleService.",
		"var ExampleServicePolicies = authz.Policies{",
		"ExampleServicePublicProcedure:      {Level: authz.LevelPublic},",
		"ExampleServiceDefaultProcedure:     {Level: authz.LevelAuthenticated},",
		"ExampleServiceUnspecifiedProcedure: {Level: authz.LevelAuthenticated},",
		`ExampleServiceInternalProcedure:    {Level: authz.LevelInternal, RequiredScopes: []string{"greeting.read", "greeting.write"}},`,
		`ExampleServiceScopedProcedure:      {Level: authz.LevelAuthenticated, RequiredScopes: []string{"tenant.claim"}, TokenUses: []string{"registration"}},`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("generated code does not contain %q:\n%s", want, content)
		}
	}
	for _, unwanted := range []string{
		"connectrpc.com/connect",
		"net/http",
		"\"context\"",
		"Verifier",
		"AuthzInterceptor",
		"HandlerWithAuthz",
	} {
		if strings.Contains(content, unwanted) {
			t.Errorf("generated code still contains %q:\n%s", unwanted, content)
		}
	}
	if got := strings.Index(content, "ExampleServiceDefaultProcedure"); got < strings.Index(content, "ExampleServicePublicProcedure") {
		t.Errorf("generated table does not preserve method declaration order:\n%s", content)
	}
}

func TestGenerateResolvesAuthPolicyMethodThenServiceThenFailsClosed(t *testing.T) {
	serviceOptions := &descriptorpb.ServiceOptions{}
	serviceOptions.ProtoReflect().SetUnknown(authPolicyUnknown(internal, scopes("service.scope"), tokenUses("service")))
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
		`ExampleServiceInheritedProcedure:           {Level: authz.LevelInternal, RequiredScopes: []string{"service.scope"}, TokenUses: []string{"service"}},`,
		"ExampleServiceOverriddenProcedure:          {Level: authz.LevelPublic},",
		"ExampleServiceExplicitUnspecifiedProcedure: {Level: authz.LevelAuthenticated},",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("generated code does not contain %q:\n%s", want, content)
		}
	}
}

func TestGenerateRejectsPublicWithScopesOrTokenUses(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		unknown []byte
		want    string
	}{
		{"required_scopes", authPolicyUnknown(public, scopes("greeting.read")), "must not declare required_scopes"},
		{"token_uses", authPolicyUnknown(public, tokenUses("access")), "must not declare token_uses"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			methodOptions := &descriptorpb.MethodOptions{}
			methodOptions.ProtoReflect().SetUnknown(testCase.unknown)
			file := &descriptorpb.FileDescriptorProto{
				Name:        new("example/v1/example.proto"),
				Package:     new("example.v1"),
				Options:     &descriptorpb.FileOptions{GoPackage: new("example.com/example/gen/examplev1;examplev1")},
				MessageType: []*descriptorpb.DescriptorProto{{Name: new("Request")}, {Name: new("Response")}},
				Service: []*descriptorpb.ServiceDescriptorProto{{
					Name: new("ExampleService"),
					Method: []*descriptorpb.MethodDescriptorProto{
						{Name: new("Public"), InputType: new(".example.v1.Request"), OutputType: new(".example.v1.Response"), Options: methodOptions},
					},
				}},
			}
			plugin, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{ProtoFile: []*descriptorpb.FileDescriptorProto{file}, FileToGenerate: []string{file.GetName()}})
			if err != nil {
				t.Fatal(err)
			}
			err = Generate(plugin)
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("Generate() error = %v, want an error containing %q", err, testCase.want)
			}
		})
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
	if supportedProtoVersion != 2 {
		t.Fatalf("supportedProtoVersion = %d, want 2", supportedProtoVersion)
	}
}

// TestSupportedProtoVersionMatchesBundledSchema pins the plugin to the schema
// shipped in this module, so bumping only one of the two fails here.
func TestSupportedProtoVersionMatchesBundledSchema(t *testing.T) {
	options, ok := authzv1.File_authz_v1_options_proto.Options().(*descriptorpb.FileOptions)
	if !ok {
		t.Fatal("bundled schema has no file options")
	}
	version, ok := proto.GetExtension(options, authzv1.E_AuthzProtoVersion).(uint32)
	if !ok {
		t.Fatal("bundled schema does not declare authz_proto_version")
	}
	if int(version) != supportedProtoVersion {
		t.Fatalf("bundled schema declares version %d, plugin supports %d", version, supportedProtoVersion)
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

type policyField func([]byte) []byte

func scopes(values ...string) policyField    { return repeatedString(2, values) }
func tokenUses(values ...string) policyField { return repeatedString(3, values) }

func repeatedString(number protowire.Number, values []string) policyField {
	return func(policy []byte) []byte {
		for _, value := range values {
			policy = protowire.AppendTag(policy, number, protowire.BytesType)
			policy = protowire.AppendString(policy, value)
		}
		return policy
	}
}

func authPolicyUnknown(level int, fields ...policyField) []byte {
	policy := protowire.AppendTag(nil, 1, protowire.VarintType)
	policy = protowire.AppendVarint(policy, uint64(level))
	for _, field := range fields {
		policy = field(policy)
	}
	unknown := protowire.AppendTag(nil, authPolicyFieldNumber, protowire.BytesType)
	return protowire.AppendBytes(unknown, policy)
}
