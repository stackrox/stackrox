package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegerSetting(t *testing.T) {
	a := assert.New(t)

	cases := map[string]struct {
		value        string
		defaultValue int
		opts         []IntegerSettingOption
		wantPanic    bool
		wantValue    int
	}{
		"shall pass validation with no options": {
			value:        "1",
			defaultValue: 5,
			opts:         nil,
			wantPanic:    false,
			wantValue:    1,
		},
		"shall pass validation with minimum": {
			value:        "1",
			defaultValue: 5,
			opts:         []IntegerSettingOption{WithMinimum(0)},
			wantPanic:    false,
			wantValue:    1,
		},
		"shall fail validation with minimum": {
			value:        "1",
			defaultValue: 5,
			opts:         []IntegerSettingOption{WithMinimum(5)},
			wantPanic:    false,
			wantValue:    5,
		},
		"shall panic with minimum": {
			value:        "1",
			defaultValue: 5,
			opts:         []IntegerSettingOption{WithMinimum(10)},
			wantPanic:    true,
			wantValue:    1,
		},
		"shall pass validation with min and max": {
			value:        "1",
			defaultValue: 5,
			opts:         []IntegerSettingOption{WithMinimum(0), WithMaximum(10)},
			wantPanic:    false,
			wantValue:    1,
		},
		"shall fail validation with min and max": {
			value:        "11",
			defaultValue: 5,
			opts:         []IntegerSettingOption{WithMinimum(5), WithMaximum(10)},
			wantPanic:    false,
			wantValue:    5,
		},
		"shall panic with min and max": {
			value:        "1",
			defaultValue: 5,
			opts:         []IntegerSettingOption{WithMinimum(10), WithMaximum(15)},
			wantPanic:    true,
			wantValue:    1,
		},
		"default value of min should have no influence if no WithMinimum is used explicitly": {
			value:        "-5",
			defaultValue: -1,
			opts:         []IntegerSettingOption{},
			wantPanic:    false,
			wantValue:    -5,
		},
		"options used incorrectly should panic": {
			value:        "1",
			defaultValue: 5,
			opts:         []IntegerSettingOption{WithMinimum(10), WithMaximum(0)},
			wantPanic:    true,
			wantValue:    1,
		},
	}

	for tname, tt := range cases {
		t.Run(tname, func(t *testing.T) {
			name := newRandomName()
			s := &IntegerSetting{}
			if tt.wantPanic {
				assert.Panics(t, func() {
					s = RegisterIntegerSetting(name, tt.defaultValue, tt.opts...)
					defer unregisterSetting(name)
				})
				return
			}
			s = RegisterIntegerSetting(name, tt.defaultValue, tt.opts...)
			defer unregisterSetting(name)
			a.NoError(os.Setenv(name, tt.value))

			a.Equal(tt.wantValue, s.IntegerSetting())
		})
	}
}
