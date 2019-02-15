// Package features helps enable or disable features.
package features

// A Feature is a product behavior that can be enabled or disabled.
type Feature interface {
	Name() string
	EnvVar() string
	Enabled() bool
}

// A FeatureFlag is an environment variable configuration that can be
// provided to control whether one or more features is enabled.
type FeatureFlag = Feature

var (
	// Features contains all defined Features by name.
	Features = make(map[string]Feature)

	// Flags contains all defined FeatureFlags by name.
	Flags = make(map[string]FeatureFlag)
)

func registerFeature(name, envVar string, defaultValue bool) Feature {
	f := &feature{
		name:         name,
		envVar:       envVar,
		defaultValue: defaultValue,
	}
	Features[f.Name()] = f
	Flags[f.Name()] = f
	return f
}
