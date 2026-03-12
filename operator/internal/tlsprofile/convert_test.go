package tlsprofile

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertMinVersion(t *testing.T) {
	tests := []struct {
		input   configv1.TLSProtocolVersion
		want    string
		wantErr bool
	}{
		{configv1.VersionTLS10, "TLSv1.0", false},
		{configv1.VersionTLS11, "TLSv1.1", false},
		{configv1.VersionTLS12, "TLSv1.2", false},
		{configv1.VersionTLS13, "TLSv1.3", false},
		// These are in library-go's format; only the configv1 constants are valid.
		{"TLSv1.2", "", true},
		{"1.2", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got, err := convertMinVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertCiphersToIANA(t *testing.T) {
	t.Run("intermediate profile ciphers", func(t *testing.T) {
		input := []string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
			"ECDHE-ECDSA-AES128-GCM-SHA256",
			"ECDHE-RSA-AES128-GCM-SHA256",
			"ECDHE-ECDSA-AES256-GCM-SHA384",
			"ECDHE-RSA-AES256-GCM-SHA384",
			"ECDHE-ECDSA-CHACHA20-POLY1305",
			"ECDHE-RSA-CHACHA20-POLY1305",
			"DHE-RSA-AES128-GCM-SHA256",
			"DHE-RSA-AES256-GCM-SHA384",
		}
		result := convertCiphersToIANA(input)

		assert.NotContains(t, result, "TLS_AES_128_GCM_SHA256", "TLS 1.3 ciphers should be excluded")
		assert.Contains(t, result, "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256")
		assert.Contains(t, result, "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384")
		assert.Contains(t, result, "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256")
		assert.NotContains(t, result, "DHE-RSA", "DHE ciphers not supported by Go should be excluded")
	})

	t.Run("unknown ciphers are skipped", func(t *testing.T) {
		input := []string{"ECDHE-ECDSA-AES128-GCM-SHA256", "UNKNOWN-CIPHER"}
		result := convertCiphersToIANA(input)
		assert.Equal(t, "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := convertCiphersToIANA(nil)
		assert.Equal(t, "", result)
	})
}

func TestConvertCiphersToOpenSSL(t *testing.T) {
	t.Run("skips TLS 1.3 ciphers", func(t *testing.T) {
		input := []string{
			"TLS_AES_128_GCM_SHA256",
			"ECDHE-ECDSA-AES128-GCM-SHA256",
			"ECDHE-RSA-AES256-GCM-SHA384",
		}
		result := convertCiphersToOpenSSL(input)
		assert.Equal(t, "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384", result)
	})

	t.Run("empty input", func(t *testing.T) {
		result := convertCiphersToOpenSSL(nil)
		assert.Equal(t, "", result)
	})
}
