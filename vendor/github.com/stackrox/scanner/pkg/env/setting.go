package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/stackrox/rox/pkg/set"
)

// A Setting is a runtime configuration set using an environment variable.
type Setting interface {
	EnvVar() string
	Value() string
}

type settingOptions struct {
	defaul          string
	allowWithoutRox bool
}

type setting struct {
	envVar string
	settingOptions
}

// SettingOption is an interface that abstracts additional options for a setting (e.g., a default value).
type SettingOption interface {
	apply(*settingOptions)
}

type settingOptionFn func(*settingOptions)

func (f settingOptionFn) apply(so *settingOptions) {
	f(so)
}

var (
	// settings is the set which tracks all environment variables and ensures uniqueness.
	settings = set.NewStringSet()
)

// WithDefault sets the default value for a newly created setting.
func WithDefault(value string) SettingOption {
	return settingOptionFn(func(so *settingOptions) {
		so.defaul = value
	})
}

// AllowWithoutRox allows the setting to not be prefixed with ROX_.
func AllowWithoutRox() SettingOption {
	return settingOptionFn(func(so *settingOptions) {
		so.allowWithoutRox = true
	})
}

func registerSetting(envVar string, opts ...SettingOption) Setting {
	if !strings.HasPrefix(envVar, "ROX_") {
		panic(fmt.Sprintf("invalid env var: %s, must start with ROX_", envVar))
	}

	if !settings.Add(envVar) {
		panic(fmt.Sprintf("duplicate env var: %s", envVar))
	}

	s := &setting{
		envVar: envVar,
	}
	for _, opt := range opts {
		opt.apply(&s.settingOptions)
	}

	return s
}

func (s *setting) EnvVar() string {
	return s.envVar
}

func (s *setting) Value() string {
	val := os.Getenv(s.envVar)
	if val == "" && s.settingOptions.allowWithoutRox {
		// Remove ROX_ prefix.
		val = os.Getenv(s.envVar[4:])
	}
	if val != "" {
		return val
	}
	return s.defaul
}
