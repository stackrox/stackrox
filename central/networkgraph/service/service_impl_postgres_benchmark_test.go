//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	configDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	networkEntityDS "github.com/stackrox/rox/central/networkgraph/entity/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	networkFlowDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

var (
	globalWriteAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph, resources.Deployment)))

	// all entity IPs of the form 1.2.x.y/32
	baseIp = "1.2"
)

type networkGraphServiceBenchmarks struct {
	db *pgtest.TestPostgres

	service Service

	clusterDataStore     clusterDS.DataStore
	deploymentsDataStore deploymentDS.DataStore
	entityDataStore      networkEntityDS.EntityDataStore
	flowDataStore        networkFlowDS.ClusterDataStore
	policyDataStore      networkPolicyDS.DataStore
	configDataStore      configDS.DataStore
	treeMgr              networktree.Manager
}

func setupBenchmarkForB(b *testing.B) networkGraphServiceBenchmarks {
	var n networkGraphServiceBenchmarks
	db := pgtest.ForT(b)

	var err error
	n.clusterDataStore, err = clusterDS.GetTestPostgresDataStore(b, db.DB)
	require.NoError(b, err)

	n.deploymentsDataStore, err = deploymentDS.GetTestPostgresDataStore(b, db.DB)
	require.NoError(b, err)

	n.entityDataStore, err = networkEntityDS.GetTestPostgresDataStore(b, db.DB)
	require.NoError(b, err)

	n.flowDataStore, err = networkFlowDS.GetTestPostgresClusterDataStore(b, db.DB)
	require.NoError(b, err)

	n.policyDataStore, err = networkPolicyDS.GetTestPostgresDataStore(b, db.DB)
	require.NoError(b, err)

	n.configDataStore, err = configDS.GetTestPostgresDataStore(b, db.DB)
	require.NoError(b, err)

	n.treeMgr = networktree.Singleton()

	n.service = New(
		n.flowDataStore,
		n.entityDataStore,
		n.treeMgr,
		n.deploymentsDataStore,
		n.clusterDataStore,
		n.policyDataStore,
		n.configDataStore,
	)

	n.db = db
	return n
}

// setupTables setups up the database for a large number of deployments all
// communicating with the same flow - it is intended for benchmarks of the
// GetExternalNetworkFlows API endpoint
func (suite *networkGraphServiceBenchmarks) setupTables(b *testing.B) string {
	cidr := "192.168.0.1/32"
	id, err := externalsrcs.NewClusterScopedID(fixtureconsts.Cluster1, cidr)
	require.NoError(b, err)

	entity := testutils.GetExtSrcNetworkEntity(id.String(), cidr, cidr, false, fixtureconsts.Cluster1)

	err = suite.entityDataStore.CreateExternalNetworkEntity(globalWriteAccessCtx, entity, true)
	require.NoError(b, err)

	flows := make([]*storage.NetworkFlow, 0, 2000)

	ts := time.Now()
	for i := 0; i < cap(flows); i++ {
		name := fmt.Sprintf("deployment-%d", i)
		deployment := &storage.Deployment{
			Name:      name,
			Id:        uuid.NewV5FromNonUUIDs(fixtureconsts.Namespace1, name).String(),
			ClusterId: fixtureconsts.Cluster1,
			Namespace: fixtureconsts.Namespace1,
		}

		deploymentEnt := testutils.GetDeploymentNetworkEntity(deployment.Id, deployment.Name)
		err := suite.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment)
		require.NoError(b, err)

		flows = append(flows, testutils.GetNetworkFlow(deploymentEnt, entity.Info, 1337, storage.L4Protocol_L4_PROTOCOL_TCP, &ts))
	}

	flowStore, err := suite.flowDataStore.CreateFlowStore(globalWriteAccessCtx, fixtureconsts.Cluster1)
	require.NoError(b, err)

	err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
	require.NoError(b, err)

	return id.String()
}

