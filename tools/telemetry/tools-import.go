//go:build tools

package telemetry

// This file declares dependencies on tool for `go mod` purposes.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// for an explanation of the approach.

import (
	_ "golang.org/x/telemetry/cmd/gotelemetry"
)
