//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchNodeComponents(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	store := postgres.New(testDB)

	ds := &datastoreImpl{
		storage:             store,
		nodeComponentRanker: ranking.NodeComponentRanker(),
	}

	component1 := fixtures.GetEmbeddedNodeComponent1x1()
	component2 := fixtures.GetEmbeddedNodeComponent1x2()
	component3 := fixtures.GetEmbeddedNodeComponent1s2x3()

	// Convert embedded components to full storage components
	nodeComponent1 := &storage.NodeComponent{
		Id:              "comp1",
		Name:            component1.GetName(),
		Version:         component1.GetVersion(),
		OperatingSystem: "ubuntu:20.04",
		RiskScore:       5.0,
	}
	nodeComponent2 := &storage.NodeComponent{
		Id:              "comp2",
		Name:            component2.GetName(),
		Version:         component2.GetVersion(),
		OperatingSystem: "rhel:8",
		RiskScore:       3.0,
	}
	nodeComponent3 := &storage.NodeComponent{
		Id:              "comp3",
		Name:            component3.GetName(),
		Version:         component3.GetVersion(),
		OperatingSystem: "ubuntu:20.04",
		RiskScore:       7.5,
	}

	require.NoError(t, store.Upsert(ctx, nodeComponent1))
	require.NoError(t, store.Upsert(ctx, nodeComponent2))
	require.NoError(t, store.Upsert(ctx, nodeComponent3))

	t.Run("SearchNodeComponents with nil query", func(t *testing.T) {
		searchResults, err := ds.SearchNodeComponents(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, searchResults, 3)

		for _, result := range searchResults {
			assert.Equal(t, v1.SearchCategory_NODE_COMPONENTS, result.GetCategory(), "Result category should be NODE_COMPONENTS")
		}
	})

	t.Run("SearchNodeComponents with empty query", func(t *testing.T) {
		searchResults, err := ds.SearchNodeComponents(ctx, pkgSearch.EmptyQuery())
		assert.NoError(t, err)
		assert.Len(t, searchResults, 3)

		resultMap := make(map[string]*v1.SearchResult)
		for _, result := range searchResults {
			resultMap[result.GetId()] = result
		}

		result1 := resultMap[nodeComponent1.GetId()]
		require.NotNil(t, result1)
		assert.Equal(t, nodeComponent1.GetId(), result1.GetId())
		assert.Equal(t, nodeComponent1.GetName(), result1.GetName())
		assert.Equal(t, v1.SearchCategory_NODE_COMPONENTS, result1.GetCategory())
		assert.NotNil(t, result1.GetFieldToMatches())

		result2 := resultMap[nodeComponent2.GetId()]
		require.NotNil(t, result2)
		assert.Equal(t, nodeComponent2.GetId(), result2.GetId())
		assert.Equal(t, nodeComponent2.GetName(), result2.GetName())
		assert.Equal(t, v1.SearchCategory_NODE_COMPONENTS, result2.GetCategory())
		assert.NotNil(t, result2.GetFieldToMatches())

		result3 := resultMap[nodeComponent3.GetId()]
		require.NotNil(t, result3)
		assert.Equal(t, nodeComponent3.GetId(), result3.GetId())
		assert.Equal(t, nodeComponent3.GetName(), result3.GetName())
		assert.Equal(t, v1.SearchCategory_NODE_COMPONENTS, result3.GetCategory())
		assert.NotNil(t, result3.GetFieldToMatches())
	})

	t.Run("SearchNodeComponents with name filter", func(t *testing.T) {
		q := pkgSearch.NewQueryBuilder().
			AddExactMatches(pkgSearch.Component, nodeComponent1.GetName()).
			ProtoQuery()

		searchResults, err := ds.SearchNodeComponents(ctx, q)
		assert.NoError(t, err)
		assert.Len(t, searchResults, 1)

		result := searchResults[0]
		assert.Equal(t, nodeComponent1.GetId(), result.GetId())
		assert.Equal(t, nodeComponent1.GetName(), result.GetName())
		assert.Equal(t, v1.SearchCategory_NODE_COMPONENTS, result.GetCategory())
	})

}
