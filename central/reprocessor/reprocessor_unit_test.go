package reprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	mockDeploymentDataStore "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/deployment/views"
	mockImageDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	mockImageV2DataStore "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	imageEnricher "github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/enricher/mocks"
	nodesEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

var (
	isInvalidateImageCache = gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "InvalidateImageCache"
	})
	isRefreshImageCacheTTL = gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "RefreshImageCacheTtl"
	})
	isUpdatedImage = gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "UpdatedImage"
	})
	isReprocessDeployments = gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "ReprocessDeployments"
	})
	isReprocessDeploymentsSkipFlush = gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "ReprocessDeployments" &&
			msg.GetReprocessDeployments().GetSkipCacheFlush()
	})
	isReprocessDeploymentsDoFlush = gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "ReprocessDeployments" &&
			!msg.GetReprocessDeployments().GetSkipCacheFlush()
	})

	v1ReprocessUpdate = func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.Image) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{ImageUpdated: true}, nil
	}
	v1ReprocessNoUpdate = func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.Image) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{ImageUpdated: false}, nil
	}
	v1ReprocessError = func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.Image) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{}, errors.New("some error")
	}

	v2ReprocessUpdate = func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.ImageV2) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{ImageUpdated: true}, nil
	}
	v2ReprocessNoUpdate = func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.ImageV2) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{ImageUpdated: false}, nil
	}
	v2ReprocessError = func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.ImageV2) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{}, errors.New("some error")
	}
)

// newConnWithCapability creates a mock connection and manager for a single
// cluster "a" that advertises TargetedImageCacheInvalidation.
func newConnWithCapability(ctrl *gomock.Controller) (*connectionMocks.MockSensorConnection, *connectionMocks.MockManager) {
	conn := connectionMocks.NewMockSensorConnection(ctrl)
	mgr := connectionMocks.NewMockManager(ctrl)
	conn.EXPECT().ClusterID().AnyTimes().Return("a")
	conn.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(true)
	mgr.EXPECT().GetConnection("a").AnyTimes().Return(conn)
	mgr.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{conn})
	return conn, mgr
}

func Test_loopImpl_reprocessNode(t *testing.T) {
	type args struct {
		id string
	}
	type mocks struct {
		nodes        *nodeDatastoreMocks.MockDataStore
		risk         *riskManagerMocks.MockManager
		nodeEnricher *nodesEnricherMocks.MockNodeEnricher
	}
	tests := []struct {
		name       string
		args       args
		node       *storage.Node
		want       bool
		setUpMocks func(t *testing.T, a *args, m *mocks)
	}{
		{
			name: "when node is RHCOS then nothing is done",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				node := &storage.Node{
					OsImage: "Red Hat Enterprise Linux CoreOS 412.86.202302091419-0 (Ootpa)",
				}
				m.nodes.EXPECT().GetNode(gomock.Any(), a.id).Return(node, true, nil)
			},
		},
		{
			name: "when node is not RHCOS then scanner is called and node is upserted",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				node := &storage.Node{
					OsImage:     "Something that is not RHCOS",
					LastUpdated: protocompat.TimestampNow(),
				}
				gomock.InOrder(
					m.nodes.EXPECT().GetNode(gomock.Any(), gomock.Eq(a.id)).Times(1).Return(node, true, nil),
					m.nodeEnricher.EXPECT().EnrichNode(node).Times(1).Return(nil),
					m.risk.EXPECT().CalculateRiskAndUpsertNode(gomock.Any()).Return(nil).Times(1),
				)
			},
			want: true,
		},
		{
			name: "when node storage returns err then returns false",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				m.nodes.EXPECT().GetNode(gomock.Any(), a.id).Times(1).Return(nil, false, errors.New("foobar"))
			},
			want: false,
		},
		{
			name: "when node storage is successful but node is not found then returns false",
			setUpMocks: func(t *testing.T, a *args, m *mocks) {
				m.nodes.EXPECT().GetNode(gomock.Any(), a.id).Times(1).Return(nil, false, nil)
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				nodes:        nodeDatastoreMocks.NewMockDataStore(ctrl),
				risk:         riskManagerMocks.NewMockManager(ctrl),
				nodeEnricher: nodesEnricherMocks.NewMockNodeEnricher(ctrl),
			}
			tt.setUpMocks(t, &tt.args, &m)
			l := &loopImpl{
				nodes:        m.nodes,
				risk:         m.risk,
				nodeEnricher: m.nodeEnricher,
			}
			if got := l.reprocessNode(tt.args.id); got != tt.want {
				t.Errorf("reprocessNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReprocessWatchedImageDelegation(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, false)
	t.Run("delegation disabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.DelegateWatchedImageReprocessing, false)

		enrichmentCtx := gomock.Cond(func(ctxRaw any) bool {
			// Ensure that the enrichment isn't delegable.
			ectx := ctxRaw.(imageEnricher.EnrichmentContext)
			return !ectx.Delegable
		})

		ctrl := gomock.NewController(t)
		enricher := mocks.NewMockImageEnricher(ctrl)
		enricher.EXPECT().EnrichImage(emptyCtx, enrichmentCtx, gomock.Any())

		loop := &loopImpl{imageEnricher: enricher}
		loop.reprocessWatchedImage("example.com/repo/path:tag")
	})

	t.Run("delegation enabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.DelegateWatchedImageReprocessing, true)

		ctx := gomock.Cond(func(ctxRaw any) bool {
			// Delegation will fail if context does not have image read access.
			ctx := ctxRaw.(context.Context)
			scopeChecker := sac.GlobalAccessScopeChecker(ctx).
				AccessMode(storage.Access_READ_ACCESS).
				Resource(resources.Image)

			return scopeChecker.IsAllowed()
		})
		enrichmentCtx := gomock.Cond(func(ctxRaw any) bool {
			// The enrichment must be delegable.
			ectx := ctxRaw.(imageEnricher.EnrichmentContext)
			return ectx.Delegable
		})

		ctrl := gomock.NewController(t)
		enricher := mocks.NewMockImageEnricher(ctrl)
		enricher.EXPECT().EnrichImage(ctx, enrichmentCtx, gomock.Any())

		loop := &loopImpl{imageEnricher: enricher}
		loop.reprocessWatchedImage("example.com/repo/path:tag")
	})
}

