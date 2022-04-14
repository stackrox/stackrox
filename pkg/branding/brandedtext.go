package branding

const (
	productBrandingNameRHACS = "Red Hat Advanced Cluster Security for Kubernetes"

	productBrandingNameStackrox = "StackRox"
)

func GetBrandedProductName() string {
	if GetProductBrandingEnv() == "RHACS_BRANDING" {
		return productBrandingNameRHACS
	}
	return productBrandingNameStackrox
}
