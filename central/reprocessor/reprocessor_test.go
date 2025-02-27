//go:build sql_integration

package reprocessor

import (
	"context"
	"testing"
	"time"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imagePG "github.com/stackrox/rox/central/image/datastore/store/postgres"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
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

	imageDS = imageDatastore.NewWithPostgres(imagePG.New(pool, false, concurrency.NewKeyFence()), nil, ranking.ImageRanker(), ranking.ComponentRanker())
	deploymentsDS, err = deploymentDatastore.New(pool, nil, nil, nil, nil, nil, filter.NewFilter(5, 5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker(), platformmatcher.Singleton())
	require.NoError(t, err)

	loop := NewLoop(nil, nil, nil, deploymentsDS, imageDS, nil, nil, nil, nil, nil).(*loopImpl)

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

func TestImagesWithSignaturesQuery(t *testing.T) {
	t.Parallel()

	testCtx := sac.WithAllAccess(context.Background())

	testingDB := pgtest.ForT(t)
	pool := testingDB.DB
	defer pool.Close()

	imageDS := imageDatastore.NewWithPostgres(imagePG.New(pool, false,
		concurrency.NewKeyFence()), nil, ranking.ImageRanker(), ranking.ComponentRanker())

	imgWithSignature := fixtures.GetImage()
	imgWithoutSignature := fixtures.GetImageWithUniqueComponents(10)

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	const imgWithSignatureID = "sha256:image-with-signature"
	const imgWithoutSignatureID = "sha256:image-without-signature"

	imgWithSignature.Id = imgWithSignatureID
	imgWithSignature.Signature = &storage.ImageSignature{
		Fetched: protocompat.ConvertTimeToTimestampOrNil(&oneHourAgo),
	}
	imgWithoutSignature.Id = imgWithoutSignatureID

	require.NoError(t, imageDS.UpsertImage(testCtx, imgWithSignature))
	require.NoError(t, imageDS.UpsertImage(testCtx, imgWithoutSignature))

	results, err := imageDS.Search(testCtx, imagesWithSignaturesQuery)
	assert.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, results[0].ID, imgWithSignatureID)

	imgWithSignature.Signature = &storage.ImageSignature{
		Fetched: protocompat.ConvertTimeToTimestampOrNil(&oneMinuteAgo),
	}
	require.NoError(t, imageDS.UpsertImage(testCtx, imgWithSignature))

	results, err = imageDS.Search(testCtx, imagesWithSignaturesQuery)
	assert.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, results[0].ID, imgWithSignatureID)
}
