package branding

import "github.com/stackrox/rox/pkg/env"

const (
	// ProductBranding should hold RHACS_BRANDING or STACKROX_BRANDING
	ProductBranding = "ROX_PRODUCT_BRANDING"
)

var (
	productBrandingSetting = env.RegisterSetting(ProductBranding)
)

// GetProductBrandingEnv returns the environment variable ROX_BRANDING_NAME value
func GetProductBrandingEnv() string {
	return productBrandingSetting.Setting()
}
