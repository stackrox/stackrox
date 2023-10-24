//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	dDS "github.com/stackrox/rox/central/deployment/datastore"
	deploymentTypes "github.com/stackrox/rox/central/deployment/store/types"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	nsDS "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	otherNamespace = "OtherNamespace"
)

func TestDeploymentDataStoreSAC(t *testing.T) {
	suite.Run(t, new(deploymentDatastoreSACSuite))
}

type deploymentIDs struct {
	clusterID    string
	deploymentID string
}

type deploymentDatastoreSACSuite struct {
	suite.Suite

	// Elements for postgres mode
	pool postgres.DB

	datastore      dDS.DataStore
	namespaceStore nsDS.DataStore
	imageStore     imageDS.DataStore

	testContexts                    map[string]context.Context
	testContextsWithImageAccess     map[string]context.Context
	testContextsWithImageOnlyAccess map[string]context.Context
	deploymentExtensionTestContexts map[string]context.Context

	testDeploymentIDs []deploymentIDs
	testNamespaceIDs  []string

	optionsMap searchPkg.OptionsMap
}

func (s *deploymentDatastoreSACSuite) SetupSuite() {
	var err error
	pgtestbase := pgtest.ForT(s.T())
	s.Require().NotNil(pgtestbase)
	s.pool = pgtestbase.DB
	s.datastore, err = dDS.GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)
	s.namespaceStore, err = nsDS.GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)
	s.imageStore = imageDS.GetTestPostgresDataStore(s.T(), s.pool)
	s.optionsMap = schema.DeploymentsSchema.OptionsMap

	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Deployment)
	s.testContextsWithImageAccess = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Deployment, resources.Image)
	s.testContextsWithImageOnlyAccess = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Image)
	s.deploymentExtensionTestContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.DeploymentExtension)
}

func (s *deploymentDatastoreSACSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *deploymentDatastoreSACSuite) SetupTest() {
	s.testDeploymentIDs = make([]deploymentIDs, 0)
	s.testNamespaceIDs = make([]string, 0)
}

func (s *deploymentDatastoreSACSuite) TearDownTest() {
	for _, deploymentIDpair := range s.testDeploymentIDs {
		s.deleteDeployment(deploymentIDpair.clusterID, deploymentIDpair.deploymentID)
	}
	for _, namespaceID := range s.testNamespaceIDs {
		s.deleteNamespace(namespaceID)
	}
}

func (s *deploymentDatastoreSACSuite) deleteDeployment(clusterID string, deploymentID string) {
	err := s.datastore.RemoveDeployment(s.testContexts[testutils.UnrestrictedReadWriteCtx], clusterID, deploymentID)
	s.NoError(err)
}

func (s *deploymentDatastoreSACSuite) deleteNamespace(namespaceID string) {
	err := s.namespaceStore.RemoveNamespace(sac.WithAllAccess(context.Background()), namespaceID)
	s.NoError(err)
}

func (s *deploymentDatastoreSACSuite) pushDeploymentToStore(clusterID string, namespaceName string) *storage.Deployment {
	var err error
	globalReadWriteCtx := s.testContexts[testutils.UnrestrictedReadWriteCtx]
	namespace := fixtures.GetNamespace(clusterID, clusterID, namespaceName)
	err = s.namespaceStore.AddNamespace(sac.WithAllAccess(context.Background()), namespace)
	s.Require().NoError(err)
	deployment := fixtures.GetScopedDeployment(uuid.NewV4().String(), clusterID, namespaceName)
	deployment.NamespaceId = namespace.GetId()
	err = s.datastore.UpsertDeployment(globalReadWriteCtx, deployment)
	s.Require().NoError(err)
	s.testDeploymentIDs = append(s.testDeploymentIDs, deploymentIDs{
		clusterID:    deployment.GetClusterId(),
		deploymentID: deployment.GetId(),
	})
	return deployment
}

type multipleDeploymentReadTestCase struct {
	ScopeKey              string
	ExpectedDeploymentIDs []string
}

