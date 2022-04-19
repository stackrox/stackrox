package branding

import "github.com/stackrox/rox/pkg/env"

const (
	// ProductBrandingEnvName is the name of environment variable that defines product branding: commercial or open source.
	ProductBrandingEnvName = "ROX_PRODUCT_BRANDING"

	// ProductBrandingRHACS is the value ProductBrandingEnvName should have for Red Hat Advanced Cluster Security branded builds.
	ProductBrandingRHACS = "RHACS_BRANDING"
)

var (
	// TODO(ROX-10208): Remove the default in the followup task of adding the new env variable to CI
	productBrandingSetting = env.RegisterSetting(ProductBrandingEnvName, env.WithDefault("RHACS_BRANDING"))
)

// getProductBrandingEnv returns a value of the environment variable that defines the product branding.
func getProductBrandingEnv() string {
	return productBrandingSetting.Setting()
}
