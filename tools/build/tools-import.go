//go:build tools

package tools

// This file declares dependencies on build tools for `go mod` purposes.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

import (
	// Build tool dependencies, not used anywhere in the code.
	_ "github.com/stackrox/ossls"
)
