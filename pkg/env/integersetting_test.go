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
		value        string
		defaultValue int
		minOpt       func() int
		maxOpt       func() int
		wantPanic    bool
		wantValue    int
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
		"using border value for int64 should not yield default value": {
			value:        strconv.Itoa(math.MaxInt),
			defaultValue: 5,
			wantPanic:    false,
			wantValue:    math.MaxInt,
		},
	}

	for tname, tt := range cases {
		t.Run(tname, func(t *testing.T) {
			name := newRandomName()
			s := &IntegerSetting{}

			registerFunc := func(name string, defaultValue int, min, max func() int) *IntegerSetting {
				if tt.minOpt != nil && tt.maxOpt != nil {
					return RegisterIntegerSetting(name, defaultValue).WithMinimum(min()).WithMaximum(max())
				}
				if tt.minOpt != nil {
					return RegisterIntegerSetting(name, defaultValue).WithMinimum(min())
				}
				if tt.maxOpt != nil {
					return RegisterIntegerSetting(name, defaultValue).WithMaximum(max())
				}
				return RegisterIntegerSetting(name, defaultValue)
			}
			defer unregisterSetting(name)
			if tt.wantPanic {
				assert.Panics(t, func() {
					s = registerFunc(name, tt.defaultValue, tt.minOpt, tt.maxOpt)
				})
				return
			}
			s = registerFunc(name, tt.defaultValue, tt.minOpt, tt.maxOpt)
			assert.NoError(t, os.Setenv(name, tt.value))

			assert.Equal(t, tt.wantValue, s.IntegerSetting())
		})
	}
}
