//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var (
	someNamespace = "someNamespace"
)

func TestClusterDatastoreSAC(t *testing.T) {
	suite.Run(t, new(clusterDatastoreSACSuite))
}

type clusterDatastoreSACSuite struct {
	suite.Suite

	datastore DataStore

	// Elements for postgres mode
	pgtestbase *pgtest.TestPostgres

	optionsMap searchPkg.OptionsMap

	testContexts   map[string]context.Context
	testClusterIDs []string
}

func (s *clusterDatastoreSACSuite) SetupSuite() {
	var err error
	s.pgtestbase = pgtest.ForT(s.T())
	s.NotNil(s.pgtestbase)
	s.datastore, err = GetTestPostgresDataStore(s.T(), s.pgtestbase.DB)
	s.Require().NoError(err)
	s.optionsMap = schema.ClustersSchema.OptionsMap
	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Cluster)
}

func (s *clusterDatastoreSACSuite) TearDownSuite() {
	s.pgtestbase.DB.Close()
}

func (s *clusterDatastoreSACSuite) SetupTest() {
	s.testClusterIDs = make([]string, 0)
}

func (s *clusterDatastoreSACSuite) TearDownTest() {
	for _, id := range s.testClusterIDs {
		s.deleteCluster(id)
	}
}

func (s *clusterDatastoreSACSuite) deleteCluster(id string) {
	exists, err := s.datastore.Exists(s.testContexts[testutils.UnrestrictedReadCtx], id)
	if !exists || err != nil {
		return
	}
	doneSignal := concurrency.NewSignal()
	err = s.datastore.RemoveCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], id, &doneSignal)
	s.NoError(err)
	<-doneSignal.Done()
}

type mutiClusterTest struct {
	Name                 string
	Context              context.Context
	ExpectedClusterIDs   []string
	ExpectedClusterNames []string
}

func getMultiClusterTestCases(baseContext context.Context, clusterID1 string, clusterID2 string, otherClusterID string) []mutiClusterTest {
	return []mutiClusterTest{
		{
			Name: "Cluster full read-only",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster))),
			ExpectedClusterIDs:   []string{clusterID1, clusterID2},
			ExpectedClusterNames: []string{testconsts.Cluster1, testconsts.Cluster2},
		},
		{
			Name: "Cluster full read-write",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster))),
			ExpectedClusterIDs:   []string{clusterID1, clusterID2},
			ExpectedClusterNames: []string{testconsts.Cluster1, testconsts.Cluster2},
		},
		{
			Name: "Cluster read Cluster1",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID1))),
			ExpectedClusterIDs:   []string{clusterID1},
			ExpectedClusterNames: []string{testconsts.Cluster1},
		},
		{
			Name: "Cluster partial read Cluster1",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID1),
					sac.NamespaceScopeKeys(someNamespace))),
			ExpectedClusterIDs:   []string{clusterID1},
			ExpectedClusterNames: []string{testconsts.Cluster1},
		},
		{
			Name: "Cluster read Cluster2",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID2))),
			ExpectedClusterIDs:   []string{clusterID2},
			ExpectedClusterNames: []string{testconsts.Cluster2},
		},
		{
			Name: "Cluster partial read Cluster2",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID2),
					sac.NamespaceScopeKeys(someNamespace))),
			ExpectedClusterIDs:   []string{clusterID2},
			ExpectedClusterNames: []string{testconsts.Cluster2},
		},
		{
			Name: "Cluster read other cluster",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(otherClusterID))),
			ExpectedClusterIDs:   []string{},
			ExpectedClusterNames: []string{},
		},
		{
			Name: "Cluster partial read other cluster",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(otherClusterID),
					sac.NamespaceScopeKeys(someNamespace))),
			ExpectedClusterIDs:   []string{},
			ExpectedClusterNames: []string{},
		},
		{
			Name: "Cluster read Cluster1 and Cluster2",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID1, clusterID2))),
			ExpectedClusterIDs:   []string{clusterID1, clusterID2},
			ExpectedClusterNames: []string{testconsts.Cluster1, testconsts.Cluster2},
		},
		{
			Name: "Cluster partial read Cluster1 and Cluster2",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID1, clusterID2),
					sac.NamespaceScopeKeys(someNamespace))),
			ExpectedClusterIDs:   []string{clusterID1, clusterID2},
			ExpectedClusterNames: []string{testconsts.Cluster1, testconsts.Cluster2},
		},
		{
			Name: "Cluster read Cluster1 and some other cluster",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID1, otherClusterID))),
			ExpectedClusterIDs:   []string{clusterID1},
			ExpectedClusterNames: []string{testconsts.Cluster1},
		},
		{
			Name: "Cluster partial read Cluster1 and some other cluster",
			Context: sac.WithGlobalAccessScopeChecker(baseContext,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(clusterID1, otherClusterID),
					sac.NamespaceScopeKeys(someNamespace))),
			ExpectedClusterIDs:   []string{clusterID1},
			ExpectedClusterNames: []string{testconsts.Cluster1},
		},
	}
}

