package cmd

import (
	"context"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validMappingJSON is the minimal content cpemapping.ValidateMapping accepts.
// otherValidMappingJSON is a distinct valid payload for tests that need two
// different hashes (e.g. deferred-apply across a simulated scan).
const (
	validMappingJSON      = `{"data":{}}`
	otherValidMappingJSON = `{"data":{"rhel-9-server-rpms":{"cpes":["cpe:/o:redhat:enterprise_linux:9"]}}}`
)

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

// registerSensorPersistDrain waits for SensorUpdater's fire-and-forget cache
// writes before t.TempDir() cleanup, so AtomicWriteFile's sibling .tmp files
// cannot race directory removal.
func registerSensorPersistDrain(t *testing.T, updater vsockserver.MappingUpdater) {
	t.Helper()
	u, ok := updater.(*vsockserver.SensorUpdater)
	require.True(t, ok, "registerSensorPersistDrain requires a *SensorUpdater")
	t.Cleanup(u.WaitPersist)
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
			vmRescanner, provider, _, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{repoCPEURL: repoCPEURL})

			assert.False(t, provider.Ready(), "no cached mapping file must leave the provider not-Ready, not erroring")

			var scanCalls int
			vmRescanner.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
				scanCalls++
				return &v4.IndexReport{}, nil
			}
			require.NoError(t, vmRescanner.scanOnce(t.Context()))
			assert.Zero(t, scanCalls, "a not-Ready provider must never trigger a disk scan")
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
// mapping arriving (without any real VSOCK connection) and checks it
// triggers an immediate scan attempt via the onChange wiring.
// Runs on the real clock (not synctest): SensorUpdater.Update's fire-and-
// forget cache-persistence goroutine is real background I/O outside the
// rescanner's own fake-clock-driven loop.
func TestNewRescannerAndProvider_SyncKicksFirstScan(t *testing.T) {
	cachePath := withMappingCachePath(t)
	vmRescanner, provider, updater, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{rescanInterval: time.Hour})
	require.False(t, provider.Ready())
	require.NotNil(t, updater, "Sensor path must produce a non-nil updater")
	registerSensorPersistDrain(t, updater)

	var mu sync.Mutex
	var calls int
	vmRescanner.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
		concurrency.WithLock(&mu, func() { calls++ })
		return &v4.IndexReport{}, nil
	}
	vmRescanner.factsFn = func(string) map[string]string { return nil }

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

	// SensorUpdater.Update persists to disk via a fire-and-forget goroutine
	// (see vsockserver.SensorUpdater.persistAndNotify), so the on-disk
	// content lands independently of, and possibly after, the scan kick
	// asserted above.
	assert.Eventually(t, func() bool {
		content, err := os.ReadFile(cachePath)
		return err == nil && string(content) == validMappingJSON
	}, waitTimeout, waitTick, "a synced mapping should also land in the on-disk cache")
}

// TestNewRescannerAndProvider_BundledPathSeedsSensor covers cmd passing
// --repo-cpe-bundled-path through to NewSensorUpdater: a seed file is
// enough to become Ready and trigger the first scan, without a Sensor push.
func TestNewRescannerAndProvider_BundledPathSeedsSensor(t *testing.T) {
	withMappingCachePath(t)
	bundledPath := filepath.Join(t.TempDir(), "bundled.json")
	require.NoError(t, os.WriteFile(bundledPath, []byte(validMappingJSON), 0o600))

	vmRescanner, provider, updater, urlUpdater := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{
		repoCPEBundledPath: bundledPath,
		rescanInterval:     time.Hour,
	})
	require.True(t, provider.Ready(), "a Sensor-managed seed file must bootstrap the provider Ready")
	require.NotNil(t, updater)
	require.Nil(t, urlUpdater)
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), provider.Hash())

	var mu sync.Mutex
	var calls int
	vmRescanner.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
		concurrency.WithLock(&mu, func() { calls++ })
		return &v4.IndexReport{}, nil
	}
	vmRescanner.factsFn = func(string) map[string]string { return nil }

	ctx, cancel := context.WithCancel(t.Context())
	stopped := vmRescanner.runAsync(ctx)
	defer func() {
		cancel()
		<-stopped
	}()

	assert.Eventually(t, func() bool {
		return concurrency.WithLock1(&mu, func() int { return calls }) == 1
	}, waitTimeout, waitTick, "bootstrapping from the bundled seed should kick an immediate scan attempt")
}

