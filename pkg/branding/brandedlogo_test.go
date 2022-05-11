package branding

import (
	"embed"
	"encoding/base64"
	"testing"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
)

var (
	//go:embed files/*.png
	logoFStest embed.FS

	logoRHACSBase64test = func() string {
		bytes, err := logoFStest.ReadFile("files/red_hat_acs_logo_rgb.png")
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()

	logoStackRoxBase64test = func() string {
		bytes, err := logoFStest.ReadFile("files/stackrox_logo_wide_dkblue.png")
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()
)

func TestGetBrandedLogo(t *testing.T) {
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
		t.Run(name, func(t *testing.T) {
			t.Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
			receivedLogo := GetLogoBase64()
			assert.Equal(t, tt.brandedLogo, receivedLogo)
		})
	}

}
