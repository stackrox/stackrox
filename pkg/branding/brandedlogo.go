package branding

import (
	"embed"
	"encoding/base64"

	"github.com/stackrox/rox/pkg/utils"
)

const (
	rhacslogoFile    = "files/red-hat-acs-logo-rgb.png"
	stackroxLogoFile = "files/StackRox_Logo_Wide_DkBlue.png"
)

var (
	//go:embed files/red-hat-acs-logo-rgb.png
	logoRHACS       embed.FS
	logoRHACSBase64 = func() string {
		bytes, err := logoRHACS.ReadFile(rhacslogoFile)
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()
	//go:embed files/StackRox_Logo_Wide_DkBlue.png
	logoStackRox       embed.FS
	logoStackRoxBase64 = func() string {
		bytes, err := logoStackRox.ReadFile(stackroxLogoFile)
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()
)

func getLogoFile() string {
	if getProductBrandingEnv() == ProductBrandingRHACS {
		return rhacslogoFile
	}
	return stackroxLogoFile
}

// GetLogoBase64 returns the logo bytes in base64 encoded string.
func GetLogoBase64() string {
	if getProductBrandingEnv() == ProductBrandingRHACS {
		return logoRHACSBase64
	}
	return logoStackRoxBase64
}
