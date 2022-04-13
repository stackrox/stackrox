package scheduler

import "github.com/stackrox/rox/pkg/env"

const (
	// ProductBranding should hold RHACS_BRANDING or STACKROX_BRANDING
	productBranding = "RHACS_BRANDING"
)

var (
	productBrandingSetting = env.RegisterSetting(productBranding)
)

// GetBrandedProductName returns the environment variable ROX_BRANDING_NAME value
func GetProductBranding() string {
	return productBrandingSetting.Setting()
}
