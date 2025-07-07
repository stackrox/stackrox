package env

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// IntegerSetting represents an environment variable which should be parsed into an integer
type IntegerSetting struct {
	envVar       string
	defaultValue int

	// Optional validation of the value
	minimumValue int
	maximumValue int
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
	if err != nil || (v < s.minimumValue) || (v > s.maximumValue) {
		return s.defaultValue
	}
	return v
}

// RegisterIntegerSetting globally registers and returns a new integer setting.
func RegisterIntegerSetting(envVar string, defaultValue int) *IntegerSetting {
	s := &IntegerSetting{
		envVar:       envVar,
		defaultValue: defaultValue,
		minimumValue: math.MinInt,
		maximumValue: math.MaxInt,
	}
	Settings[s.EnvVar()] = s
	return s
}

// WithMinimum specifies the minimal allowed value that passes the validation.
func (s *IntegerSetting) WithMinimum(min int) *IntegerSetting {
	s.minimumValue = min
	return s.mustValidate()
}

// WithMaximum specifies the maximal allowed value that passes the validation.
func (s *IntegerSetting) WithMaximum(max int) *IntegerSetting {
	s.maximumValue = max
	return s.mustValidate()
}

func (s *IntegerSetting) mustValidate() *IntegerSetting {
	if s.defaultValue < s.minimumValue {
		panic(fmt.Errorf("programmer error: default value %d is smaller than the minimum %d for %q",
			s.defaultValue, s.minimumValue, s.envVar,
		).Error())
	}
	if s.defaultValue > s.maximumValue {
		panic(fmt.Errorf("programmer error: default value %d is larger than the maximum %d for %q",
			s.defaultValue, s.maximumValue, s.envVar,
		).Error())
	}
	if s.minimumValue > s.maximumValue {
		panic(fmt.Errorf("programmer error: incorrect integer bounds for %q: "+
			"minimum value %d must be smaller or equal to maximum value %d",
			s.EnvVar(), s.minimumValue, s.maximumValue,
		).Error())
	}
	return s
}