func TestReprocessWatchedImageV2Delegation(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	t.Run("delegation disabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.DelegateWatchedImageReprocessing, false)

		enrichmentCtx := gomock.Cond(func(ctxRaw any) bool {
			// Ensure that the enrichment isn't delegable.
			ectx := ctxRaw.(imageEnricher.EnrichmentContext)
			return !ectx.Delegable
		})

		ctrl := gomock.NewController(t)
		enricher := mocks.NewMockImageEnricherV2(ctrl)
		enricher.EXPECT().EnrichImage(emptyCtx, enrichmentCtx, gomock.Any())

		loop := &loopImpl{imageEnricherV2: enricher}
		loop.reprocessWatchedImageV2("example.com/repo/path:tag")
	})

	t.Run("delegation enabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.DelegateWatchedImageReprocessing, true)

		ctx := gomock.Cond(func(ctxRaw any) bool {
			// Delegation will fail if context does not have image read access.
			ctx := ctxRaw.(context.Context)
			scopeChecker := sac.GlobalAccessScopeChecker(ctx).
				AccessMode(storage.Access_READ_ACCESS).
				Resource(resources.Image)

			return scopeChecker.IsAllowed()
		})
		enrichmentCtx := gomock.Cond(func(ctxRaw any) bool {
			// The enrichment must be delegable.
			ectx := ctxRaw.(imageEnricher.EnrichmentContext)
			return ectx.Delegable
		})

		ctrl := gomock.NewController(t)
		enricher := mocks.NewMockImageEnricherV2(ctrl)
		enricher.EXPECT().EnrichImage(ctx, enrichmentCtx, gomock.Any())

		loop := &loopImpl{imageEnricherV2: enricher}
		loop.reprocessWatchedImageV2("example.com/repo/path:tag")
	})
}

