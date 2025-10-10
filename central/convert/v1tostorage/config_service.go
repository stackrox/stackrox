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
	dayOptions := make([]*storage.DayOption, 0, len(expiryOptions.GetDayOptions()))
	for _, op := range expiryOptions.GetDayOptions() {
		dayOption := &storage.DayOption{}
		dayOption.SetNumDays(op.GetNumDays())
		dayOption.SetEnabled(op.GetEnabled())
		dayOptions = append(dayOptions, dayOption)
	}

	fixableCveOptions := &storage.VulnerabilityExceptionConfig_FixableCVEOptions{}
	fixableCveOptions.SetAllFixable(expiryOptions.GetFixableCveOptions().GetAllFixable())
	fixableCveOptions.SetAnyFixable(expiryOptions.GetFixableCveOptions().GetAnyFixable())

	expiryOpts := &storage.VulnerabilityExceptionConfig_ExpiryOptions{}
	expiryOpts.SetDayOptions(dayOptions)
	expiryOpts.SetIndefinite(expiryOptions.GetIndefinite())
	expiryOpts.SetFixableCveOptions(fixableCveOptions)
	expiryOpts.SetCustomDate(expiryOptions.GetCustomDate())

	result := &storage.VulnerabilityExceptionConfig{}
	result.SetExpiryOptions(expiryOpts)
	return result
}
