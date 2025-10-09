//go:build sql_integration

package reprocessor

import (
	"context"
	"testing"
	"time"

	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imagePG "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imagePostgresV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImagesWithSignaturesQuery(t *testing.T) {
	testCtx := sac.WithAllAccess(context.Background())

	testingDB := pgtest.ForT(t)
	pool := testingDB.DB
	defer pool.Close()

	var imageDS imageDatastore.DataStore
	if features.FlattenCVEData.Enabled() {
		imageDS = imageDatastore.NewWithPostgres(imagePostgresV2.New(pool, false, concurrency.NewKeyFence()), nil, ranking.ImageRanker(), ranking.ComponentRanker())
	} else {
		imageDS = imageDatastore.NewWithPostgres(imagePG.New(pool, false, concurrency.NewKeyFence()), nil, ranking.ImageRanker(), ranking.ComponentRanker())
	}

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
