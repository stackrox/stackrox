//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageutils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(DeploymentPostgresDataStoreTestSuite))
}

type DeploymentPostgresDataStoreTestSuite struct {
	suite.Suite

	testDB              *pgtest.TestPostgres
	ctx                 context.Context
	imageDatastore      imageDataStore.DataStore
	imageV2Datastore    imageV2Datastore.DataStore
	deploymentDatastore DataStore
}

func (s *DeploymentPostgresDataStoreTestSuite) SetupSuite() {

	s.ctx = context.Background()

	s.testDB = pgtest.ForT(s.T())

	if features.FlattenImageData.Enabled() {
		imageV2DS := imageV2Datastore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
		s.imageV2Datastore = imageV2DS
	} else {
		imageDS := imageDataStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
		s.imageDatastore = imageDS
	}

	deploymentDS, err := GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	s.deploymentDatastore = deploymentDS
}

func (s *DeploymentPostgresDataStoreTestSuite) TestSearchWithPostgres() {
	ctx := sac.WithAllAccess(context.Background())
	var imgV21, imgV22, imgV23 *storage.ImageV2
	img1 := fixtures.GetImageWithUniqueComponents(5)
	img1.Id = uuid.NewV4().String()
	if features.FlattenImageData.Enabled() {
		imgV21 = imageutils.ConvertToV2(img1)
	}
	img2 := fixtures.GetImageWithUniqueComponents(5)
	img2.Id = uuid.NewV4().String()
	if features.FlattenImageData.Enabled() {
		imgV22 = imageutils.ConvertToV2(img2)
	}
	img2.Scan.OperatingSystem = "pluto"
	img3 := fixtures.GetImageWithUniqueComponents(5)
	img3.Id = uuid.NewV4().String()
	if features.FlattenImageData.Enabled() {
		imgV23 = imageutils.ConvertToV2(img3)
	}
	img1ID := img1.GetId()
	img2ID := img2.GetId()
	img3ID := img3.GetId()
	if features.FlattenImageData.Enabled() {
		img1ID = imgV21.GetId()
		img2ID = imgV22.GetId()
		img3ID = imgV23.GetId()
	}
	for _, component := range img2.GetScan().GetComponents() {
		component.Name = img2ID + component.GetName()
		for _, vuln := range component.GetVulns() {
			vuln.Cve = img2ID + vuln.GetCve()
		}
	}
	img3.Scan.OperatingSystem = "saturn"
	dep1 := fixtures.GetDeploymentWithImage(testconsts.Cluster1, "n1", img1)
	dep2 := fixtures.GetDeploymentWithImage(testconsts.Cluster1, "n2", img2)
	dep3 := fixtures.GetDeploymentWithImage(testconsts.Cluster2, "n1", img3)
	if features.FlattenImageData.Enabled() {
		dep1 = fixtures.GetDeploymentWithImageV2(testconsts.Cluster1, "n1", imgV21)
		dep2 = fixtures.GetDeploymentWithImageV2(testconsts.Cluster1, "n2", imgV22)
		dep3 = fixtures.GetDeploymentWithImageV2(testconsts.Cluster2, "n1", imgV23)
	}

	// Upsert images.
	if features.FlattenImageData.Enabled() {
		s.NoError(s.imageV2Datastore.UpsertImage(ctx, imgV21))
		s.NoError(s.imageV2Datastore.UpsertImage(ctx, imgV22))
		s.NoError(s.imageV2Datastore.UpsertImage(ctx, imgV23))
	} else {
		s.NoError(s.imageDatastore.UpsertImage(ctx, img1))
		s.NoError(s.imageDatastore.UpsertImage(ctx, img2))
		s.NoError(s.imageDatastore.UpsertImage(ctx, img3))
	}
	// Upsert Deployments.
	s.NoError(s.deploymentDatastore.UpsertDeployment(ctx, dep1))
	s.NoError(s.deploymentDatastore.UpsertDeployment(ctx, dep2))
	s.NoError(s.deploymentDatastore.UpsertDeployment(ctx, dep3))

	componentIDImg2 := scancomponent.ComponentIDV2(
		img2.GetScan().GetComponents()[0],
		img2ID, 0)

	componentIDImg1 := scancomponent.ComponentIDV2(
		img1.GetScan().GetComponents()[0],
		img1ID, 0)
	cveID := cve.IDV2(
		img1.GetScan().GetComponents()[0].GetVulns()[0],
		componentIDImg1, 0)

	imageSearchCategory := v1.SearchCategory_IMAGES
	if features.FlattenImageData.Enabled() {
		imageSearchCategory = v1.SearchCategory_IMAGES_V2
	}

	for _, tc := range []struct {
		desc         string
		ctx          context.Context
		query        *v1.Query
		orderMatters bool
		expectedIDs  []string
		queryImages  bool
	}{
		{
			desc:         "Search deployments with empty query",
			ctx:          ctx,
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.GetId(), dep2.GetId(), dep3.GetId()},
		},
		{
			desc:         "Search deployments with query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, dep1.GetId()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.GetId()},
		},
		{
			desc:         "Search deployments with image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, img2.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.GetId()},
		},
		{
			desc:         "Search deployments with non-matching image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, "mars").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search deployments with deployments+image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).AddExactMatches(pkgSearch.ImageOS, img2.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.GetId()},
		},
		{
			desc:         "Search deployments with deployment scope",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep1.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.GetId()},
		},
		{
			desc:         "Search deployments with deployments scope and in-scope deployments query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep1.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep1.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.GetId()},
		},
		{
			desc:         "Search deployments with deployments scope and out-of-scope deployments query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep1.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search deployments with deployment scope and in-scope image query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep2.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, img2.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.GetId()},
		},
		{
			desc:         "Search deployments with deployment scope and out-of-scope image query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep2.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, img3.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search deployments with image scope",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{img2ID}, Level: imageSearchCategory}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.GetId()},
		},
		{
			desc:         "Search deployments with image scope and in-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{img2ID}, Level: imageSearchCategory}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.GetId()},
		},
		{
			desc:         "Search deployments with image scope and out-of-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{img2ID}, Level: imageSearchCategory}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep3.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc: "Search deployments with image component scope",
			ctx: scoped.Context(ctx, scoped.Scope{
				IDs:   []string{componentIDImg2},
				Level: v1.SearchCategory_IMAGE_COMPONENTS_V2,
			}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.GetId()},
		},
		{
			desc: "Search deployments with image vuln scope",
			ctx: scoped.Context(ctx, scoped.Scope{
				IDs:   []string{cveID},
				Level: v1.SearchCategory_IMAGE_VULNERABILITIES_V2,
			}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.GetId()},
		},
		{
			desc:         "Search images with empty query",
			ctx:          ctx,
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1ID, img2ID, img3ID},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).ProtoQuery(),
			orderMatters: true,
			expectedIDs:  []string{img2ID},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment+image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddExactMatches(pkgSearch.ImageName, img1.GetName().GetFullName()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1ID, img3ID},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment+image non-matching search fields",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddExactMatches(pkgSearch.ImageSHA, img2ID).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryImages:  true,
		},
		{
			desc:         "Search images with image scope and in-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{img2ID}, Level: imageSearchCategory}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{img2ID},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment scope",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep1.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1ID},
			queryImages:  true,
		},
		{
			desc:         "Search images with image scope and out-of-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{img2ID}, Level: imageSearchCategory}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep1.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment scope and in-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep1.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1ID},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment scope and out-of-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{IDs: []string{dep1.GetId()}, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryImages:  true,
		},
		{
			desc:         "Search images by operating system",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.OperatingSystem, "pluto").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{img2ID},
			queryImages:  true,
		},
		{
			desc: "Sort images by operating system",
			ctx:  ctx,
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{
							Field: pkgSearch.OperatingSystem.String(),
						},
					},
				},
			},
			orderMatters: true,
			expectedIDs:  []string{img1ID, img2ID, img3ID},
			queryImages:  true,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			var actual []pkgSearch.Result
			var err error
			if tc.queryImages {
				if features.FlattenImageData.Enabled() {
					actual, err = s.imageV2Datastore.Search(tc.ctx, tc.query)
				} else {
					actual, err = s.imageDatastore.Search(tc.ctx, tc.query)
				}
			} else {
				actual, err = s.deploymentDatastore.Search(tc.ctx, tc.query)
				searchRes, errRes := s.deploymentDatastore.SearchDeployments(tc.ctx, tc.query)
				assert.NoError(t, errRes)
				assert.Len(t, searchRes, len(tc.expectedIDs))

				// Verify SearchResult fields are properly populated with exact values
				for _, result := range searchRes {
					assert.Equal(t, v1.SearchCategory_DEPLOYMENTS, result.GetCategory(), "Result category should be DEPLOYMENTS")
					assert.NotNil(t, result.GetFieldToMatches(), "FieldToMatches should not be nil")

					// Fetch the actual deployment to verify exact name and location
					deployment, found, fetchErr := s.deploymentDatastore.GetDeployment(tc.ctx, result.GetId())
					assert.NoError(t, fetchErr, "Should be able to fetch deployment")
					assert.True(t, found, "Deployment should exist")
					assert.Equal(t, deployment.GetName(), result.GetName(), "SearchResult name should match deployment name")
					// Verify exact location matches expected format "/ClusterName/Namespace"
					expectedLocation := ""
					if deployment.GetClusterName() != "" && deployment.GetNamespace() != "" {
						expectedLocation = fmt.Sprintf("/%s/%s", deployment.GetClusterName(), deployment.GetNamespace())
					}
					assert.Equal(t, expectedLocation, result.GetLocation(), "SearchResult location should match /ClusterName/Namespace format")
				}
			}
			assert.NoError(t, err)
			assert.Len(t, actual, len(tc.expectedIDs))
			actualIDs := pkgSearch.ResultsToIDs(actual)
			if tc.orderMatters {
				assert.Equal(t, tc.expectedIDs, actualIDs)
			} else {
				assert.ElementsMatch(t, tc.expectedIDs, actualIDs)
			}
		})
	}
}

