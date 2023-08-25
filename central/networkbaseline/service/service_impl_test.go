package service

import (
	"context"
	"testing"

	networkBaselineDSMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	allAllowedCtx = sac.WithAllAccess(context.Background())
)

func TestNetworkBaselineService(t *testing.T) {
	suite.Run(t, new(NetworkBaselineServiceTestSuite))
}

type NetworkBaselineServiceTestSuite struct {
	suite.Suite

	mockCtrl  *gomock.Controller
	baselines *networkBaselineDSMocks.MockDataStore
	manager   *networkBaselineMocks.MockManager

	service Service
}

func (s *NetworkBaselineServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.baselines = networkBaselineDSMocks.NewMockDataStore(s.mockCtrl)
	s.manager = networkBaselineMocks.NewMockManager(s.mockCtrl)
	s.service = New(s.baselines, s.manager)
}

func (s *NetworkBaselineServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkBaselineServiceTestSuite) getBaselineWithCustomFlow(
	entityID, entityClusterID string,
	entityType storage.NetworkEntityInfo_Type,
	flowIsIngress bool,
	flowPort uint32,
) *storage.NetworkBaseline {
	baseline := fixtures.GetNetworkBaseline()
	baseline.Peers = []*storage.NetworkBaselinePeer{
		{
			Entity: &storage.NetworkEntity{
				Info: &storage.NetworkEntityInfo{
					Type: entityType,
					Id:   entityID,
					Desc: nil,
				},
				Scope: &storage.NetworkEntity_Scope{ClusterId: entityClusterID},
			},
			Properties: []*storage.NetworkBaselineConnectionProperties{
				{
					Ingress:  flowIsIngress,
					Port:     flowPort,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				},
			},
		},
	}

	return baseline
}

func (s *NetworkBaselineServiceTestSuite) getBaselineWithSampleFlow() *storage.NetworkBaseline {
	entityID, entityClusterID := "entity-id", "another-cluster"
	entityType := storage.NetworkEntityInfo_DEPLOYMENT
	flowIsIngress := true
	flowPort := uint32(8080)
	return s.getBaselineWithCustomFlow(entityID, entityClusterID, entityType, flowIsIngress, flowPort)
}

func (s *NetworkBaselineServiceTestSuite) TestGetNetworkBaselineStatusForFlows() {
	baseline := s.getBaselineWithSampleFlow()
	peer := baseline.GetPeers()[0]
	port, isIngress := peer.GetProperties()[0].GetPort(), peer.GetProperties()[0].GetIngress()
	entityID := peer.GetEntity().GetInfo().GetId()
	request := &v1.NetworkBaselineStatusRequest{
		DeploymentId: baseline.GetDeploymentId(),
		Peers: []*v1.NetworkBaselineStatusPeer{
			{
				Entity: &v1.NetworkBaselinePeerEntity{
					Id:   entityID,
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
				},
				Port:     port,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				Ingress:  isIngress,
			},
		},
	}

	// If we don't have any baseline, then it is in observation and not created yet, so we will create
	// one
	// First call returns not found
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(nil, false, nil)
	s.manager.EXPECT().CreateNetworkBaseline(request.GetDeploymentId()).Return(nil)
	// Second call returns a baseline that was created in the call to CreateNetworkBaseline
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	testBase, err := s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, request)
	s.Nil(err)
	s.NotNil(testBase)

	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	rsp, err := s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, request)
	s.Nil(err)
	s.Equal(1, len(rsp.Statuses))
	s.Equal(v1.NetworkBaselinePeerStatus_BASELINE, rsp.Statuses[0].Status)

	// If we change some baseline details, then the flow should be marked as anomaly
	baseline =
		s.getBaselineWithCustomFlow(
			entityID,
			baseline.GetClusterId(),
			peer.GetEntity().GetInfo().GetType(),
			!isIngress,
			port)
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	rsp, err = s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, request)
	s.Nil(err)
	s.Equal(1, len(rsp.Statuses))
	s.Equal(v1.NetworkBaselinePeerStatus_ANOMALOUS, rsp.Statuses[0].Status)
}

func (s *NetworkBaselineServiceTestSuite) TestGetNetworkBaseline() {
	baseline := s.getBaselineWithSampleFlow()

	// When no baseline, create one
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(nil, false, nil)
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	s.manager.EXPECT().CreateNetworkBaseline(gomock.Any())
	newBase, err := s.service.GetNetworkBaseline(allAllowedCtx, &v1.ResourceByID{Id: baseline.GetDeploymentId()})
	s.NotNil(newBase)
	s.Nil(err)

	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	rsp, err := s.service.GetNetworkBaseline(allAllowedCtx, &v1.ResourceByID{Id: baseline.GetDeploymentId()})
	s.Nil(err)
	s.Equal(rsp, baseline, "network baselines do not match")
}

func (s *NetworkBaselineServiceTestSuite) TestLockBaseline() {
	sampleID := "sample-ID"
	// Make sure when we call lock we are indeed locking in the manager
	s.manager.EXPECT().ProcessBaselineLockUpdate(gomock.Any(), sampleID, true).Return(nil)
	_, err := s.service.LockNetworkBaseline(allAllowedCtx, &v1.ResourceByID{Id: sampleID})
	s.Nil(err)
	// and when we call unlock we are indeed unlocking in the manager
	s.manager.EXPECT().ProcessBaselineLockUpdate(gomock.Any(), sampleID, false).Return(nil)
	_, err = s.service.UnlockNetworkBaseline(allAllowedCtx, &v1.ResourceByID{Id: sampleID})
	s.Nil(err)
}

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}
