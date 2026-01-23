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

// DefaultValue returns a copy of the default value for the setting
func (s *IntegerArraySetting) DefaultValue() []int {
	if s.defaultValue == nil {
		return nil
	}
	result := make([]int, len(s.defaultValue))
	copy(result, s.defaultValue)
	return result
}

func arrayToString(arr []int) string {
	strArr := make([]string, len(arr))
	for i, v := range arr {
		strArr[i] = strconv.Itoa(v)
	}
	return "[" + strings.Join(strArr, ",") + "]"

}

// Setting returns the string form of the integer array environment variable
func (s *IntegerArraySetting) Setting() string {
	arr, _ := s.IntegerArraySetting()
	return arrayToString(arr)
}

// IntegerArraySetting returns the integer slice represented by the environment variable
// and a warning message if there were any issues parsing the value.
// The warning message is empty if parsing was successful.
func (s *IntegerArraySetting) IntegerArraySetting() ([]int, string) {
	val, ok := os.LookupEnv(s.envVar)
	originalVal := val

	if !ok {
		return s.DefaultValue(), ""
	}

	if val == "" {
		return s.DefaultValue(), fmt.Sprintf("Empty value for environment variable %s. Using default value of %s.", s.envVar, arrayToString(s.DefaultValue()))
	}

	// Strip brackets if present
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		val = val[1 : len(val)-1]
	}

	// If the value is empty after stripping brackets (i.e., "[]"), return empty array
	val = strings.TrimSpace(val)
	if val == "" {
		if s.minLength == 0 {
			// Empty array "[]" is allowed when minLength is 0
			return []int{}, ""
		} else {
			// Empty array not allowed, return default
			return s.DefaultValue(), fmt.Sprintf("Invalid value for environment variable %s. It cannot be empty. Using default value of %s", s.envVar, arrayToString(s.DefaultValue()))
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
			return s.DefaultValue(), fmt.Sprintf("Unable to parse environment variable %s with value %s. Using default value of %s", s.envVar, originalVal, arrayToString(s.DefaultValue()))
		}

		// Validate individual value
		if v < s.minValue || v > s.maxValue {
			// Out of bounds, return default
			return s.DefaultValue(), fmt.Sprintf("Element %d in environment variable %s with value %s is out of bounds. Min value %d. Max value %d. Using default value of %s", v, s.envVar, originalVal, s.minValue, s.maxValue, arrayToString(s.DefaultValue()))
		}

		result = append(result, v)
	}

	// Validate array length
	if len(result) < s.minLength || len(result) > s.maxLength {
		return s.DefaultValue(), fmt.Sprintf("Array length %d for environment variable %s is out of bounds. Min length %d. Max length %d. Using default value of %s", len(result), s.envVar, s.minLength, s.maxLength, arrayToString(s.DefaultValue()))
	}

	return result, ""
}

// RegisterIntegerArraySetting globally registers and returns a new integer array setting.
func RegisterIntegerArraySetting(envVar string, defaultValue []int) *IntegerArraySetting {
	// Make a defensive copy of the input to ensure immutability
	var defaultCopy []int
	if defaultValue != nil {
		defaultCopy = make([]int, len(defaultValue))
		copy(defaultCopy, defaultValue)
	}

	s := &IntegerArraySetting{
		envVar:       envVar,
		defaultValue: defaultCopy,
		minValue:     math.MinInt,
		maxValue:     math.MaxInt,
		minLength:    0,
		maxLength:    math.MaxInt,
	}
	Settings[s.EnvVar()] = s
	return s
}

// WithMinimumElementValue specifies the minimal allowed value for each element in the array.
func (s *IntegerArraySetting) WithMinimumElementValue(min int) *IntegerArraySetting {
	s.minValue = min
	return s.mustValidate()
}

// WithMaximumElementValue specifies the maximal allowed value for each element in the array.
func (s *IntegerArraySetting) WithMaximumElementValue(max int) *IntegerArraySetting {
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
