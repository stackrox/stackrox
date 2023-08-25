//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"

	alertDatastore "github.com/stackrox/rox/central/alert/datastore"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageIntegrationDataStoreMocks "github.com/stackrox/rox/central/imageintegration/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	nodeMocks "github.com/stackrox/rox/central/node/datastore/mocks"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	policyIndex "github.com/stackrox/rox/central/policy/index"
	policySearcher "github.com/stackrox/rox/central/policy/search"
	policyPostgres "github.com/stackrox/rox/central/policy/store/postgres"
	categoryDataStoreMocks "github.com/stackrox/rox/central/policycategory/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	roleMocks "github.com/stackrox/rox/central/rbac/k8srole/datastore/mocks"
	roleBindingsMocks "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore/mocks"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	serviceAccountMocks "github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestSearchCategoryToOptionsMultiMap(t *testing.T) {
	t.Parallel()

	for cat := range autocompleteCategories {
		_, ok := categoryToOptionsMultimap[cat]
		assert.True(t, ok, "no options multimap for category", cat)
	}
}

func TestSearchFuncs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	builder := NewBuilder().
		WithAlertStore(alertMocks.NewMockDataStore(mockCtrl)).
		WithDeploymentStore(deploymentMocks.NewMockDataStore(mockCtrl)).
		WithImageStore(imageMocks.NewMockDataStore(mockCtrl)).
		WithPolicyStore(policyMocks.NewMockDataStore(mockCtrl)).
		WithSecretStore(secretMocks.NewMockDataStore(mockCtrl)).
		WithServiceAccountStore(serviceAccountMocks.NewMockDataStore(mockCtrl)).
		WithNodeStore(nodeMocks.NewMockDataStore(mockCtrl)).
		WithNamespaceStore(namespaceMocks.NewMockDataStore(mockCtrl)).
		WithRiskStore(riskDatastoreMocks.NewMockDataStore(mockCtrl)).
		WithRoleStore(roleMocks.NewMockDataStore(mockCtrl)).
		WithRoleBindingStore(roleBindingsMocks.NewMockDataStore(mockCtrl)).
		WithClusterDataStore(clusterDataStoreMocks.NewMockDataStore(mockCtrl)).
		WithImageIntegrationStore(imageIntegrationDataStoreMocks.NewMockDataStore(mockCtrl)).
		WithAggregator(nil).
		WithPolicyCategoryDataStore(categoryDataStoreMocks.NewMockDataStore(mockCtrl))

	s := builder.Build()

	searchFuncMap := s.(*serviceImpl).getSearchFuncs()
	for _, searchCategory := range GetAllSearchableCategories() {
		_, ok := searchFuncMap[searchCategory]
		// This is a programming error. If you see this, add the new category you've added to the
		// SearchCategoryToResource map!
		assert.True(t, ok, "Please add category %s to the map in getSearchFuncs()", searchCategory.String())
	}
}

func TestSearchService(t *testing.T) {
	suite.Run(t, new(SearchOperationsTestSuite))
}

type SearchOperationsTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
	pool     postgres.DB
}

func (s *SearchOperationsTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	testingDB := pgtest.ForT(s.T())
	s.pool = testingDB.DB
}

func (s *SearchOperationsTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
	s.pool.Close()
}

