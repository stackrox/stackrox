package connection

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	hashManager "github.com/stackrox/rox/central/hash/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	clusterMgrMock "github.com/stackrox/rox/central/sensor/service/common/mocks"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dedupingqueue"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

const deduperIntegrationClusterId = "ecabcdef-bbbb-4011-0000-111111111111"
const deduperIntegrationStaticBindingId1 = "203a7381-5bff-4aa2-a8a9-000000000001"
const deduperIntegrationStaticBindingId2 = "203a7381-5bff-4aa2-a8a9-000000000002"
const deduperIntegrationStaticBindingId3 = "203a7381-5bff-4aa2-a8a9-000000000003"
const deduperIntegrationStaticBindingId4 = "203a7381-5bff-4aa2-a8a9-000000000004"
const deduperIntegrationStaticBindingId5 = "203a7381-5bff-4aa2-a8a9-000000000005"
const deduperIntegrationDropConnectionMsg = "drop-connection"
const deduperIntegrationMsgResends = 2

var deduperIntegrationStaticTime = time.Unix(2345678901, 123)

var deduperIntegrationMsgCh = make(chan string)
var deduperIntegrationPipelineFinishCh = make(chan int)
var deduperIntegrationMsgWg sync.WaitGroup

type testDeduperIntegrationSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
}

func TestDeduperIntegrationHandler(t *testing.T) {
	suite.Run(t, new(testDeduperIntegrationSuite))
}

func (s *testDeduperIntegrationSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)

	s.mockCtrl = gomock.NewController(s.T())
}

// deduperIntegrationRetryError - represent retryable error in queue pipeline processing
// that will cause sensor message to be put back in queue (not processed!)
type deduperIntegrationRetryError struct {
	status string
}

func (e *deduperIntegrationRetryError) SafeToRetry() bool {
	return true
}

func (e *deduperIntegrationRetryError) Error() string {
	return e.status
}

// mockDeduperIntegrationHashDatastore - mocks data store. We need it to create instance
// but there is no functionality we need from it. Only that datastore is empty on init.
type mockDeduperIntegrationHashDatastore struct{}

func (d *mockDeduperIntegrationHashDatastore) UpsertHash(_ context.Context, _ *storage.Hash) error {
	return nil
}

func (d *mockDeduperIntegrationHashDatastore) GetHashes(_ context.Context, _ string) (*storage.Hash, bool, error) {
	// Return false - to represent that hash does not exist. Clean central installation.
	return nil, false, nil
}

func (d *mockDeduperIntegrationHashDatastore) DeleteHashes(_ context.Context, _ string) error {
	return nil
}

// mockDeduperIntegrationServer - mocking in order to simulate msg receiving from sensor.
type mockDeduperIntegrationServer struct {
	grpc.ServerStream
}

func (srv *mockDeduperIntegrationServer) Send(msg *central.MsgToSensor) error {
	return nil
}

func (srv *mockDeduperIntegrationServer) Recv() (*central.MsgFromSensor, error) {
	msgId := <-deduperIntegrationMsgCh

	log.Info(fmt.Sprintf("Msg Recv: %v", msgId))
	if msgId == deduperIntegrationDropConnectionMsg {
		// Error only if all messages are pulled at least once from queue.
		// This is important because it can happen that we close connection
		// before any message is processed. And we want to have control over that.
		deduperIntegrationMsgWg.Wait()

		log.Info("Connection dropped!")
		return nil, errors.New("Connection closed by client")
	}

	return getDeduperIntegrationMsgWithId(msgId)
}

//func toBindingEvent(binding *storage.K8SRoleBinding, action central.ResourceAction) *central.SensorEvent {
//	return &central.SensorEvent{
//		Id:     binding.GetId(),
//		Action: action,
//		Resource: &central.SensorEvent_Binding{
//			Binding: binding.CloneVT(),
//		},
//	}
//}

// getDeduperIntegrationMsgWithId - will use the same id for binding and event.
func getDeduperIntegrationMsgWithId(id string) (*central.MsgFromSensor, error) {
	return &central.MsgFromSensor{Msg: &central.MsgFromSensor_Event{Event: &central.SensorEvent{
		Id:     id,
		Action: 1,
		Resource: &central.SensorEvent_Binding{Binding: &storage.K8SRoleBinding{
			Id:          id,
			Name:        "test-binding",
			Namespace:   "default",
			ClusterId:   deduperIntegrationClusterId,
			ClusterName: "remote",
			ClusterRole: false,
			CreatedAt:   protocompat.ConvertTimeToTimestampOrNil(&deduperIntegrationStaticTime),
			RoleId:      "test-binding",
		}},
	}}}, nil
}

