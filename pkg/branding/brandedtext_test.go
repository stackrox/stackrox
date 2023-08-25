package branding

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	brandedProductNameRHACS    = "Red Hat Advanced Cluster Security for Kubernetes"
	brandedProductNameStackrox = "StackRox"
)

func TestBrandedText(t *testing.T) {
	suite.Run(t, new(BrandedTextTestSuite))
}

type BrandedTextTestSuite struct {
	suite.Suite
}

func (s *BrandedTextTestSuite) TestGetBrandedProductName() {
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
		"Unset env": {
			productBrandingEnv: "",
			brandedProductName: brandedProductNameStackrox,
		},
	}
	for name, tt := range tests {
		s.Run(name, func() {
			s.T().Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
			receivedProductName := GetProductName()
			s.Equal(tt.brandedProductName, receivedProductName)
		})
	}
}

func (s *BrandedTextTestSuite) TestGetBrandedProductNameShort() {
	tests := map[string]struct {
		productBrandingEnv      string
		brandedProductNameShort string
	}{
		"RHACS branding": {
			productBrandingEnv:      "RHACS_BRANDING",
			brandedProductNameShort: productNameRHACSShort,
		},
		"Stackrox branding": {
			productBrandingEnv:      "STACKROX_BRANDING",
			brandedProductNameShort: brandedProductNameStackrox,
		},
		"Unset env": {
			productBrandingEnv:      "",
			brandedProductNameShort: brandedProductNameStackrox,
		},
	}
	for name, tt := range tests {
		s.Run(name, func() {
			s.T().Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
			receivedProductNameShort := GetProductNameShort()
			s.Equal(tt.brandedProductNameShort, receivedProductNameShort)
		})
	}
}
