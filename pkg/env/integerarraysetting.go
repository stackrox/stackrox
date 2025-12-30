package env

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// IntegerArraySetting represents an environment variable which should be parsed into a slice of integers
type IntegerArraySetting struct {
	envVar       string
	defaultValue []int

	// Optional validation of the values
	minValue  int
	maxValue  int
	minLength int
	maxLength int
}

// EnvVar returns the string name of the environment variable
func (s *IntegerArraySetting) EnvVar() string {
	return s.envVar
}

// DefaultValue returns the default value for the setting
func (s *IntegerArraySetting) DefaultValue() []int {
	return s.defaultValue
}

// Setting returns the string form of the integer array environment variable
func (s *IntegerArraySetting) Setting() string {
	arr := s.IntegerArraySetting()
	strArr := make([]string, len(arr))
	for i, v := range arr {
		strArr[i] = strconv.Itoa(v)
	}
	return strings.Join(strArr, ",")
}

// IntegerArraySetting returns the integer slice represented by the environment variable
func (s *IntegerArraySetting) IntegerArraySetting() []int {
	val := os.Getenv(s.envVar)
	if val == "" {
		if s.minLength == 0 {
			// Empty string returns empty array (allows for 0-length arrays)
			return []int{}
		} else {
			return s.defaultValue
		}
	}

	// Split by comma and parse each element
	parts := strings.Split(val, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			// Skip empty parts (e.g., from trailing commas)
			continue
		}

		v, err := strconv.Atoi(trimmed)
		if err != nil {
			// Invalid format, return default
			return s.defaultValue
		}

		// Validate individual value
		if v < s.minValue || v > s.maxValue {
			// Out of bounds, return default
			return s.defaultValue
		}

		result = append(result, v)
	}

	// Validate array length
	if len(result) < s.minLength || len(result) > s.maxLength {
		return s.defaultValue
	}

	return result
}

// RegisterIntegerArraySetting globally registers and returns a new integer array setting.
func RegisterIntegerArraySetting(envVar string, defaultValue []int) *IntegerArraySetting {
	s := &IntegerArraySetting{
		envVar:       envVar,
		defaultValue: defaultValue,
		minValue:     math.MinInt,
		maxValue:     math.MaxInt,
		minLength:    0,
		maxLength:    math.MaxInt,
	}
	Settings[s.EnvVar()] = s
	return s
}

// WithMinimumValue specifies the minimal allowed value for each element in the array.
func (s *IntegerArraySetting) WithMinimumValue(min int) *IntegerArraySetting {
	s.minValue = min
	return s.mustValidate()
}

// WithMaximumValue specifies the maximal allowed value for each element in the array.
func (s *IntegerArraySetting) WithMaximumValue(max int) *IntegerArraySetting {
	s.maxValue = max
	return s.mustValidate()
}

// WithMinLength specifies the minimum allowed length of the array.
func (s *IntegerArraySetting) WithMinLength(min int) *IntegerArraySetting {
	s.minLength = min
	return s.mustValidate()
}

// WithMaxLength specifies the maximum allowed length of the array.
func (s *IntegerArraySetting) WithMaxLength(max int) *IntegerArraySetting {
	s.maxLength = max
	return s.mustValidate()
}

func (s *IntegerArraySetting) mustValidate() *IntegerArraySetting {
	// Validate bounds
	if s.minValue > s.maxValue {
		panic(fmt.Errorf("programmer error: incorrect integer bounds for %q: "+
			"minimum value %d must be smaller or equal to maximum value %d",
			s.EnvVar(), s.minValue, s.maxValue,
		).Error())
	}
	if s.minLength < 0 {
		panic(fmt.Errorf("programmer error: minimum length %d must be non-negative for %q",
			s.minLength, s.envVar,
		).Error())
	}
	if s.minLength > s.maxLength {
		panic(fmt.Errorf("programmer error: incorrect length bounds for %q: "+
			"minimum length %d must be smaller or equal to maximum length %d",
			s.EnvVar(), s.minLength, s.maxLength,
		).Error())
	}

	// Validate default value
	if len(s.defaultValue) < s.minLength {
		panic(fmt.Errorf("programmer error: default value length %d is smaller than minimum length %d for %q",
			len(s.defaultValue), s.minLength, s.envVar,
		).Error())
	}
	if len(s.defaultValue) > s.maxLength {
		panic(fmt.Errorf("programmer error: default value length %d is larger than maximum length %d for %q",
			len(s.defaultValue), s.maxLength, s.envVar,
		).Error())
	}

	// Validate each element in default value
	for i, v := range s.defaultValue {
		if v < s.minValue {
			panic(fmt.Errorf("programmer error: default value[%d]=%d is smaller than minimum %d for %q",
				i, v, s.minValue, s.envVar,
			).Error())
		}
		if v > s.maxValue {
			panic(fmt.Errorf("programmer error: default value[%d]=%d is larger than maximum %d for %q",
				i, v, s.maxValue, s.envVar,
			).Error())
		}
	}

	return s
}
