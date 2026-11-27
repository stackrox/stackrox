//go:build scanner_db_integration

package inmem_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"unique"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quay/claircore"
	"github.com/quay/claircore/alpine"
	"github.com/quay/claircore/aws"
	ccpg "github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/debian"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/claircore/enricher/kev"
	"github.com/quay/claircore/gobin"
	"github.com/quay/claircore/java"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/matchers/registry"
	"github.com/quay/claircore/nodejs"
	"github.com/quay/claircore/oracle"
	"github.com/quay/claircore/photon"
	"github.com/quay/claircore/pkg/ctxlock/v2"
	"github.com/quay/claircore/pkg/rhctag"
	"github.com/quay/claircore/python"
	"github.com/quay/claircore/rhel"
	rheljava "github.com/quay/claircore/rhel/java"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/ruby"
	"github.com/quay/claircore/suse"
	"github.com/quay/claircore/test"
	"github.com/quay/claircore/toolkit/types"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/quay/claircore/ubuntu"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/scanner/datastore/inmem"
	scannerpg "github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/enricher/csaf"
	"github.com/stackrox/rox/scanner/enricher/fixedby"
	"github.com/stackrox/rox/scanner/enricher/nvd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Mirror scanner/matcher/matcher.go: ClairCore does not register nodejs by default.
	m := nodejs.Matcher{}
	registry.Register(m.Name(), driver.MatcherStatic(&m))
}

func parityTestDB(t *testing.T, ctx context.Context, name string) *pgxpool.Pool {
	t.Helper()

	pgConn := "postgresql://postgres@127.0.0.1:5432/postgres?sslmode=disable"
	pgPool, err := ccpg.Connect(ctx, pgConn, name)
	require.NoError(t, err)
	_, err = pgPool.Exec(ctx, "DROP DATABASE IF EXISTS "+name)
	require.NoError(t, err)
	_, err = pgPool.Exec(ctx, "CREATE DATABASE "+name)
	require.NoError(t, err)
	t.Cleanup(pgPool.Close)

	dbConn := fmt.Sprintf("postgresql://postgres@127.0.0.1:5432/%s?sslmode=disable", name)
	dbPool, err := ccpg.Connect(ctx, dbConn, name)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pgPool.Exec(ctx, "DROP DATABASE IF EXISTS "+name)
	})
	t.Cleanup(dbPool.Close)
	return dbPool
}

type canonicalVuln struct {
	Name               string
	Description        string
	Link               string
	Severity           string
	NormalizedSeverity string
	FixedInVersion     string
	Issued             int64
	Cvss               string
	Epss               string
	Advisory           string
	Exploit            string
}

type canonicalReport struct {
	ByPkg     map[string][]canonicalVuln
	Enrichers []string
}

func canonicalize(t *testing.T, ctx context.Context, ccReport *claircore.VulnerabilityReport) canonicalReport {
	t.Helper()
	protoRep, err := mappers.ToProtoV4VulnerabilityReport(ctx, ccReport)
	require.NoError(t, err)

	byPkg := make(map[string][]canonicalVuln, len(protoRep.GetPackageVulnerabilities()))
	for pkgID, vidList := range protoRep.GetPackageVulnerabilities() {
		var list []canonicalVuln
		for _, vid := range vidList.GetValues() {
			v := protoRep.GetVulnerabilities()[vid]
			require.NotNil(t, v)
			cvssJSON, err := json.Marshal(v.GetCvssMetrics())
			require.NoError(t, err)
			epssJSON, err := json.Marshal(v.GetEpssMetrics())
			require.NoError(t, err)
			advJSON, err := json.Marshal(v.GetAdvisory())
			require.NoError(t, err)
			expJSON, err := json.Marshal(v.GetExploit())
			require.NoError(t, err)
			var issued int64
			if v.GetIssued() != nil {
				issued = v.GetIssued().GetSeconds()
			}
			list = append(list, canonicalVuln{
				Name:               v.GetName(),
				Description:        v.GetDescription(),
				Link:               v.GetLink(),
				Severity:           v.GetSeverity(),
				NormalizedSeverity: v.GetNormalizedSeverity().String(),
				FixedInVersion:     v.GetFixedInVersion(),
				Issued:             issued,
				Cvss:               string(cvssJSON),
				Epss:               string(epssJSON),
				Advisory:           string(advJSON),
				Exploit:            string(expJSON),
			})
		}
		slices.SortFunc(list, func(a, b canonicalVuln) int {
			if a.Name != b.Name {
				if a.Name < b.Name {
					return -1
				}
				return 1
			}
			if a.FixedInVersion < b.FixedInVersion {
				return -1
			}
			if a.FixedInVersion > b.FixedInVersion {
				return 1
			}
			return 0
		})
		if len(list) > 0 {
			byPkg[pkgID] = list
		}
	}

	enrichers := make([]string, 0, len(ccReport.Enrichments))
	for k, v := range ccReport.Enrichments {
		if len(v) > 0 {
			enrichers = append(enrichers, k)
		}
	}
	slices.Sort(enrichers)
	return canonicalReport{ByPkg: byPkg, Enrichers: enrichers}
}

