//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func BenchmarkCollections(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())

	db := pgtest.ForT(b)
	defer db.Teardown(b)

	datastore, _, err := GetTestPostgresDataStore(b, db)
	require.NoError(b, err)

	numSeedObjects := 5000

	ids := make([]string, 0, numSeedObjects)
	collections := make([]*storage.ResourceCollection, 0, numSeedObjects)
	for i := 0; i < numSeedObjects; i++ {
		name := fmt.Sprintf("%d", i)
		collections = append(collections, getTestCollection(name, nil))
		id, err := datastore.AddCollection(ctx, collections[i])
		require.NoError(b, err)
		ids = append(ids, id)
	}

	// DryRun Add
	b.Run("DryRunAddCollection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var start, end int
			start = i % numSeedObjects
			if start < numSeedObjects-5 {
				end = start + 5
			} else {
				end = numSeedObjects - 1
			}
			collection := getTestCollection("name", ids[start:end])
			err = datastore.DryRunAddCollection(ctx, collection)
			require.NoError(b, err)
		}
	})

	// DryRun Update
	b.Run("DryRunUpdateCollection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var start, end int
			start = rand.Intn(numSeedObjects-i-1) + i + 1
			if start < numSeedObjects-5 {
				end = start + 5
			} else {
				end = numSeedObjects - 1
			}
			collection := getTestCollection("name", ids[start:end])
			collection.Id = collections[i].GetId()
			err = datastore.DryRunUpdateCollection(ctx, collection)
			require.NoError(b, err)
		}
	})

	// Update
	b.Run("UpdateCollection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var start, end int
			start = rand.Intn(numSeedObjects-i-1) + i + 1
			if start < numSeedObjects-5 {
				end = start + 5
			} else {
				end = numSeedObjects - 1
			}
			collection := getTestCollection(uuid.NewV4().String(), ids[start:end])
			collection.Id = collections[i].GetId()
			err = datastore.UpdateCollection(ctx, collection)
			require.NoError(b, err)
		}
	})

	// graphInit
	dsImpl := datastore.(*datastoreImpl)
	b.Run("graphInit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			require.NoError(b, resetLocalGraph(dsImpl))
		}
	})

	// Add, last so we know how many entries for previous runs
	b.Run("AddCollection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var start, end int
			start = i % numSeedObjects
			if start < numSeedObjects-5 {
				end = start + 5
			} else {
				end = numSeedObjects - 1
			}
			collection := getTestCollection(uuid.NewV4().String(), ids[start:end])
			_, err = datastore.AddCollection(ctx, collection)
			require.NoError(b, err)
		}
	})
}
