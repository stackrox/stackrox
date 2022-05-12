package branding

import (
	"embed"
	"encoding/base64"

	"github.com/stackrox/rox/pkg/utils"
)

const (
	rhacslogoFile    = "files/red_hat_acs_logo_rgb.png"
	stackroxLogoFile = "files/stackrox_integration_logo.png"
)

var (
	//go:embed files/*.png
	logoFS          embed.FS
	logoRHACSBase64 = func() string {
		bytes, err := logoFS.ReadFile(rhacslogoFile)
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()

	logoStackRoxBase64 = func() string {
		bytes, err := logoFS.ReadFile(stackroxLogoFile)
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()
)

// GetLogoBase64 returns the logo bytes in base64 encoded string.
func GetLogoBase64() string {
	if getProductBrandingEnv() == ProductBrandingRHACS {
		return logoRHACSBase64
	}
	return logoStackRoxBase64
}