func allMatcherNames() []string {
	return []string{
		(*alpine.Matcher)(nil).Name(),
		(*aws.Matcher)(nil).Name(),
		(*debian.Matcher)(nil).Name(),
		(*gobin.Matcher)(nil).Name(),
		(*java.Matcher)(nil).Name(),
		(*nodejs.Matcher)(nil).Name(),
		(*oracle.Matcher)(nil).Name(),
		(*photon.Matcher)(nil).Name(),
		(*python.Matcher)(nil).Name(),
		rhcc.Matcher.Name(),
		(*rhel.Matcher)(nil).Name(),
		(*rheljava.RedHatMatcher)(nil).Name(),
		(*ruby.Matcher)(nil).Name(),
		(*suse.Matcher)(nil).Name(),
		(*ubuntu.Matcher)(nil).Name(),
	}
}

func allEnrichers() []driver.Enricher {
	return []driver.Enricher{
		&fixedby.Enricher{},
		&nvd.Enricher{},
		&epss.Enricher{},
		&csaf.Enricher{},
		&kev.Enricher{},
	}
}

type matcherCase struct {
	name          string
	report        *claircore.IndexReport
	wantVulnPkgs  []string
	wantCleanPkgs []string
}

// TestAllMatchersAndEnrichersPostgresParity loads a synthetic fixture covering
// all 15 enabled Scanner V4 matchers and all 5 enrichers into both Postgres and
// inmem stores and asserts exact canonical VulnerabilityReport and
// Distributions() parity.
//
// Run with:
//
//	go test -tags scanner_db_integration -v ./scanner/datastore/inmem -run '^TestAllMatchersAndEnrichersPostgresParity$'
func TestAllMatchersAndEnrichersPostgresParity(t *testing.T) {
	ctx := test.Logging(t)

	pool := parityTestDB(t, ctx, "inmem_parity_test")
	pgStore, err := scannerpg.InitPostgresMatcherStore(ctx, pool, true)
	require.NoError(t, err)
	pgLocker, err := ctxlock.New(ctx, pool)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgLocker.Close(ctx) })

	imStore, err := inmem.NewMatcherStore(inmem.Options{BlobDir: t.TempDir()})
	require.NoError(t, err)
	t.Cleanup(imStore.Close)
	imLocker := inmem.NewLocker()

	vulnsByUpdater, enrichByKind, cases := buildParityFixture(t)

	imStore.BeginBatch()
	for upd, vs := range vulnsByUpdater {
		_, err := pgStore.UpdateVulnerabilities(ctx, upd, driver.Fingerprint("fp-"+upd), vs)
		require.NoError(t, err, "postgres UpdateVulnerabilities(%s)", upd)
		_, err = imStore.UpdateVulnerabilities(ctx, upd, driver.Fingerprint("fp-"+upd), vs)
		require.NoError(t, err, "inmem UpdateVulnerabilities(%s)", upd)
	}
	for kind, es := range enrichByKind {
		_, err := pgStore.UpdateEnrichments(ctx, kind, driver.Fingerprint("fp-e-"+kind), es)
		require.NoError(t, err, "postgres UpdateEnrichments(%s)", kind)
		_, err = imStore.UpdateEnrichments(ctx, kind, driver.Fingerprint("fp-e-"+kind), es)
		require.NoError(t, err, "inmem UpdateEnrichments(%s)", kind)
	}
	imStore.EndBatch()

	// 1. Verify Distributions() parity (including RHEL & Hummingbird CPEs).
	pgDists, err := pgStore.Distributions(ctx)
	require.NoError(t, err)
	imDists, err := imStore.Distributions(ctx)
	require.NoError(t, err)
	assert.ElementsMatch(t, pgDists, imDists, "Distributions() must match Postgres")
	assert.Contains(t, imDists, claircore.Distribution{DID: "rhel", VersionID: "9", Version: "9"})
	assert.Contains(t, imDists, claircore.Distribution{DID: "hummingbird"})

	pgLV, err := libvuln.New(ctx, &libvuln.Options{
		Store:                    pgStore,
		Locker:                   pgLocker,
		MatcherNames:             allMatcherNames(),
		Enrichers:                allEnrichers(),
		DisableBackgroundUpdates: true,
		Client:                   http.DefaultClient,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgLV.Close(ctx) })

	imLV, err := libvuln.New(ctx, &libvuln.Options{
		Store:                    imStore,
		Locker:                   imLocker,
		MatcherNames:             allMatcherNames(),
		Enrichers:                allEnrichers(),
		DisableBackgroundUpdates: true,
		Client:                   http.DefaultClient,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = imLV.Close(ctx) })

	seenEnrichers := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pgRep, err := pgLV.Scan(ctx, tc.report)
			require.NoError(t, err)
			imRep, err := imLV.Scan(ctx, tc.report)
			require.NoError(t, err)

			pgCanon := canonicalize(t, ctx, pgRep)
			imCanon := canonicalize(t, ctx, imRep)

			assert.Equal(t, pgCanon, imCanon, "canonical VulnerabilityReport mismatch for %s", tc.name)

			// Verify every matcher actually exercised positive + negative paths.
			for _, pkgID := range tc.wantVulnPkgs {
				assert.NotEmpty(t, imCanon.ByPkg[pkgID], "expected vulnerability match for %s/%s", tc.name, pkgID)
			}
			for _, pkgID := range tc.wantCleanPkgs {
				assert.Empty(t, imCanon.ByPkg[pkgID], "expected no vulnerability match for %s/%s", tc.name, pkgID)
			}
			for _, e := range imCanon.Enrichers {
				seenEnrichers[e] = true
			}
		})
	}

	// Verify all 5 enrichers actually fired across the fixture.
	assert.Len(t, seenEnrichers, 5, "expected all 5 enrichers (fixedby, nvd, epss, csaf, kev) to fire; got %v", seenEnrichers)

	// Verify proto vulnerability report serialization on known vs EOL distros.
	eolDist := &claircore.Distribution{ID: "eol", DID: "centos", VersionID: "7", Version: "7"}
	eolKnown := slices.ContainsFunc(imDists, func(d claircore.Distribution) bool {
		return d.DID == eolDist.DID && mappers.VersionID(&d) == eolDist.VersionID
	})
	assert.False(t, eolKnown, "EOL centos:7 must not be in KnownDistributions")
	assert.Equal(t, v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED.String(), "NOTE_OS_UNSUPPORTED")
}

