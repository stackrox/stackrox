//go:build scanner_db_integration

package notaffected

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
	pkgnotaffected "github.com/stackrox/rox/pkg/scannerv4/enricher/notaffected"
	roxpostgres "github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDB(t *testing.T, ctx context.Context, name string) *pgxpool.Pool {
	t.Helper()

	pgConn := "postgresql://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"
	pgPool, err := postgres.Connect(ctx, pgConn, name)
	require.NoError(t, err)
	createDatabase := `CREATE DATABASE ` + name
	_, err = pgPool.Exec(ctx, createDatabase)
	require.NoError(t, err)
	t.Cleanup(pgPool.Close)

	dbConn := fmt.Sprintf("postgresql://postgres:postgres@127.0.0.1:5432/%s?sslmode=disable", name)
	dbPool, err := postgres.Connect(ctx, dbConn, name)
	require.NoError(t, err)
	t.Cleanup(func() {
		dropDatabase := `DROP DATABASE IF EXISTS ` + name
		_, _ = pgPool.Exec(ctx, dropDatabase)
	})
	t.Cleanup(dbPool.Close)

	return dbPool
}

func TestNotAffectedIntegration(t *testing.T) {
	ctx := zlog.Test(context.Background(), t)
	pool := testDB(t, ctx, "notaffected_integration_test")

	store, err := roxpostgres.InitPostgresMatcherStore(ctx, pool, true)
	require.NoError(t, err)

	serverURL, httpClient := ServeSecDB(t, "testdata/server.txtar")
	enricher := &Enricher{}

	err = enricher.Configure(ctx, func(v interface{}) error {
		cfg := v.(*Config)
		cfg.URL = serverURL + "/"
		cfg.MaxCVEsPerRecord = 5
		return nil
	}, httpClient)
	require.NoError(t, err)

	data, fingerprint, err := enricher.FetchEnrichment(ctx, "")
	require.NoError(t, err)
	defer func() {
		_ = data.Close()
	}()
	require.NotEmpty(t, fingerprint, "Fingerprint should not be empty")

	records, err := enricher.ParseEnrichment(ctx, data)
	require.NoError(t, err)
	require.NotEmpty(t, records, "Should have parsed enrichment records")

	_, err = store.UpdateEnrichments(ctx, enricher.Name(), fingerprint, records)
	require.NoError(t, err, "Failed to store enrichments in database")

	getter := &storeEnrichmentGetter{store: store, enricherName: enricher.Name()}

	t.Run("Red Hat Image Gets Enrichments", func(t *testing.T) {
		vr := createRedHatVulnReport()

		enrichmentType, data, err := enricher.Enrich(ctx, getter, vr)
		require.NoError(t, err)

		assert.Equal(t, pkgnotaffected.Type, enrichmentType)
		require.NotEmpty(t, data, "Should have enrichment data from database")

		var enrichments map[string][]json.RawMessage
		err = json.Unmarshal(data[0], &enrichments)
		require.NoError(t, err)

		var cveList []string

		// Check all Red Hat products.
		assert.Contains(t, enrichments, pkgnotaffected.RedHatProducts)
		redHatEnrichments := enrichments[pkgnotaffected.RedHatProducts]
		require.NotEmpty(t, redHatEnrichments, "Should have enrichments for red_hat_products")
		err = json.Unmarshal(redHatEnrichments[0], &cveList)
		require.NoError(t, err)
		assert.Contains(t, cveList, "CVE-2024-21613", "Should contain CVE from fixture")

		// Check specific not-affected for RHACS.
		assert.Contains(t, enrichments, pkgnotaffected.RedHatProducts)
		acsEnrichments := enrichments["advanced-cluster-security/rhacs-scanner-v4-rhel8"]
		require.NotEmpty(t, acsEnrichments, "Should have enrichments for red_hat_products")
		err = json.Unmarshal(acsEnrichments[0], &cveList)
		require.NoError(t, err)
		assert.Contains(t, cveList, "CVE-2025-7783", "Should contain CVE-2025-7783 for ACS roxctl product")

	})

	t.Run("Non Red Hat Image Gets No Enrichments", func(t *testing.T) {
		vr := createUbuntuVulnReport()

		enrichmentType, enrichments, err := enricher.Enrich(ctx, getter, vr)
		require.NoError(t, err)
		assert.Equal(t, pkgnotaffected.Type, enrichmentType)
		assert.Empty(t, enrichments, "Non-Red Hat image should get no enrichments")
	})

	t.Run("Database Error Handling", func(t *testing.T) {
		vr := createRedHatVulnReport()
		brokenGetter := &errorEnrichmentGetter{}
		enrichmentType, enrichments, err := enricher.Enrich(ctx, brokenGetter, vr)
		require.Error(t, err, "Should propagate database errors")
		assert.Equal(t, pkgnotaffected.Type, enrichmentType)
		assert.Empty(t, enrichments)
	})
}

