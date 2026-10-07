//go:build scanner_db_integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quay/claircore"
	"github.com/quay/claircore/alpine"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/java"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/toolkit/types"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/quay/claircore/ubuntu"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/enricher/nvd"
	"github.com/stackrox/rox/scanner/updater"
	"github.com/stretchr/testify/require"
)

// TestFixtureMatching imports through the production importer into fresh isolated
// databases. Set SCANNER_CI_TEST_DB to a disposable PostgreSQL admin connection.
// Optionally compare three cold imports with SCANNER_CI_BASELINE_BUNDLE.
func TestFixtureMatching(t *testing.T) {
	conn := os.Getenv("SCANNER_CI_TEST_DB")
	if conn == "" {
		t.Skip("SCANNER_CI_TEST_DB is required (creates isolated databases)")
	}
	candidate := filepath.Join(t.TempDir(), "candidate.zip")
	require.NoError(t, generate(candidate, false, fixtures()))
	bundles := map[string]string{"candidate": candidate}
	if baseline := os.Getenv("SCANNER_CI_BASELINE_BUNDLE"); baseline != "" {
		bundles["baseline"] = baseline
	}
	for name, path := range bundles {
		for sample := range 3 {
			t.Run(fmt.Sprintf("%s/%d", name, sample), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				admin, err := pgxpool.New(ctx, conn)
				require.NoError(t, err)
				defer admin.Close()
				dbName := "ci_fixture_" + strings.ReplaceAll(uuid.NewV4().String(), "-", "")
				_, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{dbName}.Sanitize())
				require.NoError(t, err)
				defer func() {
					_, err := admin.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{dbName}.Sanitize())
					require.NoError(t, err)
				}()
				dbConn := conn + " dbname=" + dbName
				if strings.HasPrefix(conn, "postgres://") || strings.HasPrefix(conn, "postgresql://") {
					u, err := url.Parse(conn)
					require.NoError(t, err)
					u.Path = "/" + dbName
					q := u.Query()
					q.Del("dbname")
					u.RawQuery = q.Encode()
					dbConn = u.String()
				}
				cfg, err := pgxpool.ParseConfig(dbConn)
				require.NoError(t, err)

				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, path) }))
				defer srv.Close()
				started := time.Now()
				require.NoError(t, updater.Load(ctx, cfg.ConnString(), srv.URL))
				t.Logf("cold import %s sample %d: %s", name, sample, time.Since(started))
				pool, err := postgres.Connect(ctx, cfg.ConnString(), "ci-fixture-test")
				require.NoError(t, err)
				defer pool.Close()
				store, err := postgres.InitPostgresMatcherStore(ctx, pool, false)
				require.NoError(t, err)
				lv, err := libvuln.New(ctx, &libvuln.Options{Store: store, Locker: updates.NewLocalLockSource(), Client: srv.Client(), DisableBackgroundUpdates: true,
					MatcherNames: []string{new(alpine.Matcher).Name(), new(java.Matcher).Name(), new(ubuntu.Matcher).Name(), rhcc.Matcher.Name(), (*rhel.Matcher)(nil).Name()}, Enrichers: []driver.Enricher{&nvd.Enricher{}}})
				require.NoError(t, err)
				defer func() { require.NoError(t, lv.Close(context.Background())) }()
				checkMatching(t, ctx, lv)
			})
		}
	}
}

