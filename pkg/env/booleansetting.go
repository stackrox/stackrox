package env

import (
	"os"
	"strconv"
)

// BooleanSetting represents an environment variable which should be parsed into a boolean.
type BooleanSetting struct {
	envVar         string
	defaultBoolean bool
	unchangeable   bool
}

// EnvVar returns the string name of the environment variable.
func (d *BooleanSetting) EnvVar() string {
	return d.envVar
}

// Setting returns the string form of the boolean environment variable.
func (d *BooleanSetting) Setting() string {
	return strconv.FormatBool(d.BooleanSetting())
}

// DefaultBooleanSetting returns the bool object that is returned if the environment variable is empty.
func (d *BooleanSetting) DefaultBooleanSetting() bool {
	return d.defaultBoolean
}

// BooleanSetting returns the bool object represented by the environment variable.
func (d *BooleanSetting) BooleanSetting() bool {
	if d.unchangeable {
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
		envVar:         envVar,
		defaultBoolean: defaultBoolean,
	}

	Settings[s.EnvVar()] = s
	return s
}

// RegisterPermanentBooleanSetting global registers and returns a new boolean setting that is always locked to the default value.
func RegisterPermanentBooleanSetting(envVar string, defaultBoolean bool) *BooleanSetting {
	s := &BooleanSetting{
		envVar:         envVar,
		defaultBoolean: defaultBoolean,
		unchangeable:   true,
	}

	Settings[s.EnvVar()] = s
	return s
}
