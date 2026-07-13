package main

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"

	"github.com/pj-hoakari/protoc-gen-authz-go/internal/generator"
	"github.com/pj-hoakari/protoc-gen-authz-go/internal/options"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--proto-path" {
		path, err := options.ProtoPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "protoc-gen-authz-go:", err)
			os.Exit(1)
		}
		fmt.Println(path)
		return
	}
	options := protogen.Options{}
	options.Run(generator.Generate)
}
