package scheduler

import "github.com/stackrox/rox/pkg/env"

const (
	// productBrandingName is the variable storing the branding name of the product.
	productBrandingName = "ROX_BRANDING_NAME"

	// ProductBrandingNameRHACS is the name for the product using RHACS branding
	ProductBrandingNameRHACS = "Red Hat Advanced Cluster Security for Kubernetes"
	// ProductBrandingNameStackrox is the name for the product using Stackrox branding
	ProductBrandingNameStackrox = "Stackrox"
)

var (
	productBrandingSetting = env.RegisterSetting(productBrandingName)
)

// ProductBranding returns the environment variable ROX_BRANDING_NAME value
func ProductBranding() string {
	return productBrandingSetting.Setting()
}