func TestReprocessImage(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, false)
	newTestLoop := func(tt *testing.T) (*loopImpl, *mockImageDataStore.MockDataStore, *riskManagerMocks.MockManager) {
		ctrl := gomock.NewController(t)
		imageDS := mockImageDataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)
		testLoop := &loopImpl{
			images: imageDS,
			risk:   riskManager,
		}
		return testLoop, imageDS, riskManager
	}
	imageID := "id"
	t.Run("error retrieving the image", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, errors.New("some error"))
		image, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("image does not exist", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil)
		image, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("image is not pullable", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.Image{NotPullable: true}, true, nil)
		image, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("image is cluster local", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.Image{IsClusterLocal: true}, true, nil)
		image, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("reprocessingFunc error", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.Image{}, true, nil)
		image, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, v1ReprocessError)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("reprocessingFunc update calculate risk and upsert error", func(tt *testing.T) {
		testLoop, imageDS, riskManager := newTestLoop(tt)
		image := &storage.Image{}
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil)
		riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Eq(image)).Times(1).Return(errors.New("some error"))
		retImage, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, v1ReprocessUpdate)
		assert.Nil(tt, retImage)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("re-fetch error", func(tt *testing.T) {
		testLoop, imageDS, riskManager := newTestLoop(tt)
		image := &storage.Image{}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil),
			riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Eq(image)).Times(1).Return(nil),
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, errors.New("some error")),
		)
		retImage, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, v1ReprocessUpdate)
		assert.Nil(tt, retImage)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("re-fetch not found", func(tt *testing.T) {
		testLoop, imageDS, riskManager := newTestLoop(tt)
		image := &storage.Image{}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil),
			riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Eq(image)).Times(1).Return(nil),
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
		)
		retImage, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, v1ReprocessUpdate)
		assert.Nil(tt, retImage)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("re-fetch image", func(tt *testing.T) {
		testLoop, imageDS, riskManager := newTestLoop(tt)
		initialImage := &storage.Image{}
		secondImage := &storage.Image{Scan: &storage.ImageScan{}}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(initialImage, true, nil),
			riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Eq(initialImage)).Times(1).Return(nil),
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(secondImage, true, nil),
		)
		retImage, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, v1ReprocessUpdate)
		assert.NotNil(tt, retImage)
		assert.False(tt, proto.Equal(initialImage, retImage))
		assert.True(tt, proto.Equal(secondImage, retImage))
		assert.True(tt, reprocessed)
		assert.True(tt, updated)
	})
	t.Run("reprocessingFunc no scan update", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		image := &storage.Image{}
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil)
		retImage, reprocessed, updated := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, v1ReprocessNoUpdate)
		assert.NotNil(tt, retImage)
		assert.True(tt, proto.Equal(image, retImage))
		assert.True(tt, reprocessed)
		assert.False(tt, updated)
	})
}

