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
	return &v1.VulnerabilityExceptionConfig{
		ExpiryOptions: &v1.VulnerabilityExceptionConfig_ExpiryOptions{
			DayOptions: func() []*v1.DayOption {
				dayOptions := make([]*v1.DayOption, 0, len(expiryOptions.GetDayOptions()))
				for _, op := range expiryOptions.GetDayOptions() {
					dayOptions = append(dayOptions, &v1.DayOption{
						NumDays: op.GetNumDays(),
						Enabled: op.GetEnabled(),
					})
				}
				return dayOptions
			}(),
			Indefinite: expiryOptions.GetIndefinite(),
			FixableCveOptions: &v1.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: expiryOptions.GetFixableCveOptions().GetAllFixable(),
				AnyFixable: expiryOptions.GetFixableCveOptions().GetAnyFixable(),
			},
			CustomDate: expiryOptions.GetCustomDate(),
		},
	}
}
