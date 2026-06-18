package resolver

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/service"
	"github.com/stackrox/rox/sensor/common/store"
	mocksStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	mocksComponent "github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// lastDeploymentID is used to mark when we should stop waiting for the deployments to be processed.
	lastDeploymentID = "last-deployment-id"
	// queueSize is the innerQueue size.
	queueSize = 100
)

var (
	res                  component.Resolver
	mockCtrl             *gomock.Controller
	mockOutput           *mocksComponent.MockOutputQueue
	mockDeploymentStore  *mocksStore.MockDeploymentStore
	mockServiceStore     *mocksStore.MockServiceStore
	mockRBACStore        *mocksStore.MockRBACStore
	mockEndpointManager  *mocksStore.MockEndpointManager
	mockPubSubDispatcher *mocksComponent.MockPubSubDispatcher

	cases = []struct {
		numEvents      int
		numDeployments int
	}{
		{
			numEvents:      100,
			numDeployments: 1,
		},
		{
			numEvents:      100,
			numDeployments: 10,
		},
		{
			numEvents:      1000,
			numDeployments: 100,
		},
		{
			numEvents:      1000,
			numDeployments: 1000,
		},
	}
)

func dispatchEvent(b *testing.B, event *component.ResourceEvent, resolver component.Resolver, pubsubEnabled bool) {
	if pubsubEnabled {
		if err := resolver.ProcessResourceEvent(event); err != nil {
			b.Error(err)
		}
		return
	}
	res.Send(event)
}

// benchmarkProcessDeploymentReferences runs the resolver benchmark using the
// currently active feature flag value for SensorInternalPubSub.
//
// To compare legacy vs pubsub with benchstat:
//
//	ROX_SENSOR_PUBSUB=false go test -run='^$' -bench=BenchmarkProcess -benchmem -count=10 ./sensor/kubernetes/eventpipeline/resolver/ > bench_legacy.txt
//	ROX_SENSOR_PUBSUB=true  go test -run='^$' -bench=BenchmarkProcess -benchmem -count=10 ./sensor/kubernetes/eventpipeline/resolver/ > bench_pubsub.txt
//	benchstat bench_legacy.txt bench_pubsub.txt
func benchmarkProcessDeploymentReferences(b *testing.B, randomIDs bool) {
	pubsubEnabled := features.SensorInternalPubSub.Enabled()
	for _, bc := range cases {
		b.Run(fmt.Sprintf("events=%d/deployments=%d", bc.numEvents, bc.numDeployments), func(b *testing.B) {
			doneSignal := concurrency.NewSignal()
			setupMocks(b, &doneSignal, pubsubEnabled)

			for b.Loop() {
				b.StopTimer()
				doneSignal.Reset()
				events := createEvents(randomIDs, bc.numEvents, bc.numDeployments)
				setupResolver(b)
				b.StartTimer()
				for _, event := range events {
					dispatchEvent(b, event, res, pubsubEnabled)
				}
				doneSignal.Wait()
				b.StopTimer()
				res.Stop()
				// b.Loop() requires the timer to be running when called.
				b.StartTimer()
			}
		})
	}
}

func BenchmarkProcessDeploymentReferences(b *testing.B) {
	benchmarkProcessDeploymentReferences(b, false)
}

func BenchmarkProcessRandomDeploymentReferences(b *testing.B) {
	benchmarkProcessDeploymentReferences(b, true)
}

func setupResolver(b *testing.B) {
	var err error
	res, err = New(mockOutput, &fakeProvider{
		deploymentStore: mockDeploymentStore,
		serviceStore:    mockServiceStore,
		rbacStore:       mockRBACStore,
		endpointManager: mockEndpointManager,
	}, queueSize, mockPubSubDispatcher)
	if err != nil {
		b.Error(err)
	}
	err = res.Start()
	if err != nil {
		b.Error(err)
	}
}

