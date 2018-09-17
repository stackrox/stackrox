// Package features helps enable or disable features.
package features

import (
	"strings"
)

// A Feature is a product behavior that can be enabled or disabled.
type Feature interface {
	Name() string
	Enabled() bool
}

// A FeatureFlag is an environment variable configuration that can be
// provided to control whether one or more features is enabled.
type FeatureFlag interface {
	Name() string
	EnvVar() string
}

var (
	// Features contains all defined Features by name.
	Features = map[string]Feature{
		RuntimePolicies.Name(): RuntimePolicies,
	}

	// Flags contains all defined FeatureFlags by name.
	Flags = map[string]FeatureFlag{
		RuntimePolicies.Name(): RuntimePolicies,
	}
)

// These are the defined features and feature flags.
// Example usage: features.MTLS.Enabled().
var (
	RuntimePolicies = runtimePolicies{}
)

func isEnabled(val string, defaultValue bool) bool {
	switch strings.ToLower(val) {
	case "false":
		return false
	case "true":
		return true
	default:
		return defaultValue
	}

}
