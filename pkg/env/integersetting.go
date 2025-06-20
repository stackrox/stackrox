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

	// Optional validation of the value
	minimumValue int
	minSet       bool
	maximumValue int
	maxSet       bool
}

// EnvVar returns the string name of the environment variable
func (s *IntegerSetting) EnvVar() string {
	return s.envVar
}

// DefaultValue returns the default value for the setting
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
	if err != nil || (s.minSet && v < s.minimumValue) || (s.maxSet && v > s.maximumValue) {
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

// WithMinimum specifies the minimal allowed value that passes the validation.
func (s *IntegerSetting) WithMinimum(min int) *IntegerSetting {
	if s.defaultValue < min {
		panic(fmt.Errorf(
			"programmer error: default %d < minimum %d for %s",
			s.defaultValue, min, s.envVar,
		))
	}
	s.minSet = true
	s.minimumValue = min
	if s.maxSet && s.minimumValue > s.maximumValue {
		panic(fmt.Errorf("programmer error: incorrect validation config for %s: "+
			"minimum value %d must be smaller or equal to maximum value %d",
			s.EnvVar(), s.minimumValue, s.maximumValue))
	}
	return s
}

// WithMaximum specifies the maximal allowed value that passes the validation.
func (s *IntegerSetting) WithMaximum(max int) *IntegerSetting {
	if s.defaultValue > max {
		panic(fmt.Errorf(
			"programmer error: default %d > maximum %d for %s",
			s.defaultValue, max, s.envVar,
		))
	}
	s.maxSet = true
	s.maximumValue = max
	if s.minSet && s.minimumValue > s.maximumValue {
		panic(fmt.Errorf("programmer error: incorrect validation config for %s: "+
			"minimum value %d must be smaller or equal to maximum value %d",
			s.EnvVar(), s.minimumValue, s.maximumValue))
	}
	return s
}
