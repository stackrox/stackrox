//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetImage_MergesAllNamesWhenMultipleV2ImagesShareDigest(t *testing.T) {
	if !features.FlattenImageData.Enabled() {
		t.Setenv("ROX_FLATTEN_IMAGE_DATA", "true")
	}
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)
	ds := GetTestPostgresDataStore(t, testDB.DB).(*datastoreImpl)

	digest := "sha256:abcdef1234567890"

	imageA := &storage.Image{
		Id: digest,
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "repo-a/image",
			Tag:      "latest",
			FullName: "quay.io/repo-a/image:latest",
		},
	}
	imageB := &storage.Image{
		Id: digest,
		Name: &storage.ImageName{
			Registry: "quay.io",
			Remote:   "repo-b/image",
			Tag:      "v1",
			FullName: "quay.io/repo-b/image:v1",
		},
	}

	require.NoError(t, ds.UpsertImage(ctx, imageA))
	require.NoError(t, ds.UpsertImage(ctx, imageB))

	result, found, err := ds.GetImage(ctx, digest)
	require.NoError(t, err)
	require.True(t, found)

	var fullNames []string
	for _, n := range result.GetNames() {
		fullNames = append(fullNames, n.GetFullName())
	}
	assert.Contains(t, fullNames, imageA.GetName().GetFullName())
	assert.Contains(t, fullNames, imageB.GetName().GetFullName())
}
