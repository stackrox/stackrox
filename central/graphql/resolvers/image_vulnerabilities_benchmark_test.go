//go:build sql_integration

package resolvers

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func BenchmarkImageVulnerabilities(b *testing.B) {
	if !features.FlattenImageData.Enabled() {
		b.Skip("only applicable when FlattenImageData is enabled")
	}

	mockCtrl := gomock.NewController(b)
	testDB := SetupTestPostgresConn(b)

	ctx := SetAuthorizerOverride(
		loaders.WithLoaderContext(sac.WithAllAccess(context.Background())),
		allow.Anonymous(),
	)

	v1Store := CreateTestImageDatastore(b, testDB, mockCtrl)
	v2Store := CreateTestImageV2Datastore(b, testDB, mockCtrl)

	resolver, _ := SetupTestResolver(b,
		imagesView.NewImageView(testDB.DB),
		CreateTestImageComponentV2Datastore(b, testDB, mockCtrl),
		CreateTestImageCVEV2Datastore(b, testDB),
		v2Store,
		imagecveflat.NewCVEFlatView(testDB.DB),
		imagecomponentflat.NewComponentFlatView(testDB.DB),
	)

	paginatedQuery := PaginatedQuery{
		Pagination: &inputtypes.Pagination{
			Limit: pointers.Int32(20000),
		},
	}

	imageCount := 100

	seedV1AndV2Images(b, ctx, v1Store, v2Store, imageCount)

	b.Run(fmt.Sprintf("ImageVulnerabilities/images=%d", imageCount), func(b *testing.B) {
		for range b.N {
			vulns, err := resolver.ImageVulnerabilities(ctx, paginatedQuery)
			require.NoError(b, err)
			require.NotEmpty(b, vulns)
		}
	})

	truncateImageBenchData(b, ctx, testDB)
}

func seedV1AndV2Images(
	b *testing.B,
	ctx context.Context,
	v1Store imageDS.DataStore,
	v2Store imageV2DS.DataStore,
	imageCount int,
) {
	b.Helper()

	v1Images := make([]*storage.Image, 0, imageCount)
	for i := range imageCount {
		v1Image := fixtures.GetImageWithUniqueComponents(100)
		v1Image.Id = fmt.Sprintf("sha256:%s", uuid.NewV4().String())
		v1Image.Name = &storage.ImageName{
			Registry: "bench.io",
			Remote:   fmt.Sprintf("bench/img-%d", i),
			Tag:      "latest",
			FullName: fmt.Sprintf("bench.io/bench/img-%d:latest", i),
		}
		for _, comp := range v1Image.GetScan().GetComponents() {
			comp.Name = fmt.Sprintf("img%d-%s", i, comp.GetName())
			for _, vuln := range comp.GetVulns() {
				vuln.Cve = fmt.Sprintf("img%d-%s", i, vuln.GetCve())
			}
		}
		v1Images = append(v1Images, v1Image)
	}

	require.NoError(b, os.Setenv(features.FlattenImageData.EnvVar(), "false"))
	for _, v1Image := range v1Images {
		require.NoError(b, v1Store.UpsertImage(ctx, v1Image))
	}

	require.NoError(b, os.Setenv(features.FlattenImageData.EnvVar(), "true"))
	for _, v1Image := range v1Images {
		v2Image := imageUtils.ConvertToV2(v1Image)
		require.NoError(b, v2Store.UpsertImage(ctx, v2Image))
	}
}

func truncateImageBenchData(b *testing.B, ctx context.Context, testDB *pgtest.TestPostgres) {
	b.Helper()
	_, err := testDB.DB.Exec(ctx, "TRUNCATE images, images_v2, image_cves_v2, image_component_v2 CASCADE")
	require.NoError(b, err)
}
