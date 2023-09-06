package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// VulnerabilityDeferralConfigV1ToStorage returns a new instance of *storage.VulnerabilityDeferralConfig
// based on input *v1.VulnerabilityDeferralConfig.
func VulnerabilityDeferralConfigV1ToStorage(config *v1.VulnerabilityDeferralConfig) *storage.VulnerabilityDeferralConfig {
	if config == nil {
		return nil
	}
	expiryOptions := config.GetExpiryOptions()
	if expiryOptions == nil {
		return nil
	}
	return &storage.VulnerabilityDeferralConfig{
		ExpiryOptions: &storage.VulnerabilityDeferralConfig_ExpiryOptions{
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
			FixableCveOptions: &storage.VulnerabilityDeferralConfig_FixableCVEOptions{
				AllFixable: expiryOptions.GetFixableCveOptions().GetAllFixable(),
				AnyFixable: expiryOptions.GetFixableCveOptions().GetAnyFixable(),
			},
			CustomDate: expiryOptions.GetCustomDate(),
		},
	}
}

// VulnerabilityDeferralConfigStorageToV1 returns a new instance of *v1.VulnerabilityDeferralConfig
// based on input *storage.VulnerabilityDeferralConfig.
func VulnerabilityDeferralConfigStorageToV1(config *storage.VulnerabilityDeferralConfig) *v1.VulnerabilityDeferralConfig {
	if config == nil {
		return nil
	}
	expiryOptions := config.GetExpiryOptions()
	if expiryOptions == nil {
		return nil
	}
	return &v1.VulnerabilityDeferralConfig{
		ExpiryOptions: &v1.VulnerabilityDeferralConfig_ExpiryOptions{
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
			FixableCveOptions: &v1.VulnerabilityDeferralConfig_FixableCVEOptions{
				AllFixable: expiryOptions.GetFixableCveOptions().GetAllFixable(),
				AnyFixable: expiryOptions.GetFixableCveOptions().GetAnyFixable(),
			},
			CustomDate: expiryOptions.GetCustomDate(),
		},
	}
}
