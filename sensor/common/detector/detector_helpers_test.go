package detector

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/detector/baseline"
	detectorEvents "github.com/stackrox/rox/sensor/common/detector/events"
	networkBaselineEval "github.com/stackrox/rox/sensor/common/detector/networkbaseline"
	"github.com/stackrox/rox/sensor/common/detector/queue"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/pubsub/consumer"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	mockStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestDetector(tb testing.TB, pubSubEnabled bool) (*detectorImpl, *mockStore.MockDeploymentStore, *mockStore.MockNetworkPolicyStore, *mockStore.MockNodeStore) {
	return createTestDetectorWithBufferSize(tb, pubSubEnabled, 100)
}

func createTestDetectorWithBufferSize(tb testing.TB, pubSubEnabled bool, bufferSize int) (*detectorImpl, *mockStore.MockDeploymentStore, *mockStore.MockNetworkPolicyStore, *mockStore.MockNodeStore) {
	tb.Helper()

	ctrl := gomock.NewController(tb)
	tb.Cleanup(ctrl.Finish)

	deploymentStore := mockStore.NewMockDeploymentStore(ctrl)
	networkPolicyStore := mockStore.NewMockNetworkPolicyStore(ctrl)
	serviceAccountStore := mockStore.NewMockServiceAccountStore(ctrl)
	serviceAccountStore.EXPECT().GetImagePullSecrets(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	nodeStore := mockStore.NewMockNodeStore(ctrl)

	detectorStopper := concurrency.NewStopper()

	var piQueue *queue.Queue[*detectorEvents.IndicatorEvent]
	var netFlowQueue *queue.Queue[*detectorEvents.NetworkFlowEvent]
	var fileAccessQueue *queue.Queue[*detectorEvents.FileAccessEvent]
	if !pubSubEnabled {
		piQueue = queue.NewQueue[*detectorEvents.IndicatorEvent](
			detectorStopper, "PIsQueue", bufferSize, nil, nil,
		)
		netFlowQueue = queue.NewQueue[*detectorEvents.NetworkFlowEvent](
			detectorStopper, "FlowsQueue", bufferSize, nil, nil,
		)
		fileAccessQueue = queue.NewQueue[*detectorEvents.FileAccessEvent](
			detectorStopper, "FileAccessQueue", bufferSize, nil, nil,
		)
	}
	deploymentQueue := queue.NewSimpleQueue[*queue.DeploymentQueueItem](
		"DeploymentQueue", 0, nil, nil,
	)

	d := &detectorImpl{
		unifiedDetector:           &fakeUnifiedDetector{},
		output:                    make(chan *message.ExpiringMessage, 1000),
		auditEventsChan:           make(chan *sensor.AuditEvents),
		deploymentAlertOutputChan: make(chan outputResult),
		deploymentProcessingMap:   make(map[string]int64),
		enricher:                  newEnricher(&fakeClusterIDPeekWaiter{}, nil, serviceAccountStore, nil, nil),
		deploymentStore:           deploymentStore,
		nodeStore:                 nodeStore,
		networkPolicyStore:        networkPolicyStore,
		baselineEval:              baseline.NewBaselineEvaluator(),
		networkbaselineEval:       networkBaselineEval.NewNetworkBaselineEvaluator(),
		enforcer:                  &fakeEnforcer{},
		deduper:                   newDeduper(),
		detectorStopper:           detectorStopper,
		auditStopper:              concurrency.NewStopper(),
		serializerStopper:         concurrency.NewStopper(),
		alertStopSig:              concurrency.NewSignal(),
		indicatorsQueue:           piQueue,
		networkFlowsQueue:         netFlowQueue,
		fileAccessQueue:           fileAccessQueue,
		deploymentsQueue:          deploymentQueue,
		runtimeRunning:            concurrency.NewSignal(),
	}

	if pubSubEnabled {
		dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
			[]pubsub.LaneConfig{
				lane.NewConcurrentLane(pubsub.DetectorProcessIndicatorLane,
					lane.WithConcurrentLaneConsumer(
						consumer.NewBufferedConsumer(
							consumer.WithBufferedConsumerSize(bufferSize),
						),
					),
				),
				lane.NewConcurrentLane(pubsub.DetectorNetworkFlowLane,
					lane.WithConcurrentLaneConsumer(
						consumer.NewBufferedConsumer(
							consumer.WithBufferedConsumerSize(bufferSize),
						),
					),
				),
				lane.NewConcurrentLane(pubsub.DetectorFileAccessLane,
					lane.WithConcurrentLaneConsumer(
						consumer.NewBufferedConsumer(
							consumer.WithBufferedConsumerSize(bufferSize),
						),
					),
				),
			},
		))
		require.NoError(tb, err)
		tb.Cleanup(dispatcher.Stop)
		d.pubSubDispatcher = dispatcher
	}

	return d, deploymentStore, networkPolicyStore, nodeStore
}

