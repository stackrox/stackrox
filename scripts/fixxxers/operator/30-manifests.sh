#!/bin/bash
set -euo pipefail

go mod tidy
# The above is to prevent this failure:
# + protoc-gen-go
# go: downloading google.golang.org/protobuf v1.36.5
# make/protogen.mk:192: *** Cached directory of scanner dependency not found, run 'go mod tidy'.  Stop.

make -C operator manifests
