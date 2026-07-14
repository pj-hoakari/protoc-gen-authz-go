// Package options exposes the embedded custom-option schema to the plugin.
package options

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pj-hoakari/protoc-gen-authz-go/proto/authz/v1"
)

// Proto is the custom option definition bundled into protoc-gen-authz-go.
var Proto = authzv1schema.OptionsProto()

// ProtoPath materializes the bundled schema in the user's cache directory and
// returns the include root suitable for protoc's -I option.
func ProtoPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache directory: %w", err)
	}
	root := filepath.Join(cacheDir, "protoc-gen-authz-go", "v1")
	if _, err := WriteProto(root); err != nil {
		return "", err
	}
	return root, nil
}

// WriteProto writes the bundled schema below root and returns its filename.
// The resulting layout is root/authz/v1/options.proto, so root can be passed
// to protoc as an include directory.
func WriteProto(root string) (string, error) {
	filename := filepath.Join(root, "authz", "v1", "options.proto")
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return "", fmt.Errorf("create authz proto directory: %w", err)
	}
	current, err := os.ReadFile(filename)
	if err == nil && string(current) == string(Proto) {
		return filename, nil
	}
	if err := os.WriteFile(filename, Proto, 0o644); err != nil {
		return "", fmt.Errorf("write authz proto: %w", err)
	}
	return filename, nil
}