func TestReprocessImageV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	newTestLoop := func(tt *testing.T) (*loopImpl, *mockImageV2DataStore.MockDataStore, *mockImageDataStore.MockDataStore, *riskManagerMocks.MockManager) {
		ctrl := gomock.NewController(t)
		imageDS := mockImageV2DataStore.NewMockDataStore(ctrl)
		legacyImageDS := mockImageDataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)
		testLoop := &loopImpl{
			imagesV2: imageDS,
			images:   legacyImageDS,
			risk:     riskManager,
		}
		return testLoop, imageDS, legacyImageDS, riskManager
	}
	imageID := "id"
	imageDigest := "sha256:test"
	imageName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		FullName: "docker.io/library/nginx:latest",
	}
	ref := imageRef{id: imageID, digest: imageDigest, name: imageName}
	t.Run("error retrieving the image", func(tt *testing.T) {
		testLoop, imageDS, _, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, errors.New("some error"))
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("image does not exist in V2 or legacy store", func(tt *testing.T) {
		testLoop, imageDS, legacyImageDS, _ := newTestLoop(tt)
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
			legacyImageDS.EXPECT().GetImageMetadata(gomock.Any(), gomock.Eq(imageDigest)).Times(1).Return(nil, false, nil),
		)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("image does not exist in V2 and error fetching from legacy store", func(tt *testing.T) {
		testLoop, imageDS, legacyImageDS, _ := newTestLoop(tt)
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
			legacyImageDS.EXPECT().GetImageMetadata(gomock.Any(), gomock.Eq(imageDigest)).Times(1).Return(nil, false, errors.New("some error")),
		)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("image is not pullable", func(tt *testing.T) {
		testLoop, imageDS, _, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.ImageV2{NotPullable: true}, true, nil)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("image is cluster local", func(tt *testing.T) {
		testLoop, imageDS, _, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.ImageV2{IsClusterLocal: true}, true, nil)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("legacy image not pullable triggers migration", func(tt *testing.T) {
		testLoop, imageDS, legacyImageDS, _ := newTestLoop(tt)
		legacyImage := &storage.Image{Id: imageDigest, NotPullable: true}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
			legacyImageDS.EXPECT().GetImageMetadata(gomock.Any(), gomock.Eq(imageDigest)).Times(1).Return(legacyImage, true, nil),
			imageDS.EXPECT().UpsertImage(gomock.Any(), gomock.Any()).Times(1).Return(nil),
		)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("legacy image cluster local triggers migration", func(tt *testing.T) {
		testLoop, imageDS, legacyImageDS, _ := newTestLoop(tt)
		legacyImage := &storage.Image{Id: imageDigest, IsClusterLocal: true}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
			legacyImageDS.EXPECT().GetImageMetadata(gomock.Any(), gomock.Eq(imageDigest)).Times(1).Return(legacyImage, true, nil),
			imageDS.EXPECT().UpsertImage(gomock.Any(), gomock.Any()).Times(1).Return(nil),
		)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("legacy image migration upsert error", func(tt *testing.T) {
		testLoop, imageDS, legacyImageDS, _ := newTestLoop(tt)
		legacyImage := &storage.Image{Id: imageDigest, NotPullable: true}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
			legacyImageDS.EXPECT().GetImageMetadata(gomock.Any(), gomock.Eq(imageDigest)).Times(1).Return(legacyImage, true, nil),
			imageDS.EXPECT().UpsertImage(gomock.Any(), gomock.Any()).Times(1).Return(errors.New("upsert error")),
		)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("reprocessingFunc error", func(tt *testing.T) {
		testLoop, imageDS, _, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.ImageV2{}, true, nil)
		image, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, v2ReprocessError)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("reprocessingFunc update calculate risk and upsert error", func(tt *testing.T) {
		testLoop, imageDS, _, riskManager := newTestLoop(tt)
		image := &storage.ImageV2{}
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil)
		riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Eq(image)).Times(1).Return(errors.New("some error"))
		retImage, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, v2ReprocessUpdate)
		assert.Nil(tt, retImage)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("re-fetch error", func(tt *testing.T) {
		testLoop, imageDS, _, riskManager := newTestLoop(tt)
		image := &storage.ImageV2{}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil),
			riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Eq(image)).Times(1).Return(nil),
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, errors.New("some error")),
		)
		retImage, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, v2ReprocessUpdate)
		assert.Nil(tt, retImage)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("re-fetch not found", func(tt *testing.T) {
		testLoop, imageDS, _, riskManager := newTestLoop(tt)
		image := &storage.ImageV2{}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil),
			riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Eq(image)).Times(1).Return(nil),
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
		)
		retImage, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, v2ReprocessUpdate)
		assert.Nil(tt, retImage)
		assert.False(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("re-fetch image", func(tt *testing.T) {
		testLoop, imageDS, _, riskManager := newTestLoop(tt)
		initialImage := &storage.ImageV2{}
		secondImage := &storage.ImageV2{Scan: &storage.ImageScan{}}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(initialImage, true, nil),
			riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Eq(initialImage)).Times(1).Return(nil),
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(secondImage, true, nil),
		)
		retImage, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, v2ReprocessUpdate)
		assert.NotNil(tt, retImage)
		assert.False(tt, proto.Equal(initialImage, retImage))
		assert.True(tt, proto.Equal(secondImage, retImage))
		assert.True(tt, reprocessed)
		assert.True(tt, updated)
	})
	t.Run("reprocessingFunc no scan update", func(tt *testing.T) {
		testLoop, imageDS, _, _ := newTestLoop(tt)
		image := &storage.ImageV2{}
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil)
		retImage, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, v2ReprocessNoUpdate)
		assert.NotNil(tt, retImage)
		assert.True(tt, proto.Equal(image, retImage))
		assert.True(tt, reprocessed)
		assert.False(tt, updated)
	})
	t.Run("legacy image found and successfully reprocessed", func(tt *testing.T) {
		testLoop, imageDS, legacyImageDS, riskManager := newTestLoop(tt)
		legacyImage := &storage.Image{Id: imageDigest}
		secondImage := &storage.ImageV2{Id: imageID, Scan: &storage.ImageScan{}}
		gomock.InOrder(
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil),
			legacyImageDS.EXPECT().GetImageMetadata(gomock.Any(), gomock.Eq(imageDigest)).Times(1).Return(legacyImage, true, nil),
			riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Any()).Times(1).Return(nil),
			imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(secondImage, true, nil),
		)
		retImage, reprocessed, updated := testLoop.reprocessImageV2(ref, imageEnricher.UseCachesIfPossible, v2ReprocessUpdate)
		assert.NotNil(tt, retImage)
		assert.True(tt, proto.Equal(secondImage, retImage))
		assert.True(tt, reprocessed)
		assert.True(tt, updated)
	})
}

