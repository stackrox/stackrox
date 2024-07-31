package features

import (
	"os"
	"strings"
)

const (
	devPreviewString  = "dev-preview"
	techPreviewString = "tech-preview"
)

// FlagSource provides information on the origin of a flag configuration.
type FlagSource int

const (
	// The flag value is its default value (no override).
	FlagSource_DEFAULT FlagSource = 0
	// The flag value comes from the central source (central override).
	FlagSource_CENTRAL FlagSource = 1
	// The flag value comes from the component environment (environment variable override).
	FlagSource_ENVIRON FlagSource = 2
)

type feature struct {
	envVar       string
	name         string
	defaultValue bool
	unchangeable bool
	techPreview  bool
	value        bool
	source       FlagSource
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
	if f.unchangeable {
		return f.defaultValue
	}

	switch strings.ToLower(os.Getenv(f.envVar)) {
	case "true":
		return true
	case "false":
		return false
	}
	return f.value
}

func (f *feature) Set(value bool, flagSource FlagSource) {
	if !f.unchangeable && f.source < flagSource {
		f.value = value
		f.source = flagSource
	}
}

func (f *feature) Stage() string {
	if f.techPreview {
		return techPreviewString
	}
	return devPreviewString
}
