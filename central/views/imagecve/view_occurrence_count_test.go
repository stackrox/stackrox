//go:build sql_integration

package imagecve

import (
	"context"
	"testing"

	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOccurrenceCount(t *testing.T) {
	testDB := pgtest.ForT(t)
	ctx := sac.WithAllAccess(context.Background())

	// Use standard fixture image but inject a duplicate CVE entry — same CVE,
	// same component, different severity — simulating scanner producing
	// duplicate advisory entries.
	img := fixtures.GetImage()
	firstComp := img.GetScan().GetComponents()[0]
	firstComp.Vulns = append(firstComp.Vulns,
		&storage.EmbeddedVulnerability{
			Cve:               "CVE-2099-99999",
			Cvss:              9.1,
			Severity:          storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.79.3"},
		},
		&storage.EmbeddedVulnerability{
			Cve:               "CVE-2099-99999",
			Cvss:              0,
			Severity:          storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			SetFixedBy:        &storage.EmbeddedVulnerability_FixedBy{FixedBy: "1.79.3"},
		},
	)

	if features.FlattenImageData.Enabled() {
		v2Store := imageV2DS.GetTestPostgresDataStore(t, testDB.DB)
		imgV2 := imageUtils.ConvertToV2(img)
		require.NoError(t, v2Store.UpsertImage(ctx, imgV2))
	} else {
		v1Store := imageDS.GetTestPostgresDataStore(t, testDB.DB)
		require.NoError(t, v1Store.UpsertImage(ctx, img))
	}

	cveView := NewCVEView(testDB.DB)

	q := search.NewQueryBuilder().
		AddExactMatches(search.CVE, "CVE-2099-99999").
		ProtoQuery()
	q.Pagination = &v1.QueryPagination{Limit: 10}

	results, err := cveView.Get(ctx, q, views.ReadOptions{})
	require.NoError(t, err)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, "CVE-2099-99999", result.GetCVE())
	assert.Greater(t, result.GetOccurrenceCount(), 0,
		"OccurrenceCount should be > 0 when scanner produces duplicate CVE entries")
	assert.Equal(t, 2, result.GetOccurrenceCount(),
		"Expected 2 occurrences: one CRITICAL, one UNKNOWN for the same CVE+component")
}
