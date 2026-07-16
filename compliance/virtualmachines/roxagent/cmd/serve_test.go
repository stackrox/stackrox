package cmd

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validServeConfig returns a serveConfig that passes validate(), so each
// test case in TestRunServe_ValidatesFlags only needs to override the one
// field it's exercising.
func validServeConfig() serveConfig {
	return serveConfig{
		rescanInterval: minRescanInterval,
		caFetchTimeout: time.Second,
		connDeadline:   minConnDeadline,
	}
}

// TestRunServe_ValidatesFlags exercises runServe's argument validation,
// which - unlike the rest of runServe - never touches the filesystem,
// network, or VSOCK, so it is cheap to cover without a real host/agent
// environment. port/hostPath/repoCPEURL are irrelevant to these cases,
// since validation returns before any of them are used.
func TestRunServe_ValidatesFlags(t *testing.T) {
	tests := map[string]struct {
		mutate      func(*serveConfig)
		errContains string
	}{
		"should error when rescan interval is zero": {
			mutate:      func(c *serveConfig) { c.rescanInterval = 0 },
			errContains: "rescan-interval",
		},
		"should error when rescan interval is negative": {
			mutate:      func(c *serveConfig) { c.rescanInterval = -time.Second },
			errContains: "rescan-interval",
		},
		"should error when rescan interval is below the minimum": {
			mutate:      func(c *serveConfig) { c.rescanInterval = minRescanInterval - time.Second },
			errContains: "rescan-interval",
		},
		"should error when rescan interval is above the maximum": {
			mutate:      func(c *serveConfig) { c.rescanInterval = maxRescanInterval + time.Hour },
			errContains: "rescan-interval",
		},
		"should error when ca fetch timeout is zero": {
			mutate:      func(c *serveConfig) { c.caFetchTimeout = 0 },
			errContains: "ca-fetch-timeout",
		},
		"should error when ca fetch timeout is negative": {
			mutate:      func(c *serveConfig) { c.caFetchTimeout = -time.Second },
			errContains: "ca-fetch-timeout",
		},
		"should error when conn deadline is below the minimum": {
			mutate:      func(c *serveConfig) { c.connDeadline = minConnDeadline - time.Second },
			errContains: "conn-deadline",
		},
		"should error when conn deadline is above the maximum": {
			mutate:      func(c *serveConfig) { c.connDeadline = maxConnDeadline + time.Second },
			errContains: "conn-deadline",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := validServeConfig()
			tt.mutate(&cfg)
			err := runServe(t.Context(), cfg)
			assert.ErrorContains(t, err, tt.errContains)
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
