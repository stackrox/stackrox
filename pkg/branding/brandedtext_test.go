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
			productBrandingEnv: "RHACS_BRANDING",
			brandedProductName: productBrandingNameRHACS,
		},
		"Stackrox branding": {
			productBrandingEnv: "STACKROX_BRANDING",
			brandedProductName: productBrandingNameStackrox,
		},
		"Default setting": {
			productBrandingEnv: "ROX_PRODUCT_BRANDING",
			brandedProductName: productBrandingNameStackrox,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			envIsolator.Setenv(ProductBranding, tt.productBrandingEnv)
			receivedProductName := GetBrandedProductName()
			assert.Equal(t, tt.brandedProductName, receivedProductName)
		})
	}
}
