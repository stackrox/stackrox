package branding

const (
	productBrandingNameRHACS = "Red Hat Advanced Cluster Security for Kubernetes"

	productBrandingNameStackrox = "StackRox"
)

// GetBrandedProductName returns the product name based on ProductBranding env variable
func GetBrandedProductName() string {
	if getProductBrandingEnv() == "RHACS_BRANDING" {
		return productBrandingNameRHACS
	}
	return productBrandingNameStackrox
}
