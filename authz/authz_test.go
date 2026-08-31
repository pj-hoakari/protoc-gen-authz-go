package authz

import (
	"errors"
	"strings"
	"testing"
)

func TestLevelString(t *testing.T) {
	for _, testCase := range []struct {
		level Level
		want  string
	}{
		{LevelUnspecified, "unspecified"},
		{LevelPublic, "public"},
		{LevelAuthenticated, "authenticated"},
		{LevelInternal, "internal"},
		{Level(7), "Level(7)"},
	} {
		if got := testCase.level.String(); got != testCase.want {
			t.Errorf("Level(%d).String() = %q, want %q", int(testCase.level), got, testCase.want)
		}
	}
}

func TestPoliciesLookup(t *testing.T) {
	policies := Policies{
		"/example.v1.ExampleService/Public": {Level: LevelPublic},
		"/example.v1.ExampleService/Scoped": {Level: LevelAuthenticated, RequiredScopes: []string{"example.read"}, TokenUses: []string{"access"}},
	}

	policy, ok := policies.Lookup("/example.v1.ExampleService/Scoped")
	if !ok {
		t.Fatal("Lookup() did not find a declared procedure")
	}
	if policy.Level != LevelAuthenticated {
		t.Errorf("Lookup() level = %v, want %v", policy.Level, LevelAuthenticated)
	}
	if len(policy.RequiredScopes) != 1 || policy.RequiredScopes[0] != "example.read" {
		t.Errorf("Lookup() required scopes = %v, want [example.read]", policy.RequiredScopes)
	}
	if len(policy.TokenUses) != 1 || policy.TokenUses[0] != "access" {
		t.Errorf("Lookup() token uses = %v, want [access]", policy.TokenUses)
	}

	if _, ok := policies.Lookup("/example.v1.ExampleService/Unknown"); ok {
		t.Error("Lookup() found an undeclared procedure")
	}

	var empty Policies
	if _, ok := empty.Lookup("/example.v1.ExampleService/Public"); ok {
		t.Error("Lookup() on a nil table found a procedure")
	}
}

func TestMerge(t *testing.T) {
	first := Policies{"/example.v1.First/Get": {Level: LevelPublic}}
	second := Policies{"/example.v1.Second/Get": {Level: LevelInternal}}

	merged, err := Merge(first, second)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 {
		t.Fatalf("Merge() returned %d procedures, want 2", len(merged))
	}
	if policy, ok := merged.Lookup("/example.v1.Second/Get"); !ok || policy.Level != LevelInternal {
		t.Errorf("Merge() lost /example.v1.Second/Get: (%v, %t)", policy, ok)
	}

	merged["/example.v1.First/Get"] = Policy{Level: LevelInternal}
	if policy, _ := first.Lookup("/example.v1.First/Get"); policy.Level != LevelPublic {
		t.Error("Merge() returned a table aliasing its input")
	}
}

func TestMergeWithoutTables(t *testing.T) {
	merged, err := Merge()
	if err != nil {
		t.Fatal(err)
	}
	if merged == nil {
		t.Fatal("Merge() returned a nil table")
	}
	if len(merged) != 0 {
		t.Fatalf("Merge() returned %d procedures, want 0", len(merged))
	}
}

func TestMergeRejectsDuplicateProcedure(t *testing.T) {
	first := Policies{"/example.v1.ExampleService/Get": {Level: LevelPublic}}
	second := Policies{"/example.v1.ExampleService/Get": {Level: LevelInternal}}

	merged, err := Merge(first, second)
	if !errors.Is(err, ErrDuplicateProcedure) {
		t.Fatalf("Merge() error = %v, want ErrDuplicateProcedure", err)
	}
	if merged != nil {
		t.Errorf("Merge() returned %v, want nil on error", merged)
	}
	if !strings.Contains(err.Error(), "/example.v1.ExampleService/Get") {
		t.Errorf("Merge() error %q does not name the duplicate procedure", err)
	}
}