func (s *clusterDatastoreSACSuite) TestAddCluster() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbAdd)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			cluster := fixtures.GetCluster(testconsts.Cluster2)
			// Erase cluster Id to allow insertion
			cluster.Id = ""
			id, err := s.datastore.AddCluster(ctx, cluster)
			if len(id) > 0 {
				s.testClusterIDs = append(s.testClusterIDs, id)
				defer s.deleteCluster(id)
			}
			if c.ExpectError {
				s.ErrorIs(err, c.ExpectedError)
				s.Equal("", id)
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *clusterDatastoreSACSuite) TestExists() {
	cluster := fixtures.GetCluster(testconsts.Cluster2)
	clusterID, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	cases := testutils.GenericClusterSACGetTestCases(context.Background(), s.T(), clusterID, "not-"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			exists, err := s.datastore.Exists(ctx, clusterID)
			s.NoError(err)
			s.Equal(c.ExpectedFound, exists)
		})
	}
}

func (s *clusterDatastoreSACSuite) TestGetCluster() {
	cluster := fixtures.GetCluster(testconsts.Cluster2)
	clusterID, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	cluster.Id = clusterID
	cluster.Priority = 1

	cases := testutils.GenericClusterSACGetTestCases(context.Background(), s.T(), clusterID, testconsts.Cluster3, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			fetchedCluster, found, err := s.datastore.GetCluster(ctx, clusterID)
			s.NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Require().NotNil(fetchedCluster)
				s.Equal(*cluster, *fetchedCluster)
			} else {
				s.False(found)
				s.Nil(fetchedCluster)
			}
		})
	}
}

func (s *clusterDatastoreSACSuite) TestGetClusterName() {
	cluster := fixtures.GetCluster(testconsts.Cluster2)
	clusterID, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	cluster.Id = clusterID
	cluster.Priority = 1

	cases := testutils.GenericClusterSACGetTestCases(context.Background(), s.T(), clusterID, "not-"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			clusterName, found, err := s.datastore.GetClusterName(ctx, clusterID)
			s.NoError(err)
			if c.ExpectedFound {
				s.True(found)
				s.Equal(cluster.GetName(), clusterName)
			} else {
				s.False(found)
				s.Equal(0, len(clusterName))
			}
		})
	}
}

func (s *clusterDatastoreSACSuite) TestGetClusters() {
	cluster1 := fixtures.GetCluster(testconsts.Cluster1)
	clusterID1, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster1)
	defer s.deleteCluster(clusterID1)
	s.Require().NoError(err)
	cluster1.Id = clusterID1
	cluster2 := fixtures.GetCluster(testconsts.Cluster2)
	clusterID2, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster2)
	defer s.deleteCluster(clusterID2)
	s.Require().NoError(err)
	cluster2.Id = clusterID2
	otherClusterID := testconsts.Cluster3

	cases := getMultiClusterTestCases(context.Background(), clusterID1, clusterID2, otherClusterID)

	for _, c := range cases {
		s.Run(c.Name, func() {
			ctx := c.Context
			clusters, err := s.datastore.GetClusters(ctx)
			s.NoError(err)
			clusterNames := make([]string, 0, len(clusters))
			for _, cluster := range clusters {
				clusterNames = append(clusterNames, cluster.GetName())
			}
			s.ElementsMatch(c.ExpectedClusterNames, clusterNames)
		})
	}
}

func (s *clusterDatastoreSACSuite) TestRemoveCluster() {
	// The cluster ID is generated at cluster insertion.
	// In order to have a context having the cluster in or out of scope, the context has to be generated
	// after the insertion. Considering each case is trying to remove the cluster, addition of the cluster
	// has to be performed within the test case. This means each test case has to be written independently.
	baseContext := context.Background()
	s.Run("(full) read-only cannot delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.ErrorIs(removeErr, sac.ErrResourceAccessDenied)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was not removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.True(exists)
		s.NoError(err)
	})
	s.Run("full read-write can delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.NoError(removeErr)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.False(exists)
		s.NoError(err)
	})
	s.Run("read-write on wrong cluster cannot delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys("not"+clusterID)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.ErrorIs(removeErr, sac.ErrResourceAccessDenied)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was not removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.True(exists)
		s.NoError(err)
	})
	s.Run("read-write on wrong cluster and partial namespace access cannot delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys("not"+clusterID),
				sac.NamespaceScopeKeys(someNamespace)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.ErrorIs(removeErr, sac.ErrResourceAccessDenied)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was not removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.True(exists)
		s.NoError(err)
	})
	s.Run("read-write on right cluster can delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys(clusterID)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.NoError(removeErr)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.False(exists)
		s.NoError(err)
	})
	s.Run("read-write on the right cluster and partial namespace access cannot delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys(clusterID),
				sac.NamespaceScopeKeys(someNamespace)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.ErrorIs(removeErr, sac.ErrResourceAccessDenied)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was not removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.True(exists)
		s.NoError(err)
	})
	s.Run("read-write on at least the right cluster can delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys(clusterID, testconsts.Cluster3)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.NoError(removeErr)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.False(exists)
		s.NoError(err)
	})
	s.Run("read-write on at least the right cluster but partial namespace access cannot delete", func() {
		doneSignal := concurrency.NewSignal()
		globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
		cluster := fixtures.GetCluster(testconsts.Cluster2)
		clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, cluster)
		defer s.deleteCluster(clusterID)
		s.Require().NoError(err)
		ctx := sac.WithGlobalAccessScopeChecker(baseContext,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Cluster),
				sac.ClusterScopeKeys(clusterID, "otherthan"+clusterID),
				sac.NamespaceScopeKeys(someNamespace)))
		removeErr := s.datastore.RemoveCluster(ctx, clusterID, &doneSignal)
		s.ErrorIs(removeErr, sac.ErrResourceAccessDenied)
		if removeErr == nil {
			// Wait for post-delete asynchronous cleanup in case the removal was allowed
			<-doneSignal.Done()
		}
		// Verify cluster was not removed
		exists, err := s.datastore.Exists(globalReadWriteCtx, clusterID)
		s.True(exists)
		s.NoError(err)
	})
}

