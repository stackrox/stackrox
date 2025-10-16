package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// VulnerabilityExceptionConfig returns a new instance of *v1.VulnerabilityExceptionConfig
// based on input *storage.VulnerabilityExceptionConfig.
func VulnerabilityExceptionConfig(config *storage.VulnerabilityExceptionConfig) *v1.VulnerabilityExceptionConfig {
	if config == nil {
		return nil
	}
	expiryOptions := config.GetExpiryOptions()
	if expiryOptions == nil {
		return nil
	}
	vf := &v1.VulnerabilityExceptionConfig_FixableCVEOptions{}
	vf.SetAllFixable(expiryOptions.GetFixableCveOptions().GetAllFixable())
	vf.SetAnyFixable(expiryOptions.GetFixableCveOptions().GetAnyFixable())
	ve := &v1.VulnerabilityExceptionConfig_ExpiryOptions{}
	ve.SetDayOptions(func() []*v1.DayOption {
		dayOptions := make([]*v1.DayOption, 0, len(expiryOptions.GetDayOptions()))
		for _, op := range expiryOptions.GetDayOptions() {
			dayOption := &v1.DayOption{}
			dayOption.SetNumDays(op.GetNumDays())
			dayOption.SetEnabled(op.GetEnabled())
			dayOptions = append(dayOptions, dayOption)
		}
		return dayOptions
	}())
	ve.SetIndefinite(expiryOptions.GetIndefinite())
	ve.SetFixableCveOptions(vf)
	ve.SetCustomDate(expiryOptions.GetCustomDate())
	vec := &v1.VulnerabilityExceptionConfig{}
	vec.SetExpiryOptions(ve)
	return vec
}