type storeEnrichmentGetter struct {
	store        roxpostgres.MatcherStore
	enricherName string
}

func (g *storeEnrichmentGetter) GetEnrichment(ctx context.Context, tags []string) ([]driver.EnrichmentRecord, error) {
	return g.store.GetEnrichment(ctx, g.enricherName, tags)
}

type errorEnrichmentGetter struct{}

func (g *errorEnrichmentGetter) GetEnrichment(ctx context.Context, tags []string) ([]driver.EnrichmentRecord, error) {
	return nil, assert.AnError
}

func createUbuntuVulnReport() *claircore.VulnerabilityReport {
	ubuntuRepo := &claircore.Repository{
		Name: "ubuntu",
		Key:  "ubuntu-key",
		URI:  "http://archive.ubuntu.com/ubuntu",
	}
	pkg := &claircore.Package{
		ID:      "pkg1",
		Name:    "libc6",
		Version: "2.31-0ubuntu9.9",
		Kind:    "binary",
		Arch:    "amd64",
	}
	env := []*claircore.Environment{
		{
			PackageDB:     "var/lib/dpkg/status",
			IntroducedIn:  claircore.MustParseDigest("sha256:fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"),
			RepositoryIDs: []string{"repo1"},
		},
	}
	return &claircore.VulnerabilityReport{
		Packages: map[string]*claircore.Package{
			"pkg1": pkg,
		},
		Repositories: map[string]*claircore.Repository{
			"repo1": ubuntuRepo,
		},
		Environments: map[string][]*claircore.Environment{
			"pkg1": env,
		},
		Vulnerabilities:        make(map[string]*claircore.Vulnerability),
		PackageVulnerabilities: make(map[string][]string),
	}
}

func createRedHatVulnReport() *claircore.VulnerabilityReport {
	acsRepo := &claircore.Repository{
		ID:   "1",
		Name: "Red Hat Container Catalog",
		URI:  "https://catalog.redhat.com/software/containers/explore",
	}
	pkg := &claircore.Package{
		ID:      "1",
		Name:    "advanced-cluster-security/rhacs-scanner-v4-rhel8",
		Version: "4.8.0-5",
		Kind:    "binary",
		Arch:    "x86_64",
	}
	env := []*claircore.Environment{
		{
			PackageDB:     "root/buildinfo/Dockerfile-rhacs-scanner-v4-rhel8-4.8.0-5",
			IntroducedIn:  claircore.MustParseDigest("sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef123456789a"),
			RepositoryIDs: []string{"1"},
		},
	}
	return &claircore.VulnerabilityReport{
		Packages: map[string]*claircore.Package{
			"1": pkg,
		},
		Repositories: map[string]*claircore.Repository{
			"1": acsRepo,
		},
		Environments: map[string][]*claircore.Environment{
			"1": env,
		},
		Vulnerabilities:        make(map[string]*claircore.Vulnerability),
		PackageVulnerabilities: make(map[string][]string),
	}
}