const benchBufferSize = 20000

func createBenchDetector(b *testing.B, pubSubEnabled bool) *detectorImpl {
	b.Helper()

	d, ds, nps, _ := createTestDetectorWithBufferSize(b, pubSubEnabled, benchBufferSize)

	ds.EXPECT().GetSnapshot(gomock.Any()).DoAndReturn(func(id string) *storage.Deployment {
		return &storage.Deployment{Id: id, Name: "bench-" + id, Namespace: "default"}
	}).AnyTimes()
	nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

	d.unifiedDetector = &fakeUnifiedDetector{
		alerts: []*storage.Alert{{
			Id:     "alert-1",
			Policy: &storage.Policy{Id: "policy-1"},
		}},
	}

	return d
}

// fakes

type fakeClusterIDPeekWaiter struct{}

func (f *fakeClusterIDPeekWaiter) Get() string       { return "fake-cluster-id" }
func (f *fakeClusterIDPeekWaiter) GetNoWait() string { return "fake-cluster-id" }

type fakeUnifiedDetector struct {
	alerts []*storage.Alert
}

func (f *fakeUnifiedDetector) ReconcilePolicies(_ []*storage.Policy) {}
func (f *fakeUnifiedDetector) DetectDeployment(_ booleanpolicy.EnhancedDeployment) []*storage.Alert {
	return f.alerts
}
func (f *fakeUnifiedDetector) DetectProcess(_ booleanpolicy.EnhancedDeployment, _ *storage.ProcessIndicator, _ bool) []*storage.Alert {
	return f.alerts
}
func (f *fakeUnifiedDetector) DetectKubeEventForDeployment(_ booleanpolicy.EnhancedDeployment, _ *storage.KubernetesEvent) []*storage.Alert {
	return nil
}
func (f *fakeUnifiedDetector) DetectNetworkFlowForDeployment(_ booleanpolicy.EnhancedDeployment, _ *augmentedobjs.NetworkFlowDetails) []*storage.Alert {
	return f.alerts
}
func (f *fakeUnifiedDetector) DetectAuditLogEvents(_ *sensor.AuditEvents) []*storage.Alert {
	return f.alerts
}
func (f *fakeUnifiedDetector) DetectNodeFileAccess(_ *storage.Node, _ *storage.FileAccess) []*storage.Alert {
	return f.alerts
}
func (f *fakeUnifiedDetector) DetectFileAccessForDeployment(_ booleanpolicy.EnhancedDeployment, _ *storage.FileAccess) []*storage.Alert {
	return f.alerts
}

type fakeEnforcer struct{}

func (f *fakeEnforcer) Start() error                                   { return nil }
func (f *fakeEnforcer) Stop()                                          {}
func (f *fakeEnforcer) Notify(_ common.SensorComponentEvent)           {}
func (f *fakeEnforcer) Capabilities() []centralsensor.SensorCapability { return nil }
func (f *fakeEnforcer) Name() string                                   { return "fake" }
func (f *fakeEnforcer) ResponsesC() <-chan *message.ExpiringMessage    { return nil }
func (f *fakeEnforcer) Accepts(_ *central.MsgToSensor) bool            { return false }
func (f *fakeEnforcer) ProcessMessage(_ context.Context, _ *central.MsgToSensor) error {
	return nil
}
func (f *fakeEnforcer) ProcessAlertResults(_ central.ResourceAction, _ storage.LifecycleStage, _ *central.AlertResults) {
}