func (s *clusterDatastoreSACSuite) TestUpdateCluster() {
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	oldCluster := fixtures.GetCluster(testconsts.Cluster2)
	clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, oldCluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	oldCluster.Id = clusterID
	oldAPIEndpoint := oldCluster.GetCentralApiEndpoint()
	newAPIEndpoint := "some.central.endpoint:443"
	newCluster := fixtures.GetCluster(testconsts.Cluster2)
	newCluster.Id = clusterID
	newCluster.CentralApiEndpoint = newAPIEndpoint
	newCluster.Priority = 1

	cases := testutils.GenericClusterSACWriteTestCases(context.Background(), s.T(), testutils.VerbUpdate, clusterID, "not"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			preUpdateCluster, preUpdateFound, preUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(preUpdateErr)
			s.True(preUpdateFound)
			updateErr := s.datastore.UpdateCluster(ctx, newCluster)
			postUpdateCluster, postUpdateFound, postUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(postUpdateErr)
			s.True(postUpdateFound)
			if c.ExpectError {
				s.Equal(oldAPIEndpoint, preUpdateCluster.GetCentralApiEndpoint())
				s.ErrorIs(updateErr, c.ExpectedError)
				s.Equal(oldAPIEndpoint, postUpdateCluster.GetCentralApiEndpoint())
			} else {
				s.Equal(oldAPIEndpoint, preUpdateCluster.GetCentralApiEndpoint())
				s.NoError(updateErr)
				s.Equal(newAPIEndpoint, postUpdateCluster.GetCentralApiEndpoint())
			}
			// Revert to pre-test state
			err := s.datastore.UpdateCluster(globalReadWriteCtx, oldCluster)
			s.Require().NoError(err)
			fetchedCluster, found, err := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.Require().NoError(err)
			s.Require().True(found)
			s.Require().Equal(oldAPIEndpoint, fetchedCluster.GetCentralApiEndpoint())
		})
	}
}

func (s *clusterDatastoreSACSuite) TestUpdateClusterCertExpiryStatus() {
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	oldCluster := fixtures.GetCluster(testconsts.Cluster2)
	oldSensorExpiry := &types.Timestamp{Seconds: 1659478729}
	oldSensorCertNotBefore := &types.Timestamp{Seconds: 1658458729}
	oldCertExpiryStatus := &storage.ClusterCertExpiryStatus{
		SensorCertExpiry:    oldSensorExpiry,
		SensorCertNotBefore: oldSensorCertNotBefore,
	}
	if oldCluster.Status == nil {
		oldCluster.Status = &storage.ClusterStatus{
			SensorVersion:         "3.71.x-88-g9798e675e5-dirty",
			DEPRECATEDLastContact: nil,
			ProviderMetadata:      nil,
			OrchestratorMetadata:  nil,
			UpgradeStatus:         nil,
			CertExpiryStatus:      oldCertExpiryStatus,
		}
	}
	newSensorExpiry := &types.Timestamp{Seconds: 1659479729}
	newSensorCertNotBefore := &types.Timestamp{Seconds: 1658468729}
	newCertExpiryStatus := &storage.ClusterCertExpiryStatus{
		SensorCertExpiry:    newSensorExpiry,
		SensorCertNotBefore: newSensorCertNotBefore,
	}
	clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, oldCluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	oldCluster.Id = clusterID

	cases := testutils.GenericClusterSACWriteTestCases(context.Background(), s.T(), "update certificate expiry status", clusterID, "not"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			preUpdateCluster, preUpdateFound, preUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(preUpdateErr)
			s.True(preUpdateFound)
			updateErr := s.datastore.UpdateClusterCertExpiryStatus(ctx, clusterID, newCertExpiryStatus)
			postUpdateCluster, postUpdateFound, postUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(postUpdateErr)
			s.True(postUpdateFound)
			if c.ExpectError {
				s.Equal(*oldSensorExpiry, *preUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertExpiry())
				s.Equal(*oldSensorCertNotBefore, *preUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertNotBefore())
				s.ErrorIs(updateErr, c.ExpectedError)
				s.Equal(*oldSensorExpiry, *postUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertExpiry())
				s.Equal(*oldSensorCertNotBefore, *postUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertNotBefore())
			} else {
				s.Equal(*oldSensorExpiry, *preUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertExpiry())
				s.Equal(*oldSensorCertNotBefore, *preUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertNotBefore())
				s.NoError(updateErr)
				s.Equal(*newSensorExpiry, *postUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertExpiry())
				s.Equal(*newSensorCertNotBefore, *postUpdateCluster.GetStatus().GetCertExpiryStatus().GetSensorCertNotBefore())
			}
			// Revert to pre-test state
			err := s.datastore.UpdateClusterCertExpiryStatus(globalReadWriteCtx, clusterID, oldCertExpiryStatus)
			s.Require().NoError(err)
			fetchedCluster, found, err := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.Require().NoError(err)
			s.Require().True(found)
			s.Require().Equal(*oldSensorExpiry, *fetchedCluster.GetStatus().GetCertExpiryStatus().GetSensorCertExpiry())
			s.Require().Equal(*oldSensorCertNotBefore, *fetchedCluster.GetStatus().GetCertExpiryStatus().GetSensorCertNotBefore())
		})
	}
}

