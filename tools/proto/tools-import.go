//go:build tools

package tools

// This file declares dependencies on tool for `go mod` purposes.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// for an explanation of the approach.

import (
	// Tool dependencies, not used anywhere in the code.
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/favadi/protoc-go-inject-tag"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
)
