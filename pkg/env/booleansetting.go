package env

import (
	"os"
	"strconv"
)

// BooleanSetting represents an environment variable which should be parsed into a boolean
type BooleanSetting struct {
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
	val := os.Getenv(d.envVar)
	if val == "" {
		return d.defaultBoolean
	}
	v, err := strconv.ParseBool(val)
	return v && err == nil
}

func registerBooleanSetting(envVar string, defaultBoolean bool) *BooleanSetting {
	s := &BooleanSetting{
		envVar:         envVar,
		defaultBoolean: defaultBoolean,
	}

	Settings[s.EnvVar()] = s
	return s
}