// TestFixturePersistentUpdate imports a changed vulnerability and enrichment
// over existing rows in the same database. A fixed Last-Modified header keeps
// HTTP freshness independent from the member revision under test; the server
// always returns 200 so repeat B reaches the ZIP member timestamp check.
func TestFixturePersistentUpdate(t *testing.T) {
	conn := os.Getenv("SCANNER_CI_TEST_DB")
	if conn == "" {
		t.Skip("SCANNER_CI_TEST_DB is required (creates isolated databases)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	admin, err := pgxpool.New(ctx, conn)
	require.NoError(t, err)
	defer admin.Close()
	dbName := "ci_fixture_update_" + strings.ReplaceAll(uuid.NewV4().String(), "-", "")
	_, err = admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{dbName}.Sanitize())
	require.NoError(t, err)
	defer func() {
		_, err := admin.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{dbName}.Sanitize())
		require.NoError(t, err)
	}()
	dbConn := conn + " dbname=" + dbName
	if strings.HasPrefix(conn, "postgres://") || strings.HasPrefix(conn, "postgresql://") {
		u, err := url.Parse(conn)
		require.NoError(t, err)
		u.Path = "/" + dbName
		q := u.Query()
		q.Del("dbname")
		u.RawQuery = q.Encode()
		dbConn = u.String()
	}
	cfg, err := pgxpool.ParseConfig(dbConn)
	require.NoError(t, err)

	const (
		member            = "persistent-fixture.json.zst"
		updaterName       = "persistent-vulnerability-fixture"
		enrichmentUpdater = "persistent-enrichment-fixture"
		cve               = "CVE-2030-0001"
	)
	revisionA := time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)
	revisionB := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	makeFixture := func(description, fixedVersion, enrichmentRevision string) []operation {
		return []operation{
			{Member: member, Updater: updaterName, Vulnerabilities: []*claircore.Vulnerability{{
				Name: cve, Description: description,
				Package:        &claircore.Package{Name: "persistent-fixture-package", Kind: types.BinaryPackage},
				Dist:           &claircore.Distribution{DID: "persistent-fixture-distro", VersionID: "1"},
				FixedInVersion: fixedVersion,
			}}},
			{Member: member, Updater: enrichmentUpdater, Enrichments: []enrichmentFixture{{
				Tags: []string{cve}, Payload: map[string]string{"id": cve, "revision": enrichmentRevision},
			}}},
		}
	}
	bundleA := filepath.Join(t.TempDir(), "a.zip")
	bundleB := filepath.Join(t.TempDir(), "b.zip")
	require.NoError(t, generateAt(bundleA, false, makeFixture("revision A", "2.0", "A"), revisionA))
	require.NoError(t, generateAt(bundleB, false, makeFixture("revision B", "3.0", "B"), revisionB))
	bundleABytes, err := os.ReadFile(bundleA)
	require.NoError(t, err)
	bundleBBytes, err := os.ReadFile(bundleB)
	require.NoError(t, err)

	var servedBundle atomic.Value
	servedBundle.Store(bundleABytes)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Last-Modified", time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC).Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(servedBundle.Load().([]byte))
	}))
	defer server.Close()

	load := func() {
		t.Helper()
		require.NoError(t, updater.Load(ctx, cfg.ConnString(), server.URL))
	}
	load()

	pool, err := postgres.Connect(ctx, cfg.ConnString(), "ci-fixture-persistent-update-test")
	require.NoError(t, err)
	defer pool.Close()
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, false)
	require.NoError(t, err)
	assertImported := func(description, fixedVersion, enrichmentRevision string, revision time.Time) {
		t.Helper()
		records, err := store.Get(ctx, []*claircore.IndexRecord{{
			Package:      &claircore.Package{ID: "persistent-fixture-package-id", Name: "persistent-fixture-package", Version: "1", Kind: types.BinaryPackage, Source: &claircore.Package{}},
			Distribution: &claircore.Distribution{DID: "persistent-fixture-distro", VersionID: "1"},
		}}, datastore.GetOpts{})
		require.NoError(t, err)
		require.Len(t, records["persistent-fixture-package-id"], 1)
		vuln := records["persistent-fixture-package-id"][0]
		require.Equal(t, cve, vuln.Name)
		require.Equal(t, description, vuln.Description)
		require.Equal(t, fixedVersion, vuln.FixedInVersion)

		enrichments, err := store.GetEnrichment(ctx, enrichmentUpdater, []string{cve})
		require.NoError(t, err)
		require.Len(t, enrichments, 1)
		var enrichment map[string]string
		require.NoError(t, json.Unmarshal(enrichments[0].Enrichment, &enrichment))
		require.Equal(t, enrichmentRevision, enrichment["revision"])

		var storedRevision time.Time
		err = pool.QueryRow(ctx, "SELECT update_timestamp FROM last_vuln_update WHERE key = $1", member).Scan(&storedRevision)
		require.NoError(t, err)
		require.True(t, storedRevision.Equal(revision), "stored member timestamp")
	}
	assertImported("revision A", "2.0", "A", revisionA)

	servedBundle.Store(bundleBBytes)
	load()
	assertImported("revision B", "3.0", "B", revisionB)
	load()
	assertImported("revision B", "3.0", "B", revisionB)
	require.EqualValues(t, 3, requests.Load(), "each import, including repeat B, should receive HTTP 200")
}

