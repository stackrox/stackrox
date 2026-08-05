package cmd

import (
	"context"
	"crypto/x509"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validMappingJSON is the minimal content repositorytocpe.ValidateMapping accepts.
const validMappingJSON = `{"data":{}}`

// waitTimeout/waitTick bound assert.Eventually polling for background
// effects of a real (non-fake-clock) mapping update, e.g. onChange firing.
const (
	waitTimeout = 2 * time.Second
	waitTick    = 10 * time.Millisecond
)

// withMappingCachePath points the package-level mappingCachePath at a fresh
// file under a per-test temp dir, so tests never share state through it or
// touch the real hardcoded path, and restores it afterward.
func withMappingCachePath(t *testing.T) string {
	t.Helper()
	original := mappingCachePath
	path := filepath.Join(t.TempDir(), "cache.json")
	mappingCachePath = path
	t.Cleanup(func() { mappingCachePath = original })
	return path
}

func TestNewRescannerAndProvider_URLSelection(t *testing.T) {
	withMappingCachePath(t)

	tests := map[string]struct {
		repoCPEURL     string
		wantUpdatePath pb.RepoCPEMappingUpdatePath
		wantUpdaterNil bool
		wantURLUpdater bool
	}{
		"empty repo-cpe-url selects the Sensor-managed path": {
			repoCPEURL:     "",
			wantUpdatePath: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR,
			wantUpdaterNil: false,
			wantURLUpdater: false,
		},
		"non-empty repo-cpe-url selects the URL-managed path": {
			repoCPEURL:     "https://example.invalid/repo-to-cpe.json",
			wantUpdatePath: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL,
			wantUpdaterNil: true,
			wantURLUpdater: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := serveConfig{repoCPEURL: tt.repoCPEURL}
			_, provider, updater, urlUpdater := newRescannerAndProvider(&vsockserver.ReportCache{}, cfg)

			assert.Equal(t, tt.wantUpdatePath, provider.UpdatePath())
			assert.Equal(t, tt.wantUpdaterNil, updater == nil)
			assert.Equal(t, tt.wantURLUpdater, urlUpdater != nil)
		})
	}
}

// TestNewRescannerAndProvider_NoMappingStaysIdle covers agent startup with
// no mapping file present yet: the provider must report not-Ready instead
// of treating a missing file as a fetch error, so runServe never attempts
// a scan before a mapping actually arrives.
func TestNewRescannerAndProvider_NoMappingStaysIdle(t *testing.T) {
	for name, repoCPEURL := range map[string]string{"Sensor-managed": "", "URL-managed": "https://example.invalid/repo-to-cpe.json"} {
		t.Run(name, func(t *testing.T) {
			withMappingCachePath(t)
			_, provider, _, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{repoCPEURL: repoCPEURL})

			assert.False(t, provider.Ready(), "no cached mapping file must leave the provider not-Ready, not erroring")
		})
	}
}

// TestNewRescannerAndProvider_CallbackWiredAfterConstruction pins down the
// construction order: the rescanner's provider field is only assigned
// after the provider itself is built, but OnMappingChanged - already bound
// to the rescanner - must be safe to call throughout, and the assignment
// must have actually taken effect by the time construction returns.
func TestNewRescannerAndProvider_CallbackWiredAfterConstruction(t *testing.T) {
	withMappingCachePath(t)
	vmRescanner, provider, _, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{})

	assert.Same(t, provider, vmRescanner.provider, "the rescanner must observe the same provider instance constructed alongside it")
	assert.NotPanics(t, vmRescanner.OnMappingChanged)
}

// TestNewRescannerAndProvider_SyncKicksFirstScan simulates a Sensor-pushed
// mapping arriving (without any real VSOCK connection) and checks it wakes
// the rescanner into an immediate scan attempt via the onChange wiring.
// Runs on the real clock (not synctest): SensorUpdater.Update's fire-and-
// forget cache-persistence goroutine is real background I/O outside the
// rescanner's own fake-clock-driven loop.
func TestNewRescannerAndProvider_SyncKicksFirstScan(t *testing.T) {
	withMappingCachePath(t)
	vmRescanner, provider, updater, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{})
	require.False(t, provider.Ready())
	require.NotNil(t, updater, "Sensor path must produce a non-nil updater")

	var mu sync.Mutex
	var calls int
	vmRescanner.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
		concurrency.WithLock(&mu, func() { calls++ })
		return &v4.IndexReport{}, nil
	}
	vmRescanner.factsFn = func(string) map[string]string { return nil }
	vmRescanner.interval = time.Hour

	ctx, cancel := context.WithCancel(t.Context())
	stopped := vmRescanner.runAsync(ctx)
	defer func() {
		cancel()
		<-stopped
	}()

	updated, err := updater.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	require.True(t, updated)

	assert.Eventually(t, func() bool {
		return concurrency.WithLock1(&mu, func() int { return calls }) == 1
	}, waitTimeout, waitTick, "a Sync-style mapping update should kick an immediate scan attempt")
}

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
