package common

import (
	"embed"
	"encoding/base64"

	"github.com/stackrox/rox/pkg/utils"
)

const (
	logoFile = "files/red-hat-acs-logo-rgb.png"
)

var (
	//go:embed files/red-hat-acs-logo-rgb.png
	logoFS     embed.FS
	logoBase64 = func() string {
		bytes, err := logoFS.ReadFile(logoFile)
		utils.Must(err)
		return base64.StdEncoding.EncodeToString(bytes)
	}()
)

// GetLogoBase64 returns the logo bytes in base64 encoded string.
func GetLogoBase64() string {
	return logoBase64
}
