// Package features helps enable or disable features.
package features

import (
	"github.com/stackrox/scanner/pkg/env"
)

// A FeatureFlag is a product behavior that can be enabled or disabled using an environment variable.
type FeatureFlag interface {
	Name() string
	EnvVar() string
	Enabled() bool
}

type feature struct {
	env.BooleanSetting
	name string
}

func (f *feature) Name() string {
	return f.name
}

func registerFeature(name, envVar string, defaul bool) FeatureFlag {
	return &feature{
		name:           name,
		BooleanSetting: env.RegisterBooleanSetting(envVar, defaul),
	}
}
