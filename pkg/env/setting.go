package env

import (
	"fmt"
	"os"
)

// A Setting is a runtime configuration set using an environment variable.
type Setting interface {
	EnvVar() string
	Setting() string
}

type settingOptions struct {
	defaultValue string
	allowEmpty   bool
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

// WithDefault sets the default value for a newly created setting.
func WithDefault(value string) SettingOption {
	return settingOptionFn(func(so *settingOptions) {
		so.defaultValue = value
	})
}

// AllowEmpty specifies that an empty value (if explicitly set) will be respected, even if a non-empty default is
// defined.
func AllowEmpty() SettingOption {
	return settingOptionFn(func(so *settingOptions) {
		so.allowEmpty = true
	})
}

// NewSetting creates a new setting for the given environment variable with the given options
func NewSetting(envVar string, opts ...SettingOption) Setting {
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

func (s *setting) Setting() string {
	val, ok := os.LookupEnv(s.envVar)
	if val != "" || (ok && s.allowEmpty) {
		return val
	}
	return s.defaultValue
}

// CombineSetting returns the a string in the form KEY=VALUE based on the Setting
func CombineSetting(s Setting) string {
	return Combine(s.EnvVar(), s.Setting())
}

// Combine concatenates a key and value into the environment variable format
func Combine(k, v string) string {
	return fmt.Sprintf("%s=%s", k, v)
}