func (s *deploymentDatastoreSACSuite) setupMultipleDeploymentReadTest() ([]string, map[string]*storage.Deployment, map[string]multipleDeploymentReadTestCase) {
	deployment1 := s.pushDeploymentToStore(testconsts.Cluster1, testconsts.NamespaceA)
	deploymentID1 := deployment1.GetId()
	deployment2 := s.pushDeploymentToStore(testconsts.Cluster2, testconsts.NamespaceA)
	deploymentID2 := deployment2.GetId()
	deployment3 := s.pushDeploymentToStore(testconsts.Cluster2, testconsts.NamespaceB)
	deploymentID3 := deployment3.GetId()
	deployment4 := s.pushDeploymentToStore(testconsts.Cluster3, otherNamespace)
	deploymentID4 := deployment4.GetId()
	pushedIDs := []string{deploymentID1, deploymentID2, deploymentID3, deploymentID4}
	IDtoDeployment := map[string]*storage.Deployment{
		deploymentID1: deployment1,
		deploymentID2: deployment2,
		deploymentID3: deployment3,
		deploymentID4: deployment4,
	}

	return pushedIDs, IDtoDeployment, map[string]multipleDeploymentReadTestCase{
		"(full) read-only can see all deployments": {
			ScopeKey:              testutils.UnrestrictedReadCtx,
			ExpectedDeploymentIDs: []string{deploymentID1, deploymentID2, deploymentID3, deploymentID4},
		},
		"full read-write can see all deployments": {
			ScopeKey:              testutils.UnrestrictedReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID1, deploymentID2, deploymentID3, deploymentID4},
		},
		"full read-write on Cluster 1 can only see deployments in Cluster 1": {
			ScopeKey:              testutils.Cluster1ReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID1},
		},
		"read-write on Cluster 1 and Namespace A can only see deployments in Cluster 1 and Namespace A": {
			ScopeKey:              testutils.Cluster1NamespaceAReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID1},
		},
		"read-write on Cluster 1 and at least Namespace A can only see deployments in Cluster 1 and Namespace A": {
			ScopeKey:              testutils.Cluster1NamespacesACReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID1},
		},
		"read-write on Cluster 1 and namespaces without deployments cannot see any deployment": {
			ScopeKey:              testutils.Cluster1NamespacesBCReadWriteCtx,
			ExpectedDeploymentIDs: []string{},
		},
		"full read-write on Cluster 2 can only see deployments in Cluster 2": {
			ScopeKey:              testutils.Cluster2ReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID2, deploymentID3},
		},
		"read-write on Cluster 2 and Namespace A can only see deployments in Cluster 2 and Namespace A": {
			ScopeKey:              testutils.Cluster2NamespaceAReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID2},
		},
		"read-write on Cluster 2 and at least Namespace A can only see deployments in Cluster 2 and Namespace A": {
			ScopeKey:              testutils.Cluster2NamespacesACReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID2},
		},
		"read-write on Cluster 2 and Namespace B can only see deployments in Cluster 1 and Namespace B": {
			ScopeKey:              testutils.Cluster2NamespaceBReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID3},
		},
		"read-write on Cluster 2 and at least Namespaces A and B can only see deployments in Cluster 1 and Namespaces A and B": {
			ScopeKey:              testutils.Cluster2NamespacesABReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID2, deploymentID3},
		},
		"read-write on Cluster 2 and at least Namespace B can only see deployments in Cluster 1 and Namespace B": {
			ScopeKey:              testutils.Cluster2NamespacesBCReadWriteCtx,
			ExpectedDeploymentIDs: []string{deploymentID3},
		},
		"read-write on Cluster 2 and namespace(s) without deployments cannot see any deployment": {
			ScopeKey:              testutils.Cluster2NamespaceCReadWriteCtx,
			ExpectedDeploymentIDs: []string{},
		},
		"read-write on other cluster but matching namespace cannot see any deployment": {
			ScopeKey:              testutils.Cluster3NamespaceAReadWriteCtx,
			ExpectedDeploymentIDs: []string{},
		},
		"read-write on a mix of cluster and namespaces can only see the deployments in the defined scope": {
			ScopeKey:              testutils.MixedClusterAndNamespaceReadCtx,
			ExpectedDeploymentIDs: []string{deploymentID1, deploymentID2, deploymentID3},
		},
	}
}