func (s *SearchOperationsTestSuite) TestAutocomplete() {
	var (
		deploymentDS deploymentDatastore.DataStore
		err          error
	)

	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(s.mockCtrl)
	// Since we are using the datastore and not the store we need to create a ranker and use it to populate the
	// risk score so the results are ordered correctly.
	deploymentRanker := ranking.NewRanker()
	deploymentDS, err = deploymentDatastore.New(s.pool, nil, nil, nil, mockRiskDatastore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), deploymentRanker)
	s.Require().NoError(err)

	allAccessCtx := sac.WithAllAccess(context.Background())

	deploymentNameOneOff := fixtures.GetDeployment()
	deploymentRanker.Add(deploymentNameOneOff.GetId(), 50)
	s.NoError(deploymentDS.UpsertDeployment(allAccessCtx, deploymentNameOneOff))

	deploymentName1 := fixtures.GetDeployment()
	deploymentName1.Id = fixtureconsts.Deployment2
	deploymentName1.Name = "name1"
	deploymentRanker.Add(fixtureconsts.Deployment2, 25)
	s.NoError(deploymentDS.UpsertDeployment(allAccessCtx, deploymentName1))

	deploymentName1Duplicate := fixtures.GetDeployment()
	deploymentName1Duplicate.Id = fixtureconsts.Deployment3
	deploymentName1Duplicate.Name = "name1"
	deploymentRanker.Add(fixtureconsts.Deployment3, 25)
	s.NoError(deploymentDS.UpsertDeployment(allAccessCtx, deploymentName1Duplicate))

	deploymentName2 := fixtures.GetDeployment()
	deploymentName2.Id = fixtureconsts.Deployment4
	deploymentName2.Name = "name12"
	deploymentName2.Labels = map[string]string{"hello": "hi", "hey": "ho"}
	deploymentRanker.Add(fixtureconsts.Deployment4, 100)
	s.NoError(deploymentDS.UpsertDeployment(allAccessCtx, deploymentName2))

	builder := NewBuilder().
		WithAlertStore(alertMocks.NewMockDataStore(s.mockCtrl)).
		WithDeploymentStore(deploymentDS).
		WithImageStore(imageMocks.NewMockDataStore(s.mockCtrl)).
		WithPolicyStore(policyMocks.NewMockDataStore(s.mockCtrl)).
		WithSecretStore(secretMocks.NewMockDataStore(s.mockCtrl)).
		WithServiceAccountStore(serviceAccountMocks.NewMockDataStore(s.mockCtrl)).
		WithNodeStore(nodeMocks.NewMockDataStore(s.mockCtrl)).
		WithNamespaceStore(namespaceMocks.NewMockDataStore(s.mockCtrl)).
		WithRiskStore(riskDatastoreMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleStore(roleMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleBindingStore(roleBindingsMocks.NewMockDataStore(s.mockCtrl)).
		WithClusterDataStore(clusterDataStoreMocks.NewMockDataStore(s.mockCtrl)).
		WithAggregator(nil).
		WithPolicyCategoryDataStore(categoryDataStoreMocks.NewMockDataStore(s.mockCtrl))

	service := builder.Build().(*serviceImpl)

	for _, testCase := range []struct {
		query           string
		expectedResults []string
		postgresResults []string
		ignoreOrder     bool
	}{
		{
			query:           search.NewQueryBuilder().AddStrings(search.DeploymentName, deploymentNameOneOff.Name).Query(),
			expectedResults: []string{deploymentNameOneOff.GetName()},
		},
		{
			query: search.NewQueryBuilder().AddStrings(search.DeploymentName, "name").Query(),
			// This is odd, but this is correct. Bleve scores name12 higher than name1
			expectedResults: []string{"name12", "name1"},
		},
		{
			query:           fmt.Sprintf("%s:", search.DeploymentName),
			expectedResults: []string{"name12", "nginx_server", "name1"},
		},
		{
			query:           fmt.Sprintf("%s:name12,", search.DeploymentName),
			expectedResults: []string{"name12", "nginx_server", "name1"},
		},
		{
			query:           fmt.Sprintf("%s:he=h", search.DeploymentLabel),
			expectedResults: []string{"hello=hi", "hey=ho"},
			ignoreOrder:     true,
		},
		{
			query:           fmt.Sprintf("%s:hey=", search.DeploymentLabel),
			expectedResults: []string{"hey=ho"},
			ignoreOrder:     true,
		},
		{
			query:           fmt.Sprintf("%s:%s+%s:", search.DeploymentName, deploymentName2.Name, search.DeploymentLabel),
			expectedResults: []string{"hello=hi", "hey=ho"},
			ignoreOrder:     true,
		},
	} {
		s.Run(fmt.Sprintf("Test case %q", testCase.query), func() {
			results, err := service.autocomplete(allAccessCtx, testCase.query, []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS})
			s.NoError(err)
			if testCase.ignoreOrder {
				s.ElementsMatch(testCase.expectedResults, results)
			} else {
				s.Equal(testCase.expectedResults, results)
			}
		})
	}
}

