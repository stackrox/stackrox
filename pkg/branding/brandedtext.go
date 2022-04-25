package branding

const (
	productNameRHACS = "Red Hat Advanced Cluster Security for Kubernetes"

	productNameStackrox = "StackRox"
)

// GetProductName returns the product name based on ProductBranding env variable
func GetProductName() string {
	if getProductBrandingEnv() == ProductBrandingRHACS {
		return productNameRHACS
	}
	return productNameStackrox
}
