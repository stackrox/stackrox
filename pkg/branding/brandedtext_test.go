package branding

import (
	"testing"

	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
)

const (
	brandedProductNameRHACS    = "Red Hat Advanced Cluster Security for Kubernetes"
	brandedProductNameStackrox = "StackRox"
)

func TestGetBrandedProductName(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)

	tests := map[string]struct {
		productBrandingEnv string
		brandedProductName string
	}{
		"RHACS branding": {
			productBrandingEnv: "RHACS_BRANDING",
			brandedProductName: brandedProductNameRHACS,
		},
		"Stackrox branding": {
			productBrandingEnv: "STACKROX_BRANDING",
			brandedProductName: brandedProductNameStackrox,
		},
		"Default setting": {
			productBrandingEnv: "ROX_PRODUCT_BRANDING",
			brandedProductName: brandedProductNameStackrox,
		},
		// TODO #ROX-10208: Change this to StackRox after changing the default value of ProductBrandingEnvName.
		"Unset env": {
			productBrandingEnv: "",
			brandedProductName: brandedProductNameRHACS,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			envIsolator.Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
			receivedProductName := GetProductName()
			assert.Equal(t, tt.brandedProductName, receivedProductName)
		})
	}
	envIsolator.RestoreAll()
}
