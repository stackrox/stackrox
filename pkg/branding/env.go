package branding

import "github.com/stackrox/rox/pkg/env"

const (
	// ProductBrandingEnvName should hold RHACS_BRANDING or STACKROX_BRANDING
	ProductBrandingEnvName = "ROX_PRODUCT_BRANDING"
)

var (
	productBrandingSetting = env.RegisterSetting(ProductBrandingEnvName)
)

// GetProductBrandingEnv returns the environment variable ROX_BRANDING_NAME value
func GetProductBrandingEnv() string {
	return productBrandingSetting.Setting()
}