func TestReprocessImagesAndResyncDeployments_SkipBrokenSensor(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, false)
	imgs := []*storage.Image{}
	for _, cluster := range []string{"a", "b"} { // two clusters
		// Create at least one more image than max semaphore size to ensure skip logic is executed.
		for i := range imageReprocessorSemaphoreSize + 1 {
			imgs = append(imgs, &storage.Image{Id: fmt.Sprintf("img%d-%s", i, cluster)})
		}
	}

	results := []search.Result{}
	for _, img := range imgs {
		results = append(results, search.Result{
			ID: img.GetId(),
			Matches: map[string][]string{
				// Last character of image ID is the cluster.
				imageClusterIDFieldPath: {img.GetId()[len(img.GetId())-1:]},
			}},
		)
	}

	newReprocessorLoop := func(t *testing.T) (*loopImpl, *connectionMocks.MockSensorConnection, *connectionMocks.MockSensorConnection) {
		ctrl := gomock.NewController(t)

		connA := connectionMocks.NewMockSensorConnection(ctrl)
		connB := connectionMocks.NewMockSensorConnection(ctrl)
		connManager := connectionMocks.NewMockManager(ctrl)
		imageDS := mockImageDataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)

		connA.EXPECT().ClusterID().AnyTimes().Return("a")
		connB.EXPECT().ClusterID().AnyTimes().Return("b")

		connManager.EXPECT().GetConnection("a").AnyTimes().Return(connA)
		connManager.EXPECT().GetConnection("b").AnyTimes().Return(connB)
		connManager.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{connA, connB})

		imageDS.EXPECT().Search(gomock.Any(), gomock.Any()).AnyTimes().Return(results, nil)
		for _, img := range imgs {
			imageDS.EXPECT().GetImage(gomock.Any(), img.GetId()).AnyTimes().Return(img, true, nil)
		}

		riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Any()).AnyTimes().Return(nil)

		testLoop := &loopImpl{
			images:      imageDS,
			risk:        riskManager,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		return testLoop, connA, connB
	}

	t.Run("send all messages when clusters are healthy", func(t *testing.T) {
		testLoop, connA, connB := newReprocessorLoop(t)

		connA.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)
		connB.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)

		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(len(imgs) / 2).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(len(imgs) / 2).Return(nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(1).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(1).Return(nil)

		testLoop.reprocessImagesAndResyncDeployments(0, v1ReprocessUpdate, nil)
	})

	t.Run("skip some messages when are broken clusters", func(t *testing.T) {
		testLoop, connA, connB := newReprocessorLoop(t)

		connA.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)
		connB.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)

		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(len(imgs) / 2).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).MaxTimes(int(imageReprocessorSemaphoreSize)).Return(errors.New("broken"))

		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(1).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(0).Return(nil)

		testLoop.reprocessImagesAndResyncDeployments(0, v1ReprocessUpdate, nil)
	})
}