// setupTablesForMetadata setups up the database for a large number of entities all
// communicating with a set of deployments - it is intended for benchmarks of the
// GetExternalNetworkFlowsMetadata API endpoint
func (s *networkGraphServiceBenchmarks) setupTablesForMetadata(b *testing.B) {
	entities := make([]*storage.NetworkEntity, 0, 255*255)

	// Generate and store all IPs between 1.2.0.0 -> 1.2.254.254 (65000 entities)
	for x := 1; x < 254; x++ {
		for y := 1; y < 254; y++ {
			cidr := fmt.Sprintf("%s.%d.%d/32", baseIp, x, y)
			id, err := externalsrcs.NewClusterScopedID(fixtureconsts.Cluster1, cidr)
			require.NoError(b, err)

			entity := testutils.GetExtSrcNetworkEntity(id.String(), cidr, cidr, false, fixtureconsts.Cluster1)
			entities = append(entities, entity)
		}
	}

	_, err := s.entityDataStore.CreateExtNetworkEntitiesForCluster(globalWriteAccessCtx, fixtureconsts.Cluster1, entities...)
	require.NoError(b, err)

	// entity 0 communicates with 2000 deployments (1.2.1.1)
	// first 1000 communicate with deployment 1 (1.2.1.1 -> 1.2.4.233)
	// first 10000 communicate with deployment 2 (1.2.1.1 -> 1.2.40.17)

	flows := make([]*storage.NetworkFlow, 0, 13000)

	for i := 0; i < 2000; i++ {
		name := fmt.Sprintf("deployment-%d", i)
		deployment := &storage.Deployment{
			Name:      name,
			Id:        uuid.NewV5FromNonUUIDs(fixtureconsts.Namespace1, name).String(),
			ClusterId: fixtureconsts.Cluster1,
			Namespace: fixtureconsts.Namespace1,
		}

		deploymentEnt := testutils.GetDeploymentNetworkEntity(deployment.Id, deployment.Name)
		err := s.deploymentsDataStore.UpsertDeployment(globalWriteAccessCtx, deployment)
		require.NoError(b, err)

		if i == 0 {
			// deployment-0 talks to the first 1000 entities
			ts := time.Now()
			for _, entity := range entities[:1000] {
				flow := testutils.GetNetworkFlow(deploymentEnt, entity.Info, 1337, storage.L4Protocol_L4_PROTOCOL_TCP, &ts)
				flows = append(flows, flow)
			}
		} else if i == 1 {
			// deployment-1 talks to the first 10000 entities
			ts := time.Now()
			for _, entity := range entities[:10000] {
				flow := testutils.GetNetworkFlow(deploymentEnt, entity.Info, 1337, storage.L4Protocol_L4_PROTOCOL_TCP, &ts)
				flows = append(flows, flow)
			}
		} else {
			// all deployments talk to entity0
			ts := time.Now()
			flow := testutils.GetNetworkFlow(deploymentEnt, entities[0].Info, 1337, storage.L4Protocol_L4_PROTOCOL_TCP, &ts)
			flows = append(flows, flow)
		}
	}

	flowStore, err := s.flowDataStore.CreateFlowStore(globalWriteAccessCtx, fixtureconsts.Cluster1)
	require.NoError(b, err)

	err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
	require.NoError(b, err)
}

func benchmarkGetExternalFlowsMetadata(suite *networkGraphServiceBenchmarks, req *v1.GetExternalNetworkFlowsMetadataRequest) func(b *testing.B) {
	return func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := suite.service.GetExternalNetworkFlowsMetadata(globalWriteAccessCtx, req)
			require.NoError(b, err)
		}
	}
}

func BenchmarkGetExternalFlowsMetadata(b *testing.B) {
	suite := setupBenchmarkForB(b)
	suite.setupTablesForMetadata(b)

	b.Run("paginated first 1000 flows", benchmarkGetExternalFlowsMetadata(
		&suite,
		&v1.GetExternalNetworkFlowsMetadataRequest{
			ClusterId: fixtureconsts.Cluster1,
			Pagination: &v1.Pagination{
				Offset: 0,
				Limit:  1000,
			},
		},
	))

	b.Run("paginated first 10000 flows", benchmarkGetExternalFlowsMetadata(
		&suite,
		&v1.GetExternalNetworkFlowsMetadataRequest{
			ClusterId: fixtureconsts.Cluster1,
			Pagination: &v1.Pagination{
				Offset: 0,
				Limit:  10000,
			},
		},
	))

	b.Run("paginated second 1000 flows", benchmarkGetExternalFlowsMetadata(
		&suite,
		&v1.GetExternalNetworkFlowsMetadataRequest{
			ClusterId: fixtureconsts.Cluster1,
			Pagination: &v1.Pagination{
				Offset: 1000,
				Limit:  1000,
			},
		},
	))

	b.Run("non-paginated with narrow CIDR filter", benchmarkGetExternalFlowsMetadata(
		&suite,
		&v1.GetExternalNetworkFlowsMetadataRequest{
			ClusterId: fixtureconsts.Cluster1,
			Query:     "External Source Address:1.2.4.0/24",
		},
	))

	b.Run("non-paginated with wide CIDR filter", benchmarkGetExternalFlowsMetadata(
		&suite,
		&v1.GetExternalNetworkFlowsMetadataRequest{
			ClusterId: fixtureconsts.Cluster1,
			Query:     "External Source Address:1.2.0.0/16",
		},
	))
}

func benchmarkGetExternalNetworkFlows(suite *networkGraphServiceBenchmarks, req *v1.GetExternalNetworkFlowsRequest) func(b *testing.B) {
	return func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := suite.service.GetExternalNetworkFlows(globalWriteAccessCtx, req)
			require.NoError(b, err)
		}
	}
}

func BenchmarkGetExternalNetworkFlows(b *testing.B) {
	suite := setupBenchmarkForB(b)
	entityId := suite.setupTables(b)

	b.Run("all entity flows", benchmarkGetExternalNetworkFlows(&suite, &v1.GetExternalNetworkFlowsRequest{
		ClusterId: fixtureconsts.Cluster1,
		EntityId:  entityId,
	}))

	b.Run("paginated entity flows first 1000", benchmarkGetExternalNetworkFlows(&suite, &v1.GetExternalNetworkFlowsRequest{
		ClusterId: fixtureconsts.Cluster1,
		EntityId:  entityId,
		Pagination: &v1.Pagination{
			Limit:  1000,
			Offset: 0,
		},
	}))

	b.Run("paginated entity flows second 1000", benchmarkGetExternalNetworkFlows(&suite, &v1.GetExternalNetworkFlowsRequest{
		ClusterId: fixtureconsts.Cluster1,
		EntityId:  entityId,
		Pagination: &v1.Pagination{
			Limit:  1000,
			Offset: 1000,
		},
	}))
}
