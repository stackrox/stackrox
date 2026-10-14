//go:build scanner_db_integration

package index

import (
	"context"
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/toolkit/types"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/pkg/scannerv4/rhcos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRHCOSNameMatching exercises ClairCore's name/kind query and RHCC matcher
// against the serialized node inventory, using a real database. It verifies that
// both the canonical and legacy image names select binary and source assertions,
// that another RHEL image name is rejected, and that the product CPE filter and
// fixed-version comparison still apply. It does not validate VEX version
// extraction or node/advisory version comparability.
//
// It runs under the scanner_db_integration tag (make -C scanner
// db-integration-test) and requires PostgreSQL on 127.0.0.1:5432 accepting user
// postgres without a password.
func TestRHCOSNameMatching(t *testing.T) {
	ctx := t.Context()
	const baseDSN = "postgresql://postgres@127.0.0.1:5432/"
	admin, err := postgres.Connect(ctx, baseDSN+"postgres?sslmode=disable", t.Name())
	require.NoError(t, err)
	t.Cleanup(admin.Close)
	const db = "rhcos_name_matching_test"
	_, err = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+db)
	require.NoError(t, err)
	_, err = admin.Exec(ctx, "CREATE DATABASE "+db)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+db)
		assert.NoError(t, err)
	})
	pool, err := postgres.Connect(ctx, baseDSN+db+"?sslmode=disable", t.Name())
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	require.NoError(t, err)

	rel, err := osRelease(ctx, "testdata-rhcos")
	require.NoError(t, err)

	// Serialize and deserialize the inventory so the records under test match
	// what Scanner receives, rather than the in-memory report.
	ir := &claircore.IndexReport{}
	addRHCOS(rel, "x86_64", ir)
	protoReport, err := mappers.ToProtoV4IndexReport(ir)
	require.NoError(t, err)
	ir, err = mappers.ToClairCoreIndexReport(protoReport.GetContents())
	require.NoError(t, err)

	// "openshift/ose-rhel-coreos-8" is a different RHEL image name and must not
	// match the RHEL 9 node inventory, even though it shares the product CPE.
	const fixedVersion = "9.6.20260325-0"
	var vulns []*claircore.Vulnerability
	for _, name := range []string{"rhcos", rel.imageName, "openshift/ose-rhel-coreos-8"} {
		for _, kind := range []types.PackageKind{types.BinaryPackage, types.SourcePackage} {
			var fixedIn string
			if kind == types.BinaryPackage {
				fixedIn = fixedVersion
			}
			vulns = append(vulns, &claircore.Vulnerability{
				Name:           name,
				Updater:        "rhcos-test",
				Package:        &claircore.Package{Name: name, Kind: kind},
				Repo:           &claircore.Repository{Name: rel.repoCPE.String(), Key: rhcc.RepositoryKey},
				FixedInVersion: fixedIn,
			})
		}
	}
	_, err = store.UpdateVulnerabilities(ctx, "rhcos-test", "rhcos-test", vulns)
	require.NoError(t, err)

	for _, id := range []string{rhcos.PackageID, rhcos.LegacyPackageID} {
		t.Run(id, func(t *testing.T) {
			record := &claircore.IndexRecord{
				Package:    ir.Packages[id],
				Repository: ir.Repositories["rhcos-repo"],
			}
			found, err := store.Get(ctx, []*claircore.IndexRecord{record}, datastore.GetOpts{Matchers: rhcc.Matcher.Query()})
			require.NoError(t, err)
			require.Len(t, found[id], 2, "binary and source assertions must both be selected")

			for _, vuln := range found[id] {
				assert.Equal(t, record.Package.Name, vuln.Package.Name)

				matched, err := rhcc.Matcher.Vulnerable(ctx, record, vuln)
				require.NoError(t, err)
				assert.True(t, matched)

				if vuln.FixedInVersion != "" {
					fixedPkg := *record.Package
					fixedPkg.Version = fixedVersion
					fixedRecord := *record
					fixedRecord.Package = &fixedPkg
					matched, err := rhcc.Matcher.Vulnerable(ctx, &fixedRecord, vuln)
					require.NoError(t, err)
					assert.False(t, matched, "installed fixed version must not match")
				}

				// A matching name must not bypass the product CPE filter.
				otherCPE, err := cpe.Unbind("cpe:/a:redhat:openshift:4.20::el9")
				require.NoError(t, err)
				vuln.Repo = &claircore.Repository{Name: otherCPE.String(), Key: rhcc.RepositoryKey}
				matched, err = rhcc.Matcher.Vulnerable(ctx, record, vuln)
				require.NoError(t, err)
				assert.False(t, matched)
			}
		})
	}
}
