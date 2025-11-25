//go:build sql_integration

package datastore

import (
	"context"
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
