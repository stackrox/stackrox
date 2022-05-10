package branding

import (
	"embed"
	"encoding/base64"
	"testing"

	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/suite"
)

var (
	//go:embed files/red-hat-acs-logo-rgb.png
	logoRHACStest       embed.FS
	logoRHACSBase64test = func() string {
		bytes, err := logoRHACStest.ReadFile("files/red-hat-acs-logo-rgb.png")
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()
	//go:embed files/StackRox_Logo_Wide_DkBlue.png
	logoStackRoxtest       embed.FS
	logoStackRoxBase64test = func() string {
		bytes, err := logoStackRoxtest.ReadFile("files/StackRox_Logo_Wide_DkBlue.png")
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()
)

var _ suite.SetupAllSuite = (*BrandedLogoTestSuite)(nil)
var _ suite.TearDownTestSuite = (*BrandedLogoTestSuite)(nil)

func TestBrandedLogo(t *testing.T) {
	suite.Run(t, new(BrandedLogoTestSuite))
}

type BrandedLogoTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *BrandedLogoTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *BrandedLogoTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *BrandedLogoTestSuite) TestGetBrandedLogo() {
	tests := map[string]struct {
		productBrandingEnv string
		brandedLogo        string
	}{
		"RHACS branding": {
			productBrandingEnv: "RHACS_BRANDING",
			brandedLogo:        logoRHACSBase64test,
		},
		"Stackrox branding": {
			productBrandingEnv: "STACKROX_BRANDING",
			brandedLogo:        logoStackRoxBase64test,
		},
		"Unset env": {
			productBrandingEnv: "",
			brandedLogo:        logoStackRoxBase64test,
		},
	}
	for name, tt := range tests {
		s.Run(name, func() {
			s.envIsolator.Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
			receivedLogo := GetLogoBase64()
			s.Equal(tt.brandedLogo, receivedLogo)
		})
	}

}
