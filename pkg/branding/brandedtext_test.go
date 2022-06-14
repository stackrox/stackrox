package branding

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const (
	brandedProductNameRHACS    = "Red Hat Advanced Cluster Security for Kubernetes"
	brandedProductNameStackrox = "StackRox"
)

var _ suite.SetupAllSuite = (*BrandedTextTestSuite)(nil)
var _ suite.TearDownTestSuite = (*BrandedTextTestSuite)(nil)

func TestBrandedText(t *testing.T) {
	suite.Run(t, new(BrandedTextTestSuite))
}

type BrandedTextTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *BrandedTextTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *BrandedTextTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
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
			s.envIsolator.Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
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
			s.envIsolator.Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
			receivedProductNameShort := GetProductNameShort()
			s.Equal(tt.brandedProductNameShort, receivedProductNameShort)
		})
	}
}
