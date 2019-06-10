package datastore

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/deployment/datastore/internal/search"
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/store"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	search2 "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearchAllDeployments(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		))

	tempPath, err := ioutil.TempDir("", "")
	require.NoError(b, err)

	boltPath := filepath.Join(tempPath, "bolt.db")
	blevePath := filepath.Join(tempPath, "scorch.bleve")

	db, err := bolthelper.New(boltPath)
	require.NoError(b, err)

	bleveIndex, err := globalindex.InitializeIndices(blevePath)
	require.NoError(b, err)

	deploymentsStore, err := store.New(db)
	require.NoError(b, err)

	deploymentsIndexer := index.New(bleveIndex)
	deploymentsSearcher := search.New(deploymentsStore, deploymentsIndexer)

	imageDS, err := imageDatastore.New(db, bleveIndex, false)
	require.NoError(b, err)

	deploymentsDatastore, err := newDatastoreImpl(deploymentsStore, deploymentsIndexer, deploymentsSearcher, imageDS, nil, nil, nil)
	require.NoError(b, err)

	deploymentPrototype := proto.Clone(fixtures.GetDeployment()).(*storage.Deployment)

	const numDeployments = 1000
	for i := 0; i < numDeployments; i++ {
		if i > 0 && i%100 == 0 {
			fmt.Println("Added", i, "deployments")
		}
		deploymentPrototype.Id = fmt.Sprintf("deployment%d", i)
		require.NoError(b, deploymentsDatastore.UpsertDeployment(ctx, deploymentPrototype))
	}

	b.Run("SearchRetrievalList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.SearchListDeployments(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})

	b.Run("GetAllRetrievalList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.ListDeployments(ctx)
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})

	b.Run("SearchRetrievalFull", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.SearchRawDeployments(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})

	b.Run("GetAllRetrievalFull", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.GetDeployments(ctx)
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})
}
