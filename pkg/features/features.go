// Package features helps enable or disable features.
package features

import (
	"fmt"
	"strings"
)

// A FeatureFlag is a product behavior that can be enabled or disabled using an environment variable.
type FeatureFlag interface {
	Name() string
	EnvVar() string
	Enabled() bool
	Default() bool
	Stage() string
}

var (
	// Flags contains all defined FeatureFlags by name.
	Flags = make(map[string]FeatureFlag)
)

// registerFeature global registers and returns a new feature flag that can be overridden from the default state regardless of build.
func registerFeature(m mode, name, envVar string) FeatureFlag {
	if !strings.HasPrefix(envVar, "ROX_") {
		panic(fmt.Sprintf("invalid env var: %s, must start with ROX_", envVar))
	}
	f := &feature{
		name:   name,
		envVar: envVar,
		mode:   m,
	}
	Flags[f.envVar] = f
	return f
}