func TestReprocessImagesV2AndResyncDeployments_SkipBrokenSensor(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	imgs := []*storage.ImageV2{}
	for _, cluster := range []string{"a", "b"} { // two clusters
		// Create at least one more image than max semaphore size to ensure skip logic is executed.
		for i := range imageReprocessorSemaphoreSize + 1 {
			imgs = append(imgs, &storage.ImageV2{Id: fmt.Sprintf("img%d-%s", i, cluster)})
		}
	}

	containerImageViews := []*views.ContainerImageView{}
	for _, img := range imgs {
		containerImageViews = append(containerImageViews, &views.ContainerImageView{
			ImageIDV2:         img.GetId(),
			ImageNameFullName: fmt.Sprintf("docker.io/library/%s:latest", img.GetId()),
			// Last character of image ID is the cluster.
			ClusterIDs: []string{img.GetId()[len(img.GetId())-1:]},
		})
	}

	newReprocessorLoop := func(t *testing.T) (*loopImpl, *connectionMocks.MockSensorConnection, *connectionMocks.MockSensorConnection) {
		ctrl := gomock.NewController(t)

		connA := connectionMocks.NewMockSensorConnection(ctrl)
		connB := connectionMocks.NewMockSensorConnection(ctrl)
		connManager := connectionMocks.NewMockManager(ctrl)
		deploymentDS := mockDeploymentDataStore.NewMockDataStore(ctrl)
		imageDS := mockImageV2DataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)

		connA.EXPECT().ClusterID().AnyTimes().Return("a")
		connB.EXPECT().ClusterID().AnyTimes().Return("b")

		connManager.EXPECT().GetConnection("a").AnyTimes().Return(connA)
		connManager.EXPECT().GetConnection("b").AnyTimes().Return(connB)
		connManager.EXPECT().GetActiveConnections().Return([]connection.SensorConnection{connA, connB})
		connManager.EXPECT().AllSensorsHaveCapability(gomock.Any()).AnyTimes().Return(false)

		deploymentDS.EXPECT().GetContainerImageViews(gomock.Any(), gomock.Any()).AnyTimes().Return(containerImageViews, nil)
		for _, img := range imgs {
			imageDS.EXPECT().GetImage(gomock.Any(), img.GetId()).AnyTimes().Return(img, true, nil)
			imageDS.EXPECT().GetImageNames(gomock.Any(), img.GetDigest()).
				AnyTimes().
				Return([]*storage.ImageName{img.GetName()}, nil)
		}

		riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Any()).AnyTimes().Return(nil)

		testLoop := &loopImpl{
			deployments: deploymentDS,
			imagesV2:    imageDS,
			risk:        riskManager,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		return testLoop, connA, connB
	}

	t.Run("send all messages when clusters are healthy", func(t *testing.T) {
		testLoop, connA, connB := newReprocessorLoop(t)

		connA.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)
		connB.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)

		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(len(imgs) / 2).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(len(imgs) / 2).Return(nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(1).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(1).Return(nil)

		testLoop.reprocessImagesV2AndResyncDeployments(0, v2ReprocessUpdate, search.EmptyQuery())
	})

	t.Run("skip some messages when are broken clusters", func(t *testing.T) {
		testLoop, connA, connB := newReprocessorLoop(t)

		connA.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)
		connB.EXPECT().HasCapability(centralsensor.TargetedImageCacheInvalidation).AnyTimes().Return(false)

		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(len(imgs) / 2).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).MaxTimes(int(imageReprocessorSemaphoreSize)).Return(errors.New("broken"))

		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(1).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(0).Return(nil)

		testLoop.reprocessImagesV2AndResyncDeployments(0, v2ReprocessUpdate, search.EmptyQuery())
	})
}

func TestInjectMessage(t *testing.T) {
	ctx := context.Background()
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ReprocessDeployments{
			ReprocessDeployments: &central.ReprocessDeployments{},
		},
	}
	contextWithTimeout := gomock.Cond(func(ctx context.Context) bool {
		_, hasTimeout := ctx.Deadline()
		return hasTimeout
	})
	contextWithoutTimeout := gomock.Cond(func(ctx context.Context) bool {
		_, hasTimeout := ctx.Deadline()
		return !hasTimeout
	})
	t.Run("use timeout when duration set", func(t *testing.T) {
		testLoop := &loopImpl{
			injectMessageTimeoutDur: 1 * time.Millisecond, // Can be anything non-zero
		}

		ctrl := gomock.NewController(t)
		conn := connectionMocks.NewMockSensorConnection(ctrl)

		// Validate created context has a timeout.
		conn.EXPECT().InjectMessage(contextWithTimeout, gomock.Any()).Return(nil)

		err := testLoop.injectMessage(ctx, conn, msg)
		require.NoError(t, err)
	})

	t.Run("no timeout when duration zero", func(t *testing.T) {
		testLoop := &loopImpl{
			injectMessageTimeoutDur: 0,
		}

		ctrl := gomock.NewController(t)
		conn := connectionMocks.NewMockSensorConnection(ctrl)

		// Validate created context DOES NOT have a timeout.
		conn.EXPECT().InjectMessage(contextWithoutTimeout, gomock.Any()).Return(nil)

		err := testLoop.injectMessage(ctx, conn, msg)
		require.NoError(t, err)
	})
}