func TestSelectQueryOnDeployments(t *testing.T) {

	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)

	deploymentDS, err := GetTestPostgresDataStore(t, testDB.DB)
	assert.NoError(t, err)

	for _, deployment := range []*storage.Deployment{
		{
			Id:   uuid.NewV4().String(),
			Name: "dep1",
			Type: "pod",
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "dep2",
			Type: "daemonset",
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "dep3",
			Type: "daemonset",
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "dep4",
			Type: "replicaset",
		},
	} {
		require.NoError(t, deploymentDS.UpsertDeployment(ctx, deployment))
	}

	q := pkgSearch.NewQueryBuilder().
		AddSelectFields(pkgSearch.NewQuerySelect(pkgSearch.DeploymentID).AggrFunc(aggregatefunc.Count)).
		AddGroupBy(pkgSearch.DeploymentType).ProtoQuery()

	type deploymentCountByType struct {
		DeploymentIDCount int    `db:"deployment_id_count"`
		DeploymentType    string `db:"deployment_type"`
	}

	expected := []*deploymentCountByType{
		{1, "pod"},
		{2, "daemonset"},
		{1, "replicaset"},
	}
	results, err := postgres.RunSelectRequestForSchema[deploymentCountByType](ctx, testDB.DB, schema.DeploymentsSchema, q)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expected, results)
}

