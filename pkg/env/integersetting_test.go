package env

import (
	"math"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegerSetting(t *testing.T) {
	cases := map[string]struct {
		value            string
		defaultValue     int
		minOpt           func() int
		maxOpt           func() int
		allowList        func() []int
		disallowAllOther bool
		wantPanic        bool
		wantValue        int
	}{
		"shall pass validation with no options": {
			value:        "1",
			defaultValue: 5,
			wantPanic:    false,
			wantValue:    1,
		},
		"shall pass validation with minimum": {
			value:        "1",
			defaultValue: 5,
			minOpt:       func() int { return 0 },
			wantPanic:    false,
			wantValue:    1,
		},
		"shall fail validation with minimum": {
			value:        "1",
			defaultValue: 5,
			minOpt:       func() int { return 5 },
			wantPanic:    false,
			wantValue:    5,
		},
		"shall panic with minimum": {
			value:        "1",
			defaultValue: 5,
			minOpt:       func() int { return 10 },
			wantPanic:    true,
			wantValue:    1,
		},
		"shall pass validation with min and max": {
			value:        "1",
			defaultValue: 5,
			minOpt:       func() int { return 0 },
			maxOpt:       func() int { return 10 },
			wantPanic:    false,
			wantValue:    1,
		},
		"shall fail validation with min and max": {
			value:        "11",
			defaultValue: 5,
			minOpt:       func() int { return 5 },
			maxOpt:       func() int { return 10 },
			wantPanic:    false,
			wantValue:    5,
		},
		"shall panic with min and max": {
			value:        "1",
			defaultValue: 5,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			wantPanic:    true,
			wantValue:    1,
		},
		"shall pass validation with min and max and allowList covering the value and the default": {
			value:        "12",
			defaultValue: 11,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			allowList:    func() []int { return []int{11, 12} },
			wantPanic:    false,
			wantValue:    12,
		},
		"shall pass validation with min and max and allowList covering the value but not the default": {
			value:        "12",
			defaultValue: 11,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			allowList:    func() []int { return []int{12} },
			wantPanic:    false,
			wantValue:    12,
		},
		"shall pass validation with min and max and allowList covering the default but not the value": {
			value:        "12",
			defaultValue: 11,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			allowList:    func() []int { return []int{11} },
			wantPanic:    false,
			wantValue:    12,
		},
		"shall respect allowList when value is outside of min max range": {
			value:        "0",
			defaultValue: 12,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			allowList:    func() []int { return []int{0} },
			wantPanic:    false,
			wantValue:    0,
		},
		"shall respect allowList when the value and the default value is on allowList": {
			value:        "0",
			defaultValue: 5,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			allowList:    func() []int { return []int{0, 5} },
			wantPanic:    false,
			wantValue:    0,
		},
		"shall panic even if value is on allowList but the default is not": {
			value:        "0",
			defaultValue: 5,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			allowList:    func() []int { return []int{0} },
			wantPanic:    true,
			wantValue:    0,
		},
		"shall return default if value is not on allowList but the default is": {
			value:        "0",
			defaultValue: 5,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 15 },
			allowList:    func() []int { return []int{5} },
			wantPanic:    false,
			wantValue:    5,
		},
		"only values from allowList should be accepted - default on allowList, value outside": {
			value:            "0",
			defaultValue:     5,
			allowList:        func() []int { return []int{5} },
			disallowAllOther: true,
			wantPanic:        false,
			wantValue:        5,
		},
		"default outside of allowList should yield panic": {
			value:            "1",
			defaultValue:     5,
			allowList:        func() []int { return []int{0, 1, 2} },
			disallowAllOther: true,
			wantPanic:        true,
			wantValue:        1,
		},
		"default and the value on allowList should return the value": {
			value:            "1",
			defaultValue:     2,
			allowList:        func() []int { return []int{0, 1, 2} },
			disallowAllOther: true,
			wantPanic:        false,
			wantValue:        1,
		},
		"when atoi fails and 0 is in the allowList, then the default value should be returned": {
			value:            "foo",
			defaultValue:     2,
			allowList:        func() []int { return []int{0, 1, 2} },
			disallowAllOther: false,
			wantPanic:        false,
			wantValue:        2,
		},
		"when atoi fails and 0 is not in the allowList, then the default value should be returned": {
			value:            "foo",
			defaultValue:     2,
			allowList:        func() []int { return []int{1, 2, 3} },
			disallowAllOther: false,
			wantPanic:        false,
			wantValue:        2,
		},
		"default value of min should have no influence if no WithMinimum is used explicitly": {
			value:        "-5",
			defaultValue: -1,
			wantPanic:    false,
			wantValue:    -5,
		},
		"options used incorrectly should panic": {
			value:        "1",
			defaultValue: 5,
			minOpt:       func() int { return 10 },
			maxOpt:       func() int { return 0 },
			wantPanic:    true,
			wantValue:    1,
		},
		"crossing the max/min value of int should return default value": {
			// We use bigInt here to get the max value (because int may be int32 or int64) and do plus-one arithmetic.
			value:        new(big.Int).Add(big.NewInt(math.MaxInt), big.NewInt(1)).Text(10), // math.MaxInt + 1
			defaultValue: 5,
			wantPanic:    false,
			wantValue:    5,
		},
		"using border value for int should not yield default value": {
			value:        strconv.Itoa(math.MaxInt),
			defaultValue: 5,
			wantPanic:    false,
			wantValue:    math.MaxInt,
		},
	}

	for tname, tt := range cases {
		t.Run(tname, func(t *testing.T) {
			name := newRandomName()
			defer unregisterSetting(name)
			if tt.wantPanic {
				assert.Panics(t, func() {
					_ = testRegisterSetting(t, name, tt.defaultValue, tt.minOpt, tt.maxOpt, tt.allowList, tt.disallowAllOther)
				})
				return
			}
			s := testRegisterSetting(t, name, tt.defaultValue, tt.minOpt, tt.maxOpt, tt.allowList, tt.disallowAllOther)
			assert.NoError(t, os.Setenv(name, tt.value))

			assert.Equal(t, tt.wantValue, s.IntegerSetting())
		})
	}
}

// testRegisterSetting is a helper to the function-under-test with its options.
// It was created to avoid code repetition, as it must be called in two places depending on whether we test for panics.
func testRegisterSetting(_ *testing.T, name string, defaultValue int, min, max func() int, allowList func() []int, disallowAllOther bool) *IntegerSetting {
	s := RegisterIntegerSetting(name, defaultValue)
	if allowList != nil {
		s = s.AllowExplicitly(allowList()...)
	}
	if disallowAllOther { // must be called after `AllowExplicitly`
		s = s.DisallowRest()
	}
	if min != nil {
		s = s.WithMinimum(min())
	}
	if max != nil {
		s = s.WithMaximum(max())
	}
	return s
}
