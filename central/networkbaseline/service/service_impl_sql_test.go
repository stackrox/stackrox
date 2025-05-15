//go:build sql_integration

package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	networkBaselineDS "github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/networkbaseline/manager"
	"github.com/stackrox/rox/central/networkbaseline/testutils"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	networkFlowDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	networkgraphTestutils "github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	testEntityId = "entity-1"

	// the idea is to have some IPs across a filterable
	// range, to test queries against both 1.1.0.0/16
	// and 1.1.1.0/24
	externalIps = []string{
		"1.1.0.1",
		"1.1.0.2",
		"1.1.1.3",
		"1.1.1.4",
		"1.1.1.5",
	}
)

func TestNetworkBaselinePostgres(t *testing.T) {
	suite.Run(t, new(networkBaselineServiceSuite))
}

type networkBaselineServiceSuite struct {
	suite.Suite

	db *pgtest.TestPostgres

	service Service
	manager manager.Manager

	deploymentDataStore deploymentDS.DataStore
	entityDataStore     networkEntityDS.EntityDataStore
	flowDataStore       networkFlowDS.ClusterDataStore
	policyDataStore     networkPolicyDS.DataStore
	baselineDataStore   networkBaselineDS.DataStore
	connectionManager   connection.Manager
}

func (s *networkBaselineServiceSuite) SetupTest() {
	db := pgtest.ForT(s.T())

	var err error

	s.deploymentDataStore, err = deploymentDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.entityDataStore = networkEntityDS.GetTestPostgresDataStore(s.T(), db.DB)

	s.flowDataStore, err = networkFlowDS.GetTestPostgresClusterDataStore(s.T(), db.DB)
	s.NoError(err)

	s.policyDataStore, err = networkPolicyDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.baselineDataStore, err = networkBaselineDS.GetTestPostgresDataStore(s.T(), db.DB)
	s.NoError(err)

	s.connectionManager = connection.ManagerSingleton()

	s.manager, err = manager.New(
		s.baselineDataStore,
		s.entityDataStore,
		s.deploymentDataStore,
		s.policyDataStore,
		s.flowDataStore,
		s.connectionManager,
	)
	s.NoError(err)

	s.service = New(s.baselineDataStore, s.manager)
}

func (s *networkBaselineServiceSuite) setupTablesExternalFlows() {
	// this baseline is created for fixtureconsts.Deployment1
	baseline := testutils.GetBaselineWithInternet(fixtureconsts.Cluster1, false, 1234)
	s.NoError(s.baselineDataStore.UpsertNetworkBaselines(allAllowedCtx, []*storage.NetworkBaseline{baseline}))

	deployment := fixtures.LightweightDeployment()
	s.NoError(s.deploymentDataStore.UpsertDeployment(allAllowedCtx, deployment))

	var entities []*storage.NetworkEntity

	for _, ip := range externalIps {
		cidr := fmt.Sprintf("%s/32", ip)
		id, err := externalsrcs.NewClusterScopedID(fixtureconsts.Cluster1, cidr)
		s.NoError(err)

		entities = append(entities, networkgraphTestutils.GetExtSrcNetworkEntity(
			id.String(),
			ip,
			cidr,
			false,
			fixtureconsts.Cluster1,
			true,
		))
	}

	_, err := s.entityDataStore.CreateExtNetworkEntitiesForCluster(allAllowedCtx, fixtureconsts.Cluster1, entities...)
	s.NoError(err)

	var flows []*storage.NetworkFlow

	ts := time.Now().Add(-10 * time.Minute)

	deploymentEntity := &storage.NetworkEntityInfo{
		Id:   fixtureconsts.Deployment1,
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
	}

	// for every entity, have a baseline flow and an
	// anomalous flow (using a different port)
	for _, entity := range entities {
		flows = append(flows, networkgraphTestutils.GetNetworkFlow(
			deploymentEntity,
			entity.Info,
			1234,
			storage.L4Protocol_L4_PROTOCOL_TCP,
			&ts,
		))

		flows = append(flows, networkgraphTestutils.GetNetworkFlow(
			deploymentEntity,
			entity.Info,
			4567,
			storage.L4Protocol_L4_PROTOCOL_TCP,
			&ts,
		))
	}

	fs, err := s.flowDataStore.GetFlowStore(allAllowedCtx, fixtureconsts.Cluster1)
	s.NoError(err)

	err = fs.UpsertFlows(allAllowedCtx, flows, timestamp.FromGoTime(ts))
	s.NoError(err)
}

