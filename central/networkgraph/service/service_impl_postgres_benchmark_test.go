//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
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

	n.entityDataStore = networkEntityDS.GetTestPostgresDataStore(b, db.DB)
	require.NoError(b, err)

	n.flowDataStore, err = networkFlowDS.GetTestPostgresClusterDataStore(b, db.DB)
	require.NoError(b, err)

	n.policyDataStore, err = networkPolicyDS.GetTestPostgresDataStore(b, db.DB)
	require.NoError(b, err)

	n.configDataStore = configDS.GetTestPostgresDataStore(b, db.DB)
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

func (s *networkGraphServiceBenchmarks) createExternalIps(b *testing.B, baseIp string, n int) []*storage.NetworkEntity {
	entities := make([]*storage.NetworkEntity, 0, n)

	baseIpParts := strings.Split(baseIp, ".")

	if len(baseIpParts) > 4 {
		require.FailNow(b, "Invalid baseIp provided", baseIp)
	}

	for _, part := range baseIpParts {
		if part == "" {
			// possible trailing dot, e.g. 1.2.
			continue
		}

		if i, err := strconv.Atoi(part); err != nil || i < 0 || i > 255 {
			require.FailNow(b, "Invalid octet in baseIp", part)
		}
	}

	// pad zeros if given small IP e.g. 1.2
	for len(baseIpParts) < 4 {
		baseIpParts = append(baseIpParts, "0")
	}

	ip := net.ParseIP(strings.Join(baseIpParts, ".")).To4()
	if ip == nil {
		require.FailNow(b, "Invalid IP parts", baseIpParts)
	}

	base := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])

	for i := 1; i <= n; i++ {
		next := base + uint32(i)

		b1 := byte(next >> 24)
		b2 := byte(next >> 16)
		b3 := byte(next >> 8)
		b4 := byte(next)

		cidr := fmt.Sprintf("%d.%d.%d.%d/32", b1, b2, b3, b4)
		id, err := externalsrcs.NewClusterScopedID(fixtureconsts.Cluster1, cidr)
		require.NoError(b, err)

		entity := testutils.GetExtSrcNetworkEntity(id.String(), cidr, cidr, false, fixtureconsts.Cluster1, true)
		entities = append(entities, entity)
	}

	return entities
}

// First deployment will always talk to all the entities
// All other deployments will talk to an even split of the entities
func (s *networkGraphServiceBenchmarks) createFlows(b *testing.B, numDeployments int, entities []*storage.NetworkEntity) []*storage.NetworkFlow {
	flows := make([]*storage.NetworkFlow, 0)

	chunkSize := len(entities) / numDeployments

	ts := time.Now()
	for i := 0; i < numDeployments; i++ {
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
			// deployment-0 talks to all the entities
			for _, entity := range entities {
				flow := testutils.GetNetworkFlow(deploymentEnt, entity.Info, rand.IntN(65565), storage.L4Protocol_L4_PROTOCOL_TCP, &ts)
				flows = append(flows, flow)
			}
		} else {
			for _, entity := range entities[chunkSize*(i-1) : chunkSize*i] {
				flow := testutils.GetNetworkFlow(deploymentEnt, entity.Info, rand.IntN(65565), storage.L4Protocol_L4_PROTOCOL_TCP, &ts)
				flows = append(flows, flow)
			}
		}
	}

	return flows
}

// setupTables setups up the database for a large number of deployments all
// communicating with the same external IP - it is intended for benchmarks of the
// GetExternalNetworkFlows API endpoint
func (s *networkGraphServiceBenchmarks) setupTables(b *testing.B, numFlows int) string {
	cidr := "192.168.0.1/32"
	id, err := externalsrcs.NewClusterScopedID(fixtureconsts.Cluster1, cidr)
	require.NoError(b, err)

	entity := testutils.GetExtSrcNetworkEntity(id.String(), cidr, cidr, false, fixtureconsts.Cluster1, true)

	err = s.entityDataStore.CreateExternalNetworkEntity(globalWriteAccessCtx, entity, true)
	require.NoError(b, err)

	flows := s.createFlows(b, numFlows, []*storage.NetworkEntity{entity})

	flowStore, err := s.flowDataStore.CreateFlowStore(globalWriteAccessCtx, fixtureconsts.Cluster1)
	require.NoError(b, err)

	_, err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
	require.NoError(b, err)

	return id.String()
}

// setupTablesForMetadata sets up the database for a large number of entities all
// communicating with a set of deployments - it is intended for benchmarks of the
// GetExternalNetworkFlowsMetadata API endpoint
func (s *networkGraphServiceBenchmarks) setupTablesForMetadata(b *testing.B) {
	entities := s.createExternalIps(b, "1.2", 255*255)

	_, err := s.entityDataStore.CreateExtNetworkEntitiesForCluster(globalWriteAccessCtx, fixtureconsts.Cluster1, entities...)
	require.NoError(b, err)

	flows := s.createFlows(b, 2000, entities)

	flowStore, err := s.flowDataStore.CreateFlowStore(globalWriteAccessCtx, fixtureconsts.Cluster1)
	require.NoError(b, err)

	_, err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
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
	entityId := suite.setupTables(b, 2000)

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

func BenchmarkNetworkGraphExternalFlows(b *testing.B) {
	suite := setupBenchmarkForB(b)

	entities := suite.createExternalIps(b, "1.2", 10000)

	_, err := suite.entityDataStore.CreateExtNetworkEntitiesForCluster(globalWriteAccessCtx, fixtureconsts.Cluster1, entities...)
	require.NoError(b, err)

	flows := suite.createFlows(b, 2000, entities)

	flowStore, err := suite.flowDataStore.CreateFlowStore(globalWriteAccessCtx, fixtureconsts.Cluster1)
	require.NoError(b, err)

	_, err = flowStore.UpsertFlows(globalWriteAccessCtx, flows, timestamp.FromGoTime(time.Now()))
	require.NoError(b, err)

	b.Run("all graph", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := suite.service.GetNetworkGraph(globalWriteAccessCtx, &v1.NetworkGraphRequest{
				ClusterId: fixtureconsts.Cluster1,
			})
			require.NoError(b, err)
		}
	})
}
