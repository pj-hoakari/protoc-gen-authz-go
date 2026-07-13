package options

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteProto(t *testing.T) {
	root := t.TempDir()
	filename, err := WriteProto(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "authz", "v1", "authz.proto")
	if filename != want {
		t.Fatalf("WriteProto() returned %q, want %q", filename, want)
	}
	got, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(Proto) {
		t.Fatal("written proto does not match embedded proto")
	}
}
