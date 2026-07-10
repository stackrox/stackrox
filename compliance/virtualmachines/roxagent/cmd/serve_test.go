package cmd

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunServe_ValidatesFlags exercises runServe's argument validation,
// which - unlike the rest of runServe - never touches the filesystem,
// network, or VSOCK, so it is cheap to cover without a real host/agent
// environment. port/hostPath/repoCPEURL are irrelevant to these cases,
// since validation returns before any of them are used.
func TestRunServe_ValidatesFlags(t *testing.T) {
	tests := map[string]struct {
		rescanEvery    time.Duration
		caFetchTimeout time.Duration
		errContains    string
	}{
		"should error when rescan interval is zero": {
			rescanEvery:    0,
			caFetchTimeout: time.Second,
			errContains:    "rescan-interval",
		},
		"should error when rescan interval is negative": {
			rescanEvery:    -time.Second,
			caFetchTimeout: time.Second,
			errContains:    "rescan-interval",
		},
		"should error when ca fetch timeout is zero": {
			rescanEvery:    time.Second,
			caFetchTimeout: 0,
			errContains:    "ca-fetch-timeout",
		},
		"should error when ca fetch timeout is negative": {
			rescanEvery:    time.Second,
			caFetchTimeout: -time.Second,
			errContains:    "ca-fetch-timeout",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := runServe(t.Context(), 0, "", "", tt.rescanEvery, tt.caFetchTimeout)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestDiscoverFacts(t *testing.T) {
	facts := discoverFacts(t.TempDir())

	assert.Contains(t, facts, "detected_os")
	assert.Contains(t, facts, "os_version")
	assert.Contains(t, facts, "activation_status")
	assert.Contains(t, facts, "dnf_metadata_status")
}

func TestSelfSignedCert(t *testing.T) {
	cert, err := selfSignedCert()
	require.NoError(t, err)
	require.Len(t, cert.Certificate, 1)
	require.NotNil(t, cert.PrivateKey)

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	assert.Contains(t, parsed.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	assert.True(t, parsed.NotBefore.Before(time.Now()), "cert should already be valid")
	assert.True(t, parsed.NotAfter.After(time.Now()), "cert should not be expired")

	// Self-signed: issuer and subject are the same key, so it must verify
	// against a pool containing only itself.
	pool := x509.NewCertPool()
	pool.AddCert(parsed)
	_, err = parsed.Verify(x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
	assert.NoError(t, err)
}
