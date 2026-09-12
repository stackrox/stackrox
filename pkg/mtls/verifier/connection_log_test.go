package verifier

import (
	"crypto/tls"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamedGroupString(t *testing.T) {
	assert.Equal(t, "RSA", namedGroupString(0))
	assert.Equal(t, tls.CurveP256.String(), namedGroupString(tls.CurveP256))
	assert.Equal(t, tls.X25519MLKEM768.String(), namedGroupString(tls.X25519MLKEM768))
}

func TestApplyServerConnectionLogging(t *testing.T) {
	t.Parallel()

	t.Run("wraps existing VerifyConnection", func(t *testing.T) {
		t.Parallel()
		called := false
		serverConnectionLogger = func(tls.ConnectionState) {}
		t.Cleanup(func() {
			serverConnectionLogger = defaultServerConnectionLogger
		})

		cfg := &tls.Config{
			VerifyConnection: func(tls.ConnectionState) error {
				called = true
				return errors.New("verify failed")
			},
		}
		applyServerConnectionLogging(cfg)
		require.NotNil(t, cfg.VerifyConnection)

		err := cfg.VerifyConnection(tls.ConnectionState{
			Version:     tls.VersionTLS13,
			CipherSuite: tls.TLS_AES_128_GCM_SHA256,
			CurveID:     tls.X25519,
		})
		assert.EqualError(t, err, "verify failed")
		assert.True(t, called)
	})

	t.Run("no-op on nil config", func(t *testing.T) {
		t.Parallel()
		assert.NotPanics(t, func() {
			applyServerConnectionLogging(nil)
		})
	})
}
