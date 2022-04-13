package scheduler

import "github.com/stackrox/rox/pkg/env"

const (
	// ProductBranding should hold RHACS_BRANDING or STACKROX_BRANDING
	productBrandingEnvName = "ROX_PRODUCT_BRANDING"
)

var (
	productBrandingSetting = env.RegisterSetting(productBrandingEnvName)
)

// GetBrandedProductName returns the environment variable ROX_BRANDING_NAME value
func GetProductBrandingEnvName() string {
	return productBrandingSetting.Setting()
}
