//go:build sql_integration

package m176tom177

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchemav73 "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/m_176_to_m_177_network_baselines_cidr/networkbaselinestore"
	"github.com/stackrox/rox/migrator/migrations/m_176_to_m_177_network_baselines_cidr/networkentitystore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func deploymentPeer(id string) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Entity: &storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   id,
				Desc: &storage.NetworkEntityInfo_Deployment_{Deployment: &storage.NetworkEntityInfo_Deployment{}},
			},
		},
		Properties: []*storage.NetworkBaselineConnectionProperties{},
	}
}

func externalPeer(id string) *storage.NetworkBaselinePeer {
	return &storage.NetworkBaselinePeer{
		Entity: &storage.NetworkEntity{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   id,
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name:   "provider/region",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{Cidr: ""},
					},
				},
			},
		},
		Properties: []*storage.NetworkBaselineConnectionProperties{},
	}
}

var (
	ctx = sac.WithAllAccess(context.Background())

	ext1Cidr = "42.42.0.0/16"
	ext2Cidr = "24.24.0.0/16"

	dep1 = uuid.NewV4()
	dep2 = uuid.NewV4()
	dep3 = uuid.NewV4()
	dep4 = uuid.NewV4()

	cluster1 = uuid.NewV4()

	networkEntities = []*storage.NetworkEntity{
		{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   "ext1",
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "provider/region",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: ext1Cidr,
						},
					},
				},
			},
		},
		{
			Info: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
				Id:   "ext2",
				Desc: &storage.NetworkEntityInfo_ExternalSource_{
					ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
						Name: "provider/region",
						Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
							Cidr: ext2Cidr,
						},
					},
				},
			},
		},
	}

	networkBaselines = []*storage.NetworkBaseline{
		{
			DeploymentId: dep1.String(),
			ClusterId:    cluster1.String(),
			Namespace:    "ns1",
			Peers: []*storage.NetworkBaselinePeer{
				deploymentPeer(dep2.String()),
				externalPeer("ext1"),
			},
			DeploymentName: "Has ext1 as peer",
		},
		{
			DeploymentId: dep2.String(),
			ClusterId:    cluster1.String(),
			Namespace:    "ns1",
			Peers: []*storage.NetworkBaselinePeer{
				deploymentPeer(dep1.String()),
			},
			DeploymentName: "No external peers",
		},
		{
			DeploymentId: dep3.String(),
			ClusterId:    cluster1.String(),
			Namespace:    "ns1",
			ForbiddenPeers: []*storage.NetworkBaselinePeer{
				externalPeer("ext1"),
			},
			DeploymentName: "Has ext as forbidden peer",
		},
		{
			DeploymentId: dep4.String(),
			ClusterId:    cluster1.String(),
			Namespace:    "ns1",
			Peers: []*storage.NetworkBaselinePeer{
				externalPeer("ext1"),
				externalPeer("ext2"),
			},
			DeploymentName: "Has two (ext1 and ext2) as peers",
		},
	}
)

type networkBaselineCidrTestSuite struct {
	suite.Suite
	db *pghelper.TestPostgres
}

func TestNetworkBaselinesCIDR(t *testing.T) {
	suite.Run(t, new(networkBaselineCidrTestSuite))
}

func (n *networkBaselineCidrTestSuite) SetupSuite() {
	n.db = pghelper.ForT(n.T(), true)
	pgutils.CreateTableFromModel(ctx, n.db.GetGormDB(), frozenSchemav73.CreateTableNetworkBaselinesStmt)
	pgutils.CreateTableFromModel(ctx, n.db.GetGormDB(), frozenSchemav73.CreateTableNetworkEntitiesStmt)
}

func (n *networkBaselineCidrTestSuite) TearDownSuite() {
	n.db.Teardown(n.T())
}

func (n *networkBaselineCidrTestSuite) peerShouldHaveCidrBlock(peer *storage.NetworkBaselinePeer) {
	info := peer.GetEntity().GetInfo()
	if info.GetType() == storage.NetworkEntityInfo_EXTERNAL_SOURCE {
		n.Require().NotNilf(info.GetExternalSource(), "should not have external source nil if type is NetworkEntityInfo_EXTERNAL_SOURCE")
		externalSource := info.GetExternalSource()
		n.Assert().Equal(externalSource.GetName(), "provider/region")
		switch info.GetId() {
		case "ext1":
			n.Assert().Equal(externalSource.GetCidr(), ext1Cidr)
		case "ext2":
			n.Assert().Equal(externalSource.GetCidr(), ext2Cidr)
		default:
			n.Fail("external source should be ext1 or ext2, instead it is: %s", info.GetId())
		}
	}
}

func (n *networkBaselineCidrTestSuite) Test_Migration() {
	baselineStore := networkbaselinestore.New(n.db.DB)
	networkEntityStore := networkentitystore.New(n.db.DB)

	n.Require().NoError(networkEntityStore.UpsertMany(ctx, networkEntities))
	n.Require().NoError(baselineStore.UpsertMany(ctx, networkBaselines))

	dbs := &types.Databases{
		GormDB:     n.db.GetGormDB(),
		PostgresDB: n.db.DB,
	}

	n.Require().NoError(migration.Run(dbs))

	n.NoError(baselineStore.Walk(ctx, func(baseline *storage.NetworkBaseline) error {
		n.Run(fmt.Sprintf("Baseline: %s", baseline.DeploymentName), func() {
			for _, peer := range baseline.GetPeers() {
				n.peerShouldHaveCidrBlock(peer)
			}

			for _, peer := range baseline.GetForbiddenPeers() {
				n.peerShouldHaveCidrBlock(peer)
			}
		})
		return nil
	}))
}