func singlePackageReport(p *claircore.Package, d *claircore.Distribution, r *claircore.Repository) *claircore.IndexReport {
	p.ID = "1"
	if p.Source == nil {
		p.Source = &claircore.Package{}
	}
	if d != nil && p.Source.Name == "" {
		p.Source = &claircore.Package{Name: p.Name, Version: p.Version, Kind: types.SourcePackage}
	}
	ir := &claircore.IndexReport{Success: true, Packages: map[string]*claircore.Package{"1": p}, Distributions: map[string]*claircore.Distribution{}, Repositories: map[string]*claircore.Repository{}, Environments: map[string][]*claircore.Environment{"1": {{}}}}
	if d != nil {
		d.ID = "d"
		ir.Distributions[d.ID] = d
		ir.Environments["1"][0].DistributionID = d.ID
	}
	if r != nil {
		r.ID = "r"
		ir.Repositories[r.ID] = r
		ir.Environments["1"][0].RepositoryIDs = []string{r.ID}
	}
	return ir
}

func hasFinding(vr *claircore.VulnerabilityReport, name string) bool {
	for _, ids := range vr.PackageVulnerabilities {
		for _, id := range ids {
			v := vr.Vulnerabilities[id]
			if v.Name == name {
				return true
			}
			for _, a := range v.Aliases {
				if a.Valid() && a.String() == name {
					return true
				}
			}
		}
	}
	return false
}