func (s *clusterDatastoreSACSuite) TestUpdateClusterHealth() {
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	oldCluster := fixtures.GetCluster(testconsts.Cluster2)
	oldLastContact := &types.Timestamp{Seconds: 1659478729}
	oldHealthStatus := &storage.ClusterHealthStatus{
		Id:                           "",
		CollectorHealthInfo:          nil,
		AdmissionControlHealthInfo:   nil,
		ScannerHealthInfo:            nil,
		SensorHealthStatus:           storage.ClusterHealthStatus_HEALTHY,
		CollectorHealthStatus:        storage.ClusterHealthStatus_UNHEALTHY,
		OverallHealthStatus:          storage.ClusterHealthStatus_UNHEALTHY,
		AdmissionControlHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		ScannerHealthStatus:          storage.ClusterHealthStatus_HEALTHY,
		LastContact:                  oldLastContact,
		HealthInfoComplete:           true,
	}
	oldCluster.HealthStatus = oldHealthStatus
	newLastContact := &types.Timestamp{Seconds: 1659479729}
	newHealthStatus := &storage.ClusterHealthStatus{
		Id:                           "",
		CollectorHealthInfo:          nil,
		AdmissionControlHealthInfo:   nil,
		ScannerHealthInfo:            nil,
		SensorHealthStatus:           storage.ClusterHealthStatus_HEALTHY,
		CollectorHealthStatus:        storage.ClusterHealthStatus_HEALTHY,
		OverallHealthStatus:          storage.ClusterHealthStatus_HEALTHY,
		AdmissionControlHealthStatus: storage.ClusterHealthStatus_HEALTHY,
		ScannerHealthStatus:          storage.ClusterHealthStatus_HEALTHY,
		LastContact:                  newLastContact,
		HealthInfoComplete:           true,
	}
	clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, oldCluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	oldCluster.Id = clusterID
	oldCluster.HealthStatus = oldHealthStatus

	var cases map[string]testutils.ClusterSACCrudTestCase
	testedVerb := "update cluster health"
	cases = testutils.GenericGlobalClusterSACWriteTestCases(context.Background(), s.T(), testedVerb, clusterID, "not"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			preUpdateCluster, preUpdateFound, preUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(preUpdateErr)
			s.True(preUpdateFound)
			updateErr := s.datastore.UpdateClusterHealth(ctx, clusterID, newHealthStatus)
			postUpdateCluster, postUpdateFound, postUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(postUpdateErr)
			s.True(postUpdateFound)
			if c.ExpectError {
				s.Equal(oldHealthStatus.GetCollectorHealthStatus(), preUpdateCluster.GetHealthStatus().GetCollectorHealthStatus())
				s.Equal(oldHealthStatus.GetOverallHealthStatus(), preUpdateCluster.GetHealthStatus().GetOverallHealthStatus())
				s.Equal(*oldHealthStatus.GetLastContact(), *preUpdateCluster.GetHealthStatus().GetLastContact())
				s.ErrorIs(updateErr, c.ExpectedError)
				s.Equal(oldHealthStatus.GetCollectorHealthStatus(), postUpdateCluster.GetHealthStatus().GetCollectorHealthStatus())
				s.Equal(oldHealthStatus.GetOverallHealthStatus(), postUpdateCluster.GetHealthStatus().GetOverallHealthStatus())
				s.Equal(*oldHealthStatus.GetLastContact(), *preUpdateCluster.GetHealthStatus().GetLastContact())
			} else {
				s.Equal(oldHealthStatus.GetCollectorHealthStatus(), preUpdateCluster.GetHealthStatus().GetCollectorHealthStatus())
				s.Equal(oldHealthStatus.GetOverallHealthStatus(), preUpdateCluster.GetHealthStatus().GetOverallHealthStatus())
				s.Equal(*oldHealthStatus.GetLastContact(), *preUpdateCluster.GetHealthStatus().GetLastContact())
				s.NoError(updateErr)
				s.Equal(newHealthStatus.GetCollectorHealthStatus(), postUpdateCluster.GetHealthStatus().GetCollectorHealthStatus())
				s.Equal(newHealthStatus.GetOverallHealthStatus(), postUpdateCluster.GetHealthStatus().GetOverallHealthStatus())
				s.Equal(*newHealthStatus.GetLastContact(), *postUpdateCluster.GetHealthStatus().GetLastContact())
			}
			// Revert to pre-test state
			err := s.datastore.UpdateClusterHealth(globalReadWriteCtx, clusterID, oldHealthStatus)
			s.Require().NoError(err)
			fetchedCluster, found, err := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.Require().NoError(err)
			s.Require().True(found)
			s.Require().Equal(oldHealthStatus.GetCollectorHealthStatus(), fetchedCluster.GetHealthStatus().GetCollectorHealthStatus())
			s.Require().Equal(oldHealthStatus.GetOverallHealthStatus(), fetchedCluster.GetHealthStatus().GetOverallHealthStatus())
			s.Require().Equal(*oldHealthStatus.GetLastContact(), *preUpdateCluster.GetHealthStatus().GetLastContact())
		})
	}
}

