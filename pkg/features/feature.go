package features

import (
	"os"
	"strings"
)

type feature struct {
	envVar       string
	name         string
	defaultValue bool
	overridable  bool
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
	if !f.overridable {
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