func checkMatching(t *testing.T, ctx context.Context, lv *libvuln.Libvuln) {
	t.Helper()
	// nginx version-boundary probes on Alpine 3.9; independent of the fixture tables.
	for name, tc := range map[string]struct {
		version, distro string
		affected        bool
	}{
		"nginx vulnerable":   {"1.14.0", "alpine", true},
		"nginx fixed":        {"1.14.2-r5", "alpine", false},
		"wrong distribution": {"1.14.2", "debian", false},
	} {
		t.Run(name, func(t *testing.T) {
			ir := singlePackageReport(&claircore.Package{Name: "nginx", Version: tc.version, Kind: types.BinaryPackage}, &claircore.Distribution{DID: tc.distro, Name: "Alpine Linux", VersionID: "3.9", PrettyName: "Alpine Linux v3.9"}, nil)
			if tc.distro != "alpine" {
				ir.Distributions["d"].Name = "Debian"
			}
			vr, err := lv.Scan(ctx, ir)
			require.NoError(t, err)
			require.Equal(t, tc.affected, hasFinding(vr, "CVE-2019-20372"))
			if tc.affected {
				require.NotEmpty(t, vr.Enrichments)
				converted, err := mappers.ToProtoV4VulnerabilityReport(ctx, vr)
				require.NoError(t, err)
				var important bool
				for _, v := range converted.GetVulnerabilities() {
					if v.GetFixedInVersion() != "" && v.GetCvss().GetV3().GetBaseScore() >= 7 {
						important = true
					}
				}
				require.True(t, important, "nginx must have a fixable Important-or-higher CVSS")
				for _, v := range vr.Vulnerabilities {
					if v.Name == "CVE-2019-20372" {
						require.Equal(t, "1.14.2-r5", v.FixedInVersion)
					}
				}
			}
		})
	}
	for name, tc := range map[string]struct {
		version, repo string
		affected      bool
	}{
		"struts vulnerable": {"2.3.12", "maven", true},
		"struts fixed":      {"2.3.32", "maven", false},
		"wrong repository":  {"2.3.12", "npm", false},
	} {
		t.Run(name, func(t *testing.T) {
			vr, err := lv.Scan(ctx, singlePackageReport(&claircore.Package{Name: "org.apache.struts:struts2-core", Version: tc.version, Kind: types.BinaryPackage}, nil, &claircore.Repository{Name: tc.repo}))
			require.NoError(t, err)
			require.Equal(t, tc.affected, hasFinding(vr, "CVE-2017-5638"))
		})
	}
	t.Run("RHEL 9 python3.9 source package", func(t *testing.T) {
		const repoName = "cpe:2.3:a:redhat:enterprise_linux:9:*:*:*:*:*:*:*"
		repoCPE, err := cpe.Unbind(repoName)
		require.NoError(t, err)
		pkg := &claircore.Package{
			Name:    "python3",
			Version: "3.9.25-2.el9_7",
			Kind:    types.BinaryPackage,
			Source:  &claircore.Package{Name: "python3.9", Version: "3.9.25-2.el9_7", Kind: types.SourcePackage},
		}
		repo := &claircore.Repository{Name: repoName, Key: "rhel-cpe-repository", CPE: repoCPE}
		distro := &claircore.Distribution{DID: "rhel", Name: "Red Hat Enterprise Linux", VersionID: "9"}
		vr, err := lv.Scan(ctx, singlePackageReport(pkg, distro, repo))
		require.NoError(t, err)
		require.True(t, hasFinding(vr, "CVE-2025-11468"), "RHEL 9 python3.9 should match CVE-2025-11468")

		converted, err := mappers.ToProtoV4VulnerabilityReport(ctx, vr)
		require.NoError(t, err)
		var matched bool
		for _, vuln := range converted.GetVulnerabilities() {
			if vuln.GetName() != "CVE-2025-11468" {
				continue
			}
			matched = true
			require.Equal(t, v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE, vuln.GetNormalizedSeverity())
			require.Equal(t, float32(4.5), vuln.GetCvss().GetV3().GetBaseScore())
		}
		require.True(t, matched, "mapped report should include CVE-2025-11468")
	})
	t.Run("exact description and CVSS", func(t *testing.T) {
		// Same input and expectations as TestImage/sandbox-scannerremovejar.
		vr, err := lv.Scan(ctx, singlePackageReport(&claircore.Package{Name: "com.fasterxml.jackson.core:jackson-databind", Version: "2.9.10.4", Kind: types.BinaryPackage}, nil, &claircore.Repository{Name: "maven"}))
		require.NoError(t, err)
		var found bool
		for _, v := range vr.Vulnerabilities {
			for _, a := range v.Aliases {
				if a.Valid() && a.String() == "CVE-2020-24616" {
					require.Equal(t, "Code Injection in jackson-databind", v.Description)
					found = true
				}
			}
		}
		require.True(t, found)
		var scored bool
		for _, blobs := range vr.Enrichments {
			for _, blob := range blobs {
				var payload map[string][]schema.CVEAPIJSON20CVEItem
				require.NoError(t, json.Unmarshal(blob, &payload))
				for _, items := range payload {
					for _, item := range items {
						if item.ID != "CVE-2020-24616" || item.Metrics == nil {
							continue
						}
						for _, metric := range item.Metrics.CvssMetricV31 {
							if metric.CvssData != nil && metric.CvssData.BaseScore == 8.1 {
								require.Equal(t, "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", metric.CvssData.VectorString)
								scored = true
							}
						}
					}
				}
			}
		}
		require.True(t, scored, "NVD CVSS must remain connected to matching records")
	})
	t.Run("struts aggregate", func(t *testing.T) {
		var inventory []struct{ Name, Version, Arch, Source, SourceVersion string }
		b, err := os.ReadFile("testdata/struts-packages.json")
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(b, &inventory))
		ir := singlePackageReport(&claircore.Package{Name: "org.apache.struts:struts2-core", Version: "2.3.12", Kind: types.BinaryPackage}, nil, &claircore.Repository{Name: "maven"})
		ir.Distributions["ubuntu"] = &claircore.Distribution{ID: "ubuntu", DID: "ubuntu", Name: "Ubuntu", VersionID: "20.04"}
		for i, p := range inventory {
			id := fmt.Sprint(i + 2)
			ir.Packages[id] = &claircore.Package{ID: id, Name: p.Name, Version: p.Version, Arch: p.Arch, Kind: types.BinaryPackage, Source: &claircore.Package{Name: p.Source, Version: p.SourceVersion, Kind: types.SourcePackage}}
			ir.Environments[id] = []*claircore.Environment{{DistributionID: "ubuntu"}}
		}
		vr, err := lv.Scan(ctx, ir)
		require.NoError(t, err)
		count := 0
		for _, ids := range vr.PackageVulnerabilities {
			names := map[string]bool{}
			for _, id := range ids {
				names[vr.Vulnerabilities[id].Name] = true
			}
			count += len(names)
		}
		require.GreaterOrEqual(t, count, 138)
		require.True(t, hasFinding(vr, "CVE-2017-5638"))
		t.Logf("Struts package/advisory findings: %d", count)
	})
	t.Run("rhel unaffected range", func(t *testing.T) {
		const repo = "cpe:2.3:a:redhat:advanced_cluster_security:4:*:*:*:*:*:*:*"
		c, err := cpe.Unbind(repo)
		require.NoError(t, err)
		ir := singlePackageReport(&claircore.Package{Name: "advanced-cluster-security/rhacs-scanner-rhel8", Version: "4.3.0", Kind: types.AncestryPackage, NormalizedVersion: claircore.Version{Kind: "rhctag", V: [10]int32{4, 3, 0}}}, nil, &claircore.Repository{Name: repo, Key: "rhcc-container-repository", CPE: c})
		vr, err := lv.Scan(ctx, ir)
		require.NoError(t, err)
		require.False(t, hasFinding(vr, "CVE-2024-45337"))
		var excluded bool
		for _, ids := range vr.PackageNotVulnerable {
			for _, id := range ids {
				excluded = excluded || vr.Vulnerabilities[id].Name == "CVE-2024-45337"
			}
		}
		require.True(t, excluded, "must exercise the unaffected record, not merely return no findings")
	})
}