func (s *clusterDatastoreSACSuite) TestUpdateClusterStatus() {
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	oldCluster := fixtures.GetCluster(testconsts.Cluster2)
	oldStatus := &storage.ClusterStatus{
		SensorVersion: "3.71.x-88-g9798e675e5-dirty",
	}
	newStatus := &storage.ClusterStatus{
		SensorVersion: "3.71.x-95-gb7a8e625e9-dirty",
	}
	oldCluster.Status = oldStatus
	clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, oldCluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	oldCluster.Id = clusterID

	cases := testutils.GenericClusterSACWriteTestCases(context.Background(), s.T(), "update cluster status", clusterID, "not"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			preUpdateCluster, preUpdateFound, preUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(preUpdateErr)
			s.True(preUpdateFound)
			updateErr := s.datastore.UpdateClusterStatus(ctx, clusterID, newStatus)
			postUpdateCluster, postUpdateFound, postUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(postUpdateErr)
			s.True(postUpdateFound)
			if c.ExpectError {
				s.Equal(oldStatus.GetSensorVersion(), preUpdateCluster.GetStatus().GetSensorVersion())
				s.ErrorIs(updateErr, c.ExpectedError)
				s.Equal(oldStatus.GetSensorVersion(), postUpdateCluster.GetStatus().GetSensorVersion())
			} else {
				s.Equal(oldStatus.GetSensorVersion(), preUpdateCluster.GetStatus().GetSensorVersion())
				s.NoError(updateErr)
				s.Equal(newStatus.GetSensorVersion(), postUpdateCluster.GetStatus().GetSensorVersion())
			}
			// Revert to pre-test state
			err := s.datastore.UpdateClusterStatus(globalReadWriteCtx, clusterID, oldStatus)
			s.Require().NoError(err)
			fetchedCluster, found, err := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.Require().NoError(err)
			s.Require().True(found)
			s.Require().Equal(oldStatus.GetSensorVersion(), fetchedCluster.GetStatus().GetSensorVersion())
		})
	}
}

func (s *clusterDatastoreSACSuite) TestUpdateClusterUpgradeStatus() {
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	oldCluster := fixtures.GetCluster(testconsts.Cluster2)
	oldUpgradeStatus := &storage.ClusterUpgradeStatus{
		Upgradability:             storage.ClusterUpgradeStatus_MANUAL_UPGRADE_REQUIRED,
		UpgradabilityStatusReason: "Manual upgrade required",
	}
	oldStatus := &storage.ClusterStatus{
		SensorVersion: "3.71.x-88-g9798e675e5-dirty",
		UpgradeStatus: oldUpgradeStatus,
	}
	oldCluster.Status = oldStatus
	clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, oldCluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	oldCluster.Id = clusterID
	newUpgradeStatus := &storage.ClusterUpgradeStatus{
		Upgradability:             storage.ClusterUpgradeStatus_AUTO_UPGRADE_POSSIBLE,
		UpgradabilityStatusReason: "Automatic upgrade possible",
	}

	cases := testutils.GenericClusterSACWriteTestCases(context.Background(), s.T(), "update cluster upgrade status", clusterID, "not"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			preUpdateCluster, preUpdateFound, preUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(preUpdateErr)
			s.True(preUpdateFound)
			updateErr := s.datastore.UpdateClusterUpgradeStatus(ctx, clusterID, newUpgradeStatus)
			postUpdateCluster, postUpdateFound, postUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(postUpdateErr)
			s.True(postUpdateFound)
			if c.ExpectError {
				s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradability(), preUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradability())
				s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradabilityStatusReason(), preUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradabilityStatusReason())
				s.ErrorIs(updateErr, c.ExpectedError)
				s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradability(), postUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradability())
				s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradabilityStatusReason(), postUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradabilityStatusReason())
			} else {
				s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradability(), preUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradability())
				s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradabilityStatusReason(), preUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradabilityStatusReason())
				s.NoError(updateErr)
				s.Require().Equal(newUpgradeStatus.GetUpgradability(), postUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradability())
				s.Require().Equal(newUpgradeStatus.GetUpgradabilityStatusReason(), postUpdateCluster.GetStatus().GetUpgradeStatus().GetUpgradabilityStatusReason())
			}
			// Revert to pre-test state
			err := s.datastore.UpdateClusterUpgradeStatus(globalReadWriteCtx, clusterID, oldUpgradeStatus)
			s.Require().NoError(err)
			fetchedCluster, found, err := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.Require().NoError(err)
			s.Require().True(found)
			s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradability(), fetchedCluster.GetStatus().GetUpgradeStatus().GetUpgradability())
			s.Require().Equal(oldStatus.GetUpgradeStatus().GetUpgradabilityStatusReason(), fetchedCluster.GetStatus().GetUpgradeStatus().GetUpgradabilityStatusReason())
		})
	}
}

