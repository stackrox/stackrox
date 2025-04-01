#!/bin/bash
set -euo pipefail

go mod tidy
# The above is to prevent the following failure. Should be done by a script in parent directory
# but we're keeping it here to keep this script standalone.
# + protoc-gen-go
# go: downloading google.golang.org/protobuf v1.36.5
# make/protogen.mk:192: *** Cached directory of scanner dependency not found, run 'go mod tidy'.  Stop.

make -C operator manifests