// TestNewRescannerAndProvider_URLIgnoresBundledPath covers URL-managed
// plus --repo-cpe-bundled-path: the seed is Sensor-only, so a URL that
// has never succeeded stays not-ready instead of looking configured.
func TestNewRescannerAndProvider_URLIgnoresBundledPath(t *testing.T) {
	withMappingCachePath(t)
	bundledPath := filepath.Join(t.TempDir(), "bundled.json")
	require.NoError(t, os.WriteFile(bundledPath, []byte(validMappingJSON), 0o600))

	_, provider, updater, urlUpdater := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{
		repoCPEURL:         "https://example.invalid/repo-to-cpe.json",
		repoCPEBundledPath: bundledPath,
	})
	require.NotNil(t, urlUpdater)
	require.Nil(t, updater)
	assert.False(t, provider.Ready(), "URL-managed must not seed from --repo-cpe-bundled-path")
	assert.Equal(t, pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL, provider.UpdatePath())
}

// TestNewRescannerAndProvider_URLNeverSucceeds_StaysNotSensorManaged covers
// a configured --repo-cpe-url that never produces a usable mapping: the
// update path must stay URL, with no fallback to Sensor, even once the
// downloader has actually attempted and failed a fetch. Log-level (ERROR
// vs WARN) is not asserted here: the codebase has no lightweight log-
// capture convention (see report), and URLUpdater.onDownloadComplete only
// ever logs WARN today regardless of whether a mapping was ever
// successfully fetched — see the task report for this gap.
func TestNewRescannerAndProvider_URLNeverSucceeds_StaysNotSensorManaged(t *testing.T) {
	withMappingCachePath(t)

	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, provider, updater, urlUpdater := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{repoCPEURL: srv.URL})
	require.Nil(t, updater)
	require.NotNil(t, urlUpdater)

	ctx, cancel := context.WithCancel(t.Context())
	require.NoError(t, urlUpdater.Start(ctx))
	defer func() {
		cancel()
		urlUpdater.Stop()
	}()

	assert.Eventually(t, func() bool { return requests.Load() > 0 }, waitTimeout, waitTick,
		"the downloader should have attempted at least one fetch against the configured URL")

	assert.False(t, provider.Ready(), "a URL that never succeeds must never make the provider Ready")
	assert.Equal(t, pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL, provider.UpdatePath(),
		"a failing URL must not fall back to Sensor management")
}

// TestNewRescannerAndProvider_RestartSwitchesUpdatePath simulates an
// operator adding or removing --repo-cpe-url between two agent restarts
// that share the same on-disk mappingCachePath. Per the design, the
// update path is re-derived purely from cfg.repoCPEURL on every restart,
// never persisted; this table only pins that re-derivation, not whether
// the previous updater's cached bytes get re-read (see report: cmd wires
// both updaters to the same mappingCachePath file, so unlike the design's
// "own cache file" wording, content bootstrapped by one updater is in
// fact visible to the other after a switch).
func TestNewRescannerAndProvider_RestartSwitchesUpdatePath(t *testing.T) {
	tests := map[string]struct {
		firstURL  string
		secondURL string
		wantFirst pb.RepoCPEMappingUpdatePath
		wantAfter pb.RepoCPEMappingUpdatePath
	}{
		"adding --repo-cpe-url switches Sensor-managed to URL-managed": {
			firstURL:  "",
			secondURL: "https://example.invalid/repo-to-cpe.json",
			wantFirst: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR,
			wantAfter: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL,
		},
		"removing --repo-cpe-url switches URL-managed to Sensor-managed": {
			firstURL:  "https://example.invalid/repo-to-cpe.json",
			secondURL: "",
			wantFirst: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_URL,
			wantAfter: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			withMappingCachePath(t)

			_, firstProvider, _, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{repoCPEURL: tt.firstURL})
			require.Equal(t, tt.wantFirst, firstProvider.UpdatePath())

			_, secondProvider, _, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{repoCPEURL: tt.secondURL})
			assert.Equal(t, tt.wantAfter, secondProvider.UpdatePath(),
				"a fresh restart must re-derive the update path from the current --repo-cpe-url, not remember the prior one")
		})
	}
}

