package reprocessor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	mockImageDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	nodeDatastoreMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	"github.com/stackrox/rox/central/sensor/service/connection"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
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

func TestReprocessImage(t *testing.T) {
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
	reprocessFuncError := func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.Image) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{}, errors.New("some error")
	}
	reprocessFuncUpdate := func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.Image) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{ImageUpdated: true}, nil
	}
	reprocessFuncNoUpdate := func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.Image) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{ImageUpdated: false}, nil
	}
	imageID := "id"
	t.Run("error retrieving the image", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, errors.New("some error"))
		image, reprocessed := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
	})
	t.Run("image does not exist", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(nil, false, nil)
		image, reprocessed := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
	})
	t.Run("image is not pullable", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.Image{NotPullable: true}, true, nil)
		image, reprocessed := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
	})
	t.Run("image is cluster local", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.Image{IsClusterLocal: true}, true, nil)
		image, reprocessed := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, nil)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
	})
	t.Run("reprocessingFunc error", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(&storage.Image{}, true, nil)
		image, reprocessed := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, reprocessFuncError)
		assert.Nil(tt, image)
		assert.False(tt, reprocessed)
	})
	t.Run("reprocessingFunc update calculate risk and upsert error", func(tt *testing.T) {
		testLoop, imageDS, riskManager := newTestLoop(tt)
		image := &storage.Image{}
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil)
		riskManager.EXPECT().CalculateRiskAndUpsertImage(gomock.Eq(image)).Times(1).Return(errors.New("some error"))
		retImage, reprocessed := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, reprocessFuncUpdate)
		assert.Nil(tt, retImage)
		assert.False(tt, reprocessed)
	})
	t.Run("reprocessingFunc no scan update", func(tt *testing.T) {
		testLoop, imageDS, _ := newTestLoop(tt)
		image := &storage.Image{}
		imageDS.EXPECT().GetImage(gomock.Any(), gomock.Eq(imageID)).Times(1).Return(image, true, nil)
		retImage, reprocessed := testLoop.reprocessImage(imageID, imageEnricher.UseCachesIfPossible, reprocessFuncNoUpdate)
		assert.NotNil(tt, retImage)
		assert.True(tt, proto.Equal(image, retImage))
		assert.True(tt, reprocessed)
	})
}

func TestReprocessImagesAndResyncDeployments_SkipBrokenSensor(t *testing.T) {
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
			ID: img.Id,
			Matches: map[string][]string{
				// Last character of image ID is the cluster.
				imageClusterIDFieldPath: {img.Id[len(img.Id)-1:]},
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
			imageDS.EXPECT().GetImage(gomock.Any(), img.Id).AnyTimes().Return(img, true, nil)
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

	reprocessFuncUpdate := func(_ context.Context, _ imageEnricher.EnrichmentContext, _ *storage.Image) (imageEnricher.EnrichmentResult, error) {
		return imageEnricher.EnrichmentResult{ImageUpdated: true}, nil
	}
	updatedImageTypeCond := gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "UpdatedImage"
	})
	reprocessDeploymentsTypeCond := gomock.Cond(func(msg *central.MsgToSensor) bool {
		return event.GetEventTypeWithoutPrefix(msg.GetMsg()) == "ReprocessDeployments"
	})

	t.Run("send all messages when clusters are healthy", func(t *testing.T) {
		testLoop, connA, connB := newReprocessorLoop(t)

		// Expect every updated image to be sent.
		connA.EXPECT().InjectMessage(gomock.Any(), updatedImageTypeCond).Times(len(imgs) / 2).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), updatedImageTypeCond).Times(len(imgs) / 2).Return(nil)

		// Expect each cluster to be sent a reprocess deployments message.
		connA.EXPECT().InjectMessage(gomock.Any(), reprocessDeploymentsTypeCond).Times(1).Return(nil)
		connB.EXPECT().InjectMessage(gomock.Any(), reprocessDeploymentsTypeCond).Times(1).Return(nil)

		testLoop.reprocessImagesAndResyncDeployments(0, reprocessFuncUpdate, nil)
	})

	t.Run("skip some messages when are broken clusters", func(t *testing.T) {
		testLoop, connA, connB := newReprocessorLoop(t)

		// Cluster "a" is healthy, expect all applicable images to be sent.
		connA.EXPECT().InjectMessage(gomock.Any(), updatedImageTypeCond).Times(len(imgs) / 2).Return(nil)
		// Cluster "b" is not healthy, expect at MOST imageReprocessorSemaphoreSize attempted messages.
		connB.EXPECT().InjectMessage(gomock.Any(), updatedImageTypeCond).MaxTimes(int(imageReprocessorSemaphoreSize)).Return(errors.New("broken"))

		connA.EXPECT().InjectMessage(gomock.Any(), reprocessDeploymentsTypeCond).Times(1).Return(nil)
		// No reprocess deployments message is sent due to previous failures.
		connB.EXPECT().InjectMessage(gomock.Any(), reprocessDeploymentsTypeCond).Times(0).Return(nil)

		testLoop.reprocessImagesAndResyncDeployments(0, reprocessFuncUpdate, nil)
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