func (s *SearchOperationsTestSuite) TestAutocompleteForEnums() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Create Policy Searcher
	var policyIndexer policyIndex.Indexer
	var ds policyDatastore.DataStore

	categoriesDS := categoryDataStoreMocks.NewMockDataStore(s.mockCtrl)
	policyStore := policyPostgres.New(s.pool)
	policyIndexer = policyPostgres.NewIndexer(s.pool)
	s.NoError(policyStore.Upsert(ctx, fixtures.GetPolicy()))
	policySearcher := policySearcher.New(policyStore, policyIndexer)
	ds = policyDatastore.New(policyStore, policySearcher, nil, nil, categoriesDS)

	builder := NewBuilder().
		WithAlertStore(alertMocks.NewMockDataStore(s.mockCtrl)).
		WithDeploymentStore(deploymentMocks.NewMockDataStore(s.mockCtrl)).
		WithImageStore(imageMocks.NewMockDataStore(s.mockCtrl)).
		WithPolicyStore(ds).
		WithSecretStore(secretMocks.NewMockDataStore(s.mockCtrl)).
		WithServiceAccountStore(serviceAccountMocks.NewMockDataStore(s.mockCtrl)).
		WithNodeStore(nodeMocks.NewMockDataStore(s.mockCtrl)).
		WithNamespaceStore(namespaceMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleStore(roleMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleBindingStore(roleBindingsMocks.NewMockDataStore(s.mockCtrl)).
		WithClusterDataStore(clusterDataStoreMocks.NewMockDataStore(s.mockCtrl)).
		WithAggregator(nil).
		WithPolicyCategoryDataStore(categoriesDS)

	service := builder.Build().(*serviceImpl)

	results, err := service.autocomplete(ctx, fmt.Sprintf("%s:", search.Severity), []v1.SearchCategory{v1.SearchCategory_POLICIES})
	s.NoError(err)
	s.Equal([]string{fixtures.GetPolicy().GetSeverity().String()}, results)
}

func (s *SearchOperationsTestSuite) TestAutocompleteAuthz() {
	deploymentAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment)))
	alertAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	noAccessCtx := sac.WithNoAccess(context.Background())

	var (
		alertsDS     alertDatastore.DataStore
		deploymentDS deploymentDatastore.DataStore
		err          error
	)

	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(s.mockCtrl)
	deploymentDS, err = deploymentDatastore.New(s.pool, nil, nil, nil, mockRiskDatastore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	s.Require().NoError(err)

	alertsDS, err = alertDatastore.GetTestPostgresDataStore(s.T(), s.pool)
	s.NoError(err)

	deployment := fixtures.GetDeployment()
	s.NoError(deploymentDS.UpsertDeployment(deploymentAccessCtx, deployment))

	alert := fixtures.GetAlert()
	s.NoError(alertsDS.UpsertAlert(alertAccessCtx, alert))

	builder := NewBuilder().
		WithAlertStore(alertsDS).
		WithDeploymentStore(deploymentDS).
		WithImageStore(imageMocks.NewMockDataStore(s.mockCtrl)).
		WithPolicyStore(policyMocks.NewMockDataStore(s.mockCtrl)).
		WithSecretStore(secretMocks.NewMockDataStore(s.mockCtrl)).
		WithServiceAccountStore(serviceAccountMocks.NewMockDataStore(s.mockCtrl)).
		WithNodeStore(nodeMocks.NewMockDataStore(s.mockCtrl)).
		WithNamespaceStore(namespaceMocks.NewMockDataStore(s.mockCtrl)).
		WithRiskStore(riskDatastoreMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleStore(roleMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleBindingStore(roleBindingsMocks.NewMockDataStore(s.mockCtrl)).
		WithClusterDataStore(clusterDataStoreMocks.NewMockDataStore(s.mockCtrl)).
		WithAggregator(nil)

	builder = builder.WithPolicyCategoryDataStore(categoryDataStoreMocks.NewMockDataStore(s.mockCtrl))
	service := builder.Build().(*serviceImpl)

	deploymentQuery := search.NewQueryBuilder().AddStrings(search.DeploymentName, deployment.Name).Query()
	alertQuery := search.NewQueryBuilder().AddStrings(search.DeploymentName, alert.GetDeployment().GetName()).Query()

	// If caller has "Deployment" permission, return results in "Deployment" category
	results, err := service.autocomplete(deploymentAccessCtx, deploymentQuery, []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS})
	s.NoError(err)
	s.Equal([]string{deployment.GetName()}, results)

	// If caller has no "Deployment" permission, return no results in "Deployment" category
	results, err = service.autocomplete(noAccessCtx, deploymentQuery, []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS})
	s.NoError(err)
	s.Equal([]string(nil), results)

	// If caller has "Alert" permission, return results in "Alert" category
	results, err = service.autocomplete(alertAccessCtx, alertQuery, []v1.SearchCategory{v1.SearchCategory_ALERTS})
	s.NoError(err)
	s.Equal([]string{alert.GetDeployment().GetName()}, results)

	// If caller has no "Alert" permission but "Deployment" permission, return no results in "Alert" category
	results, err = service.autocomplete(deploymentAccessCtx, alertQuery, []v1.SearchCategory{v1.SearchCategory_ALERTS})
	s.NoError(err)
	s.Equal([]string(nil), results)
}