func setupMocks(b *testing.B, doneSignal *concurrency.Signal, pubsubEnabled bool) {
	// Create the mocks
	mockCtrl = gomock.NewController(b)
	mockOutput = mocksComponent.NewMockOutputQueue(mockCtrl)
	mockDeploymentStore = mocksStore.NewMockDeploymentStore(mockCtrl)
	mockServiceStore = mocksStore.NewMockServiceStore(mockCtrl)
	mockRBACStore = mocksStore.NewMockRBACStore(mockCtrl)
	mockEndpointManager = mocksStore.NewMockEndpointManager(mockCtrl)
	mockPubSubDispatcher = mocksComponent.NewMockPubSubDispatcher(mockCtrl)
	// Set up the EXPECT
	if pubsubEnabled {
		mockPubSubDispatcher.EXPECT().RegisterConsumerToLane(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
		mockPubSubDispatcher.EXPECT().Publish(gomock.Any()).AnyTimes().DoAndReturn(func(event pubsub.Event) error {
			resourceEvent, ok := event.(*component.ResourceEvent)
			if !ok {
				return nil
			}
			for _, m := range resourceEvent.ForwardMessages {
				if m.GetDeployment().GetId() == lastDeploymentID {
					doneSignal.Signal()
				}
			}
			return nil
		})
	}
	mockOutput.EXPECT().Send(gomock.Any()).AnyTimes().DoAndReturn(func(resourceEvent *component.ResourceEvent) {
		for _, m := range resourceEvent.ForwardMessages {
			if m.GetDeployment().GetId() == lastDeploymentID {
				doneSignal.Signal()
			}
		}
	})
	mockDeploymentStore.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(id string) *storage.Deployment {
		return &storage.Deployment{
			Id: id,
		}
	})
	mockEndpointManager.EXPECT().OnDeploymentCreateOrUpdateByID(gomock.Any()).AnyTimes()
	mockRBACStore.EXPECT().GetPermissionLevelForDeployment(gomock.Any()).AnyTimes().DoAndReturn(func(d *storage.Deployment) storage.PermissionLevel {
		return storage.PermissionLevel_NONE
	})
	mockServiceStore.EXPECT().GetExposureInfos(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(ns string, labels map[string]string) []map[service.PortRef][]*storage.PortConfig_ExposureInfo {
		return []map[service.PortRef][]*storage.PortConfig_ExposureInfo{
			{
				service.PortRef{
					Port:     intstr.FromInt32(80),
					Protocol: v1.ProtocolTCP,
				}: []*storage.PortConfig_ExposureInfo{
					{
						Level: storage.PortConfig_INTERNAL,
					},
				},
			},
		}
	})
	mockDeploymentStore.EXPECT().BuildDeploymentWithDependencies(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(id string, _ store.Dependencies) (*storage.Deployment, bool, error) {
		return &storage.Deployment{
			Id: id,
		}, true, nil
	})
}

func createEvents(randomIDs bool, numEvents, numDeploymentRefs int) []*component.ResourceEvent {
	ret := make([]*component.ResourceEvent, numEvents+1)
	var ids []string
	if !randomIDs {
		ids = createIds(numDeploymentRefs)
	}
	for i := 0; i < numEvents; i++ {
		var event component.ResourceEvent
		if randomIDs {
			ids = createRandomIds(numDeploymentRefs)
		}
		event.AddDeploymentReference(resolver.ResolveDeploymentIds(ids...))
		ret[i] = &event
	}
	// Add the last-deployment, this way we know when all the messages have been processed.
	var event component.ResourceEvent
	event.AddDeploymentReference(resolver.ResolveDeploymentIds(lastDeploymentID))
	ret[numEvents] = &event
	return ret
}

func createIds(n int) []string {
	ret := make([]string, n)
	for i := 0; i < n; i++ {
		ret[i] = fmt.Sprintf("deployment-%d", i)
	}
	return ret
}

const charset = "abcdef0123456789"

func randStringWithLength(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func createRandomIds(n int) []string {
	ret := make([]string, n)
	for i := 0; i < n; i++ {
		ret[i] = randStringWithLength(10)
	}
	return ret
}
