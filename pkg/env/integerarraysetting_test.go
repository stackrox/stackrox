package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegerArraySetting(t *testing.T) {
	cases := map[string]struct {
		value         string
		defaultValue  []int
		minValue      int
		maxValue      int
		minLength     int
		maxLength     int
		expectedPanic bool
		expectedValue []int
	}{
		"Valid comma-separated integers": {
			value:         "1,2,3",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Single integer": {
			value:         "42",
			defaultValue:  []int{5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{42},
		},
		"Whitespace around commas": {
			value:         "1, 2, 3",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Whitespace around values": {
			value:         " 1 , 2 , 3 ",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Empty string returns default value": {
			value:         "",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{5, 5, 5},
		},
		"Empty array [] returns empty slice": {
			value:         "[]",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{},
		},
		"Bracket format with values": {
			value:         "[1,2,3]",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Bracket format with whitespace": {
			value:         "[ 1 , 2 , 3 ]",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Return default for invalid format": {
			value:         "1,abc,3",
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{5, 5, 5},
		},
		"Return default for partially invalid format": {
			value:         "1,2,",
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2},
		},
		"Validation with minimum value": {
			value:         "5,10,15",
			defaultValue:  []int{5, 6, 7},
			minValue:      5,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{5, 10, 15},
		},
		"Fail validation with minimum value": {
			value:         "1,2,3",
			defaultValue:  []int{10, 10, 10},
			minValue:      5,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{10, 10, 10},
		},
		"Pass validation with maximum value": {
			value:         "5,10,15",
			defaultValue:  []int{1, 2, 3},
			minValue:      0,
			maxValue:      20,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{5, 10, 15},
		},
		"Fail validation with maximum value": {
			value:         "5,10,25",
			defaultValue:  []int{1, 2, 3},
			minValue:      0,
			maxValue:      20,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Pass validation with min and max value": {
			value:         "5,10,15",
			defaultValue:  []int{1, 2, 3},
			minValue:      0,
			maxValue:      20,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{5, 10, 15},
		},
		"Pass validation with minimum length": {
			value:         "1,2,3",
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     2,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Fail validation with minimum length": {
			value:         "1",
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     2,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{5, 5},
		},
		"Pass validation with maximum length": {
			value:         "1,2,3",
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     5,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Fail validation with maximum length": {
			value:         "1,2,3,4,5,6",
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     5,
			expectedPanic: false,
			expectedValue: []int{5, 5},
		},
		"Pass validation with min and max length": {
			value:         "1,2,3",
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     1,
			maxLength:     5,
			expectedPanic: false,
			expectedValue: []int{1, 2, 3},
		},
		"Handle negative numbers": {
			value:         "-1,-2,-3",
			defaultValue:  []int{0, 0, 0},
			minValue:      -10,
			maxValue:      10,
			minLength:     0,
			maxLength:     100,
			expectedPanic: false,
			expectedValue: []int{-1, -2, -3},
		},
		"Panic with invalid minimum value greater than maximum": {
			defaultValue:  []int{5, 5},
			minValue:      10,
			maxValue:      5,
			minLength:     0,
			maxLength:     100,
			expectedPanic: true,
		},
		"Panic with invalid minimum length negative": {
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     -1,
			maxLength:     100,
			expectedPanic: true,
		},
		"Panic with invalid minimum length greater than maximum": {
			defaultValue:  []int{5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     10,
			maxLength:     5,
			expectedPanic: true,
		},
		"Panic with default value length less than minimum": {
			defaultValue:  []int{5},
			minValue:      0,
			maxValue:      100,
			minLength:     2,
			maxLength:     100,
			expectedPanic: true,
		},
		"Panic with default value length greater than maximum": {
			defaultValue:  []int{5, 5, 5},
			minValue:      0,
			maxValue:      100,
			minLength:     0,
			maxLength:     2,
			expectedPanic: true,
		},
		"Panic with default value element less than minimum": {
			defaultValue:  []int{1, 2, 3},
			minValue:      5,
			maxValue:      100,
			minLength:     0,
			maxLength:     100,
			expectedPanic: true,
		},
		"Panic with default value element greater than maximum": {
			defaultValue:  []int{5, 10, 15},
			minValue:      0,
			maxValue:      10,
			minLength:     0,
			maxLength:     100,
			expectedPanic: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			envVar := "ROX_TEST_INT_ARRAY_SETTING_" + name
			defer unregisterSetting(envVar)
			if tc.value != "" {
				err := os.Setenv(envVar, tc.value)
				assert.NoError(t, err)
				defer func() {
					_ = os.Unsetenv(envVar)
				}()
			}

			if tc.expectedPanic {
				assert.Panics(t, func() {
					testRegisterArraySetting(envVar, tc.defaultValue, tc.minValue, tc.maxValue, tc.minLength, tc.maxLength)
				})
			} else {
				s := testRegisterArraySetting(envVar, tc.defaultValue, tc.minValue, tc.maxValue, tc.minLength, tc.maxLength)
				value, _ := s.IntegerArraySetting()
				assert.Equal(t, tc.expectedValue, value)
			}
		})
	}
}

func testRegisterArraySetting(name string, defaultValue []int, minValue, maxValue, minLength, maxLength int) *IntegerArraySetting {
	s := RegisterIntegerArraySetting(name, defaultValue)

	s = s.WithMinimumElementValue(minValue)
	s = s.WithMaximumElementValue(maxValue)
	s = s.WithMinLength(minLength)
	s = s.WithMaxLength(maxLength)

	return s
}