func (s *clusterDatastoreSACSuite) TestUpdateSensorDeploymentIdentification() {
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	oldCluster := fixtures.GetCluster(testconsts.Cluster2)
	oldSensorDeploymentIdentification := oldCluster.GetMostRecentSensorId()
	clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, oldCluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	oldCluster.Id = clusterID
	newSensorDeploymentIdentification := &storage.SensorDeploymentIdentification{
		SystemNamespaceId:   "fcab1a6d-07a3-4da9-a9cf-e286537ed4e3",
		DefaultNamespaceId:  "cd14a849-21d3-4351-9a56-8a066c2e83e1",
		AppNamespace:        "stackrox",
		AppNamespaceId:      "dbcbf202-6086-4bf9-8bc1-d10af3e36883",
		AppServiceaccountId: "",
		K8SNodeName:         "colima",
	}

	cases := testutils.GenericClusterSACWriteTestCases(context.Background(), s.T(), "update sensor deployment identification", clusterID, "not"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			preUpdateCluster, preUpdateFound, preUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(preUpdateErr)
			s.True(preUpdateFound)
			updateErr := s.datastore.UpdateSensorDeploymentIdentification(ctx, clusterID, newSensorDeploymentIdentification)
			postUpdateCluster, postUpdateFound, postUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(postUpdateErr)
			s.True(postUpdateFound)
			if c.ExpectError {
				s.Require().Equal(oldSensorDeploymentIdentification.GetSystemNamespaceId(), preUpdateCluster.GetMostRecentSensorId().GetSystemNamespaceId())
				s.Require().Equal(oldSensorDeploymentIdentification.GetDefaultNamespaceId(), preUpdateCluster.GetMostRecentSensorId().GetDefaultNamespaceId())
				s.ErrorIs(updateErr, c.ExpectedError)
				s.Require().Equal(oldSensorDeploymentIdentification.GetSystemNamespaceId(), postUpdateCluster.GetMostRecentSensorId().GetSystemNamespaceId())
				s.Require().Equal(oldSensorDeploymentIdentification.GetDefaultNamespaceId(), postUpdateCluster.GetMostRecentSensorId().GetDefaultNamespaceId())
			} else {
				s.Require().Equal(oldSensorDeploymentIdentification.GetSystemNamespaceId(), preUpdateCluster.GetMostRecentSensorId().GetSystemNamespaceId())
				s.Require().Equal(oldSensorDeploymentIdentification.GetDefaultNamespaceId(), preUpdateCluster.GetMostRecentSensorId().GetDefaultNamespaceId())
				s.NoError(updateErr)
				s.Require().Equal(newSensorDeploymentIdentification.GetSystemNamespaceId(), postUpdateCluster.GetMostRecentSensorId().GetSystemNamespaceId())
				s.Require().Equal(newSensorDeploymentIdentification.GetDefaultNamespaceId(), postUpdateCluster.GetMostRecentSensorId().GetDefaultNamespaceId())
			}
			// Revert to pre-test state
			err := s.datastore.UpdateSensorDeploymentIdentification(globalReadWriteCtx, clusterID, oldSensorDeploymentIdentification)
			s.Require().NoError(err)
			fetchedCluster, found, err := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.Require().NoError(err)
			s.Require().True(found)
			s.Require().Equal(oldSensorDeploymentIdentification.GetSystemNamespaceId(), fetchedCluster.GetMostRecentSensorId().GetSystemNamespaceId())
			s.Require().Equal(oldSensorDeploymentIdentification.GetDefaultNamespaceId(), fetchedCluster.GetMostRecentSensorId().GetDefaultNamespaceId())
		})
	}
}

