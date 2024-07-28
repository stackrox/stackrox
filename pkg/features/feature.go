package features

import (
	"os"
	"strings"
)

const (
	devPreviewString  = "dev-preview"
	techPreviewString = "tech-preview"
	releasedString    = "released"
)

type feature struct {
	envVar       string
	name         string
	released     bool
	unchangeable bool
	techPreview  bool
}

func (f *feature) EnvVar() string {
	return f.envVar
}

func (f *feature) Name() string {
	return f.name
}

func (f *feature) Released() bool {
	return f.released
}

func (f *feature) Enabled() bool {
	if f.unchangeable {
		return f.released
	}

	switch strings.ToLower(os.Getenv(f.envVar)) {
	case "false":
		return false
	case "true":
		return true
	default:
		return f.released
	}
}

func (f *feature) Stage() string {
	switch {
	case f.techPreview:
		return techPreviewString
	case f.released:
		// Allow tech-preview features to be enabled by default for backward
		// compatibility.
		return releasedString
	default:
		return devPreviewString
	}
}
