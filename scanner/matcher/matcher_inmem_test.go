package matcher

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/toolkit/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/scanner/config"
	"github.com/stackrox/rox/scanner/datastore/inmem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestBundleZip(t *testing.T, modTime time.Time) []byte {
	t.Helper()
	ctx := context.Background()

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)

	writeEntry := func(entryName, updaterName string, vulns []*claircore.Vulnerability) {
		js, err := jsonblob.New()
		require.NoError(t, err)
		_, err = js.UpdateVulnerabilities(ctx, updaterName, driver.Fingerprint("fp-"+updaterName), vulns)
		require.NoError(t, err)

		var raw bytes.Buffer
		require.NoError(t, js.Store(&raw))

		hdr := &zip.FileHeader{
			Name:     entryName,
			Method:   zip.Store,
			Modified: modTime,
		}
		w, err := zw.CreateHeader(hdr)
		require.NoError(t, err)
		zwEnc, err := zstd.NewWriter(w)
		require.NoError(t, err)
		_, err = zwEnc.Write(raw.Bytes())
		require.NoError(t, err)
		require.NoError(t, zwEnc.Close())
	}

	writeEntry("bundles/alpine.json.zst", "alpine-3.19-main", []*claircore.Vulnerability{
		{
			Name:               "CVE-2025-9999",
			Description:        "test alpine vulnerability",
			Links:              "https://security.alpinelinux.org/vuln/CVE-2025-9999",
			Severity:           "High",
			NormalizedSeverity: claircore.High,
			FixedInVersion:     "3.1.0-r0",
			Package: &claircore.Package{
				Name: "openssl",
				Kind: types.SourcePackage,
			},
			Dist: &claircore.Distribution{
				DID:       "alpine",
				Name:      "Alpine Linux",
				VersionID: "3.19",
				Version:   "3.19",
			},
		},
	})

	writeEntry("bundles/rhel-vex.json.zst", "rhel-vex", []*claircore.Vulnerability{
		{
			Name:               "CVE-2025-8888",
			Description:        "test rhel vulnerability",
			Links:              "https://access.redhat.com/security/cve/CVE-2025-8888",
			NormalizedSeverity: claircore.Medium,
			Package: &claircore.Package{
				Name: "bash",
				Kind: types.BinaryPackage,
			},
			Repo: &claircore.Repository{
				Name: "cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*",
				Key:  "rhel-cpe-repository",
			},
		},
	})

	require.NoError(t, zw.Close())
	return zipBuf.Bytes()
}

func TestNewMatcherInMemEndToEnd(t *testing.T) {
	modTime := time.Now().UTC().Truncate(time.Second)
	zipBytes := buildTestBundleZip(t, modTime)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "vulnerabilities.zip", modTime, bytes.NewReader(zipBytes))
	}))
	defer srv.Close()

	ctx := context.Background()
	cfg := config.MatcherConfig{
		Enable:              true,
		Store:               config.MatcherStoreInMem,
		InMemBlobDir:        t.TempDir(),
		Readiness:           config.ReadinessDatabase, // must still wait for vulns in inmem mode
		VulnerabilitiesURL:  srv.URL,
		VulnerabilitiesURLs: []string{srv.URL},
	}
	m, err := NewMatcher(ctx, cfg, nil)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, m.Close(ctx))
	}()

	require.Eventually(t, func() bool {
		return m.Initialized(ctx) == nil && m.Ready(ctx) == nil
	}, 15*time.Second, 50*time.Millisecond, "inmem matcher should load bundle and become ready without Postgres")

	// Known distributions must include both Alpine and CPE-derived RHEL 9.
	dists := m.GetKnownDistributions(ctx)
	assert.Contains(t, dists, claircore.Distribution{DID: "alpine", VersionID: "3.19", Version: "3.19"})
	assert.Contains(t, dists, claircore.Distribution{DID: "rhel", VersionID: "9", Version: "9"})

	// Scan an IndexReport with one vulnerable and one patched package to prove
	// libvuln.Scan + alpine.Matcher + opts.Vulnerable + mmap display work end-to-end.
	alpineDist := &claircore.Distribution{
		ID:        "dist-1",
		DID:       "alpine",
		Name:      "Alpine Linux",
		VersionID: "3.19",
		Version:   "3.19",
	}
	vulnPkg := &claircore.Package{
		ID:      "pkg-vuln",
		Name:    "openssl",
		Version: "3.0.0-r0",
		Kind:    types.BinaryPackage,
		Source: &claircore.Package{
			ID:      "src-vuln",
			Name:    "openssl",
			Version: "3.0.0-r0",
			Kind:    types.SourcePackage,
		},
	}
	fixedPkg := &claircore.Package{
		ID:      "pkg-fixed",
		Name:    "openssl",
		Version: "3.2.0-r0",
		Kind:    types.BinaryPackage,
		Source: &claircore.Package{
			ID:      "src-fixed",
			Name:    "openssl",
			Version: "3.2.0-r0",
			Kind:    types.SourcePackage,
		},
	}
	ir := &claircore.IndexReport{
		State:   "IndexFinished",
		Success: true,
		Packages: map[string]*claircore.Package{
			vulnPkg.ID:  vulnPkg,
			fixedPkg.ID: fixedPkg,
		},
		Distributions: map[string]*claircore.Distribution{
			alpineDist.ID: alpineDist,
		},
		Environments: map[string][]*claircore.Environment{
			vulnPkg.ID:  {{DistributionID: alpineDist.ID}},
			fixedPkg.ID: {{DistributionID: alpineDist.ID}},
		},
	}

	vr, err := m.GetVulnerabilities(ctx, ir)
	require.NoError(t, err)
	require.Len(t, vr.PackageVulnerabilities[vulnPkg.ID], 1, "vulnerable package should match CVE-2025-9999")
	assert.Empty(t, vr.PackageVulnerabilities[fixedPkg.ID], "patched package must be filtered out by opts.Vulnerable")

	matchedID := vr.PackageVulnerabilities[vulnPkg.ID][0]
	v := vr.Vulnerabilities[matchedID]
	require.NotNil(t, v)
	assert.Equal(t, "CVE-2025-9999", v.Name)
	assert.Equal(t, "test alpine vulnerability", v.Description)
	assert.Equal(t, "https://security.alpinelinux.org/vuln/CVE-2025-9999", v.Links)
}

