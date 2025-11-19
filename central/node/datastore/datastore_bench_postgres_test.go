//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/node/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/nodes/converter"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkGetManyNodes(b *testing.B) {

	ctx := sac.WithAllAccess(context.Background())

	pool := pgtest.ForT(b)
	defer pool.Close()

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	store := pgStore.New(pool, false, concurrency.NewKeyFence())
	datastore := NewWithPostgres(store, mockRisk, ranking.NewRanker(), ranking.NewRanker())

	ids := make([]string, 0, 100)
	nodes := make([]*storage.Node, 0, 100)

	for i := 0; i < 100; i++ {
		node := fixtures.GetNodeWithUniqueComponents(5, 5)
		converter.MoveNodeVulnsToNewField(node)
		id := uuid.NewV4().String()
		ids = append(ids, id)
		node.Id = id
		nodes = append(nodes, node)
	}

	for _, node := range nodes {
		require.NoError(b, datastore.UpsertNode(ctx, node))
	}

	var err error

	b.Run("GetNodesBatch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err = datastore.GetNodesBatch(ctx, ids)
			require.NoError(b, err)
		}
	})

	b.Run("GetManyNodeMetadata", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err = datastore.GetManyNodeMetadata(ctx, ids)
			require.NoError(b, err)
		}
	})
}
