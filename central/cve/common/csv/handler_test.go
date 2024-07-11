package csv

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/audit"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	componentMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	nsMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestCVEScoping(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(CVEScopingTestSuite))
}

type CVEScopingTestSuite struct {
	suite.Suite
	ctx                 context.Context
	mockCtrl            *gomock.Controller
	clusterDataStore    *clusterMocks.MockDataStore
	nsDataStore         *nsMocks.MockDataStore
	deploymentDataStore *deploymentMocks.MockDataStore
	imageDataStore      *imageMocks.MockDataStore
	nodeDataStore       *nodeMocks.MockDataStore
	componentDataStore  *componentMocks.MockDataStore
	resolver            *resolvers.Resolver
	handler             *HandlerImpl
}

func (suite *CVEScopingTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.clusterDataStore = clusterMocks.NewMockDataStore(suite.mockCtrl)
	suite.nsDataStore = nsMocks.NewMockDataStore(suite.mockCtrl)
	suite.deploymentDataStore = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.imageDataStore = imageMocks.NewMockDataStore(suite.mockCtrl)
	suite.nodeDataStore = nodeMocks.NewMockDataStore(suite.mockCtrl)
	suite.componentDataStore = componentMocks.NewMockDataStore(suite.mockCtrl)
	notifierMock := notifierMocks.NewMockProcessor(suite.mockCtrl)

	notifierMock.EXPECT().HasEnabledAuditNotifiers().Return(false).AnyTimes()

	suite.resolver = &resolvers.Resolver{
		ClusterDataStore:        suite.clusterDataStore,
		NamespaceDataStore:      suite.nsDataStore,
		DeploymentDataStore:     suite.deploymentDataStore,
		ImageDataStore:          suite.imageDataStore,
		NodeDataStore:           suite.nodeDataStore,
		ImageComponentDataStore: suite.componentDataStore,
		AuditLogger:             audit.New(notifierMock),
	}

	suite.handler = newTestHandler(suite.resolver)

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
}

func (suite *CVEScopingTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *CVEScopingTestSuite) TestSingleResourceQuery() {
	imgSha := "img1"

	query := search.ConjunctionQuery(
		search.NewQueryBuilder().AddStrings(search.ImageSHA, imgSha).ProtoQuery(),
		search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())

	suite.imageDataStore.EXPECT().
		Search(suite.ctx, search.NewQueryBuilder().AddStrings(search.ImageSHA, imgSha).ProtoQuery()).
		Return([]search.Result{{ID: imgSha}}, nil)

	expected := scoped.Context(suite.ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    imgSha,
	})
	actual, err := suite.handler.GetScopeContext(suite.ctx, query)
	suite.NoError(err)
	suite.Equal(expected, actual)
}

func (suite *CVEScopingTestSuite) TestMultipleResourceQuery() {
	imgSha := "img1"

	query := search.ConjunctionQuery(
		search.NewQueryBuilder().AddStrings(search.DeploymentName, "dep").ProtoQuery(),
		search.NewQueryBuilder().AddStrings(search.ImageSHA, imgSha).ProtoQuery(),
		search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())

	suite.imageDataStore.EXPECT().
		Search(suite.ctx, search.NewQueryBuilder().AddStrings(search.ImageSHA, imgSha).ProtoQuery()).
		Return([]search.Result{{ID: imgSha}}, nil)

	expected := scoped.Context(suite.ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    imgSha,
	})
	// Lowest resource scope should be applied.
	actual, err := suite.handler.GetScopeContext(suite.ctx, query)
	suite.NoError(err)
	suite.Equal(expected, actual)
}

func (suite *CVEScopingTestSuite) TestMultipleMatchesQuery() {
	img := "img"

	query := search.ConjunctionQuery(
		search.NewQueryBuilder().AddStrings(search.ImageName, img).ProtoQuery(),
		search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())

	suite.imageDataStore.EXPECT().
		Search(suite.ctx, search.NewQueryBuilder().AddStrings(search.ImageName, img).ProtoQuery()).
		Return([]search.Result{{ID: "img1"}, {ID: "img2"}}, nil)

	// No scope should be applied.
	actual, err := suite.handler.GetScopeContext(suite.ctx, query)
	suite.NoError(err)
	suite.Equal(suite.ctx, actual)
}

func (suite *CVEScopingTestSuite) TestNoReScope() {
	img := "img"

	query := search.ConjunctionQuery(
		search.NewQueryBuilder().AddStrings(search.ImageName, img).ProtoQuery(),
		search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery())

	expected := scoped.Context(suite.ctx, scoped.Scope{
		Level: v1.SearchCategory_DEPLOYMENTS,
		ID:    "dep",
	})
	actual, err := suite.handler.GetScopeContext(expected, query)
	suite.NoError(err)
	suite.Equal(expected, actual)
}

func newTestHandler(resolver *resolvers.Resolver) *HandlerImpl {
	return NewCSVHandler(
		resolver,
		// CVEs must be scoped from lowest entities to highest entities. DO NOT CHANGE THE ORDER.
		[]*SearchWrapper{
			NewSearchWrapper(v1.SearchCategory_IMAGE_COMPONENTS, schema.ImageComponentsSchema.OptionsMap,
				resolver.ImageComponentDataStore),
			NewSearchWrapper(v1.SearchCategory_IMAGES, ImageOnlyOptionsMap, resolver.ImageDataStore),
			NewSearchWrapper(v1.SearchCategory_DEPLOYMENTS, DeploymentOnlyOptionsMap, resolver.DeploymentDataStore),
			NewSearchWrapper(v1.SearchCategory_NAMESPACES, NamespaceOnlyOptionsMap, resolver.NamespaceDataStore),
			NewSearchWrapper(v1.SearchCategory_NODES, NodeOnlyOptionsMap, resolver.NodeDataStore),
			NewSearchWrapper(v1.SearchCategory_CLUSTERS, schema.ClustersSchema.OptionsMap, resolver.ClusterDataStore),
		},
	)
}
