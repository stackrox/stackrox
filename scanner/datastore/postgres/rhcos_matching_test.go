//go:build scanner_db_integration

package postgres

import (
	"math"
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/pkg/rhctag"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/claircore/test"
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
// normalized-version filtering and fixed-version comparison still apply. It does
// not validate VEX version extraction or node/advisory version comparability.
//
// Run with go test -tags scanner_db_integration ./scanner/datastore/postgres
// -run '^TestRHCOSNameMatching$'. It requires PostgreSQL on 127.0.0.1:5432
// accepting user postgres without a password.
func TestRHCOSNameMatching(t *testing.T) {
	// The database helper uses ctx during cleanup, after t.Context is canceled.
	ctx := test.Logging(t)
	pool := testDB(t, ctx, "rhcos_name_matching_test")
	store, err := InitPostgresMatcherStore(ctx, pool, true)
	require.NoError(t, err)

	const imageName = "openshift/ose-rhel-coreos-9"
	const installedVersion = "9.6.20260324-0"
	installed, err := rhctag.Parse(installedVersion)
	require.NoError(t, err)
	minorStart := installed.MinorStart()
	normalizedVersion := minorStart.Version(true)
	repoCPE, err := cpe.Unbind("cpe:/a:redhat:openshift:4.21::el9")
	require.NoError(t, err)

	ir := &claircore.IndexReport{
		Packages: make(map[string]*claircore.Package),
		Repositories: map[string]*claircore.Repository{
			"rhcos-repo": {ID: "rhcos-repo", Name: repoCPE.String(), Key: rhcc.RepositoryKey, CPE: repoCPE},
		},
	}
	for _, pair := range []struct{ name, binaryID, sourceID string }{
		{imageName, rhcos.PackageID, rhcos.SourceID},
		{"rhcos", rhcos.LegacyPackageID, rhcos.LegacySourceID},
	} {
		source := &claircore.Package{
			ID: pair.sourceID, Name: pair.name, Kind: types.SourcePackage,
			Version: installedVersion, NormalizedVersion: normalizedVersion,
		}
		ir.Packages[source.ID] = source
		ir.Packages[pair.binaryID] = &claircore.Package{
			ID: pair.binaryID, Name: pair.name, Kind: types.BinaryPackage, Version: source.Version, Source: source,
			NormalizedVersion: normalizedVersion,
		}
	}
	protoReport, err := mappers.ToProtoV4IndexReport(ir)
	require.NoError(t, err)
	ir, err = mappers.ToClairCoreIndexReport(protoReport.GetContents())
	require.NoError(t, err)
	for _, pkg := range ir.Packages {
		assert.Equal(t, normalizedVersion, pkg.NormalizedVersion, "serialization must preserve normalized versions")
	}

	// "openshift/ose-rhel-coreos-8" is a different RHEL image name and must not
	// match the RHEL 9 node inventory, even though it shares the product CPE.
	const fixedVersion = "9.6.20260325-0"
	fixed, err := rhctag.Parse(fixedVersion)
	require.NoError(t, err)
	zero := new(rhctag.Version).Version(true)
	unbounded := (&rhctag.Version{Major: math.MaxInt32}).Version(true)
	otherStream := (&rhctag.Version{Major: 10}).Version(true)
	var vulns []*claircore.Vulnerability
	for _, name := range []string{"rhcos", imageName, "openshift/ose-rhel-coreos-8"} {
		for _, kind := range []types.PackageKind{types.BinaryPackage, types.SourcePackage} {
			var fixedIn string
			upper := unbounded
			if kind == types.BinaryPackage {
				fixedIn = fixedVersion
				upper = fixed.Version(false)
			}
			vulns = append(vulns, &claircore.Vulnerability{
				Name:           name,
				Updater:        "rhcos-test",
				Package:        &claircore.Package{Name: name, Kind: kind},
				Repo:           &claircore.Repository{Name: repoCPE.String(), Key: rhcc.RepositoryKey},
				FixedInVersion: fixedIn,
				Range:          &claircore.Range{Lower: zero, Upper: upper},
			})
			// The name, kind, and repository match, but the normalized version
			// is outside this range. Only database-side filtering rejects it.
			vulns = append(vulns, &claircore.Vulnerability{
				Name:    name + "-other-stream",
				Updater: "rhcos-test",
				Package: &claircore.Package{Name: name, Kind: kind},
				Repo:    &claircore.Repository{Name: repoCPE.String(), Key: rhcc.RepositoryKey},
				Range:   &claircore.Range{Lower: otherStream, Upper: unbounded},
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
			opts := datastore.GetOpts{Matchers: rhcc.Matcher.Query(), VersionFiltering: true}
			found, err := store.Get(ctx, []*claircore.IndexRecord{record}, opts)
			require.NoError(t, err)
			require.Len(t, found[id], 2, "binary and source assertions must both be selected")
			for _, vuln := range found[id] {
				assert.Equal(t, record.Package.Name, vuln.Name, "out-of-range assertions must be excluded")
			}

			opts.VersionFiltering = false
			unfiltered, err := store.Get(ctx, []*claircore.IndexRecord{record}, opts)
			require.NoError(t, err)
			require.Len(t, unfiltered[id], 4, "without version filtering, both out-of-range assertions must also be selected")

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
