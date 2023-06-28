//go:build sql_integration

package reprocessor

import (
	"context"
	"testing"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imagePG "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
)

func TestGetActiveImageIDs(t *testing.T) {
	t.Parallel()

	testCtx := sac.WithAllAccess(context.Background())

	var (
		pool          postgres.DB
		imageDS       imageDatastore.DataStore
		deploymentsDS deploymentDatastore.DataStore
		err           error
	)

	testingDB := pgtest.ForT(t)
	pool = testingDB.DB
	defer pool.Close()

	imageDS = imageDatastore.NewWithPostgres(imagePG.New(pool, false, concurrency.NewKeyFence()), imagePG.NewIndexer(pool), nil, ranking.ImageRanker(), ranking.ComponentRanker())
	deploymentsDS, err = deploymentDatastore.New(pool, nil, nil, nil, nil, nil, filter.NewFilter(5, 5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

	loop := NewLoop(nil, nil, nil, deploymentsDS, imageDS, nil, nil, nil, nil).(*loopImpl)

	ids, err := loop.getActiveImageIDs()
	require.NoError(t, err)
	require.Equal(t, 0, len(ids))

	deployment := fixtures.GetDeployment()
	require.NoError(t, deploymentsDS.UpsertDeployment(testCtx, deployment))

	images := fixtures.DeploymentImages()
	imageIDs := make([]string, 0, len(images))
	for _, image := range images {
		require.NoError(t, imageDS.UpsertImage(testCtx, image))
		imageIDs = append(imageIDs, image.GetId())
	}

	ids, err = loop.getActiveImageIDs()
	require.NoError(t, err)
	require.ElementsMatch(t, imageIDs, ids)
}
