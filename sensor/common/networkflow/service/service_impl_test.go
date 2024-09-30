package service

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	mocksNetworkflowManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	mocksOrchestrator "github.com/stackrox/rox/sensor/common/orchestrator/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
)

//import (
//	"context"
//	//"net"
//	//"sync/atomic"
//	"testing"
//	//"time"
//
//	"github.com/stackrox/rox/generated/internalapi/sensor"
//	//"github.com/stackrox/rox/generated/storage"
//	"github.com/stackrox/rox/pkg/concurrency"
//	//"github.com/stackrox/rox/pkg/set"
//	//"github.com/stackrox/rox/pkg/utils"
//	//"github.com/stackrox/rox/sensor/common"
//	mocksNetworkflowManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
//	//"github.com/stackrox/rox/sensor/common/orchestrator"
//	mocksOrchestrator "github.com/stackrox/rox/sensor/common/orchestrator/mocks"
//	"github.com/stretchr/testify/mock"
//	"github.com/stretchr/testify/require"
//	"github.com/stretchr/testify/suite"
//	"go.uber.org/mock/gomock"
//	//"google.golang.org/grpc"
//	//"google.golang.org/grpc/credentials/insecure"
//	"google.golang.org/grpc/metadata"
//	//"google.golang.org/grpc/test/bufconn"
//)

func TestNetworkflowService(t *testing.T) {
	suite.Run(t, new(networkflowServiceSuite))
}

type networkflowServiceSuite struct {
	suite.Suite
	mockCtrl            *gomock.Controller
	mockOrchestrator    *mocksOrchestrator.MockOrchestrator
	mockNetworkflowManager *mocksNetworkflowManager.MockManager
	srv                 *serviceImpl
	stream              sensor.NetworkConnectionInfoService_CommunicateServer
	collectorConfigProtoStream *concurrency.ValueStream[*sensor.CollectorConfig]
}

func (s *networkflowServiceSuite) SetupTest() {
	s.collectorConfigProtoStream = concurrency.NewValueStream[*sensor.CollectorConfig](nil)
	//s.mockCtrl = gomock.NewController(s.T())
	//s.mockOrchestrator = mocksOrchestrator.NewMockOrchestrator(s.mockCtrl)
	//s.mockNetworkflowManager = mocksNetworkflowManager.NewMockManager(s.mockCtrl)
	//s.srv = &serviceImpl{
	//	manager:          s.mockNetworkflowManager,
	//	authFuncOverride: authFuncOverride,
	//	writer:           nil,
	//}
	//s.createMockService()
}

type MockStream struct {
	mock.Mock
	//recorder *MockStreamMockRecorder
}

//type MockStreamMockRecorder struct {
//	mock *MockStream
//}
//
//// EXPECT returns an object that allows the caller to indicate expected use.
//func (m *MockStream) EXPECT() *MockStreamMockRecorder {
//	return m.recorder
//}

func (m *MockStream) Send(msg *sensor.MsgToCollector) error {
	args := m.Called(msg)
	log.Info("In MockStream Send")
	return args.Error(0)
}

//// AuditEvents indicates an expected call of AuditEvents.
//func (mr *MockServiceMockRecorder) AuditEvents() *gomock.Call {
//	mr.mock.ctrl.T.Helper()
//	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuditEvents", reflect.TypeOf((*MockService)(nil).AuditEvents))
//}

func (m *MockStream) SendMsg(msg interface{}) error {
    args := m.Called(msg)
    return args.Error(0)
}

func (m *MockStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *MockStream) SetHeader(md metadata.MD) error {
    args := m.Called(md)
    return args.Error(0)
}

func (m *MockStream) SetTrailer(md metadata.MD) {
    m.Called(md)
}

func (m *MockStream) Recv() (*sensor.NetworkConnectionInfoMessage, error) {
	args := m.Called()
	if msg, ok := args.Get(0).(*sensor.NetworkConnectionInfoMessage); ok {
		return msg, nil
	}
	return nil, args.Error(1)
}

func (m *MockStream) RecvMsg(_ interface{}) error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStream) Context() context.Context {
	args := m.Called()
	if ctx, ok := args.Get(0).(context.Context); ok {
		return ctx
	}
	return context.Background()
}

func (s *networkflowServiceSuite) CollectorConfigValueStream() concurrency.ReadOnlyValueStream[*sensor.CollectorConfig] {
	return s.collectorConfigProtoStream
}

func (s *networkflowServiceSuite) TestSendCollectorConfig() {
	log.Info("In TestSendCollectorConfig")
    collectorConfig := &sensor.CollectorConfig{
        NetworkConnectionConfig: &sensor.NetworkConnectionConfig{
            EnableExternalIps: true,
        },
    }
    log.Info("Set collectorConfig")
    //var collectorConfigIterator concurrency.ValueStreamIter[*sensor.CollectorConfig]
    log.Info("Declared collectorConfigIterator")

    //collectorConfigIterator = s.CollectorConfigValueStream().Iterator(false)
    collectorValueStream := s.CollectorConfigValueStream()
    log.Infof("collectorValueStream= %+v", collectorValueStream)
    collectorConfigIterator := collectorValueStream.Iterator(false)

    log.Info("Before Push")
    log.Infof("collectorConfigIterator= %+v", collectorConfigIterator)
    //log.Infof("s.collectorConfigProtoStream= %+v", s.collectorConfigProtoStream.Curr())
    s.collectorConfigProtoStream.Push(collectorConfig)
    time.Sleep(2 * time.Second)
    log.Info("After Push")
    log.Infof("collectorConfigIterator= %+v", collectorConfigIterator)
    collectorConfigIterator = collectorConfigIterator.TryNext()

    mockStream := new(MockStream)

    // Expect the Send method to be called exactly once with a MsgToCollector
    mockStream.On("Send", mock.AnythingOfType("*sensor.MsgToCollector")).Return(nil).Once()
    //mockStream.EXPECT().Send(gomock.Any()).Once()
    //s.mockAuditLogManager.EXPECT().RemoveEligibleComplianceNode(gomock.Any()).AnyTimes()

    //service := &serviceImpl{}
    service := NewService(s.mockNetworkflowManager)
//func NewService(networkFlowManager manager.Manager, opts ...Option) Service {

    //_ = service.sendCollectorConfig(mockStream, collectorConfigIterator)
    log.Info("Before SendCollectorConfig")
    err := service.SendCollectorConfig(mockStream, collectorConfigIterator)

    require.NoError(s.T(), err)

    mockStream.AssertExpectations(s.T())
}


//func (s *networkflowServiceSuite) sendCollectorConfig() {
//	collectorConfig := &sensor.CollectorConfig{
//		NetworkConnectionConfig: &sensor.NetworkConnectionConfig{
//			EnableExternalIps: true,
//		},
//	}
//	var collectorConfigIterator concurrency.ValueStreamIter[*sensor.CollectorConfig]
//
//	collectorConfigIterator = s.CollectorConfigValueStream().Iterator(false)
//	
//	s.collectorConfigProtoStream.Push(collectorConfig)
//	
//	mockStream := new(MockStream)
//	mockStream.On("Send", mock.AnythingOfType("*sensor.MsgToCollector")).Return(nil).Once()
//	
//	service := &serviceImpl{}
//	
//	err := service.sendCollectorConfig(mockStream, collectorConfigIterator)
//	require.NoError(s.T(), err)
//	
//	mockStream.AssertExpectations(s.T())
//}
