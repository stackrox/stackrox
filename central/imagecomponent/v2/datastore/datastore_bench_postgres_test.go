//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/imagecomponent/v2/views"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/require"
)

// BenchmarkInitializeRankers compares the current Walk-based approach for
// initializing component rankers against a lightweight select query that
// fetches only the component ID and risk score.
func BenchmarkInitializeRankers(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)
	store := pgStore.New(testDB.DB)

	// Insert a parent image row to satisfy the foreign key constraint.
	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		_, err := testDB.DB.Exec(ctx, "INSERT INTO images_v2 (Id, serialized) VALUES ($1, $2) ON CONFLICT DO NOTHING", "bench-image", []byte{})
		require.NoError(b, err)
	} else {
		_, err := testDB.DB.Exec(ctx, "INSERT INTO images (Id, serialized) VALUES ($1, $2) ON CONFLICT DO NOTHING", "bench-image", []byte{})
		require.NoError(b, err)
	}

	// Insert test components with realistic serialized blob sizes.
	numComponents := 500
	for i := 0; i < numComponents; i++ {
		component := &storage.ImageComponentV2{
			Id:              fmt.Sprintf("component-%d", i),
			Name:            fmt.Sprintf("pkg-%d", i),
			Version:         fmt.Sprintf("%d.0.0", i),
			RiskScore:       float32(i) / float32(numComponents),
			Source:          storage.SourceType_OS,
			OperatingSystem: "linux",
			Location:        fmt.Sprintf("/usr/lib/pkg-%d", i),
		}
		// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
		if features.FlattenImageData.Enabled() {
			component.ImageIdV2 = "bench-image"
		} else {
			component.ImageId = "bench-image"
		}
		require.NoError(b, store.Upsert(ctx, component))
	}

	b.Run("Walk", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			ranker := ranking.NewRanker()
			count := 0
			err := store.Walk(ctx, func(component *storage.ImageComponentV2) error {
				ranker.Add(component.GetId(), component.GetRiskScore())
				count++
				return nil
			})
			require.NoError(b, err)
			require.Equal(b, numComponents, count)
		}
	})

	b.Run("SelectRiskView", func(b *testing.B) {
		b.ReportAllocs()
		selects := []*v1.QuerySelect{
			search.NewQuerySelect(search.ComponentID).Proto(),
			search.NewQuerySelect(search.ComponentRiskScore).Proto(),
		}
		query := search.EmptyQuery()
		query.Selects = selects

		for b.Loop() {
			ranker := ranking.NewRanker()
			count := 0
			err := pgSearch.RunSelectRequestForSchemaFn[views.ComponentRiskView](
				ctx, testDB.DB, pkgSchema.ImageComponentV2Schema(), query,
				func(r *views.ComponentRiskView) error {
					ranker.Add(r.ComponentID, r.ComponentRiskScore)
					count++
					return nil
				},
			)
			require.NoError(b, err)
			require.Equal(b, numComponents, count)
		}
	})
}
