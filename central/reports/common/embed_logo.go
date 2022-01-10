package common

import (
	"embed"
	"encoding/base64"

	"github.com/pkg/errors"
)

const (
	logoFile = "files/red-hat-acs-logo-rgb.png"
)

var (
	//go:embed files/red-hat-acs-logo-rgb.png
	logoFS embed.FS
)

// GetLogo reads and returns the logo bytes in base64 encoded string.
func GetLogo() (string, error) {
	bytes, err := logoFS.ReadFile(logoFile)
	if err != nil {
		return "", errors.Wrapf(err, "could not read logo from %q", logoFile)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
