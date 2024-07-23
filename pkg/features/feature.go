package features

import (
	"os"
	"strings"

	"github.com/stackrox/rox/pkg/buildinfo"
)

type mode int

const (
	techPreview mode = 1 << iota
	enabled
	unchangeable

	devPreview mode = 0
	disabled   mode = 0
	// Const integer 1 for release, 0 for development build:
	releaseBuild       int  = 1 - (len(buildinfo.BuildFlavor)-len("release"))/4
	unchangeableInProd mode = unchangeable * mode(releaseBuild)
)

type feature struct {
	envVar string
	name   string
	mode   mode
}

func (f *feature) EnvVar() string {
	return f.envVar
}

func (f *feature) Name() string {
	return f.name
}

func (f *feature) Default() bool {
	return f.mode&enabled != 0
}

func (f *feature) Enabled() bool {
	if f.mode&unchangeable != 0 {
		return f.Default()
	}

	switch strings.ToLower(os.Getenv(f.envVar)) {
	case "false":
		return false
	case "true":
		return true
	default:
		return f.Default()
	}
}

func (f *feature) Stage() string {
	if f.mode&techPreview != 0 {
		return "tech-preview"
	}
	return "dev-preview"
}
