package matcher

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/toolkit/types"
	"github.com/stackrox/rox/scanner/config"
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
		Readiness:           config.ReadinessVulnerability,
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
