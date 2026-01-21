package env

import (
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
)

// FloatSetting represents an environment variable which should be parsed into a float64.
type FloatSetting struct {
	envVar       string
	defaultValue float64

	// Optional validation of the value.
	minimumValue float64
	maximumValue float64
	allowList    []float64
	disallowRest bool
}

// EnvVar returns the string name of the environment variable.
func (s *FloatSetting) EnvVar() string {
	return s.envVar
}

// DefaultValue returns the default value for the setting.
func (s *FloatSetting) DefaultValue() float64 {
	return s.defaultValue
}

// Setting returns the string form of the float environment variable.
func (s *FloatSetting) Setting() string {
	return strconv.FormatFloat(s.FloatSetting(), 'f', -1, 64)
}

// FloatSetting returns the float64 object represented by the environment variable.
func (s *FloatSetting) FloatSetting() float64 {
	val := os.Getenv(s.envVar)
	if val == "" {
		return s.defaultValue
	}
	v, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return s.defaultValue
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return s.defaultValue
	}
	if slices.Contains(s.allowList, v) {
		return v
	}
	if s.disallowRest {
		return s.defaultValue
	}
	if (v < s.minimumValue) || (v > s.maximumValue) {
		return s.defaultValue
	}
	return v
}

// RegisterFloatSetting globally registers and returns a new float setting.
func RegisterFloatSetting(envVar string, defaultValue float64) *FloatSetting {
	s := &FloatSetting{
		envVar:       envVar,
		defaultValue: defaultValue,
		minimumValue: math.Inf(-1),
		maximumValue: math.Inf(1),
	}
	s.mustValidate()
	Settings[s.EnvVar()] = s
	return s
}

// WithMinimum specifies the minimal allowed value that passes the validation.
func (s *FloatSetting) WithMinimum(min float64) *FloatSetting {
	s.minimumValue = min
	return s.mustValidate()
}

// WithMaximum specifies the maximal allowed value that passes the validation.
func (s *FloatSetting) WithMaximum(max float64) *FloatSetting {
	s.maximumValue = max
	return s.mustValidate()
}

// AllowExplicitly specifies the values that are explicitly allowed for the FloatSetting.
// Those values will not be affected by `WithMinimum` and `WithMaximum`.
// This is mainly useful for allowing 0 as a special value to disable a setting.
func (s *FloatSetting) AllowExplicitly(values ...float64) *FloatSetting {
	s.allowList = values
	return s.mustValidate()
}

// DisallowRest configures the validation, so that only the numbers on specified by `AllowExplicitly` will pass.
func (s *FloatSetting) DisallowRest() *FloatSetting {
	s.disallowRest = true
	return s.mustValidate()
}

func (s *FloatSetting) mustValidate() *FloatSetting {
	if math.IsNaN(s.defaultValue) || math.IsInf(s.defaultValue, 0) {
		panic(fmt.Errorf("programmer error: default value %v is not finite for %q", s.defaultValue, s.envVar).Error())
	}
	if math.IsNaN(s.minimumValue) || math.IsNaN(s.maximumValue) {
		panic(fmt.Errorf("programmer error: incorrect float bounds for %q: NaN is not allowed", s.EnvVar()).Error())
	}
	for _, v := range s.allowList {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			panic(fmt.Errorf("programmer error: allow-list value %v is not finite for %q", v, s.envVar).Error())
		}
	}
	if slices.Contains(s.allowList, s.defaultValue) {
		return s
	}
	if s.disallowRest {
		if len(s.allowList) == 0 {
			panic(fmt.Errorf("programmer error: no values are allowed - allow-list is empty for %q."+
				"`DisallowAllOther` must be called after `AllowExplicitly`", s.envVar).Error())
		}
		panic(fmt.Errorf("programmer error: default value %v is not on allow-list: %v for %q", s.defaultValue,
			s.allowList, s.envVar,
		).Error())
	}
	if s.defaultValue < s.minimumValue {
		panic(fmt.Errorf("programmer error: default value %v is smaller than the minimum %v for %q",
			s.defaultValue, s.minimumValue, s.envVar,
		).Error())
	}
	if s.defaultValue > s.maximumValue {
		panic(fmt.Errorf("programmer error: default value %v is larger than the maximum %v for %q",
			s.defaultValue, s.maximumValue, s.envVar,
		).Error())
	}
	if s.minimumValue > s.maximumValue {
		panic(fmt.Errorf("programmer error: incorrect float bounds for %q: "+
			"minimum value %v must be smaller or equal to maximum value %v",
			s.EnvVar(), s.minimumValue, s.maximumValue,
		).Error())
	}
	return s
}
