package branding

import "github.com/stackrox/rox/pkg/env"

const (
	// ProductBranding should hold RHACS_BRANDING or STACKROX_BRANDING
	ProductBrandingEnvName = "ROX_PRODUCT_BRANDING"
)

var (
	productBrandingSetting = env.RegisterSetting(ProductBrandingEnvName, env.WithDefault("RHACS_BRANDING"))
)

// GetBrandedProductName returns the environment variable ROX_BRANDING_NAME value
func GetProductBrandingEnvName() string {
	return productBrandingSetting.Setting()
}
