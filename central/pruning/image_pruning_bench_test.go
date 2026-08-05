//go:build sql_integration

package pruning

import (
	"context"
	"fmt"
	"testing"
	"time"

	configDSMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	imageV2Postgres "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	podDS "github.com/stackrox/rox/central/pod/datastore"
	processBaselineDSMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	processIndicatorDSMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	plopDSMocks "github.com/stackrox/rox/central/processlisteningonport/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskDSMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	filterMocks "github.com/stackrox/rox/pkg/process/filter/mocks"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newImage(prefix string, index int, timestamp *timestamppb.Timestamp) *storage.ImageV2 {
	digest := types.NewDigest(fmt.Sprintf("%s-%d", prefix, index)).Digest()
	name := &storage.ImageName{FullName: fmt.Sprintf("ghcr.io/bench/%s-%d:latest@%s", prefix, index, digest)}
	return &storage.ImageV2{
		Id:          utils.NewImageV2ID(name, digest),
		Digest:      digest,
		Name:        name,
		LastUpdated: timestamp,
	}
}

func setupBenchmarkTest(b *testing.B, pool postgres.DB) (
	imageV2DS.DataStore, deploymentDS.DataStore, podDS.DataStore, *garbageCollectorImpl,
) {

	ctrl := gomock.NewController(b)
	any := gomock.Any()

	mockRiskDS := riskDSMocks.NewMockDataStore(ctrl)
	mockRiskDS.EXPECT().RemoveRisk(any, any, any).AnyTimes()

	mockProcessIndicatorDS := processIndicatorDSMocks.NewMockDataStore(ctrl)
	mockProcessIndicatorDS.EXPECT().RemoveProcessIndicatorsByPod(any, any).AnyTimes().Return(nil)

	mockPlopDS := plopDSMocks.NewMockDataStore(ctrl)
	mockPlopDS.EXPECT().RemovePlopsByPod(any, any).AnyTimes().Return(nil)

	mockConfigDS := configDSMocks.NewMockDataStore(ctrl)
	mockConfigDS.EXPECT().GetPrivateConfig(any).AnyTimes().Return(testConfig.GetPrivateConfig(), nil)

	mockFilter := filterMocks.NewMockFilter(ctrl)
	mockFilter.EXPECT().UpdateByPod(any).AnyTimes()
	mockFilter.EXPECT().DeleteByPod(any).AnyTimes()

	deployments, err := deploymentDS.New(
		pool, nil, nil,
		processBaselineDSMocks.NewMockDataStore(ctrl),
		nil, mockRiskDS, nil, mockFilter,
		ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker(),
		platformmatcher.GetTestPlatformMatcherWithDefaultPlatformComponentConfig(ctrl),
	)
	require.NoError(b, err)

	imagesV2 := imageV2DS.NewWithPostgres(
		imageV2Postgres.New(pool, true, concurrency.NewKeyFence()),
		mockRiskDS,
		ranking.ImageRanker(),
		ranking.ComponentRanker(),
	)

	pods := podDS.NewPostgresDB(pool, mockProcessIndicatorDS, mockPlopDS, mockFilter)

	gc := newGarbageCollector(nil, nil, nil, imagesV2, nil, deployments, pods,
		nil, nil, nil, mockConfigDS, nil,
		nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil).(*garbageCollectorImpl)
	gc.postgres = pool

	return imagesV2, deployments, pods, gc
}

