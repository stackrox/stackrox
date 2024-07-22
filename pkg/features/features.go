// Package features helps enable or disable features.
package features

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
)

// A FeatureFlag is a product behavior that can be enabled or disabled using an
// environment variable.
type FeatureFlag interface {
	Name() string
	EnvVar() string
	Enabled() bool
	Default() bool
	Stage() string
	LoggingContext() interface{}
}

var (
	// Flags contains all defined FeatureFlags by name.
	Flags = make(map[string]FeatureFlag)

	log = logging.LoggerForModule()
)

// registerFeature registers and returns a new feature flag, configured with the
// provided options.
func registerFeature(name, envVar string, options ...option) FeatureFlag {
	if !strings.HasPrefix(envVar, "ROX_") {
		panic(fmt.Sprintf("invalid env var: %s, must start with ROX_", envVar))
	}
	f := &feature{
		name:   name,
		envVar: envVar,
	}
	for _, o := range options {
		o(f)
	}
	Flags[f.envVar] = f
	return f
}

// LogFeatureFlags logs the global state of all features flags.
func LogFeatureFlags() {
	var context []interface{}
	for _, flag := range Flags {
		context = append(context, flag.LoggingContext())
	}
	log.Infow("Feature flags", context...)
}
