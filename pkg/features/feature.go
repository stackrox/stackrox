package features

import (
	"os"
	"strings"

	"github.com/stackrox/rox/pkg/buildinfo"
)

type feature struct {
	envVar       string
	name         string
	defaultValue bool
	unchangeable bool
}

func (f *feature) EnvVar() string {
	return f.envVar
}

func (f *feature) Name() string {
	return f.name
}

func (f *feature) Default() bool {
	return f.defaultValue
}

func (f *feature) Enabled() bool {
	if buildinfo.ReleaseBuild && f.unchangeable {
		return f.defaultValue
	}

	switch strings.ToLower(os.Getenv(f.envVar)) {
	case "false":
		return false
	case "true":
		return true
	default:
		return f.defaultValue
	}
}
