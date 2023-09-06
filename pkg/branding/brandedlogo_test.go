package branding

import (
	"encoding/hex"
	"testing"

	"github.com/cloudflare/cfssl/scan/crypto/sha256"
	"github.com/stretchr/testify/assert"
)

const (
	// To re-generate hashes when files change, run:
	// find pkg/branding/files -type f -print -exec bash -c 'base64 {} | tr -d \\n | sha256sum' \;
	logoRHACSBase64hash    = "7f3e82963a705c41cac096b516dee068ec1e8693d55ec6d836546b12c617e195" //#nosec G101
	logoStackRoxBase64hash = "318908997d28eb54a31305290ded071bdd61d5dc8718b75dfb4a7a6ba4c162d8" //#nosec G101
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