func (s *clusterDatastoreSACSuite) TestUpdateAuditLogFileStates() {
	// Note: The tested function, when the scope allows it, actually adds the new state to the old one,
	// erasing old values with the new ones when a conflict arises.
	// The goal of the current test is to test whether the scope allows the action. As a consequence,
	// only one value is used in the test, and moves from initial to replaced state and back.
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	oldCluster := fixtures.GetCluster(testconsts.Cluster2)
	oldCollectTimestamp := &types.Timestamp{Seconds: 1659478729}
	oldAuditLogFileState := map[string]*storage.AuditLogFileState{
		"fileState": {
			CollectLogsSince: oldCollectTimestamp,
			LastAuditId:      "oldAuditID",
		},
	}
	oldCluster.AuditLogState = oldAuditLogFileState
	clusterID, err := s.datastore.AddCluster(globalReadWriteCtx, oldCluster)
	defer s.deleteCluster(clusterID)
	s.Require().NoError(err)
	oldCluster.Id = clusterID
	newCollectTimestamp := &types.Timestamp{Seconds: 1659479729}
	newAuditLogFileState := map[string]*storage.AuditLogFileState{
		"fileState": {
			CollectLogsSince: newCollectTimestamp,
			LastAuditId:      "newAuditID",
		},
	}

	cases := testutils.GenericClusterSACWriteTestCases(context.Background(), s.T(), "update sensor deployment identification", clusterID, "not"+clusterID, resources.Cluster)

	for name, c := range cases {
		s.Run(name, func() {
			ctx := c.Context
			preUpdateCluster, preUpdateFound, preUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(preUpdateErr)
			s.True(preUpdateFound)
			updateErr := s.datastore.UpdateAuditLogFileStates(ctx, clusterID, newAuditLogFileState)
			postUpdateCluster, postUpdateFound, postUpdateErr := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
			s.NoError(postUpdateErr)
			s.True(postUpdateFound)
			if c.ExpectError {
				s.Require().Equal(len(oldAuditLogFileState), len(preUpdateCluster.GetAuditLogState()))
				for k := range oldAuditLogFileState {
					s.Require().Equal(oldAuditLogFileState[k].GetLastAuditId(), preUpdateCluster.GetAuditLogState()[k].GetLastAuditId())
					s.Require().NotNil(*preUpdateCluster.GetAuditLogState()[k])
					s.Require().Equal(*oldAuditLogFileState[k].GetCollectLogsSince(), *preUpdateCluster.GetAuditLogState()[k].GetCollectLogsSince())
				}
				s.ErrorIs(updateErr, c.ExpectedError)
				s.Require().Equal(len(oldAuditLogFileState), len(postUpdateCluster.GetAuditLogState()))
				for k := range oldAuditLogFileState {
					s.Require().Equal(oldAuditLogFileState[k].GetLastAuditId(), postUpdateCluster.GetAuditLogState()[k].GetLastAuditId())
					s.Require().NotNil(*postUpdateCluster.GetAuditLogState()[k])
					s.Require().Equal(*oldAuditLogFileState[k].GetCollectLogsSince(), *postUpdateCluster.GetAuditLogState()[k].GetCollectLogsSince())
				}
			} else {
				s.Require().Equal(len(oldAuditLogFileState), len(preUpdateCluster.GetAuditLogState()))
				for k := range oldAuditLogFileState {
					s.Require().Equal(oldAuditLogFileState[k].GetLastAuditId(), preUpdateCluster.GetAuditLogState()[k].GetLastAuditId())
					s.Require().NotNil(*preUpdateCluster.GetAuditLogState()[k])
					s.Require().Equal(*oldAuditLogFileState[k].GetCollectLogsSince(), *preUpdateCluster.GetAuditLogState()[k].GetCollectLogsSince())
				}
				s.NoError(updateErr)
				s.Require().Equal(len(newAuditLogFileState), len(postUpdateCluster.GetAuditLogState()))
				for k := range newAuditLogFileState {
					s.Require().Equal(newAuditLogFileState[k].GetLastAuditId(), postUpdateCluster.GetAuditLogState()[k].GetLastAuditId())
					s.Require().NotNil(*postUpdateCluster.GetAuditLogState()[k])
					s.Require().Equal(*newAuditLogFileState[k].GetCollectLogsSince(), *postUpdateCluster.GetAuditLogState()[k].GetCollectLogsSince())
				}
				// Revert to pre-test state
				err := s.datastore.UpdateAuditLogFileStates(globalReadWriteCtx, clusterID, oldAuditLogFileState)
				s.Require().NoError(err)
				fetchedCluster, found, err := s.datastore.GetCluster(globalReadWriteCtx, clusterID)
				s.Require().NoError(err)
				s.Require().True(found)
				s.Require().Equal(len(oldAuditLogFileState), len(fetchedCluster.GetAuditLogState()))
				for k := range oldAuditLogFileState {
					s.Require().Equal(oldAuditLogFileState[k].GetLastAuditId(), fetchedCluster.GetAuditLogState()[k].GetLastAuditId())
					s.Require().NotNil(*fetchedCluster.GetAuditLogState()[k])
					s.Require().Equal(*oldAuditLogFileState[k].GetCollectLogsSince(), *fetchedCluster.GetAuditLogState()[k].GetCollectLogsSince())
				}
			}
		})
	}
}

func (s *clusterDatastoreSACSuite) TestCount() {
	cluster1 := fixtures.GetCluster(testconsts.Cluster1)
	clusterID1, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster1)
	defer s.deleteCluster(clusterID1)
	s.Require().NoError(err)
	cluster1.Id = clusterID1
	cluster2 := fixtures.GetCluster(testconsts.Cluster2)
	clusterID2, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster2)
	defer s.deleteCluster(clusterID2)
	s.Require().NoError(err)
	cluster2.Id = clusterID2
	otherClusterID := testconsts.Cluster3

	cases := getMultiClusterTestCases(context.Background(), clusterID1, clusterID2, otherClusterID)

	for _, c := range cases {
		s.Run(c.Name, func() {
			ctx := c.Context
			clusterCount, err := s.datastore.Count(ctx, nil)
			s.NoError(err)
			s.Equal(len(c.ExpectedClusterNames), clusterCount)
		})
	}
}

