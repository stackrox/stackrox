package branding

import (
	"testing"

	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
)

func TestGetBrandedProductName(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)

	tests := map[string]struct {
		productBrandingEnv string
		brandedProductName string
	}{
		"RHACS branding": {
			productBrandingEnv: ProductBrandingRHACS,
			brandedProductName: productNameRHACS,
		},
		"Stackrox branding": {
			productBrandingEnv: "STACKROX_BRANDING",
			brandedProductName: productNameStackrox,
		},
		"Default setting": {
			productBrandingEnv: "ROX_PRODUCT_BRANDING",
			brandedProductName: productNameStackrox,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			envIsolator.Setenv(ProductBrandingEnvName, tt.productBrandingEnv)
			receivedProductName := GetProductName()
			assert.Equal(t, tt.brandedProductName, receivedProductName)
		})
	}
}