// sendDeduperIntegrationMessages - will simply send messages to Recv (simulate sensor sending)
// We could also send only once. Just simulating a bit more noise and that deduper will do it's job.
func sendDeduperIntegrationMessages(msgs []string) {
	for i := 0; i < deduperIntegrationMsgResends; i++ {
		for _, msg := range msgs {
			deduperIntegrationMsgCh <- msg
		}
	}

	deduperIntegrationMsgCh <- deduperIntegrationDropConnectionMsg
}

// TestDeduperIntegrationAfterReconnect - is intended for debug and investigation.
// Because of that there is logging.
func (s *testSuite) TestDeduperIntegrationAfterReconnect() {
	ctx := sac.WithAllAccess(context.Background())

	clusterRef := &storage.Cluster{Id: deduperIntegrationClusterId}

	ctrl := gomock.NewController(s.T())
	mgrMock := clusterMgrMock.NewMockClusterManager(ctrl)

	mockHashDatastore := &mockDeduperIntegrationHashDatastore{}
	hashMngr := hashManager.NewManager(mockHashDatastore)
	deduper := hashMngr.GetDeduper(ctx, clusterRef.GetId())

	pipeline := pipelineMock.NewMockClusterPipeline(ctrl)
	pipeline.EXPECT().OnFinish(gomock.Any()).Times(2).Do(func(clusterID string) {
		log.Info("Pipeline OnFinish")

		go func() {
			deduperIntegrationPipelineFinishCh <- 0
		}()
	})
	pipeline.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).Times(6).Do(func(ctx context.Context, msg *central.MsgFromSensor, injector common.MessageInjector) error {
		defer deduperIntegrationMsgWg.Done()

		log.Info(fmt.Sprintf("Msg Pull: %v", msg.GetEvent().GetBinding().GetId()))

		// This simulates that message is not processed from queue!!!
		if msg.GetEvent().GetId() == deduperIntegrationStaticBindingId3 {
			// Return error that is recoverable - so msg is pushed to queue again!
			return &deduperIntegrationRetryError{status: "Safe to retry!"}
		}

		deduper.MarkSuccessful(msg)
		return nil
	})

	stopSig := concurrency.NewErrorSignal()
	assert.False(s.T(), stopSig.IsDone())
	eventHandler := newSensorEventHandler(clusterRef, "", pipeline, nil, &stopSig, deduper, nil)
	sensorMockConn := &sensorConnection{
		clusterMgr:         mgrMock,
		sensorEventHandler: eventHandler,
		sensorHello:        &central.SensorHello{SensorVersion: "1.0"},
		hashDeduper:        deduper,
		stopSig:            stopSig,
		queues:             make(map[string]*dedupingqueue.DedupingQueue[string]),
		sendC:              make(chan *central.MsgToSensor),
		eventPipeline:      pipeline,
	}
	server := &mockDeduperIntegrationServer{}

	deduperIntegrationMsgWg.Add(3) // we expect that 1, 2, 3 are processed (msg3 - always throws error, never removed from queue)
	go sendDeduperIntegrationMessages([]string{deduperIntegrationStaticBindingId1, deduperIntegrationStaticBindingId2, deduperIntegrationStaticBindingId3})
	sensorMockConn.runRecv(ctx, server)

	// Pipeline close -> Queues destroyed!
	<-deduperIntegrationPipelineFinishCh

	log.Info(deduper.GetSuccessfulHashes())

	// --- Do everything again - keep deduper (we can also keep pipeline, because it does not have any state - except deduper update)

	log.Info("New connection is started")
	deduper.StartSync()
	log.Info(deduper.GetSuccessfulHashes())

	stopSig = concurrency.NewErrorSignal()
	assert.False(s.T(), stopSig.IsDone())
	eventHandler = newSensorEventHandler(clusterRef, "", pipeline, nil, &stopSig, deduper, nil)
	sensorMockConn = &sensorConnection{
		clusterMgr:         mgrMock,
		sensorEventHandler: eventHandler,
		sensorHello:        &central.SensorHello{SensorVersion: "1.0"},
		hashDeduper:        deduper,
		stopSig:            stopSig,
		queues:             make(map[string]*dedupingqueue.DedupingQueue[string]),
		sendC:              make(chan *central.MsgToSensor),
		eventPipeline:      pipeline,
	}
	server = &mockDeduperIntegrationServer{}

	deduperIntegrationMsgWg.Add(3) // we expect that 3, 4, 5 are processed
	go sendDeduperIntegrationMessages([]string{deduperIntegrationStaticBindingId3, deduperIntegrationStaticBindingId4, deduperIntegrationStaticBindingId5})
	sensorMockConn.runRecv(ctx, server)

	// Pipeline close -> Queues destroyed!
	<-deduperIntegrationPipelineFinishCh
}
