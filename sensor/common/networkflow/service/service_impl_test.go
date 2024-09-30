package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	mocksNetworkflowManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/metadata"
)

func TestNetworkflowService(t *testing.T) {
	suite.Run(t, new(networkflowServiceSuite))
}

type networkflowServiceSuite struct {
	suite.Suite
	mockNetworkflowManager *mocksNetworkflowManager.MockManager
	collectorConfigProtoStream *concurrency.ValueStream[*sensor.CollectorConfig]
}

func (s *networkflowServiceSuite) SetupTest() {
	s.collectorConfigProtoStream = concurrency.NewValueStream[*sensor.CollectorConfig](nil)
}

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

func (s *networkflowServiceSuite) CollectorConfigValueStream() concurrency.ReadOnlyValueStream[*sensor.CollectorConfig] {
	return s.collectorConfigProtoStream
}

func (s *networkflowServiceSuite) TestSendCollectorConfig() {
    collectorConfig := &sensor.CollectorConfig{
        NetworkConnectionConfig: &sensor.NetworkConnectionConfig{
            EnableExternalIps: true,
        },
    }

    collectorConfigIterator := s.CollectorConfigValueStream().Iterator(false)

    s.collectorConfigProtoStream.Push(collectorConfig)
    collectorConfigIterator = collectorConfigIterator.TryNext()

    mockStream := new(MockStream)

    mockStream.On("Send", mock.AnythingOfType("*sensor.MsgToCollector")).Return(nil).Once()

    service := NewService(s.mockNetworkflowManager)

    err := service.SendCollectorConfig(mockStream, collectorConfigIterator)

    require.NoError(s.T(), err)

    mockStream.AssertExpectations(s.T())
}

func (s *networkflowServiceSuite) TestSendCollectorConfigNoConfig() {

    collectorConfigIterator := s.CollectorConfigValueStream().Iterator(false)

    mockStream := new(MockStream)

    service := NewService(s.mockNetworkflowManager)

    err := service.SendCollectorConfig(mockStream, collectorConfigIterator)

    require.NoError(s.T(), err)

    mockStream.AssertNotCalled(s.T(), "Send", mock.AnythingOfType("*sensor.MsgToCollector"))
}

func (s *networkflowServiceSuite) TestSendCollectorConfigErr() {
    collectorConfig := &sensor.CollectorConfig{
        NetworkConnectionConfig: &sensor.NetworkConnectionConfig{
            EnableExternalIps: true,
        },
    }

    collectorConfigIterator := s.CollectorConfigValueStream().Iterator(false)

    s.collectorConfigProtoStream.Push(collectorConfig)
    collectorConfigIterator = collectorConfigIterator.TryNext()

    mockStream := new(MockStream)

    sendError := errors.New("failed to send collector config")
    mockStream.On("Send", mock.AnythingOfType("*sensor.MsgToCollector")).Return(sendError).Once()

    service := NewService(s.mockNetworkflowManager)

    err := service.SendCollectorConfig(mockStream, collectorConfigIterator)

    require.Error(s.T(), err)

    mockStream.AssertExpectations(s.T())
}
