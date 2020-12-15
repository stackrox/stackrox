package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	networkBaselineDSMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

var (
	allAllowedCtx = sac.WithAllAccess(context.Background())
)

func TestNetworkBaselineService(t *testing.T) {
	suite.Run(t, new(NetworkBaselineServiceTestSuite))
}

type NetworkBaselineServiceTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator

	mockCtrl  *gomock.Controller
	baselines *networkBaselineDSMocks.MockDataStore

	service Service
}

func (s *NetworkBaselineServiceTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv("ROX_NETWORK_DETECTION", "true")
	s.mockCtrl = gomock.NewController(s.T())

	s.baselines = networkBaselineDSMocks.NewMockDataStore(s.mockCtrl)
	s.service = New(s.baselines)
}

func (s *NetworkBaselineServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *NetworkBaselineServiceTestSuite) getBaselineWithSampleFlow(
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

func (s *NetworkBaselineServiceTestSuite) TestGetNetworkBaselineStatusForFlows() {
	entityID, entityClusterID := "entity-id", "another-cluster"
	entityType := storage.NetworkEntityInfo_DEPLOYMENT
	flowIsIngress := true
	flowPort := uint32(8080)

	baseline := s.getBaselineWithSampleFlow(entityID, entityClusterID, entityType, flowIsIngress, flowPort)
	request := &v1.NetworkBaselineStatusRequest{
		DeploymentId: baseline.GetDeploymentId(),
		Peers: []*v1.NetworkBaselinePeer{
			{
				Entity: &v1.NetworkBaselinePeerEntity{
					Id:   entityID,
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
				},
				Port:     flowPort,
				Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				Ingress:  flowIsIngress,
			},
		},
	}

	// Id we don't have any baseline, it should throw error since baselines
	// should have been created when deployments are created
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(nil, false, nil)
	_, err := s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, request)
	s.Error(err, "network baseline for the deployment does not exist")

	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	rsp, err := s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, request)
	s.Nil(err)
	s.Equal(1, len(rsp.Statuses))
	s.Equal(v1.NetworkBaselinePeerStatus_BASELINE, rsp.Statuses[0].Status)

	// If we change some baseline details, then the flow should be marked as anomaly
	baseline = s.getBaselineWithSampleFlow(entityID, entityClusterID, entityType, !flowIsIngress, flowPort)
	s.baselines.EXPECT().GetNetworkBaseline(gomock.Any(), gomock.Any()).Return(baseline, true, nil)
	rsp, err = s.service.GetNetworkBaselineStatusForFlows(allAllowedCtx, request)
	s.Nil(err)
	s.Equal(1, len(rsp.Statuses))
	s.Equal(v1.NetworkBaselinePeerStatus_ANOMALOUS, rsp.Statuses[0].Status)
}
