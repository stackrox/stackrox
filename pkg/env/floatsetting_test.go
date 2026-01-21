package env

import (
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloatSetting(t *testing.T) {
	cases := map[string]struct {
		value        string
		defaultValue float64
		minOpt       func() float64
		maxOpt       func() float64
		allowList    func() []float64
		disallowRest bool
		wantPanic    bool
		wantValue    float64
	}{
		"should return value with no options": {
			value:        "1.25",
			defaultValue: 5.5,
			wantValue:    1.25,
		},
		"should return default when parse fails": {
			value:        "not-a-float",
			defaultValue: 2.5,
			wantValue:    2.5,
		},
		"should return default when value is NaN": {
			value:        "NaN",
			defaultValue: 3.25,
			wantValue:    3.25,
		},
		"should return default when value is Inf": {
			value:        "Inf",
			defaultValue: 4.75,
			wantValue:    4.75,
		},
		"should pass validation with minimum": {
			value:        "1.5",
			defaultValue: 5.5,
			minOpt:       func() float64 { return 0 },
			wantValue:    1.5,
		},
		"should return default when below minimum": {
			value:        "1.5",
			defaultValue: 5.5,
			minOpt:       func() float64 { return 2 },
			wantValue:    5.5,
		},
		"should pass validation with min and max": {
			value:        "1.5",
			defaultValue: 1,
			minOpt:       func() float64 { return 0 },
			maxOpt:       func() float64 { return 2 },
			wantValue:    1.5,
		},
		"should return default when outside min and max": {
			value:        "3.5",
			defaultValue: 1.5,
			minOpt:       func() float64 { return 0 },
			maxOpt:       func() float64 { return 2 },
			wantValue:    1.5,
		},
		"should return default when value exceeds max by tiny amount": {
			value:        "1.000000000001",
			defaultValue: 0.5,
			minOpt:       func() float64 { return 0 },
			maxOpt:       func() float64 { return 1.0 },
			wantValue:    0.5,
		},
		"should return default for b-format float values": {
			// strconv.ParseFloat does not accept the 'b' format (e.g., "-ddddp±ddd"), so it falls back to default.
			value:        "5629499534213120p-52",
			defaultValue: 0.5,
			wantValue:    0.5,
		},
		"should parse e-format float values": {
			value:        "1.25e+00",
			defaultValue: 0.5,
			wantValue:    1.25,
		},
		"should parse E-format float values": {
			value:        "1.25E+00",
			defaultValue: 0.5,
			wantValue:    1.25,
		},
		"should parse f-format float values": {
			value:        "1.250000",
			defaultValue: 0.5,
			wantValue:    1.25,
		},
		"should parse x-format float values": {
			value:        "0x1.4p+00",
			defaultValue: 0.5,
			wantValue:    1.25,
		},
		"should parse X-format float values": {
			value:        "0X1.4P+00",
			defaultValue: 0.5,
			wantValue:    1.25,
		},
		"should respect allowList when value is outside range": {
			value:        "0",
			defaultValue: 12,
			minOpt:       func() float64 { return 10 },
			maxOpt:       func() float64 { return 15 },
			allowList:    func() []float64 { return []float64{0} },
			wantValue:    0,
		},
		"should return default when disallowRest and value not allowed": {
			value:        "0",
			defaultValue: 5,
			allowList:    func() []float64 { return []float64{5} },
			disallowRest: true,
			wantValue:    5,
		},
		"should return value when disallowRest and value allowed": {
			value:        "1",
			defaultValue: 2,
			allowList:    func() []float64 { return []float64{1, 2} },
			disallowRest: true,
			wantValue:    1,
		},
		"should panic when disallowRest without allowList": {
			value:        "1",
			defaultValue: 2,
			disallowRest: true,
			wantPanic:    true,
		},
		"should panic when default not on allowList with disallowRest": {
			value:        "1",
			defaultValue: 3,
			allowList:    func() []float64 { return []float64{1, 2} },
			disallowRest: true,
			wantPanic:    true,
		},
		"should panic when default is NaN": {
			value:        "1",
			defaultValue: math.NaN(),
			wantPanic:    true,
		},
		"should panic when allowList includes NaN": {
			value:        "1",
			defaultValue: 1,
			allowList:    func() []float64 { return []float64{math.NaN()} },
			wantPanic:    true,
		},
		"should panic when minimum is NaN": {
			value:        "1",
			defaultValue: 1,
			minOpt:       func() float64 { return math.NaN() },
			wantPanic:    true,
		},
		"should panic when maximum is NaN": {
			value:        "1",
			defaultValue: 1,
			maxOpt:       func() float64 { return math.NaN() },
			wantPanic:    true,
		},
		"should panic when minimum is larger than maximum": {
			value:        "1",
			defaultValue: 1,
			minOpt:       func() float64 { return 2 },
			maxOpt:       func() float64 { return 1 },
			wantPanic:    true,
		},
	}

	for tname, tt := range cases {
		t.Run(tname, func(t *testing.T) {
			name := newRandomName()
			defer unregisterSetting(name)
			if tt.wantPanic {
				assert.Panics(t, func() {
					_ = testRegisterFloatSetting(name, tt.defaultValue, tt.minOpt, tt.maxOpt, tt.allowList, tt.disallowRest)
				})
				return
			}
			s := testRegisterFloatSetting(name, tt.defaultValue, tt.minOpt, tt.maxOpt, tt.allowList, tt.disallowRest)
			assert.NoError(t, os.Setenv(name, tt.value))

			assert.Equal(t, tt.wantValue, s.FloatSetting())
		})
	}
}

// testRegisterFloatSetting is a helper to the function-under-test with its options.
// It was created to avoid code repetition, as it must be called in two places depending on whether we test for panics.
func testRegisterFloatSetting(name string, defaultValue float64, min, max func() float64, allowList func() []float64, disallowRest bool) *FloatSetting {
	s := RegisterFloatSetting(name, defaultValue)
	if allowList != nil {
		s = s.AllowExplicitly(allowList()...)
	}
	if disallowRest { // must be called after `AllowExplicitly`
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
