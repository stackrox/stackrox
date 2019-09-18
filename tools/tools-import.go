// +build tools

package tools

// This file declares dependencies on tool for `go mod` purposes.
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
// for an explanation of the approach.

import (
	// Tool dependencies, not used anywheree in the code.
	_ "github.com/gobuffalo/packr/packr"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/mailru/easyjson/easyjson"
	_ "github.com/mattn/goveralls"
	_ "github.com/mauricelam/genny"
	_ "github.com/nilslice/protolock"
	_ "golang.org/x/lint/golint"
	_ "golang.org/x/tools/cmd/goimports"
	_ "golang.org/x/tools/cmd/stringer"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
