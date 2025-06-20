package env

import (
	"fmt"
	"os"
	"strconv"
)

// IntegerSetting represents an environment variable which should be parsed into an integer
type IntegerSetting struct {
	envVar       string
	defaultValue int
	intSettingOptions
}

// EnvVar returns the string name of the environment variable
func (s *IntegerSetting) EnvVar() string {
	return s.envVar
}

// DefaultValue returns the default vaule for the setting
func (s *IntegerSetting) DefaultValue() int {
	return s.defaultValue
}

// Setting returns the string form of the integer environment variable
func (s *IntegerSetting) Setting() string {
	return fmt.Sprintf("%d", s.IntegerSetting())
}

// IntegerSetting returns the integer object represented by the environment variable
func (s *IntegerSetting) IntegerSetting() int {
	val := os.Getenv(s.envVar)
	if val == "" {
		return s.defaultValue
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return s.defaultValue
	}
	if v < s.minimumValue && s.minSet {
		return s.defaultValue
	}
	if v > s.maximumValue && s.maxSet {
		return s.defaultValue
	}
	return v
}

// RegisterIntegerSetting globally registers and returns a new integer setting.
func RegisterIntegerSetting(envVar string, defaultValue int, opts ...IntegerSettingOption) *IntegerSetting {
	s := &IntegerSetting{
		envVar:       envVar,
		defaultValue: defaultValue,
	}
	for _, opt := range opts {
		opt.apply(&s.intSettingOptions)
	}
	if s.defaultValue < s.minimumValue && s.minSet {
		panic(fmt.Errorf("programmer error: default value for %s must be at least %d",
			s.EnvVar(), s.minimumValue))
	}
	if s.defaultValue > s.maximumValue && s.maxSet {
		panic(fmt.Errorf("programmer error: default value for %s must be maximally %d",
			s.EnvVar(), s.maximumValue))
	}
	if s.minSet && s.maxSet && s.minimumValue > s.maximumValue {
		panic(fmt.Errorf("programmer error: incorrect validation config for %s: "+
			"minimum value %d must be smaller or equal to maximum value %d",
			s.EnvVar(), s.minimumValue, s.maximumValue))
	}

	Settings[s.EnvVar()] = s
	return s
}

type intSettingOptions struct {
	minimumValue int
	minSet       bool
	maximumValue int
	maxSet       bool
}

type intSettingOptionFn func(*intSettingOptions)

func (f intSettingOptionFn) apply(so *intSettingOptions) {
	f(so)
}

// IntegerSettingOption is an interface that abstracts additional options for a setting (e.g., a default value).
type IntegerSettingOption interface {
	apply(*intSettingOptions)
}

// WithMinimum specifies the minimum allowed value that passes the validation.
func WithMinimum(val int) IntegerSettingOption {
	return intSettingOptionFn(func(so *intSettingOptions) {
		so.minimumValue = val
		so.minSet = true
	})
}

// WithMaximum specifies the minimum allowed value that passes the validation.
func WithMaximum(val int) IntegerSettingOption {
	return intSettingOptionFn(func(so *intSettingOptions) {
		so.maximumValue = val
		so.maxSet = true
	})
}
