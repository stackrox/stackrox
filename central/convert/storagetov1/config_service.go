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
	dayOptions := make([]*v1.DayOption, 0, len(expiryOptions.GetDayOptions()))
	for _, op := range expiryOptions.GetDayOptions() {
		dayOption := &v1.DayOption{}
		dayOption.SetNumDays(op.GetNumDays())
		dayOption.SetEnabled(op.GetEnabled())
		dayOptions = append(dayOptions, dayOption)
	}

	fixableCveOptions := &v1.VulnerabilityExceptionConfig_FixableCVEOptions{}
	fixableCveOptions.SetAllFixable(expiryOptions.GetFixableCveOptions().GetAllFixable())
	fixableCveOptions.SetAnyFixable(expiryOptions.GetFixableCveOptions().GetAnyFixable())

	expiryOpts := &v1.VulnerabilityExceptionConfig_ExpiryOptions{}
	expiryOpts.SetDayOptions(dayOptions)
	expiryOpts.SetIndefinite(expiryOptions.GetIndefinite())
	expiryOpts.SetFixableCveOptions(fixableCveOptions)
	expiryOpts.SetCustomDate(expiryOptions.GetCustomDate())

	result := &v1.VulnerabilityExceptionConfig{}
	result.SetExpiryOptions(expiryOpts)
	return result
}
