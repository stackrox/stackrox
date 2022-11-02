package env

import (
	"strconv"
)

// BooleanSetting represents an environment variable which should be parsed into a boolean
type BooleanSetting interface {
	Setting
	Enabled() bool
}

type booleanSetting struct {
	Setting
}

// Enabled returns the bool object represented by the environment variable
func (s *booleanSetting) Enabled() bool {
	v, err := strconv.ParseBool(s.Value())
	return v && err == nil
}

// RegisterBooleanSetting globally registers and returns a new boolean setting.
func RegisterBooleanSetting(envVar string, defaul bool, opts ...SettingOption) BooleanSetting {
	return &booleanSetting{
		Setting: registerSetting(envVar, append(opts, WithDefault(strconv.FormatBool(defaul)))...),
	}
}