func buildParityFixture(t *testing.T) (map[string][]*claircore.Vulnerability, map[string][]driver.EnrichmentRecord, []matcherCase) {
	t.Helper()
	issued := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	rhel9CPE := cpe.MustUnbind("cpe:2.3:o:redhat:enterprise_linux:9:*:*:*:*:*:*:*")
	rhelAppCPE := cpe.MustUnbind("cpe:2.3:a:redhat:enterprise_linux:9::appstream:*:*:*:*:*")
	rhccCPE := cpe.MustUnbind("cpe:/a:redhat:openshift:4.21::el9")
	rhelJavaCPE := cpe.MustUnbind("cpe:2.3:a:redhat:jboss_enterprise_application_platform:8:*:*:*:*:*:*:*")

	rhccInstalled, err := rhctag.Parse("9.6.20260324-0")
	require.NoError(t, err)
	rhccInstalledMinor := rhccInstalled.MinorStart()
	rhccFixed, err := rhctag.Parse("9.6.20260325-0")
	require.NoError(t, err)
	rhccPatched, err := rhctag.Parse("9.6.20260326-0")
	require.NoError(t, err)
	rhccPatchedMinor := rhccPatched.MinorStart()
	rhccZero := new(rhctag.Version).Version(true)

	alpineDist := &claircore.Distribution{ID: "d-alp", DID: "alpine", Name: "Alpine Linux", VersionID: "3.19", Version: "3.19"}
	awsDist := &claircore.Distribution{ID: "d-aws", DID: aws.ID, Name: aws.AL2023Dist.Name, VersionID: aws.AL2023Dist.VersionID, Version: aws.AL2023Dist.Version}
	debDist := &claircore.Distribution{ID: "d-deb", DID: "debian", Name: "Debian GNU/Linux", VersionID: "12", Version: "12 (bookworm)"}
	oraDist := &claircore.Distribution{ID: "d-ora", DID: oracle.OSReleaseID, Name: oracle.OSReleaseName, VersionID: "9", Version: "9"}
	phoDist := &claircore.Distribution{ID: "d-pho", DID: "photon", Name: "VMware Photon OS", VersionID: "5.0", Version: "5.0"}
	suseDist := &claircore.Distribution{ID: "d-suse", DID: suse.OSReleaseIDs[0], Name: suse.OSReleaseNames[0], VersionID: "15.5", Version: "15.5"}
	ubuDist := &claircore.Distribution{ID: "d-ubu", DID: "ubuntu", Name: "Ubuntu", VersionID: "22.04", Version: "22.04"}

	goRepo := &claircore.Repository{ID: "r-go", Name: "go", URI: "https://pkg.go.dev/"}
	javaRepo := &claircore.Repository{ID: "r-java", Name: java.Repository.Name, URI: java.Repository.URI}
	nodeRepo := &claircore.Repository{ID: "r-node", Name: nodejs.Repository.Name, URI: nodejs.Repository.URI}
	pyRepo := &claircore.Repository{ID: "r-py", Name: python.Repository.Name, URI: python.Repository.URI}
	rubyRepo := &claircore.Repository{ID: "r-ruby", Name: ruby.Repository.Name, URI: ruby.Repository.URI}
	rhelRepo1 := &claircore.Repository{ID: "r-rhel1", Name: rhel9CPE.String(), Key: "rhel-cpe-repository", CPE: rhel9CPE}
	rhelRepo2 := &claircore.Repository{ID: "r-rhel2", Name: rhelAppCPE.String(), Key: "rhel-cpe-repository", CPE: rhelAppCPE}
	rhccRepo := &claircore.Repository{ID: "r-rhcc", Name: rhccCPE.String(), Key: rhcc.RepositoryKey, CPE: rhccCPE}
	rhelJavaRepo := &claircore.Repository{ID: "r-rjava", Name: rhelJavaCPE.String(), Key: rheljava.RepositoryKey, CPE: rhelJavaCPE}

	vulns := map[string][]*claircore.Vulnerability{
		"alpine-3.19": {{
			Updater: "alpine-3.19", Name: "CVE-2025-2001", Description: "alpine vuln", Links: "https://alpine.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "3.1.0-r0", Issued: issued,
			Package: &claircore.Package{Name: "openssl", Kind: types.SourcePackage}, Dist: alpineDist,
		}},
		"aws-al2023": {{
			Updater: "aws-al2023", Name: "CVE-2025-2002", Description: "aws vuln", Links: "https://aws.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "8.5.0-1.amzn2023", Issued: issued,
			Package: &claircore.Package{Name: "curl", Kind: types.BinaryPackage}, Dist: awsDist,
		}},
		"debian-12": {{
			Updater: "debian-12", Name: "CVE-2025-2003", Description: "debian vuln", Links: "https://deb.example/1",
			Severity: "Medium", NormalizedSeverity: claircore.Medium, FixedInVersion: "2.36-10", Issued: issued,
			Package: &claircore.Package{Name: "glibc", Kind: types.SourcePackage}, Dist: debDist,
		}},
		"osv-go": {{
			Updater: "osv-go", Name: "CVE-2025-2004", Description: "go vuln", Links: "https://go.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "0.23.0", Issued: issued,
			Package: &claircore.Package{Name: "golang.org/x/net", Kind: types.BinaryPackage}, Repo: goRepo,
			Range: &claircore.Range{
				Lower: claircore.Version{Kind: "semver", V: [10]int32{0, 0, 0}},
				Upper: claircore.Version{Kind: "semver", V: [10]int32{0, 23, 0}},
			},
		}},
		"osv-maven": {{
			Updater: "osv-maven", Name: "CVE-2025-2005", Description: "java vuln", Links: "https://maven.example/1",
			Severity: "Critical", NormalizedSeverity: claircore.Critical, FixedInVersion: "introduced=2.0&fixed=2.17.0", Issued: issued,
			Package: &claircore.Package{Name: "org.apache.logging.log4j:log4j-core", Kind: types.BinaryPackage}, Repo: javaRepo,
		}},
		"osv-npm": {{
			Updater: "osv-npm", Name: "CVE-2025-2006", Description: "npm vuln", Links: "https://npm.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "4.17.21", Issued: issued,
			Package: &claircore.Package{Name: "lodash", Kind: types.BinaryPackage}, Repo: nodeRepo,
			Range: &claircore.Range{
				Lower: claircore.Version{Kind: "semver", V: [10]int32{4, 0, 0}},
				Upper: claircore.Version{Kind: "semver", V: [10]int32{4, 17, 21}},
			},
		}},
		"oracle-9": {{
			Updater: "oracle-9", Name: "CVE-2025-2007", Description: "oracle vuln", Links: "https://oracle.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "1.2.11-40.el9", Issued: issued,
			Package: &claircore.Package{Name: "zlib", Kind: types.BinaryPackage}, Dist: oraDist,
		}},
		"photon-5": {{
			Updater: "photon-5", Name: "CVE-2025-2008", Description: "photon vuln", Links: "https://photon.example/1",
			Severity: "Medium", NormalizedSeverity: claircore.Medium, FixedInVersion: "2.6.0-1.ph5", Issued: issued,
			Package: &claircore.Package{Name: "expat", Kind: types.BinaryPackage}, Dist: phoDist,
		}},
		"osv-pypi": {{
			Updater: "osv-pypi", Name: "CVE-2025-2009", Description: "python vuln", Links: "https://pypi.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "introduced=4.0&fixed=4.2.5", Issued: issued,
			Package: &claircore.Package{Name: "django", Kind: types.BinaryPackage}, Repo: pyRepo,
		}},
		"rhcc": {{
			Updater: "rhcc", Name: "CVE-2025-2010", Description: "rhcc vuln", Links: "https://rhcc.example/1",
			Severity: "Important", NormalizedSeverity: claircore.High, FixedInVersion: "9.6.20260325-0", Issued: issued,
			Package: &claircore.Package{Name: "openshift/ose-rhel-coreos-9", Kind: types.BinaryPackage}, Repo: rhccRepo,
			Range: &claircore.Range{Lower: rhccZero, Upper: rhccFixed.Version(false)},
		}},
		mappers.RedHatUpdaterName: {
			{
				Updater: mappers.RedHatUpdaterName, Name: "RHSA-2025:0001", Description: "CVE-2025-1001 rhel vuln",
				Links:    "https://access.redhat.com/errata/RHSA-2025:0001 https://access.redhat.com/security/cve/CVE-2025-1001",
				Severity: "Important", NormalizedSeverity: claircore.High, FixedInVersion: "0:5.1.8-9.el9", Issued: issued,
				Package: &claircore.Package{Name: "bash", Kind: types.BinaryPackage, Module: "mod:1"},
				Repo:    rhelRepo1,
				Aliases: []claircore.Alias{{Space: unique.Make("cve"), Name: "CVE-2025-1001"}},
				Self:    claircore.Alias{Space: unique.Make("rhsa"), Name: "RHSA-2025:0001"},
			},
			{
				// Second CPE repo sharing Package.ID (multi-record union).
				Updater: mappers.RedHatUpdaterName, Name: "RHSA-2025:0002", Description: "CVE-2025-1002 rhel appstream",
				Links:    "https://access.redhat.com/errata/RHSA-2025:0002",
				Severity: "Moderate", NormalizedSeverity: claircore.Medium, FixedInVersion: "0:5.1.8-9.el9", Issued: issued,
				Package: &claircore.Package{Name: "bash", Kind: types.BinaryPackage, Module: "mod:1"},
				Repo:    rhelRepo2,
			},
			{
				// Known-not-affected suppression (Invert: true).
				Updater: mappers.RedHatUpdaterName, Name: "CVE-2025-1003", Description: "not affected",
				Severity: "Low", NormalizedSeverity: claircore.Low, Invert: true, Issued: issued,
				Package: &claircore.Package{Name: "bash", Kind: types.BinaryPackage, Module: "mod:1"},
				Repo:    rhelRepo1,
			},
			{
				// Hummingbird CPE for Distributions() parity.
				Updater: mappers.RedHatUpdaterName, Name: "CVE-2025-1004", Description: "hummingbird",
				Severity: "Low", NormalizedSeverity: claircore.Low, Issued: issued,
				Package: &claircore.Package{Name: "hb-pkg", Kind: types.BinaryPackage},
				Repo:    &claircore.Repository{Name: "cpe:2.3:a:redhat:hummingbird:1:*:*:*:*:*:*:*", Key: "rhel-cpe-repository"},
			},
			{
				// Red Hat Java CPE advisory (rheljava matcher).
				Updater: mappers.RedHatUpdaterName, Name: "CVE-2025-2012", Description: "rh-java vuln",
				Links:    "https://access.redhat.com/security/cve/CVE-2025-2012",
				Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "2.15.0", Issued: issued,
				Package: &claircore.Package{Name: "com.fasterxml.jackson.core:jackson-databind", Kind: types.BinaryPackage},
				Repo:    rhelJavaRepo,
			},
		},
		"osv-ruby": {{
			Updater: "osv-ruby", Name: "CVE-2025-2013", Description: "ruby vuln", Links: "https://ruby.example/1",
			Severity: "Medium", NormalizedSeverity: claircore.Medium, FixedInVersion: "introduced=2.0.0&fixed=2.2.8", Issued: issued,
			Package: &claircore.Package{Name: "rack", Kind: types.BinaryPackage}, Repo: rubyRepo,
		}},
		"suse-15": {{
			Updater: "suse-15", Name: "CVE-2025-2014", Description: "suse vuln", Links: "https://suse.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "2.9.14-2.1", Issued: issued,
			Package: &claircore.Package{Name: "libxml2", Kind: types.BinaryPackage}, Dist: suseDist,
		}},
		"ubuntu-2204": {{
			Updater: "ubuntu-2204", Name: "CVE-2025-2015", Description: "ubuntu vuln", Links: "https://ubuntu.example/1",
			Severity: "High", NormalizedSeverity: claircore.High, FixedInVersion: "1.9.9-1ubuntu2.5", Issued: issued,
			Package: &claircore.Package{Name: "sudo", Kind: types.SourcePackage}, Dist: ubuDist,
		}},
	}

	// Single-tag enrichment records for all 4 store-backed enrichers (fixedby is computed inline).
	enrich := map[string][]driver.EnrichmentRecord{
		(&nvd.Enricher{}).Name(): {{
			Tags:       []string{"CVE-2025-1001"},
			Enrichment: []byte(`{"cvssMetricV31":[{"source":"nvd@nist.gov","type":"Primary","cvssData":{"version":"3.1","vectorString":"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H","baseScore":9.8,"baseSeverity":"CRITICAL"}}]}`),
		}},
		(&epss.Enricher{}).Name(): {{
			Tags:       []string{"CVE-2025-1001"},
			Enrichment: []byte(`{"cve":"CVE-2025-1001","epss":0.91,"percentile":0.99,"model_version":"v2025","date":"2025-06-01"}`),
		}},
		(&csaf.Enricher{}).Name(): {{
			Tags:       []string{"RHSA-2025:0001"},
			Enrichment: []byte(`{"name":"RHSA-2025:0001","description":"Red Hat Security Advisory","severity":"Important","release_date":"2025-06-01T00:00:00Z"}`),
		}},
		(&kev.Enricher{}).Name(): {{
			Tags:       []string{"CVE-2025-1001"},
			Enrichment: []byte(`{"cveID":"CVE-2025-1001","vendorProject":"GNU","product":"bash","vulnerabilityName":"Test KEV","dateAdded":"2025-06-01","shortDescription":"test","requiredAction":"patch","dueDate":"2025-06-15","knownRansomwareCampaignUse":"Known"}`),
		}},
	}

	mkDistIR := func(dist *claircore.Distribution, pkgs ...*claircore.Package) *claircore.IndexReport {
		ir := &claircore.IndexReport{
			State:         "IndexFinished",
			Success:       true,
			Packages:      map[string]*claircore.Package{},
			Distributions: map[string]*claircore.Distribution{dist.ID: dist},
			Environments:  map[string][]*claircore.Environment{},
		}
		for _, p := range pkgs {
			if p.Source == nil {
				p.Source = &claircore.Package{}
			}
			ir.Packages[p.ID] = p
			ir.Environments[p.ID] = []*claircore.Environment{{DistributionID: dist.ID}}
		}
		return ir
	}
	mkRepoIR := func(repo *claircore.Repository, pkgs ...*claircore.Package) *claircore.IndexReport {
		ir := &claircore.IndexReport{
			State:        "IndexFinished",
			Success:      true,
			Packages:     map[string]*claircore.Package{},
			Repositories: map[string]*claircore.Repository{repo.ID: repo},
			Environments: map[string][]*claircore.Environment{},
		}
		for _, p := range pkgs {
			if p.Source == nil {
				p.Source = &claircore.Package{}
			}
			ir.Packages[p.ID] = p
			ir.Environments[p.ID] = []*claircore.Environment{{RepositoryIDs: []string{repo.ID}}}
		}
		return ir
	}

	rhelIR := &claircore.IndexReport{
		State:   "IndexFinished",
		Success: true,
		Packages: map[string]*claircore.Package{
			"p-vuln": {ID: "p-vuln", Name: "bash", Version: "5.1.8-6.el9", Kind: types.BinaryPackage, Module: "mod:1", Source: &claircore.Package{}},
			"p-fix":  {ID: "p-fix", Name: "bash", Version: "5.1.8-9.el9", Kind: types.BinaryPackage, Module: "mod:1", Source: &claircore.Package{}},
		},
		Repositories: map[string]*claircore.Repository{
			rhelRepo1.ID: rhelRepo1,
			rhelRepo2.ID: rhelRepo2,
		},
		Environments: map[string][]*claircore.Environment{
			// Two repositories attached to the same Package.ID exercises multi-record union.
			"p-vuln": {{RepositoryIDs: []string{rhelRepo1.ID, rhelRepo2.ID}}},
			"p-fix":  {{RepositoryIDs: []string{rhelRepo1.ID, rhelRepo2.ID}}},
		},
	}

	cases := []matcherCase{
		{
			name: "alpine",
			report: mkDistIR(alpineDist,
				&claircore.Package{ID: "v", Name: "openssl", Version: "3.0.0-r0", Kind: types.BinaryPackage, Source: &claircore.Package{Name: "openssl", Version: "3.0.0-r0", Kind: types.SourcePackage}},
				&claircore.Package{ID: "f", Name: "openssl", Version: "3.2.0-r0", Kind: types.BinaryPackage, Source: &claircore.Package{Name: "openssl", Version: "3.2.0-r0", Kind: types.SourcePackage}},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "aws",
			report: mkDistIR(awsDist,
				&claircore.Package{ID: "v", Name: "curl", Version: "8.0.0-1.amzn2023", Kind: types.BinaryPackage},
				&claircore.Package{ID: "f", Name: "curl", Version: "8.5.0-1.amzn2023", Kind: types.BinaryPackage},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "debian",
			report: mkDistIR(debDist,
				&claircore.Package{ID: "v", Name: "glibc", Version: "2.36-9", Kind: types.BinaryPackage, Source: &claircore.Package{Name: "glibc", Version: "2.36-9", Kind: types.SourcePackage}},
				&claircore.Package{ID: "f", Name: "glibc", Version: "2.36-10", Kind: types.BinaryPackage, Source: &claircore.Package{Name: "glibc", Version: "2.36-10", Kind: types.SourcePackage}},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "gobin",
			report: mkRepoIR(goRepo,
				&claircore.Package{ID: "v", Name: "golang.org/x/net", Version: "v0.20.0", Kind: types.BinaryPackage, NormalizedVersion: claircore.Version{Kind: "semver", V: [10]int32{0, 20, 0}}},
				&claircore.Package{ID: "f", Name: "golang.org/x/net", Version: "v0.25.0", Kind: types.BinaryPackage, NormalizedVersion: claircore.Version{Kind: "semver", V: [10]int32{0, 25, 0}}},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "java-maven",
			report: mkRepoIR(javaRepo,
				&claircore.Package{ID: "v", Name: "org.apache.logging.log4j:log4j-core", Version: "2.14.1", Kind: types.BinaryPackage},
				&claircore.Package{ID: "f", Name: "org.apache.logging.log4j:log4j-core", Version: "2.17.1", Kind: types.BinaryPackage},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "nodejs",
			report: mkRepoIR(nodeRepo,
				&claircore.Package{ID: "v", Name: "lodash", Version: "4.17.20", Kind: types.BinaryPackage, NormalizedVersion: claircore.Version{Kind: "semver", V: [10]int32{4, 17, 20}}},
				&claircore.Package{ID: "f", Name: "lodash", Version: "4.17.21", Kind: types.BinaryPackage, NormalizedVersion: claircore.Version{Kind: "semver", V: [10]int32{4, 17, 21}}},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "oracle",
			report: mkDistIR(oraDist,
				&claircore.Package{ID: "v", Name: "zlib", Version: "1.2.11-30.el9", Kind: types.BinaryPackage},
				&claircore.Package{ID: "f", Name: "zlib", Version: "1.2.11-40.el9", Kind: types.BinaryPackage},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "photon",
			report: mkDistIR(phoDist,
				&claircore.Package{ID: "v", Name: "expat", Version: "2.5.0-1.ph5", Kind: types.BinaryPackage},
				&claircore.Package{ID: "f", Name: "expat", Version: "2.6.0-1.ph5", Kind: types.BinaryPackage},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "python",
			report: mkRepoIR(pyRepo,
				&claircore.Package{ID: "v", Name: "django", Version: "4.2.1", Kind: types.BinaryPackage, NormalizedVersion: claircore.Version{Kind: "pep440", V: [10]int32{4, 2, 1}}},
				&claircore.Package{ID: "f", Name: "django", Version: "4.2.10", Kind: types.BinaryPackage, NormalizedVersion: claircore.Version{Kind: "pep440", V: [10]int32{4, 2, 10}}},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "rhcc",
			report: mkRepoIR(rhccRepo,
				&claircore.Package{ID: "v", Name: "openshift/ose-rhel-coreos-9", Version: "9.6.20260324-0", Kind: types.BinaryPackage, NormalizedVersion: rhccInstalledMinor.Version(true)},
				&claircore.Package{ID: "f", Name: "openshift/ose-rhel-coreos-9", Version: "9.6.20260326-0", Kind: types.BinaryPackage, NormalizedVersion: rhccPatchedMinor.Version(true)},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name:          "rhel-and-all-enrichers",
			report:        rhelIR,
			wantVulnPkgs:  []string{"p-vuln"},
			wantCleanPkgs: []string{"p-fix"},
		},
		{
			name: "redhat-java",
			report: mkRepoIR(rhelJavaRepo,
				&claircore.Package{ID: "v", Name: "com.fasterxml.jackson.core:jackson-databind", Version: "2.13.1", Kind: types.BinaryPackage},
				&claircore.Package{ID: "f", Name: "com.fasterxml.jackson.core:jackson-databind", Version: "2.15.0", Kind: types.BinaryPackage},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "ruby-gem",
			report: mkRepoIR(rubyRepo,
				&claircore.Package{ID: "v", Name: "rack", Version: "2.2.4", Kind: types.BinaryPackage},
				&claircore.Package{ID: "f", Name: "rack", Version: "2.2.9", Kind: types.BinaryPackage},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "suse",
			report: mkDistIR(suseDist,
				&claircore.Package{ID: "v", Name: "libxml2", Version: "2.9.14-1.1", Kind: types.BinaryPackage},
				&claircore.Package{ID: "f", Name: "libxml2", Version: "2.9.14-2.1", Kind: types.BinaryPackage},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
		{
			name: "ubuntu",
			report: mkDistIR(ubuDist,
				&claircore.Package{ID: "v", Name: "sudo", Version: "1.9.9-1ubuntu2", Kind: types.BinaryPackage, Source: &claircore.Package{Name: "sudo", Version: "1.9.9-1ubuntu2", Kind: types.SourcePackage}},
				&claircore.Package{ID: "f", Name: "sudo", Version: "1.9.9-1ubuntu2.5", Kind: types.BinaryPackage, Source: &claircore.Package{Name: "sudo", Version: "1.9.9-1ubuntu2.5", Kind: types.SourcePackage}},
			),
			wantVulnPkgs: []string{"v"}, wantCleanPkgs: []string{"f"},
		},
	}

	return vulns, enrich, cases
}
