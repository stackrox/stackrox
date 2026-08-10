//go:build sql_integration

package imagecve

import (
	"context"
	"testing"

	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageSamples "github.com/stackrox/rox/pkg/fixtures/image"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/require"
)

func BenchmarkTwoPhaseGet(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		))

	testDB := pgtest.ForT(b)
	cveView := NewCVEView(testDB.DB)

	if features.FlattenImageData.Enabled() {
		imageV2Store := imageV2DS.GetTestPostgresDataStore(b, testDB.DB)
		imagesV2, err := imageSamples.GetTestImagesV2(&testing.T{})
		require.NoError(b, err)
		for _, img := range imagesV2 {
			require.NoError(b, imageV2Store.UpsertImage(ctx, img))
		}
		b.Logf("Seeded %d V2 images", len(imagesV2))
	} else {
		imageStore := imageDS.GetTestPostgresDataStore(b, testDB.DB)
		require.NoError(b, imageStore.UpsertImage(ctx, fixtures.GetImageSherlockHolmes1()))
		require.NoError(b, imageStore.UpsertImage(ctx, fixtures.GetImageDoctorJekyll2()))
		b.Logf("Seeded 2 V1 images")
	}

	paginatedQ := search.NewQueryBuilder().
		AddStrings(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).
		WithPagination(
			search.NewPagination().
				Limit(20).
				AddSortOption(search.NewSortOption(search.CVSS).AggregateBy(aggregatefunc.Max, false)),
		).ProtoQuery()

	b.Run("paginated_cvss_sort", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			results, err := cveView.Get(ctx, paginatedQ, views.ReadOptions{})
			require.NoError(b, err)
			require.NotEmpty(b, results)
		}
	})

	severitySortQ := search.NewQueryBuilder().
		AddStrings(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).
		WithPagination(
			search.NewPagination().
				Limit(20).
				AddSortOption(search.NewSortOption(search.CriticalSeverityCount)),
		).ProtoQuery()

	b.Run("paginated_severity_count_sort", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			results, err := cveView.Get(ctx, severitySortQ, views.ReadOptions{})
			require.NoError(b, err)
			require.NotEmpty(b, results)
		}
	})
}
