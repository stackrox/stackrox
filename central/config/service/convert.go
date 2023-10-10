package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// VulnerabilityDeferralConfigV1ToStorage returns a new instance of *storage.VulnerabilityExceptionConfig
// based on input *v1.VulnerabilityExceptionConfig.
func VulnerabilityDeferralConfigV1ToStorage(config *v1.VulnerabilityExceptionConfig) *storage.VulnerabilityExceptionConfig {
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
			FixableCveOptions: &storage.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: expiryOptions.GetFixableCveOptions().GetAllFixable(),
				AnyFixable: expiryOptions.GetFixableCveOptions().GetAnyFixable(),
			},
			CustomDate: expiryOptions.GetCustomDate(),
		},
	}
}

// VulnerabilityExceptionConfigStorageToV1 returns a new instance of *v1.VulnerabilityExceptionConfig
// based on input *storage.VulnerabilityExceptionConfig.
func VulnerabilityExceptionConfigStorageToV1(config *storage.VulnerabilityExceptionConfig) *v1.VulnerabilityExceptionConfig {
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
			FixableCveOptions: &v1.VulnerabilityExceptionConfig_FixableCVEOptions{
				AllFixable: expiryOptions.GetFixableCveOptions().GetAllFixable(),
				AnyFixable: expiryOptions.GetFixableCveOptions().GetAnyFixable(),
			},
			CustomDate: expiryOptions.GetCustomDate(),
		},
	}
}