func BenchmarkImagePruning(b *testing.B) {
	if !features.FlattenImageData.Enabled() {
		b.Skip("only applicable when FlattenImageData is enabled")
	}

	ctx := sac.WithAllAccess(context.Background())
	testingDB := pgtest.ForT(b)
	defer testingDB.DB.Close()
	pool := testingDB.DB

	imagesV2, deployments, pods, gc := setupBenchmarkTest(b, pool)

	retentionDays := testConfig.GetPrivateConfig().GetImageRetentionDurationDays()
	privateConfig := testConfig.GetPrivateConfig()

	for _, imageCount := range []int{50, 200, 500} {
		b.Run(fmt.Sprintf("images=%d", imageCount), func(b *testing.B) {
			for range b.N {
				b.StopTimer()
				cleanupBenchmarkData(b, ctx, pool)
				seedBenchmarkData(b, ctx, imagesV2, deployments, pods, imageCount, retentionDays)
				b.StartTimer()

				gc.collectImages(privateConfig)
			}
			b.StopTimer()
			cleanupBenchmarkData(b, ctx, pool)
		})
	}
}

// seedBenchmarkData populates the database with test images, deployments, and pods.
//
// Distribution:
//   - 50% of images: referenced by a deployment -- should NOT be pruned
//   - 20% of images: inactive with old timestamp (past retention) -- candidates for pruning
//   - 20% of images: inactive with new timestamp (within retention) -- should NOT be pruned
//   - 10% of images: inactive with old timestamp but have a stuck pod (deployment updated
//     to different digest, pod still running old digest) -- should NOT be pruned
func seedBenchmarkData(
	b *testing.B,
	ctx context.Context,
	imagesV2 imageV2DS.DataStore,
	deployments deploymentDS.DataStore,
	pods podDS.DataStore,
	imageCount int,
	retentionDays int32,
) {
	oldTimestamp := protoconv.ConvertTimeToTimestamp(
		time.Now().Add(-24 * time.Duration(retentionDays+1) * time.Hour),
	)
	newTimestamp := protoconv.ConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour))

	deployedCount := imageCount * 50 / 100                                            // active images referenced by a deployment
	inactiveOldCount := imageCount * 20 / 100                                         // inactive images past retention window
	inactiveNewCount := imageCount * 20 / 100                                         // inactive images within retention window
	stuckPodCount := imageCount - deployedCount - inactiveOldCount - inactiveNewCount // inactive images (old timestamp) with a stuck pod

	// Group 1: Images referenced by deployments
	for i := range deployedCount {
		img := newImage("deployed", i, oldTimestamp)
		require.NoError(b, imagesV2.UpsertImage(ctx, img))

		dep := &storage.Deployment{
			Id:        uuid.NewV4().String(),
			Name:      fmt.Sprintf("dep-%d", i),
			Namespace: "bench",
			Containers: []*storage.Container{
				{
					Image: &storage.ContainerImage{
						Id:   img.GetDigest(),
						IdV2: img.GetId(),
						Name: img.GetName(),
					},
				},
			},
		}
		require.NoError(b, deployments.UpsertDeployment(ctx, dep))
	}

	// Group 2: Inactive images with old timestamp (past retention, no references)
	for i := range inactiveOldCount {
		require.NoError(b, imagesV2.UpsertImage(ctx, newImage("inactive-old", i, oldTimestamp)))
	}

	// Group 3: Inactive images with new timestamp (within retention, no references)
	for i := range inactiveNewCount {
		require.NoError(b, imagesV2.UpsertImage(ctx, newImage("inactive-new", i, newTimestamp)))
	}

	// Group 4: Stuck-pod images. The pod is still running the old image digest.
	// The image should NOT be pruned.
	for i := range stuckPodCount {
		img := newImage("stuck", i, oldTimestamp)
		require.NoError(b, imagesV2.UpsertImage(ctx, img))

		pod := &storage.Pod{
			Id:           uuid.NewV4().String(),
			DeploymentId: uuid.NewV4().String(),
			LiveInstances: []*storage.ContainerInstance{
				{ImageDigest: img.GetDigest()},
			},
		}
		require.NoError(b, pods.UpsertPod(ctx, pod))
	}
}

func cleanupBenchmarkData(b *testing.B, ctx context.Context, pool postgres.DB) {
	_, err := pool.Exec(ctx, "TRUNCATE images_v2, deployments, pods CASCADE")
	require.NoError(b, err)
}