func TestConcurrentLibvulnScanDuringLiveRefresh(t *testing.T) {
	ctx := context.Background()
	store, err := inmem.NewMatcherStore(inmem.Options{BlobDir: t.TempDir()})
	require.NoError(t, err)
	defer store.Close()

	alpineDist := &claircore.Distribution{
		ID: "d-alp", DID: "alpine", Name: "Alpine Linux", VersionID: "3.19", Version: "3.19",
	}
	debDist := &claircore.Distribution{
		ID: "d-deb", DID: "debian", Name: "Debian GNU/Linux", VersionID: "12", Version: "12 (bookworm)",
	}

	seed := func(gen int) {
		_, err := store.UpdateVulnerabilities(ctx, "alpine-3.19", driver.Fingerprint("a-"+time.Duration(gen).String()), []*claircore.Vulnerability{
			{
				Name: "CVE-ALP-1", Description: "alp desc", Links: "https://alp",
				FixedInVersion: "2.0-r0",
				Package:        &claircore.Package{Name: "openssl", Kind: types.SourcePackage},
				Dist:           alpineDist,
			},
		})
		require.NoError(t, err)
		_, err = store.UpdateVulnerabilities(ctx, "debian-12", driver.Fingerprint("d-"+time.Duration(gen).String()), []*claircore.Vulnerability{
			{
				Name: "CVE-DEB-1", Description: "deb desc", Links: "https://deb",
				FixedInVersion: "2.0",
				Package:        &claircore.Package{Name: "curl", Kind: types.SourcePackage},
				Dist:           debDist,
			},
		})
		require.NoError(t, err)
	}
	seed(0)

	lv, err := libvuln.New(ctx, &libvuln.Options{
		Store:                    store,
		Locker:                   inmem.NewLocker(),
		MatcherNames:             []string{"alpine-matcher", "debian-matcher"},
		DisableBackgroundUpdates: true,
		Client:                   http.DefaultClient,
	})
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, lv.Close(ctx))
	}()

	ir := &claircore.IndexReport{
		State:   "IndexFinished",
		Success: true,
		Packages: map[string]*claircore.Package{
			"p-alp": {
				ID: "p-alp", Name: "openssl", Version: "1.0-r0", Kind: types.BinaryPackage,
				Source: &claircore.Package{ID: "s-alp", Name: "openssl", Version: "1.0-r0", Kind: types.SourcePackage},
			},
			"p-deb": {
				ID: "p-deb", Name: "curl", Version: "1.0", Kind: types.BinaryPackage,
				Source: &claircore.Package{ID: "s-deb", Name: "curl", Version: "1.0", Kind: types.SourcePackage},
			},
		},
		Distributions: map[string]*claircore.Distribution{
			alpineDist.ID: alpineDist,
			debDist.ID:    debDist,
		},
		Environments: map[string][]*claircore.Environment{
			"p-alp": {{DistributionID: alpineDist.ID}},
			"p-deb": {{DistributionID: debDist.ID}},
		},
	}

	var stop atomic.Bool
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for !stop.Load() {
				vr, scanErr := lv.Scan(ctx, ir)
				if !assert.NoError(t, scanErr) {
					return
				}
				// Neither CVE may overwrite the other across concurrent liveSet swaps.
				alpIDs := vr.PackageVulnerabilities["p-alp"]
				debIDs := vr.PackageVulnerabilities["p-deb"]
				if assert.Len(t, alpIDs, 1) && assert.Len(t, debIDs, 1) {
					assert.NotEqual(t, alpIDs[0], debIDs[0], "v.ID collision between matchers across live swap")
					assert.Equal(t, "CVE-ALP-1", vr.Vulnerabilities[alpIDs[0]].Name)
					assert.Equal(t, "CVE-DEB-1", vr.Vulnerabilities[debIDs[0]].Name)
				}
			}
		})
	}

	for gen := 1; gen <= 25; gen++ {
		seed(gen)
	}
	stop.Store(true)
	wg.Wait()
}
