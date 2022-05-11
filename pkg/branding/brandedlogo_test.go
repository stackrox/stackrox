package branding

import (
	"testing"

	"github.com/cloudflare/cfssl/scan/crypto/sha256"
	"github.com/stretchr/testify/assert"
)

var (
	logoRHACSBase64hash    = [32]uint8{0x7f, 0x3e, 0x82, 0x96, 0x3a, 0x70, 0x5c, 0x41, 0xca, 0xc0, 0x96, 0xb5, 0x16, 0xde, 0xe0, 0x68, 0xec, 0x1e, 0x86, 0x93, 0xd5, 0x5e, 0xc6, 0xd8, 0x36, 0x54, 0x6b, 0x12, 0xc6, 0x17, 0xe1, 0x95}
	logoStackRoxBase64hash = [32]uint8{0x54, 0x5f, 0xa0, 0x92, 0xc7, 0x24, 0x1e, 0xc8, 0x7f, 0x1c, 0x2b, 0x7f, 0x7a, 0x79, 0x8e, 0x35, 0x7, 0x27, 0xf0, 0x3f, 0xff, 0xe5, 0x23, 0xcc, 0x7b, 0x68, 0x98, 0xf7, 0x2c, 0xf5, 0xbc, 0xe8}
)

func TestGetBrandedLogo(t *testing.T) {
	tests := map[string]struct {
		productBrandingEnv string
		brandedLogoHash    [32]byte
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
			receivedLogoHash := sha256.Sum256([]byte(GetLogoBase64()))
			assert.Equal(t, tt.brandedLogoHash, receivedLogoHash)
		})
	}

}
