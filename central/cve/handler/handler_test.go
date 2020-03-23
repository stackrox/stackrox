package handler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	cveMocks "github.com/stackrox/rox/central/cve/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	componentMocks "github.com/stackrox/rox/central/imagecomponent/datastore/mocks"
	nsMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/suite"
)

func TestCVEScoping(t *testing.T) {
	suite.Run(t, new(CVEScopingTestSuite))
}

type CVEScopingTestSuite struct {
	suite.Suite
	ctx                 context.Context
	clusterDataStore    *clusterMocks.MockDataStore
	nsDataStore         *nsMocks.MockDataStore
	deploymentDataStore *deploymentMocks.MockDataStore
	imageDataStore      *imageMocks.MockDataStore
	componentDataStore  *componentMocks.MockDataStore
	cveDataStore        *cveMocks.MockDataStore
	resolver            *resolvers.Resolver
}

func (suite *CVEScopingTestSuite) SetupSuite() {
	mockCtrl := gomock.NewController(suite.T())
	suite.clusterDataStore = clusterMocks.NewMockDataStore(mockCtrl)
	suite.nsDataStore = nsMocks.NewMockDataStore(mockCtrl)
	suite.deploymentDataStore = deploymentMocks.NewMockDataStore(mockCtrl)
	suite.imageDataStore = imageMocks.NewMockDataStore(mockCtrl)
	suite.componentDataStore = componentMocks.NewMockDataStore(mockCtrl)
	suite.cveDataStore = cveMocks.NewMockDataStore(mockCtrl)

	suite.resolver = &resolvers.Resolver{
		ClusterDataStore:        suite.clusterDataStore,
		NamespaceDataStore:      suite.nsDataStore,
		DeploymentDataStore:     suite.deploymentDataStore,
		ImageDataStore:          suite.imageDataStore,
		ImageComponentDataStore: suite.componentDataStore,
		CVEDataStore:            suite.cveDataStore,
	}

	suite.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
}

func (suite *CVEScopingTestSuite) TestGetVulnsWithScoping() {
	deploymentID := "deployment1"

	query := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.DeploymentID.String(), Value: deploymentID},
						},
					},
				}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.Fixable.String(), Value: "true"},
						},
					},
				}},
			},
		}},
	}

	expectedVulns := []search.Result{
		{
			ID: "cve1",
		},
		{
			ID: "cve2",
		},
	}

	suite.deploymentDataStore.EXPECT().Search(suite.ctx, query).Return([]search.Result{{ID: deploymentID}}, nil)

	scopedCtx := scoped.Context(suite.ctx, scoped.Scope{
		Level: v1.SearchCategory_DEPLOYMENTS,
		ID:    deploymentID,
	})
	suite.cveDataStore.EXPECT().Search(scopedCtx, query).Return(expectedVulns, nil)

	actual, err := runAsScopedQuery(suite.ctx, suite.resolver, query)
	suite.NoError(err)

	for i, vuln := range actual {
		suite.Equal(expectedVulns[i].ID, vuln.ID)
	}
}

func (suite *CVEScopingTestSuite) TestGetVulnsWithoutScoping() {
	imageID := "image1"

	query := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.DeploymentName.String(), Value: "any"},
						},
					},
				}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.ImageSHA.String(), Value: imageID},
						},
					},
				}},
			},
		}},
	}

	expectedVulns := []search.Result{
		{
			ID: "cve1",
		},
		{
			ID: "cve2",
		},
	}

	suite.cveDataStore.EXPECT().Search(suite.ctx, query).Return(expectedVulns, nil)

	actual, err := runAsScopedQuery(suite.ctx, suite.resolver, query)
	suite.NoError(err)

	for i, vuln := range actual {
		suite.Equal(expectedVulns[i].ID, vuln.ID)
	}
}

func (suite *CVEScopingTestSuite) TestGetVulnsWithScopingOrder() {
	deploymentID := "deployment1"
	imageID := "image1"

	query := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.DeploymentID.String(), Value: deploymentID},
						},
					},
				}},
				{Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{Field: search.ImageSHA.String(), Value: imageID},
						},
					},
				}},
			},
		}},
	}

	expectedVulns := []search.Result{
		{
			ID: "cve1",
		},
		{
			ID: "cve2",
		},
	}

	suite.imageDataStore.EXPECT().Search(suite.ctx, query).Return([]search.Result{{ID: imageID}}, nil)

	scopedCtx := scoped.Context(suite.ctx, scoped.Scope{
		Level: v1.SearchCategory_IMAGES,
		ID:    imageID,
	})
	suite.cveDataStore.EXPECT().Search(scopedCtx, query).Return(expectedVulns, nil)

	actual, err := runAsScopedQuery(suite.ctx, suite.resolver, query)
	suite.NoError(err)

	for i, vuln := range actual {
		suite.Equal(expectedVulns[i].ID, vuln.ID)
	}
}
