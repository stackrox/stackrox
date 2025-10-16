package v1tostorage

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// VulnerabilityExceptionConfig returns a new instance of *storage.VulnerabilityExceptionConfig
// based on input *v1.VulnerabilityExceptionConfig.
func VulnerabilityExceptionConfig(config *v1.VulnerabilityExceptionConfig) *storage.VulnerabilityExceptionConfig {
	if config == nil {
		return nil
	}
	expiryOptions := config.GetExpiryOptions()
	if expiryOptions == nil {
		return nil
	}
	vf := &storage.VulnerabilityExceptionConfig_FixableCVEOptions{}
	vf.SetAllFixable(expiryOptions.GetFixableCveOptions().GetAllFixable())
	vf.SetAnyFixable(expiryOptions.GetFixableCveOptions().GetAnyFixable())
	ve := &storage.VulnerabilityExceptionConfig_ExpiryOptions{}
	ve.SetDayOptions(func() []*storage.DayOption {
		dayOptions := make([]*storage.DayOption, 0, len(expiryOptions.GetDayOptions()))
		for _, op := range expiryOptions.GetDayOptions() {
			dayOption := &storage.DayOption{}
			dayOption.SetNumDays(op.GetNumDays())
			dayOption.SetEnabled(op.GetEnabled())
			dayOptions = append(dayOptions, dayOption)
		}
		return dayOptions
	}())
	ve.SetIndefinite(expiryOptions.GetIndefinite())
	ve.SetFixableCveOptions(vf)
	ve.SetCustomDate(expiryOptions.GetCustomDate())
	vec := &storage.VulnerabilityExceptionConfig{}
	vec.SetExpiryOptions(ve)
	return vec
}