func (s *deploymentDatastoreSACSuite) setupSearchTest() {
	deployments := fixtures.GetSACTestResourceSet(fixtures.GetScopedDeployment)
	pushedNamespaces := make(map[string]map[string]*storage.NamespaceMetadata, 0)
	for _, d := range deployments {
		clusterID := d.GetClusterId()
		namespaceName := d.GetNamespace()
		if _, ok := pushedNamespaces[clusterID]; !ok {
			pushedNamespaces[clusterID] = make(map[string]*storage.NamespaceMetadata, 0)
		}
		if _, ok := pushedNamespaces[clusterID][namespaceName]; !ok {
			namespace := fixtures.GetNamespace(clusterID, clusterID, namespaceName)
			pushedNamespaces[clusterID][namespaceName] = namespace
			err := s.namespaceStore.AddNamespace(sac.WithAllAccess(context.Background()), namespace)
			s.NoError(err)
			s.testNamespaceIDs = append(s.testNamespaceIDs, namespace.GetId())
		}
		d.NamespaceId = pushedNamespaces[clusterID][namespaceName].GetId()
		err := s.datastore.UpsertDeployment(s.testContexts[testutils.UnrestrictedReadWriteCtx], d)
		s.Require().NoError(err)
		s.testDeploymentIDs = append(s.testDeploymentIDs, deploymentIDs{
			clusterID:    clusterID,
			deploymentID: d.GetId(),
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestUpsertDeployment() {
	cases := testutils.GenericGlobalSACUpsertTestCases(s.T(), testutils.VerbUpsert)

	for name, c := range cases {
		s.Run(name, func() {
			deployment := fixtures.GetScopedDeployment(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			deployment.Priority = 1
			s.testDeploymentIDs = append(s.testDeploymentIDs, deploymentIDs{
				clusterID:    deployment.GetClusterId(),
				deploymentID: deployment.GetId(),
			})
			ctx := s.testContexts[c.ScopeKey]
			err := s.datastore.UpsertDeployment(ctx, deployment)
			defer s.deleteDeployment(deployment.GetClusterId(), deployment.GetId())
			fetched, found, getErr := s.datastore.GetDeployment(s.testContexts[testutils.UnrestrictedReadCtx], deployment.GetId())
			s.NoError(getErr)
			if c.ExpectError {
				s.Require().Error(err)
				s.ErrorIs(err, c.ExpectedError)
				s.False(found)
				s.Nil(fetched)
			} else {
				s.NoError(err)
				s.True(found)
				s.Equal(*deployment, *fetched)
			}
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestGetDeployment() {
	deployment := s.pushDeploymentToStore(testconsts.Cluster2, testconsts.NamespaceB)
	deployment.Priority = 1

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, err := s.datastore.GetDeployment(ctx, deployment.GetId())
			s.Require().NoError(err)
			if c.ExpectedFound {
				s.Require().True(found)
				s.Require().NotNil(res)
				s.Equal(*deployment, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestGetDeployments() {
	pushedIDs, IDtoDeployment, cases := s.setupMultipleDeploymentReadTest()
	testIDs := []string{pushedIDs[0], pushedIDs[1], pushedIDs[3]}

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			expectedIDs := make([]string, 0, len(c.ExpectedDeploymentIDs))
			for _, ID := range c.ExpectedDeploymentIDs {
				found := false
				for _, testID := range testIDs {
					if ID == testID {
						found = true
						break
					}
				}
				if found {
					expectedIDs = append(expectedIDs, ID)
				}
			}
			deployments, getErr := s.datastore.GetDeployments(ctx, testIDs)
			s.Require().NoError(getErr)
			fetchedIDs := make([]string, 0, len(deployments))
			for _, d := range deployments {
				fetchedIDs = append(fetchedIDs, d.GetId())
				ref := IDtoDeployment[d.GetId()]
				s.Require().NotNil(ref)
				s.Require().NotNil(d)
				s.Equal(ref.GetName(), d.GetName())
				s.Equal(ref.GetClusterId(), d.GetClusterId())
				s.Equal(ref.GetNamespace(), d.GetNamespace())
			}
			s.ElementsMatch(fetchedIDs, expectedIDs)
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestGetDeploymentIDs() {
	_, _, cases := s.setupMultipleDeploymentReadTest()

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			fetchedIDs, getErr := s.datastore.GetDeploymentIDs(ctx)
			s.Require().NoError(getErr)
			// Note: the behaviour change may impact policy dry runs if the requester does not have full namespace scope.
			s.ElementsMatch(fetchedIDs, c.ExpectedDeploymentIDs)
		})
	}
}

/*
The function GetImagesForDeployment scans the containers in a given deployment, and tries to fetch
the container images from the image datastore. If nothing can be fetched from the datastore, then
the information available in the deployment is formatted and returned to the user.
SAC checks on image access can prevent data retrieval from store. In case where image access is not
allowed, then the image data available in the deployment object is returned too.
The test strategy is to have slightly different image information in the deployment and in the image
datastore, and to check whether the difference in the function return points at the data from the
deployment object, or at the data pushed to the image datastore.
*/

func (s *deploymentDatastoreSACSuite) runGetImagesForDeploymentTest(testContexts map[string]context.Context, expectOnlyDeploymentData bool) {
	imageToStore := fixtures.LightweightDeploymentImage()
	imageName := imageToStore.GetName()
	imageName.Tag = "beta"
	imageName.FullName = imageName.GetRegistry() + "/" + imageName.GetRemote() + ":" + imageName.GetTag()
	imageToStore.Name = imageName
	err := s.imageStore.UpsertImage(sac.WithAllAccess(context.Background()), imageToStore)
	s.Require().NoError(err)
	deployment := s.pushDeploymentToStore(testconsts.Cluster2, testconsts.NamespaceB)
	imageFromDeployment := fixtures.LightweightDeploymentImage()

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := testContexts[c.ScopeKey]
			images, err := s.datastore.GetImagesForDeployment(ctx, deployment)
			s.NoError(err)
			s.Require().Equal(1, len(images))
			if (!expectOnlyDeploymentData) && c.ExpectedFound {
				s.Equal(*imageToStore.GetName(), *images[0].GetName())
			} else {
				s.Equal(*imageFromDeployment.GetName(), *images[0].GetName())
			}
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestGetImagesForDeploymentNoImageAccess() {
	// No image access -> image data is always retrieved from the deployment data
	s.runGetImagesForDeploymentTest(s.testContexts, true)
}

func (s *deploymentDatastoreSACSuite) TestGetImagesForDeploymentWithImageAccess() {
	s.runGetImagesForDeploymentTest(s.testContextsWithImageAccess, false)
}

func (s *deploymentDatastoreSACSuite) TestGetImagesForDeploymentWithImageOnlyAccess() {
	s.runGetImagesForDeploymentTest(s.testContextsWithImageOnlyAccess, false)
}

func (s *deploymentDatastoreSACSuite) TestListDeployment() {
	deployment := fixtures.GetScopedDeployment(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
	err := s.datastore.UpsertDeployment(s.testContexts[testutils.UnrestrictedReadWriteCtx], deployment)
	s.Require().NoError(err)
	s.testDeploymentIDs = append(s.testDeploymentIDs, deploymentIDs{
		clusterID:    deployment.GetClusterId(),
		deploymentID: deployment.GetId(),
	})
	listDeployment := deploymentTypes.ConvertDeploymentToDeploymentList(deployment)
	listDeployment.Priority = 1

	cases := testutils.GenericNamespaceSACGetTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]
			res, found, getErr := s.datastore.ListDeployment(ctx, deployment.GetId())
			s.Require().NoError(getErr)
			if c.ExpectedFound {
				s.True(found)
				s.Require().NotNil(res)
				s.Equal(*listDeployment, *res)
			} else {
				s.False(found)
				s.Nil(res)
			}
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestRemoveDeployment() {
	cases := testutils.GenericGlobalSACDeleteTestCases(s.T())

	for name, c := range cases {
		s.Run(name, func() {
			deployment := fixtures.GetScopedDeployment(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB)
			listDeployment := deploymentTypes.ConvertDeploymentToDeploymentList(deployment)
			listDeployment.Priority = 1
			err := s.datastore.UpsertDeployment(s.testContexts[testutils.UnrestrictedReadWriteCtx], deployment)
			s.Require().NoError(err)
			defer s.deleteDeployment(deployment.GetClusterId(), deployment.GetId())
			preFetch, preFound, err := s.datastore.GetDeployment(s.testContexts[testutils.UnrestrictedReadCtx], deployment.GetId())
			s.NoError(err)
			ctx := s.testContexts[c.ScopeKey]
			removeErr := s.datastore.RemoveDeployment(ctx, deployment.GetClusterId(), deployment.GetId())
			postFetch, postFound, err := s.datastore.GetDeployment(s.testContexts[testutils.UnrestrictedReadCtx], deployment.GetId())
			s.NoError(err)
			s.Require().True(preFound)
			listPreFetch := deploymentTypes.ConvertDeploymentToDeploymentList(preFetch)
			s.Require().Equal(*listDeployment, *listPreFetch)
			if c.ExpectError {
				s.Error(removeErr)
				s.ErrorIs(removeErr, c.ExpectedError)
				s.True(postFound)
				listPostFetch := deploymentTypes.ConvertDeploymentToDeploymentList(postFetch)
				s.Equal(*listDeployment, *listPostFetch)
			} else {
				s.NoError(removeErr)
				s.False(postFound)
				s.Nil(postFetch)
			}
		})
	}
}

func (s *deploymentDatastoreSACSuite) runTestCount(testCase testutils.SACSearchTestCase) {
	ctx := s.testContexts[testCase.ScopeKey]
	resultCount, err := s.datastore.Count(ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	expectedResultCount := testutils.AggregateCounts(s.T(), testCase.Results)
	s.Equal(expectedResultCount, resultCount)
}

func (s *deploymentDatastoreSACSuite) TestScopedCount() {
	s.setupSearchTest()
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestCount(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestUnrestrictedCount() {
	s.setupSearchTest()
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestCount(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) runTestCountDeployments(testCase testutils.SACSearchTestCase) {
	ctx := s.testContexts[testCase.ScopeKey]
	resultCount, err := s.datastore.CountDeployments(ctx)
	s.NoError(err)
	expectedResultCount := testutils.AggregateCounts(s.T(), testCase.Results)
	s.Equal(expectedResultCount, resultCount)
}

func (s *deploymentDatastoreSACSuite) TestScopedCountDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestCountDeployments(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestUnrestrictedCountDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestCountDeployments(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) runTestSearch(testCase testutils.SACSearchTestCase) {
	ctx := s.testContexts[testCase.ScopeKey]
	globalReadCtx := s.testContexts[testutils.UnrestrictedReadCtx]
	results, err := s.datastore.Search(ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	deployments := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		d, found, getErr := s.datastore.GetDeployment(globalReadCtx, r.ID)
		s.NoError(getErr)
		s.True(found)
		if d != nil {
			deployments = append(deployments, d)
		}
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), deployments)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testCase.Results, resultCounts)
}

func (s *deploymentDatastoreSACSuite) TestScopedSearch() {
	s.setupSearchTest()
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearch(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestUnrestrictedSearch() {
	s.setupSearchTest()
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearch(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) runTestSearchDeployments(testCase testutils.SACSearchTestCase) {
	ctx := s.testContexts[testCase.ScopeKey]
	globalReadCtx := s.testContexts[testutils.UnrestrictedReadCtx]
	results, err := s.datastore.SearchDeployments(ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	deployments := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		d, found, getErr := s.datastore.GetDeployment(globalReadCtx, r.GetId())
		s.NoError(getErr)
		s.True(found)
		if d != nil {
			deployments = append(deployments, d)
		}
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), deployments)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testCase.Results, resultCounts)
}

func (s *deploymentDatastoreSACSuite) TestScopedSearchDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearchDeployments(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestUnrestrictedSearchDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearchDeployments(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) runTestSearchRawDeployments(testCase testutils.SACSearchTestCase) {
	ctx := s.testContexts[testCase.ScopeKey]
	results, err := s.datastore.SearchRawDeployments(ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	deployments := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		deployments = append(deployments, r)
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), deployments)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testCase.Results, resultCounts)
}

func (s *deploymentDatastoreSACSuite) TestScopedSearchRawDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearchRawDeployments(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestUnrestrictedSearchRawDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearchRawDeployments(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) runTestSearchListDeployments(testCase testutils.SACSearchTestCase) {
	ctx := s.testContexts[testCase.ScopeKey]
	results, err := s.datastore.SearchListDeployments(ctx, searchPkg.EmptyQuery())
	s.NoError(err)
	deployments := make([]sac.NamespaceScopedObject, 0, len(results))
	for _, r := range results {
		deployments = append(deployments, r)
	}
	resultCounts := testutils.CountSearchResultObjectsPerClusterAndNamespace(s.T(), deployments)
	testutils.ValidateSACSearchResultDistribution(&s.Suite, testCase.Results, resultCounts)
}

func (s *deploymentDatastoreSACSuite) TestScopedSearchListDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericScopedSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearchListDeployments(c)
		})
	}
}

func (s *deploymentDatastoreSACSuite) TestUnrestrictedSearchListDeployments() {
	s.setupSearchTest()
	for name, c := range testutils.GenericUnrestrictedRawSACSearchTestCases(s.T()) {
		s.Run(name, func() {
			s.runTestSearchListDeployments(c)
		})
	}
}

// Process tags should be deprecated as of 3.72.0 and will not be tested for SAC behaviour consistency.
