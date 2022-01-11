package env

import (
	"fmt"
	"os"
	"strconv"
)

// IntegerSetting represents an environment variable which should be parsed into an integer
type IntegerSetting struct {
	envVar string
	defaultValue int
}

// EnvVar returns the string name of the environment variable
func (d *IntegerSetting) EnvVar() string {
	return d.envVar
}

// Setting returns the string form of the boolean environment variable
func (d *IntegerSetting) Setting() string {
	return fmt.Sprintf("%d", d.IntegerSetting())
}

// IntegerSetting returns the integer object represented by the environment variable
func (d *IntegerSetting) IntegerSetting() int {
	val := os.Getenv(d.envVar)
	if val == "" {
		return d.defaultValue
	}
	v, err := strconv.Atoi(val)
	if err != nil {
		return d.defaultValue
	}
	return v
}

// RegisterIntegerSetting globally registers and returns a new integer setting.
func RegisterIntegerSetting(envVar string, defaultValue int) *IntegerSetting {
	s := &IntegerSetting{
		envVar:         envVar,
		defaultValue: defaultValue,
	}

	Settings[s.EnvVar()] = s
	return s
}