func TestReprocessImagesAndResyncDeployments_WithCapability(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, false)

	imgUpdated := &storage.Image{Id: "img-updated-a"}
	imgUnchanged := &storage.Image{Id: "img-unchanged-a"}

	resultsUpdated := search.Result{
		ID:      imgUpdated.GetId(),
		Matches: map[string][]string{imageClusterIDFieldPath: {"a"}},
	}
	resultsUnchanged := search.Result{
		ID:      imgUnchanged.GetId(),
		Matches: map[string][]string{imageClusterIDFieldPath: {"a"}},
	}

	t.Run("updated image sends UpdatedImage only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connA, connManager := newConnWithCapability(ctrl)
		imageDS := mockImageDataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)

		imageDS.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{resultsUpdated}, nil)
		imageDS.EXPECT().GetImage(gomock.Any(), imgUpdated.GetId()).AnyTimes().Return(imgUpdated, true, nil)
		riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Any()).AnyTimes().Return(nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isInvalidateImageCache).Times(0)
		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(1).Return(nil)
		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeploymentsSkipFlush).Times(1).Return(nil)

		testLoop := &loopImpl{
			images:      imageDS,
			risk:        riskManager,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		testLoop.reprocessImagesAndResyncDeployments(imageEnricher.UseCachesIfPossible, v1ReprocessUpdate, nil)
	})

	t.Run("unchanged image sends RefreshImageCacheTTL only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connA, connManager := newConnWithCapability(ctrl)
		imageDS := mockImageDataStore.NewMockDataStore(ctrl)

		imageDS.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{resultsUnchanged}, nil)
		imageDS.EXPECT().GetImage(gomock.Any(), imgUnchanged.GetId()).AnyTimes().Return(imgUnchanged, true, nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isRefreshImageCacheTTL).Times(1).Return(nil)
		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(0)
		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeploymentsSkipFlush).Times(1).Return(nil)

		testLoop := &loopImpl{
			images:      imageDS,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		testLoop.reprocessImagesAndResyncDeployments(imageEnricher.UseCachesIfPossible, v1ReprocessNoUpdate, nil)
	})

	t.Run("periodic path skips InvalidateImageCache and flushes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connA, connManager := newConnWithCapability(ctrl)
		imageDS := mockImageDataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)

		imageDS.EXPECT().Search(gomock.Any(), gomock.Any()).Return([]search.Result{resultsUpdated}, nil)
		imageDS.EXPECT().GetImage(gomock.Any(), imgUpdated.GetId()).AnyTimes().Return(imgUpdated, true, nil)
		riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Any()).AnyTimes().Return(nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isInvalidateImageCache).Times(0)
		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(1).Return(nil)
		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeploymentsDoFlush).Times(1).Return(nil)

		testLoop := &loopImpl{
			images:      imageDS,
			risk:        riskManager,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		testLoop.reprocessImagesAndResyncDeployments(imageEnricher.ForceRefetchCachedValuesOnly, v1ReprocessUpdate, nil)
	})

	t.Run("broken cluster with capability skips after first error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connA, connManager := newConnWithCapability(ctrl)
		imageDS := mockImageDataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)

		imgs := []*storage.Image{}
		results := []search.Result{}
		for i := range imageReprocessorSemaphoreSize + 1 {
			img := &storage.Image{Id: fmt.Sprintf("img%d-a", i)}
			imgs = append(imgs, img)
			results = append(results, search.Result{
				ID:      img.GetId(),
				Matches: map[string][]string{imageClusterIDFieldPath: {"a"}},
			})
		}

		imageDS.EXPECT().Search(gomock.Any(), gomock.Any()).Return(results, nil)
		for _, img := range imgs {
			imageDS.EXPECT().GetImage(gomock.Any(), img.GetId()).AnyTimes().Return(img, true, nil)
		}
		riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Any()).AnyTimes().Return(nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).MaxTimes(int(imageReprocessorSemaphoreSize)).Return(errors.New("broken"))
		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeployments).Times(0)

		testLoop := &loopImpl{
			images:      imageDS,
			risk:        riskManager,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		testLoop.reprocessImagesAndResyncDeployments(0, v1ReprocessUpdate, nil)
	})
}