func (s *clusterDatastoreSACSuite) TestCountClusters() {
	cluster1 := fixtures.GetCluster(testconsts.Cluster1)
	clusterID1, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster1)
	defer s.deleteCluster(clusterID1)
	s.Require().NoError(err)
	cluster1.Id = clusterID1
	cluster2 := fixtures.GetCluster(testconsts.Cluster2)
	clusterID2, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster2)
	defer s.deleteCluster(clusterID2)
	s.Require().NoError(err)
	cluster2.Id = clusterID2
	otherClusterID := testconsts.Cluster3

	cases := getMultiClusterTestCases(context.Background(), clusterID1, clusterID2, otherClusterID)

	for _, c := range cases {
		s.Run(c.Name, func() {
			ctx := c.Context
			clusterCount, err := s.datastore.CountClusters(ctx)
			s.NoError(err)
			s.Equal(len(c.ExpectedClusterNames), clusterCount)
		})
	}
}

func (s *clusterDatastoreSACSuite) TestSearch() {
	cluster1 := fixtures.GetCluster(testconsts.Cluster1)
	clusterID1, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster1)
	defer s.deleteCluster(clusterID1)
	s.Require().NoError(err)
	cluster1.Id = clusterID1
	cluster2 := fixtures.GetCluster(testconsts.Cluster2)
	clusterID2, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster2)
	defer s.deleteCluster(clusterID2)
	s.Require().NoError(err)
	cluster2.Id = clusterID2
	otherClusterID := testconsts.Cluster3

	cases := getMultiClusterTestCases(context.Background(), clusterID1, clusterID2, otherClusterID)

	for _, c := range cases {
		s.Run(c.Name, func() {
			ctx := c.Context
			results, err := s.datastore.Search(ctx, nil)
			s.NoError(err)
			fetchedIDs := make([]string, 0, len(results))
			for _, r := range results {
				fetchedIDs = append(fetchedIDs, r.ID)
			}
			s.ElementsMatch(c.ExpectedClusterIDs, fetchedIDs)
		})
	}
}

func (s *clusterDatastoreSACSuite) TestSearchRawClusters() {
	cluster1 := fixtures.GetCluster(testconsts.Cluster1)
	clusterID1, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster1)
	defer s.deleteCluster(clusterID1)
	s.Require().NoError(err)
	cluster1.Id = clusterID1
	cluster2 := fixtures.GetCluster(testconsts.Cluster2)
	clusterID2, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster2)
	defer s.deleteCluster(clusterID2)
	s.Require().NoError(err)
	cluster2.Id = clusterID2
	otherClusterID := testconsts.Cluster3

	cases := getMultiClusterTestCases(context.Background(), clusterID1, clusterID2, otherClusterID)

	for _, c := range cases {
		s.Run(c.Name, func() {
			ctx := c.Context
			results, err := s.datastore.SearchRawClusters(ctx, nil)
			s.NoError(err)
			fetchedIDs := make([]string, 0, len(results))
			for _, r := range results {
				fetchedIDs = append(fetchedIDs, r.GetId())
			}
			fetchedNames := make([]string, 0, len(results))
			for _, r := range results {
				fetchedNames = append(fetchedNames, r.GetName())
			}
			s.ElementsMatch(c.ExpectedClusterIDs, fetchedIDs)
			s.ElementsMatch(c.ExpectedClusterNames, fetchedNames)
		})
	}
}

func (s *clusterDatastoreSACSuite) TestSearchResults() {
	cluster1 := fixtures.GetCluster(testconsts.Cluster1)
	clusterID1, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster1)
	defer s.deleteCluster(clusterID1)
	s.Require().NoError(err)
	cluster1.Id = clusterID1
	cluster2 := fixtures.GetCluster(testconsts.Cluster2)
	clusterID2, err := s.datastore.AddCluster(s.testContexts[testutils.UnrestrictedReadWriteCtx], cluster2)
	defer s.deleteCluster(clusterID2)
	s.Require().NoError(err)
	cluster2.Id = clusterID2
	otherClusterID := testconsts.Cluster3

	cases := getMultiClusterTestCases(context.Background(), clusterID1, clusterID2, otherClusterID)

	for _, c := range cases {
		s.Run(c.Name, func() {
			ctx := c.Context
			results, err := s.datastore.SearchResults(ctx, nil)
			s.NoError(err)
			fetchedIDs := make([]string, 0, len(results))
			for _, r := range results {
				fetchedIDs = append(fetchedIDs, r.GetId())
			}
			s.ElementsMatch(c.ExpectedClusterIDs, fetchedIDs)
		})
	}
}

// The LookupOrCreateClusterFromConfig function seems a bit more complex to test.
// On the other hand, Scoped Access Control for that function should behave the same as
// for the other write (UpdateXXX) functions.
// It will not be tested here.