// TestNewRescannerAndProvider_SensorSync_DefersUnderScanThenApplies covers
// the cmd-level wiring for deferred apply: SensorUpdater's own deferred-
// apply logic is unit-tested in vsockserver/sensor_updater_test.go, so this
// only checks that newRescannerAndProvider's returned updater/provider
// pair exposes that behavior end-to-end through the ScanBusyGate interface.
func TestNewRescannerAndProvider_SensorSync_DefersUnderScanThenApplies(t *testing.T) {
	cachePath := withMappingCachePath(t)
	require.NoError(t, os.WriteFile(cachePath, []byte(validMappingJSON), 0o600))

	_, provider, updater, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{})
	require.True(t, provider.Ready(), "a pre-seeded Sensor cache must bootstrap the provider Ready")
	require.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), provider.Hash())
	registerSensorPersistDrain(t, updater)

	gate, ok := updater.(vsockserver.ScanBusyGate)
	require.True(t, ok, "the Sensor updater must implement ScanBusyGate")
	gate.MarkScanBusy()

	updated, err := updater.Update([]byte(otherValidMappingJSON))
	require.NoError(t, err)
	require.True(t, updated)

	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), provider.Hash(),
		"the active mapping must stay the bootstrapped content while a scan is in flight")

	gate.MarkScanIdleAndApplyPending()

	assert.Equal(t, cpemapping.HashMapping([]byte(otherValidMappingJSON)), provider.Hash(),
		"the pending mapping should apply once the scan is marked idle")

	// MarkScanIdleAndApplyPending's persist is also fire-and-forget; wait
	// for it so the background write can't still be touching cachePath's
	// directory when t.TempDir() cleans it up.
	assert.Eventually(t, func() bool {
		content, err := os.ReadFile(cachePath)
		return err == nil && string(content) == otherValidMappingJSON
	}, waitTimeout, waitTick, "the applied mapping should also land in the on-disk cache")
}

// TestNewRescannerAndProvider_SensorUpdate_InvalidContentKeepsLastGood
// covers the cmd-level wiring only; SensorUpdater's own validation
// behavior is unit-tested in vsockserver/sensor_updater_test.go.
func TestNewRescannerAndProvider_SensorUpdate_InvalidContentKeepsLastGood(t *testing.T) {
	cachePath := withMappingCachePath(t)
	_, provider, updater, _ := newRescannerAndProvider(&vsockserver.ReportCache{}, serveConfig{})
	registerSensorPersistDrain(t, updater)

	updated, err := updater.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	require.True(t, updated)
	wantHash := provider.Hash()
	// Drain the first Update's fire-and-forget persist before the test ends,
	// so its background write can't still be touching cachePath's directory
	// when t.TempDir() cleans it up.
	assert.Eventually(t, func() bool {
		content, err := os.ReadFile(cachePath)
		return err == nil && string(content) == validMappingJSON
	}, waitTimeout, waitTick, "the initial valid mapping should land in the on-disk cache")

	updated, err = updater.Update([]byte(`{not valid json`))
	assert.Error(t, err)
	assert.False(t, updated)
	assert.Equal(t, wantHash, provider.Hash(), "invalid content must not replace the last-known-good mapping")
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
