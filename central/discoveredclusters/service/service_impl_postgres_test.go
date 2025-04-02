//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/convert/storagetov1"
	"github.com/stackrox/rox/central/convert/typetostorage"
	"github.com/stackrox/rox/central/discoveredclusters/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestServicePostgres(t *testing.T) {
	suite.Run(t, new(servicePostgresTestSuite))
}

type servicePostgresTestSuite struct {
	suite.Suite

	readCtx   context.Context
	writeCtx  context.Context
	pool      *pgtest.TestPostgres
	datastore datastore.DataStore
	service   Service
}

func (s *servicePostgresTestSuite) SetupTest() {
	s.readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)
	s.datastore = datastore.GetTestPostgresDataStore(s.T(), s.pool)
	s.service = newService(s.datastore)
}

func (s *servicePostgresTestSuite) TestCount() {
	s.addDiscoveredClusters(50)

	// 1. Count discovered clusters without providing a query filter.
	resp, err := s.service.CountDiscoveredClusters(s.readCtx, &v1.CountDiscoveredClustersRequest{})
	s.Require().NoError(err)
	s.Assert().Equal(int32(50), resp.GetCount())

	// 2.a. Filter discovered clusters based on the name - no match.
	resp, err = s.service.CountDiscoveredClusters(s.readCtx, &v1.CountDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Names: []string{"this name does not exist"},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(0), resp.GetCount())

	// 2.b. Filter discovered clusters based on the name - one match.
	resp, err = s.service.CountDiscoveredClusters(s.readCtx, &v1.CountDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Names: []string{"my-cluster-00"},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(1), resp.GetCount())

	// 3. Filter discovered clusters based on the type.
	resp, err = s.service.CountDiscoveredClusters(s.readCtx, &v1.CountDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Types: []v1.DiscoveredCluster_Metadata_Type{v1.DiscoveredCluster_Metadata_GKE},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(25), resp.GetCount())

	// 4. Filter discovered clusters based on the status.
	resp, err = s.service.CountDiscoveredClusters(s.readCtx, &v1.CountDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Statuses: []v1.DiscoveredCluster_Status{v1.DiscoveredCluster_STATUS_SECURED},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(25), resp.GetCount())

	// 5. Filter discovered clusters based on the cloud source id.
	resp, err = s.service.CountDiscoveredClusters(s.readCtx, &v1.CountDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			SourceIds: []string{"fb28231c-54d1-41e1-9551-ede4c0e15c6c"},
		},
	})
	s.Require().NoError(err)
	s.Assert().Equal(int32(25), resp.GetCount())
}

func (s *servicePostgresTestSuite) TestGetDiscoveredCluster() {
	discoveredClusters := s.addDiscoveredClusters(1)

	resp, err := s.service.GetDiscoveredCluster(s.readCtx, &v1.GetDiscoveredClusterRequest{
		Id: discoveredClusters[0].GetId(),
	})
	s.Require().NoError(err)
	protoassert.Equal(s.T(), discoveredClusters[0], resp.GetCluster())
}

func (s *servicePostgresTestSuite) TestListDiscoveredClusters() {
	discoveredClusters := s.addDiscoveredClusters(50)

	// 1. Count discovered clusters without providing a query filter.
	resp, err := s.service.ListDiscoveredClusters(s.readCtx, &v1.ListDiscoveredClustersRequest{})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), discoveredClusters, resp.GetClusters())

	// 2.a. Filter discovered clusters based on the name - no match.
	resp, err = s.service.ListDiscoveredClusters(s.readCtx, &v1.ListDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Names: []string{"this name does not exist"},
		},
	})
	s.Require().NoError(err)
	s.Assert().Empty(resp.GetClusters())

	// 2.b. Filter discovered clusters based on the name - one match.
	resp, err = s.service.ListDiscoveredClusters(s.readCtx, &v1.ListDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Names: []string{"my-cluster-00"},
		},
	})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), []*v1.DiscoveredCluster{discoveredClusters[0]}, resp.GetClusters())

	// 3. Filter discovered clusters based on the type.
	resp, err = s.service.ListDiscoveredClusters(s.readCtx, &v1.ListDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Types: []v1.DiscoveredCluster_Metadata_Type{v1.DiscoveredCluster_Metadata_GKE},
		},
	})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), discoveredClusters[0:25], resp.GetClusters())

	// 4. Filter discovered clusters based on the status.
	resp, err = s.service.ListDiscoveredClusters(s.readCtx, &v1.ListDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			Statuses: []v1.DiscoveredCluster_Status{v1.DiscoveredCluster_STATUS_SECURED},
		},
	})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), discoveredClusters[0:25], resp.GetClusters())

	// 5. Filter discovered clusters based on the cloud source id.
	resp, err = s.service.ListDiscoveredClusters(s.readCtx, &v1.ListDiscoveredClustersRequest{
		Filter: &v1.DiscoveredClustersFilter{
			SourceIds: []string{"fb28231c-54d1-41e1-9551-ede4c0e15c6c"},
		},
	})
	s.Require().NoError(err)
	protoassert.SlicesEqual(s.T(), discoveredClusters[0:25], resp.GetClusters())
}

func (s *servicePostgresTestSuite) addDiscoveredClusters(num int) []*v1.DiscoveredCluster {
	fakeClusters := fixtures.GetManyDiscoveredClusters(num)
	s.Require().NoError(s.datastore.UpsertDiscoveredClusters(s.writeCtx, fakeClusters...))
	v1Clusters := []*v1.DiscoveredCluster{}
	for _, dc := range fakeClusters {
		storageCluster := typetostorage.DiscoveredCluster(dc)
		v1Clusters = append(v1Clusters, storagetov1.DiscoveredCluster(storageCluster))
	}
	return v1Clusters
}