func TestReprocessImagesV2AndResyncDeployments_WithCapability(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)

	imgUpdated := &storage.ImageV2{Id: "img-updated-a"}
	imgUnchanged := &storage.ImageV2{Id: "img-unchanged-a"}

	viewUpdated := &views.ContainerImageView{
		ImageIDV2:  imgUpdated.GetId(),
		ClusterIDs: []string{"a"},
	}
	viewUnchanged := &views.ContainerImageView{
		ImageIDV2:  imgUnchanged.GetId(),
		ClusterIDs: []string{"a"},
	}

	t.Run("updated image sends UpdatedImage only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connA, connManager := newConnWithCapability(ctrl)
		connManager.EXPECT().AllSensorsHaveCapability(gomock.Any()).AnyTimes().Return(false)
		deploymentDS := mockDeploymentDataStore.NewMockDataStore(ctrl)
		imageDS := mockImageV2DataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)

		deploymentDS.EXPECT().GetContainerImageViews(gomock.Any(), gomock.Any()).Return([]*views.ContainerImageView{viewUpdated}, nil)
		imageDS.EXPECT().GetImage(gomock.Any(), imgUpdated.GetId()).AnyTimes().Return(imgUpdated, true, nil)
		imageDS.EXPECT().GetImageNames(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageName{imgUpdated.GetName()}, nil)
		riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Any()).AnyTimes().Return(nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isInvalidateImageCache).Times(0)
		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(1).Return(nil)
		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeploymentsSkipFlush).Times(1).Return(nil)

		testLoop := &loopImpl{
			deployments: deploymentDS,
			imagesV2:    imageDS,
			risk:        riskManager,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		testLoop.reprocessImagesV2AndResyncDeployments(imageEnricher.UseCachesIfPossible, v2ReprocessUpdate, search.EmptyQuery())
	})

	t.Run("unchanged image sends RefreshImageCacheTTL only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connA, connManager := newConnWithCapability(ctrl)
		connManager.EXPECT().AllSensorsHaveCapability(gomock.Any()).AnyTimes().Return(false)
		deploymentDS := mockDeploymentDataStore.NewMockDataStore(ctrl)
		imageDS := mockImageV2DataStore.NewMockDataStore(ctrl)

		deploymentDS.EXPECT().GetContainerImageViews(gomock.Any(), gomock.Any()).Return([]*views.ContainerImageView{viewUnchanged}, nil)
		imageDS.EXPECT().GetImage(gomock.Any(), imgUnchanged.GetId()).AnyTimes().Return(imgUnchanged, true, nil)
		imageDS.EXPECT().GetImageNames(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageName{imgUnchanged.GetName()}, nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isRefreshImageCacheTTL).Times(1).Return(nil)
		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(0)
		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeploymentsSkipFlush).Times(1).Return(nil)

		testLoop := &loopImpl{
			deployments: deploymentDS,
			imagesV2:    imageDS,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		testLoop.reprocessImagesV2AndResyncDeployments(imageEnricher.UseCachesIfPossible, v2ReprocessNoUpdate, search.EmptyQuery())
	})

	t.Run("periodic path skips InvalidateImageCache and flushes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		connA, connManager := newConnWithCapability(ctrl)
		connManager.EXPECT().AllSensorsHaveCapability(gomock.Any()).AnyTimes().Return(false)
		deploymentDS := mockDeploymentDataStore.NewMockDataStore(ctrl)
		imageDS := mockImageV2DataStore.NewMockDataStore(ctrl)
		riskManager := riskManagerMocks.NewMockManager(ctrl)

		deploymentDS.EXPECT().GetContainerImageViews(gomock.Any(), gomock.Any()).Return([]*views.ContainerImageView{viewUpdated}, nil)
		imageDS.EXPECT().GetImage(gomock.Any(), imgUpdated.GetId()).AnyTimes().Return(imgUpdated, true, nil)
		imageDS.EXPECT().GetImageNames(gomock.Any(), gomock.Any()).AnyTimes().Return([]*storage.ImageName{imgUpdated.GetName()}, nil)
		riskManager.EXPECT().CalculateRiskAndUpsertImageV2(gomock.Any()).AnyTimes().Return(nil)

		connA.EXPECT().InjectMessage(gomock.Any(), isInvalidateImageCache).Times(0)
		connA.EXPECT().InjectMessage(gomock.Any(), isUpdatedImage).Times(1).Return(nil)
		connA.EXPECT().InjectMessage(gomock.Any(), isReprocessDeploymentsDoFlush).Times(1).Return(nil)

		testLoop := &loopImpl{
			deployments: deploymentDS,
			imagesV2:    imageDS,
			risk:        riskManager,
			connManager: connManager,
			stopSig:     concurrency.NewSignal(),
		}
		testLoop.reprocessImagesV2AndResyncDeployments(imageEnricher.ForceRefetchCachedValuesOnly, v2ReprocessUpdate, search.EmptyQuery())
	})
}