func TestContainerImagesView(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(t)

	deploymentDS, err := GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	// Define common image IDs that will be shared across deployments and containers
	sharedImageIDV2_1 := "imagev2-id-shared-1"
	sharedImageIDV2_2 := "imagev2-id-shared-2"
	uniqueImageIDV2 := "imagev2-id-unique"

	// Define image digests (SHA) for each image
	sharedImageDigest_1 := "sha256:shared1digest"
	sharedImageDigest_2 := "sha256:shared2digest"
	uniqueImageDigest := "sha256:uniquedigest"

	cluster1 := testconsts.Cluster1
	cluster2 := testconsts.Cluster2

	// Create deployments with multiple containers running the same images
	deployments := []*storage.Deployment{
		{
			// Deployment 1 in cluster1 with 2 containers using the same shared image
			Id:          uuid.NewV4().String(),
			Name:        "dep1",
			ClusterId:   cluster1,
			ClusterName: cluster1,
			Namespace:   "ns1",
			NamespaceId: cluster1 + "ns1",
			Containers: []*storage.Container{
				{
					Name: "container1",
					Image: &storage.ContainerImage{
						Id:   sharedImageDigest_1,
						IdV2: sharedImageIDV2_1,
						Name: &storage.ImageName{FullName: "nginx:1.0"},
					},
				},
				{
					Name: "container2",
					Image: &storage.ContainerImage{
						Id:   sharedImageDigest_1, // Same digest as container1
						IdV2: sharedImageIDV2_1,   // Same image as container1
						Name: &storage.ImageName{FullName: "nginx:1.0"},
					},
				},
			},
		},
		{
			// Deployment 2 in cluster1 with 1 container using the same shared image as dep1
			Id:          uuid.NewV4().String(),
			Name:        "dep2",
			ClusterId:   cluster1,
			ClusterName: cluster1,
			Namespace:   "ns2",
			NamespaceId: cluster1 + "ns2",
			Containers: []*storage.Container{
				{
					Name: "container1",
					Image: &storage.ContainerImage{
						Id:   sharedImageDigest_1, // Same digest as dep1
						IdV2: sharedImageIDV2_1,   // Same image as dep1
						Name: &storage.ImageName{FullName: "nginx:1.0"},
					},
				},
			},
		},
		{
			// Deployment 3 in cluster2 with containers using shared and unique images
			Id:          uuid.NewV4().String(),
			Name:        "dep3",
			ClusterId:   cluster2,
			ClusterName: cluster2,
			Namespace:   "ns1",
			NamespaceId: cluster2 + "ns1",
			Containers: []*storage.Container{
				{
					Name: "container1",
					Image: &storage.ContainerImage{
						Id:   sharedImageDigest_1, // Same digest as dep1
						IdV2: sharedImageIDV2_1,   // Same image as dep1, but in cluster2
						Name: &storage.ImageName{FullName: "nginx:1.0"},
					},
				},
				{
					Name: "container2",
					Image: &storage.ContainerImage{
						Id:   sharedImageDigest_2,
						IdV2: sharedImageIDV2_2,
						Name: &storage.ImageName{FullName: "redis:latest"},
					},
				},
				{
					Name: "container3",
					Image: &storage.ContainerImage{
						Id:   uniqueImageDigest,
						IdV2: uniqueImageIDV2,
						Name: &storage.ImageName{FullName: "postgres:15"},
					},
				},
			},
		},
		{
			// Deployment 4 in cluster2 with the second shared image
			Id:          uuid.NewV4().String(),
			Name:        "dep4",
			ClusterId:   cluster2,
			ClusterName: cluster2,
			Namespace:   "ns2",
			NamespaceId: cluster2 + "ns2",
			Containers: []*storage.Container{
				{
					Name: "container1",
					Image: &storage.ContainerImage{
						Id:   sharedImageDigest_2, // Same digest as dep3.container2
						IdV2: sharedImageIDV2_2,   // Same image as dep3.container2
						Name: &storage.ImageName{FullName: "redis:latest"},
					},
				},
			},
		},
	}

	for _, dep := range deployments {
		require.NoError(t, deploymentDS.UpsertDeployment(ctx, dep))
	}

	// Test GetContainerImageViews
	responses, err := deploymentDS.GetContainerImageViews(ctx, pkgSearch.EmptyQuery())
	require.NoError(t, err)

	// Expected results:
	// - sharedImageIDV2_1: deployed in cluster1 and cluster2
	// - sharedImageIDV2_2: deployed in cluster2 only
	// - uniqueImageIDV2: deployed in cluster2 only
	assert.Len(t, responses, 3, "Should return 3 distinct responses")

	// Build maps for easier assertion
	type imageInfo struct {
		digest     string
		clusterIDs []string
	}
	responseMap := make(map[string]imageInfo)
	for _, resp := range responses {
		responseMap[resp.GetImageID()] = imageInfo{
			digest:     resp.GetImageDigest(),
			clusterIDs: resp.GetClusterIDs(),
		}
	}

	// Verify sharedImageIDV2_1 is in both clusters with correct digest
	assert.Contains(t, responseMap, sharedImageIDV2_1)
	assert.Equal(t, sharedImageDigest_1, responseMap[sharedImageIDV2_1].digest,
		"sharedImageIDV2_1 should have correct digest")
	assert.ElementsMatch(t, []string{cluster1, cluster2}, responseMap[sharedImageIDV2_1].clusterIDs,
		"sharedImageIDV2_1 should be in both cluster1 and cluster2")

	// Verify sharedImageIDV2_2 is only in cluster2 with correct digest
	assert.Contains(t, responseMap, sharedImageIDV2_2)
	assert.Equal(t, sharedImageDigest_2, responseMap[sharedImageIDV2_2].digest,
		"sharedImageIDV2_2 should have correct digest")
	assert.ElementsMatch(t, []string{cluster2}, responseMap[sharedImageIDV2_2].clusterIDs,
		"sharedImageIDV2_2 should be only in cluster2")

	// Verify uniqueImageIDV2 is only in cluster2 with correct digest
	assert.Contains(t, responseMap, uniqueImageIDV2)
	assert.Equal(t, uniqueImageDigest, responseMap[uniqueImageIDV2].digest,
		"uniqueImageIDV2 should have correct digest")
	assert.ElementsMatch(t, []string{cluster2}, responseMap[uniqueImageIDV2].clusterIDs,
		"uniqueImageIDV2 should be only in cluster2")
}
