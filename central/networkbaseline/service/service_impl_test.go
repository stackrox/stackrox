package service

import (
	"context"
	"testing"

	deploymentUtils "github.com/stackrox/rox/central/deployment/utils"
	networkBaselineDSMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testPeerDeploymentName = "testPeerDeployment"
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
					Desc: &storage.NetworkEntityInfo_Deployment_{
						Deployment: &storage.NetworkEntityInfo_Deployment{
							Name: testPeerDeploymentName,
						},
					},
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

func (s *NetworkBaselineServiceTestSuite) getBaselineWithInternet(
	entityClusterID string,
	flowIsIngress bool,
	flowPort uint32,
) *storage.NetworkBaseline {
	baseline := fixtures.GetNetworkBaseline()
	baseline.Peers = []*storage.NetworkBaselinePeer{
		{
			Entity: &storage.NetworkEntity{
				Info: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_INTERNET,
					Id:   networkgraph.InternetExternalSourceID,
					Desc: &storage.NetworkEntityInfo_ExternalSource_{
						ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
							Name: testPeerDeploymentName,
						},
					},
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
	otherRequest := request.CloneVT()
	s.Require().NotEmpty(otherRequest.Peers)
	otherRequest.Peers[0].Entity.Id = deploymentUtils.GetMaskedDeploymentID(entityID, testPeerDeploymentName)

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

	// Check the request with the original deployment ID of the baseline peer flags the flow as baseline
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	rsp, err := s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, request)
	s.Nil(err)
	s.Equal(1, len(rsp.Statuses))
	s.Equal(v1.NetworkBaselinePeerStatus_BASELINE, rsp.Statuses[0].GetStatus())

	// Check the request with the masked ID for the baseline peer deployment flags the flow as baseline
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	rsp2, err := s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, otherRequest)
	s.Nil(err)
	s.Equal(1, len(rsp2.Statuses))
	s.Equal(v1.NetworkBaselinePeerStatus_BASELINE, rsp2.Statuses[0].GetStatus())

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
	protoassert.Equal(s.T(), rsp, baseline, "network baselines do not match")
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

func (s *NetworkBaselineServiceTestSuite) TestGetNetworkBaselineStatusForExternalFlows() {
	baseline := s.getBaselineWithInternet("cluster", true, 1234)

	externalPeers := []*v1.NetworkBaselineStatusPeer{
		// In the baseline
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "external1",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "123.0.0.4",
				Discovered: true,
			},
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  true,
		},
		// not in the baseline
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "external2",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "1.2.3.4",
				Discovered: true,
			},
			Port:     4567, // different port
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  true,
		},
	}

	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	s.manager.EXPECT().GetExternalNetworkPeers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(externalPeers, nil)

	resp, err := s.service.GetNetworkBaselineStatusForExternalFlows(
		allAllowedCtx,
		&v1.NetworkBaselineExternalStatusRequest{
			DeploymentId: "deployment",
		})

	s.Nil(err)
	s.Equal(1, len(resp.Anomalous))
	s.Equal(1, len(resp.Baseline))
	s.Equal(1, int(resp.TotalAnomalous))
	s.Equal(1, int(resp.TotalBaseline))
}

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}