func (s *SearchOperationsTestSuite) TestSearchAuthz() {
	deploymentAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment)))
	alertAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Alert)))
	noAccessCtx := sac.WithNoAccess(context.Background())

	var (
		alertsDS     alertDatastore.DataStore
		deploymentDS deploymentDatastore.DataStore
		err          error
	)

	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(s.mockCtrl)
	deploymentDS, err = deploymentDatastore.New(s.pool, nil, nil, nil, mockRiskDatastore, nil, nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	s.Require().NoError(err)

	alertsDS, err = alertDatastore.GetTestPostgresDataStore(s.T(), s.pool)
	s.NoError(err)

	deployment := fixtures.GetDeployment()
	s.NoError(deploymentDS.UpsertDeployment(deploymentAccessCtx, deployment))

	alert := fixtures.GetAlert()
	s.NoError(alertsDS.UpsertAlert(alertAccessCtx, alert))

	builder := NewBuilder().
		WithAlertStore(alertsDS).
		WithDeploymentStore(deploymentDS).
		WithImageStore(imageMocks.NewMockDataStore(s.mockCtrl)).
		WithPolicyStore(policyMocks.NewMockDataStore(s.mockCtrl)).
		WithSecretStore(secretMocks.NewMockDataStore(s.mockCtrl)).
		WithServiceAccountStore(serviceAccountMocks.NewMockDataStore(s.mockCtrl)).
		WithNodeStore(nodeMocks.NewMockDataStore(s.mockCtrl)).
		WithNamespaceStore(namespaceMocks.NewMockDataStore(s.mockCtrl)).
		WithRiskStore(riskDatastoreMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleStore(roleMocks.NewMockDataStore(s.mockCtrl)).
		WithRoleBindingStore(roleBindingsMocks.NewMockDataStore(s.mockCtrl)).
		WithClusterDataStore(clusterDataStoreMocks.NewMockDataStore(s.mockCtrl)).
		WithImageIntegrationStore(imageIntegrationDataStoreMocks.NewMockDataStore(s.mockCtrl)).
		WithAggregator(nil).
		WithPolicyCategoryDataStore(categoryDataStoreMocks.NewMockDataStore(s.mockCtrl))

	service := builder.Build().(*serviceImpl)

	deploymentQuery := search.NewQueryBuilder().AddStrings(search.DeploymentName, deployment.Name).Query()
	alertQuery := search.NewQueryBuilder().AddStrings(search.DeploymentName, alert.GetDeployment().GetName()).Query()

	// If caller has "Deployment" permission, return results in "Deployment" category
	results, err := service.Search(deploymentAccessCtx, &v1.RawSearchRequest{
		Query:      deploymentQuery,
		Categories: []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS},
	})
	s.NoError(err)
	s.Len(results.GetResults(), 1)
	s.Equal(deployment.GetName(), results.GetResults()[0].GetName())

	// If caller has no "Deployment" permission, return no results in "Deployment" category
	results, err = service.Search(noAccessCtx, &v1.RawSearchRequest{
		Query:      deploymentQuery,
		Categories: []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS},
	})
	s.NoError(err)
	s.Len(results.GetResults(), 0)

	// If caller has "Alert" permission, return results in "Alert" category
	results, err = service.Search(alertAccessCtx, &v1.RawSearchRequest{
		Query:      alertQuery,
		Categories: []v1.SearchCategory{v1.SearchCategory_ALERTS},
	})
	s.NoError(err)
	s.Len(results.GetResults(), 1)
	s.Equal(results.GetResults()[0].GetId(), alert.GetId())

	// If caller has no "Alert" permission but "Deployment" permission, return no results in "Alert" category
	results, err = service.Search(deploymentAccessCtx, &v1.RawSearchRequest{
		Query:      alertQuery,
		Categories: []v1.SearchCategory{v1.SearchCategory_ALERTS},
	})
	s.NoError(err)
	s.Len(results.GetResults(), 0)
}
