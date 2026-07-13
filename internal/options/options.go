// Package options exposes the embedded custom-option schema to the plugin.
package options

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// Proto is the custom option definition bundled into protoc-gen-authz-go.
//
//go:embed authz.proto
var Proto []byte

// ProtoPath materializes the bundled schema in the user's cache directory and
// returns the include root suitable for protoc's -I option. Callers import it
// as "authz/v1/authz.proto"; no schema file needs to be tracked by a consumer.
func ProtoPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache directory: %w", err)
	}
	root := filepath.Join(cacheDir, "protoc-gen-authz-go", "v1")
	filename := filepath.Join(root, "authz", "v1", "authz.proto")
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return "", fmt.Errorf("create authz proto cache: %w", err)
	}
	current, err := os.ReadFile(filename)
	if err == nil && string(current) == string(Proto) {
		return root, nil
	}
	if err := os.WriteFile(filename, Proto, 0o644); err != nil {
		return "", fmt.Errorf("write embedded authz proto: %w", err)
	}
	return root, nil
}
