//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	flow "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
)

type EntityFullStoreTestSuite struct {
	suite.Suite

	tested    store.EntityStore
	flowStore flow.FlowDataStore

	allAccess context.Context
	db        *pgtest.TestPostgres
}

func TestEntityFullStore(t *testing.T) {
	suite.Run(t, new(EntityFullStoreTestSuite))
}

// SetupSuite runs before any tests
func (s *EntityFullStoreTestSuite) SetupSuite() {
	s.db = pgtest.ForT(s.T())
	s.tested = NewFullStore(s.db.DB)

	clusterDataStore, err := flow.GetTestPostgresClusterDataStore(s.T(), s.db.DB)
	s.NoError(err)

	s.allAccess = sac.WithAllAccess(context.Background())

	s.flowStore, err = clusterDataStore.CreateFlowStore(s.allAccess, fixtureconsts.ClusterFake1)
	s.NoError(err)
}

func (s *EntityFullStoreTestSuite) TeardownSuite() {
	s.db.Teardown(s.T())
}

// TestPruning verifies the FullEntityStore pruning behavior
func (s *EntityFullStoreTestSuite) TestPruning() {
	t1 := time.Now().UTC()
	extEntity1 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{1, 2, 3, 4, 32})).ToProto()
	extEntity2 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{2, 3, 4, 5, 32})).ToProto()
	// extEntity3 is not linked to any flow
	extEntity3 := networkgraph.DiscoveredExternalEntity(net.IPNetworkFromCIDRBytes([]byte{3, 4, 5, 6, 32})).ToProto()
	flows := []*storage.NetworkFlow{
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  extEntity1,
				DstEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment1},
				DstPort:    1,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t1),
			ClusterId:         fixtureconsts.ClusterFake1,
		},
		{
			Props: &storage.NetworkFlowProperties{
				SrcEntity:  &storage.NetworkEntityInfo{Type: storage.NetworkEntityInfo_DEPLOYMENT, Id: fixtureconsts.Deployment2},
				DstEntity:  extEntity2,
				DstPort:    2,
				L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
			},
			LastSeenTimestamp: protoconv.ConvertTimeToTimestamp(t1),
			ClusterId:         fixtureconsts.ClusterFake1,
		},
	}
	var err error

	updateTS := timestamp.Now() - 1000000
	err = s.flowStore.UpsertFlows(s.allAccess, flows, updateTS)
	s.NoError(err, "upsert should succeed on first insert")

	err = s.tested.UpsertMany(s.allAccess, []*storage.NetworkEntity{
		{Info: extEntity1},
		{Info: extEntity2},
		{Info: extEntity3},
	})
	s.NoError(err, "upsert should succeed on first insert")

	rowsAffected, err := s.tested.RemoveOrphanedEntities(s.allAccess)

	s.NoError(err)
	s.Equal(int64(1), rowsAffected, "Only one entity is expected to be pruned")

	ids, err := s.tested.GetIDs(s.allAccess)

	s.NoError(err)
	s.ElementsMatch(ids, []string{extEntity1.GetId(), extEntity2.GetId()})
}
