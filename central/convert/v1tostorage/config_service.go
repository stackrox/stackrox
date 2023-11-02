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
	return &storage.VulnerabilityExceptionConfig{
		ExpiryOptions: &storage.VulnerabilityExceptionConfig_ExpiryOptions{
			DayOptions: func() []*storage.DayOption {
				dayOptions := make([]*storage.DayOption, 0, len(expiryOptions.GetDayOptions()))
				for _, op := range expiryOptions.GetDayOptions() {
					dayOptions = append(dayOptions, &storage.DayOption{
						NumDays: op.GetNumDays(),
						Enabled: op.GetEnabled(),
					})
				}
				return dayOptions
			}(),
			Indefinite: expiryOptions.GetIndefinite(),
			FixableCveOptions: &storage.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: expiryOptions.GetFixableCveOptions().GetAllFixable(),
				AnyFixable: expiryOptions.GetFixableCveOptions().GetAnyFixable(),
			},
			CustomDate: expiryOptions.GetCustomDate(),
		},
	}
}
