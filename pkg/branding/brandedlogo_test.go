package branding

import (
	"encoding/hex"
	"testing"

	"github.com/cloudflare/cfssl/scan/crypto/sha256"
	"github.com/stretchr/testify/assert"
)

const (
	// run "find pkg/branding/files -type f -print -exec bash -c 'base64 -w0 {} | sha256sum' \;" to generate hashes
	// or where base64 doesn't support a -w0 option (eg. MacOS)
	// run "find pkg/branding/files -type f -print -exec bash -c 'base64 {} | tr -d \\n | sha256sum' \;" instead
	logoRHACSBase64hash    = "7f3e82963a705c41cac096b516dee068ec1e8693d55ec6d836546b12c617e195"
	logoStackRoxBase64hash = "545fa092c7241ec87f1c2b7f7a798e350727f03fffe523cc7b6898f72cf5bce8"
)

func TestGetBrandedLogo(t *testing.T) {
	tests := map[string]struct {
		productBrandingEnv string
		brandedLogoHash    string
	}{
		"RHACS branding": {
			productBrandingEnv: "RHACS_BRANDING",
			brandedLogoHash:    logoRHACSBase64hash,
		},
		"Stackrox branding": {
			productBrandingEnv: "STACKROX_BRANDING",
			brandedLogoHash:    logoStackRoxBase64hash,
		},
		"Unset env": {
			productBrandingEnv: "",
			brandedLogoHash:    logoStackRoxBase64hash,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ROX_PRODUCT_BRANDING", tt.productBrandingEnv)
			logoHashBytes := sha256.Sum256([]byte(GetLogoBase64()))
			receivedLogoHash := hex.EncodeToString(logoHashBytes[:])
			assert.Equal(t, tt.brandedLogoHash, receivedLogoHash)
		})
	}

}
