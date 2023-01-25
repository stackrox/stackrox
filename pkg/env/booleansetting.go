package env

import (
	"os"
	"strconv"
)

// BooleanSetting represents an environment variable which should be parsed into a boolean
type BooleanSetting struct {
	changeable     bool
	envVar         string
	defaultBoolean bool
}

// EnvVar returns the string name of the environment variable
func (d *BooleanSetting) EnvVar() string {
	return d.envVar
}

// Setting returns the string form of the boolean environment variable
func (d *BooleanSetting) Setting() string {
	return strconv.FormatBool(d.BooleanSetting())
}

// BooleanSetting returns the bool object represented by the environment variable
func (d *BooleanSetting) BooleanSetting() bool {
	if !d.changeable {
		return d.defaultBoolean
	}
	val := os.Getenv(d.envVar)
	if val == "" {
		return d.defaultBoolean
	}
	v, err := strconv.ParseBool(val)
	return v && err == nil
}

// RegisterBooleanSetting globally registers and returns a new boolean setting.
func RegisterBooleanSetting(envVar string, defaultBoolean bool) *BooleanSetting {
	s := &BooleanSetting{
		changeable:     true,
		envVar:         envVar,
		defaultBoolean: defaultBoolean,
	}

	Settings[s.EnvVar()] = s
	return s
}

// RegisterUnchangeableBooleanSetting allows for very little code to change, but for the env var to not listen to the env var
func RegisterUnchangeableBooleanSetting(envVar string, defaultBoolean bool) *BooleanSetting {
	s := &BooleanSetting{
		changeable:     false,
		envVar:         envVar,
		defaultBoolean: defaultBoolean,
	}

	Settings[s.EnvVar()] = s
	return s
}
