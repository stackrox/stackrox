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
	return v
}

// RegisterIntegerSetting globally registers and returns a new integer setting.
func RegisterIntegerSetting(envVar string, defaultValue int) *IntegerSetting {
	s := &IntegerSetting{
		envVar:       envVar,
		defaultValue: defaultValue,
	}

	Settings[s.EnvVar()] = s
	return s
}
