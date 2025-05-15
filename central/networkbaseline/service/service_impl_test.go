package service

import (
	"context"
	"testing"

	deploymentUtils "github.com/stackrox/rox/central/deployment/utils"
	networkBaselineDSMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	networkBaselineMocks "github.com/stackrox/rox/central/networkbaseline/manager/mocks"
	"github.com/stackrox/rox/central/networkbaseline/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	grpcUtils "github.com/stackrox/rox/pkg/grpc/testutils"
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

func (s *NetworkBaselineServiceTestSuite) getBaselineWithSampleFlow() *storage.NetworkBaseline {
	entityID, entityClusterID := "entity-id", "another-cluster"
	flowIsIngress := true
	flowPort := uint32(8080)
	return testutils.GetBaselineWithCustomDeploymentFlow(testPeerDeploymentName, entityID, entityClusterID, flowIsIngress, flowPort)
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
		testutils.GetBaselineWithCustomDeploymentFlow(
			testPeerDeploymentName,
			entityID,
			baseline.GetClusterId(),
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
	baseline := testutils.GetBaselineWithInternet("cluster", true, 1234)

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
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "external2",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "123.0.0.5",
				Discovered: true,
			},
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  true,
		},
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "external3",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "123.0.0.6",
				Discovered: true,
			},
			Port:     1234,
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  true,
		},

		// not in the baseline
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "external4",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "1.2.3.4",
				Discovered: true,
			},
			Port:     4567, // different port
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  true,
		},
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "external5",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "1.2.3.5",
				Discovered: true,
			},
			Port:     9012, // different port
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  true,
		},
		{
			Entity: &v1.NetworkBaselinePeerEntity{
				Id:         "external6",
				Type:       storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Name:       "1.2.3.6",
				Discovered: true,
			},
			Port:     3456, // different port
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			Ingress:  true,
		},
	}

	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).AnyTimes().Return(baseline, true, nil)
	s.manager.EXPECT().GetExternalNetworkPeers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(externalPeers, nil)

	testCases := []struct {
		request           *v1.NetworkBaselineExternalStatusRequest
		expectedBaseline  []*v1.NetworkBaselinePeerStatus
		expectedAnomalous []*v1.NetworkBaselinePeerStatus
	}{
		{
			request: &v1.NetworkBaselineExternalStatusRequest{
				DeploymentId: "deployment",
				Pagination: &v1.Pagination{
					Limit:  1,
					Offset: 0,
				},
			},

			expectedBaseline: []*v1.NetworkBaselinePeerStatus{
				{
					Peer:   externalPeers[0],
					Status: v1.NetworkBaselinePeerStatus_BASELINE,
				},
			},

			expectedAnomalous: []*v1.NetworkBaselinePeerStatus{
				{
					Peer:   externalPeers[3],
					Status: v1.NetworkBaselinePeerStatus_ANOMALOUS,
				},
			},
		},

		{
			request: &v1.NetworkBaselineExternalStatusRequest{
				DeploymentId: "deployment",
				Pagination: &v1.Pagination{
					Limit:  1,
					Offset: 2,
				},
			},

			expectedBaseline: []*v1.NetworkBaselinePeerStatus{
				{
					Peer:   externalPeers[2],
					Status: v1.NetworkBaselinePeerStatus_BASELINE,
				},
			},

			expectedAnomalous: []*v1.NetworkBaselinePeerStatus{
				{
					Peer:   externalPeers[5],
					Status: v1.NetworkBaselinePeerStatus_ANOMALOUS,
				},
			},
		},

		{
			request: &v1.NetworkBaselineExternalStatusRequest{
				DeploymentId: "deployment",
			},

			expectedBaseline: []*v1.NetworkBaselinePeerStatus{
				{
					Peer:   externalPeers[0],
					Status: v1.NetworkBaselinePeerStatus_BASELINE,
				},
				{
					Peer:   externalPeers[1],
					Status: v1.NetworkBaselinePeerStatus_BASELINE,
				},
				{
					Peer:   externalPeers[2],
					Status: v1.NetworkBaselinePeerStatus_BASELINE,
				},
			},

			expectedAnomalous: []*v1.NetworkBaselinePeerStatus{
				{
					Peer:   externalPeers[3],
					Status: v1.NetworkBaselinePeerStatus_ANOMALOUS,
				},
				{
					Peer:   externalPeers[4],
					Status: v1.NetworkBaselinePeerStatus_ANOMALOUS,
				},
				{
					Peer:   externalPeers[5],
					Status: v1.NetworkBaselinePeerStatus_ANOMALOUS,
				},
			},
		},
	}

	for _, tc := range testCases {
		resp, err := s.service.GetNetworkBaselineStatusForExternalFlows(
			allAllowedCtx, tc.request)

		s.Nil(err)

		protoassert.ElementsMatch(s.T(), tc.expectedAnomalous, resp.Anomalous)
		protoassert.ElementsMatch(s.T(), tc.expectedBaseline, resp.Baseline)
		s.Equal(len(externalPeers), int(resp.TotalAnomalous+resp.TotalBaseline))
	}

}

func TestAuthz(t *testing.T) {
	grpcUtils.AssertAuthzWorks(t, &serviceImpl{})
}
