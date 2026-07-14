// Package authzv1schema provides the authz protobuf schema to this module.
package authzv1schema

import _ "embed"

// optionsProto is embedded so the installed CLI can provide the schema without
// requiring the repository checkout to be present.
//
//go:embed options.proto
var optionsProto []byte

// OptionsProto returns a copy of the bundled authz option schema.
func OptionsProto() []byte {
	return append([]byte(nil), optionsProto...)
}