func (s *networkBaselineServiceSuite) TestExternalStatus() {
	s.setupTablesExternalFlows()

	req := &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
		Since:        timestamppb.New(time.Now().Add(-1 * time.Hour)),
	}

	resp, err := s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	// expect len(ips) anomalous flows and len(ips) baseline flows
	s.Equal(len(externalIps), len(resp.Anomalous))
	s.Equal(len(externalIps), len(resp.Baseline))
	s.Equal(int32(len(externalIps)), resp.TotalAnomalous)
	s.Equal(int32(len(externalIps)), resp.TotalBaseline)

	for _, anomalous := range resp.Anomalous {
		s.Equal(v1.NetworkBaselinePeerStatus_ANOMALOUS, anomalous.Status)
	}

	for _, baseline := range resp.Baseline {
		s.Equal(v1.NetworkBaselinePeerStatus_BASELINE, baseline.Status)
	}
}

func (s *networkBaselineServiceSuite) TestExternalStatusPagination() {
	s.setupTablesExternalFlows()

	req := &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
		Since:        timestamppb.New(time.Now().Add(-1 * time.Hour)),
		Pagination: &v1.Pagination{
			Offset: 0,
			Limit:  2,
		},
	}

	resp, err := s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Equal(2, len(resp.Anomalous))
	s.Equal(2, len(resp.Baseline))
	s.Equal(int32(len(externalIps)), resp.TotalAnomalous)
	s.Equal(int32(len(externalIps)), resp.TotalBaseline)

	req.Pagination.Offset = 2

	firstPageAnomalous := resp.Anomalous
	firstPageBaseline := resp.Baseline

	resp, err = s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Equal(2, len(resp.Anomalous))
	s.Equal(2, len(resp.Baseline))
	s.Equal(int32(len(externalIps)), resp.TotalAnomalous)
	s.Equal(int32(len(externalIps)), resp.TotalBaseline)

	s.NotElementsMatch(firstPageAnomalous, resp.Anomalous)
	s.NotElementsMatch(firstPageBaseline, resp.Baseline)
}

func (s *networkBaselineServiceSuite) TestExternalStatusNoExternalFlows() {
	s.NoError(s.deploymentDataStore.UpsertDeployment(allAllowedCtx, fixtures.LightweightDeployment()))

	req := &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
	}

	resp, err := s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Empty(resp.Anomalous)
	s.Empty(resp.Baseline)
	s.Equal(int32(0), resp.TotalAnomalous)
	s.Equal(int32(0), resp.TotalBaseline)
}

func (s *networkBaselineServiceSuite) TestExternalStatusCIDRFilter() {
	s.setupTablesExternalFlows()

	req := &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
		Query:        "External Source Address:1.1.1.0/24",
	}

	resp, err := s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Equal(3, len(resp.Anomalous))
	s.Equal(3, len(resp.Baseline))
	s.Equal(int32(3), resp.TotalAnomalous)
	s.Equal(int32(3), resp.TotalBaseline)

	for _, anomaly := range append(resp.Baseline, resp.Anomalous...) {
		// confirm all the CIDRs match the expected range
		s.True(strings.HasPrefix(anomaly.GetPeer().GetEntity().GetName(), "1.1.1"))
	}

	req = &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
		Query:        "External Source Address:1.2.3.4/32", // non existent
	}

	resp, err = s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Equal(0, len(resp.Anomalous))
	s.Equal(0, len(resp.Baseline))
	s.Equal(int32(0), resp.TotalAnomalous)
	s.Equal(int32(0), resp.TotalBaseline)

	// empty query should return everything
	req = &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
		Query:        "", // non existent
	}

	resp, err = s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Equal(len(externalIps), len(resp.Anomalous))
	s.Equal(len(externalIps), len(resp.Baseline))
	s.Equal(int32(len(externalIps)), resp.TotalAnomalous)
	s.Equal(int32(len(externalIps)), resp.TotalBaseline)
}

func (s *networkBaselineServiceSuite) TestExternalStatusSince() {
	s.setupTablesExternalFlows()

	// very recent timestamp
	req := &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
		Since:        timestamppb.New(time.Now().Add(-5 * time.Second)),
	}

	resp, err := s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Equal(0, len(resp.Anomalous))
	s.Equal(0, len(resp.Baseline))
	s.Equal(int32(0), resp.TotalAnomalous)
	s.Equal(int32(0), resp.TotalBaseline)

	// very old timestamp should return everything
	req = &v1.NetworkBaselineExternalStatusRequest{
		DeploymentId: fixtureconsts.Deployment1,
		Since:        timestamppb.New(time.Now().Add(-24 * time.Hour)),
	}

	resp, err = s.service.GetNetworkBaselineStatusForExternalFlows(allAllowedCtx, req)
	s.NoError(err)

	s.Equal(len(externalIps), len(resp.Anomalous))
	s.Equal(len(externalIps), len(resp.Baseline))
	s.Equal(int32(len(externalIps)), resp.TotalAnomalous)
	s.Equal(int32(len(externalIps)), resp.TotalBaseline)
}
