package service

import (
	"context"
	//"net"
	//"sync/atomic"
	"testing"
	//"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	//"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	//"github.com/stackrox/rox/pkg/set"
	//"github.com/stackrox/rox/pkg/utils"
	//"github.com/stackrox/rox/sensor/common"
	mocksNetworkflowManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	//"github.com/stackrox/rox/sensor/common/orchestrator"
	mocksOrchestrator "github.com/stackrox/rox/sensor/common/orchestrator/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	//"google.golang.org/grpc"
	//"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	//"google.golang.org/grpc/test/bufconn"
)

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
	//stream              sensor.NetworkConnectionInfoService_CommunicateClient
	stopServerFn        func()
}

//var _ suite.SetupTestSuite = (*networkflowServiceSuite)(nil)
//var _ suite.TearDownTestSuite = (*networkflowServiceSuite)(nil)

//func (s *networkflowServiceSuite) SetupTest() {
//	s.mockCtrl = gomock.NewController(s.T())
//	s.mockOrchestrator = mocksOrchestrator.NewMockOrchestrator(s.mockCtrl)
//	s.mockNetworkflowManager = mocksNetworkflowManager.NewMockManager(s.mockCtrl)
//	s.srv = &serviceImpl{
//		manager:          s.mockNetworkflowManager,
//		authFuncOverride: authFuncOverride,
//		writer:           nil,
//	}
//	s.createMockService()
//}

//func authFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
//	return ctx, idcheck.CollectorOnly().Authorized(ctx, fullMethodName)
//}

//func (s *networkflowServiceSuite) TearDownTest() {
//	if s.stopServerFn != nil {
//		s.stopServerFn()
//	}
//	//assertNoGoroutineLeaks(s.T())
//}

type MockStream struct {
	mock.Mock
}

func (m *MockStream) Send(msg *sensor.MsgToCollector) error {
	args := m.Called(msg)
	return args.Error(0)
}

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

//type MockIter struct {
//	mock.Mock
//	doneCh chan struct{}
//}
//
//func (m *MockIter) Done() <-chan struct{} {
//	if m.doneCh == nil {
//		m.doneCh = make(chan struct{})
//	}
//	return m.doneCh
//}
//
//func (m *MockIter) Value() *sensor.CollectorConfig {
//	args := m.Called()
//	if val, ok := args.Get(0).(*sensor.CollectorConfig); ok {
//		return val
//	}
//	return nil
//}
//
//func (m *MockIter) Next(waitable concurrency.ErrorWaitable) (concurrency.ValueStreamIter[*sensor.CollectorConfig], error) {
//    args := m.Called(waitable)
//    if iter, ok := args.Get(0).(concurrency.ValueStreamIter[*sensor.CollectorConfig]); ok {
//        return iter, args.Error(1)
//    }
//    return nil, args.Error(1)
//}
//
//func (m *MockIter) TryNext() concurrency.ValueStreamIter[*sensor.CollectorConfig] {
//    args := m.Called()
//    if iter, ok := args.Get(0).(concurrency.ValueStreamIter[*sensor.CollectorConfig]); ok {
//        return iter
//    }
//    return nil
//}
//
//func (m *MockIter) isValueStreamIter() {}

func (s *networkflowServiceSuite) CollectorConfigValueStream() concurrency.ReadOnlyValueStream[*sensor.CollectorConfig] {
	return s.collectorConfigProtoStream
}

func (s *networkflowServiceSuite) sendCollectorConfig() {
	collectorConfig := &sensor.CollectorConfig{
	}
	
	//var collectorConfigValueStream concurrency.ReadOnlyValueStream[*sensor.CollectorConfig]
	//var collectorConfigIterator concurrency.ValueStreamIter[*sensor.CollectorConfig]
	//iter := concurrency.NewValueStreamIter([]*sensor.CollectorConfig{collectorConfig})


	var collectorConfigIterator concurrency.ValueStreamIter[*sensor.CollectorConfig]
	collectorConfigIterator = s.CollectorConfigValueStream().Iterator(false)
	
	//collectorConfigIterator.Push(collectorConfig)
	s.collectorConfigProtoStream.Push(collectorConfig)
	
	mockStream := new(MockStream)
	mockStream.On("Send", mock.AnythingOfType("*sensor.MsgToCollector")).Return(nil)
	
	service := &serviceImpl{}
	
	err := service.sendCollectorConfig(mockStream, collectorConfigIterator)
	require.NoError(s.T(), err)
	
	mockStream.AssertExpectations(s.T())
}

//func (s *networkflowServiceSuite) sendCollectorConfig() {
//	mockStream := new(MockStream)
//	mockIter := new(MockIter)
//
//	// Create a sample collector configuration to be sent
//	collectorConfig := &sensor.CollectorConfig{
//	}
//
//	// Set up the mock iter to return the sample collector config
//	mockIter.On("Value").Return(collectorConfig)
//
//	// Expect the Send method to be called with a specific message
//	expectedMsg := &sensor.MsgToCollector{
//		Msg: &sensor.MsgToCollector_CollectorConfig{
//			CollectorConfig: collectorConfig,
//		},
//	}
//	mockStream.On("Send", expectedMsg).Return(nil)
//
//	// Create a service instance and call sendCollectorConfig
//	service := &serviceImpl{}
//	err := service.sendCollectorConfig(mockStream, mockIter)
//	require.NoError(s.T(), err)
//
//	// Validate that all expectations were met
//	mockIter.AssertExpectations(s.T())
//	mockStream.AssertExpectations(s.T())
//}
