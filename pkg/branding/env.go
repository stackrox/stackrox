package branding

import "github.com/stackrox/rox/pkg/env"

const (
	// ProductBrandingEnvName is the name of environment variable that defines product branding: commercial or open source.
	ProductBrandingEnvName = "ROX_PRODUCT_BRANDING"
)

var (
	// TODO @jschnath: Remove the default in the followup task of adding the new env variable to CI
	productBrandingSetting = env.RegisterSetting(ProductBrandingEnvName, env.WithDefault("RHACS_BRANDING"))
)

// GetProductBrandingEnv returns the environment variable ROX_BRANDING_NAME value
func GetProductBrandingEnv() string {
	return productBrandingSetting.Setting()
}
