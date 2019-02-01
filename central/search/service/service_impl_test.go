package service

import (
	"testing"

	"github.com/golang/mock/gomock"
	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	"github.com/stackrox/rox/central/deployment/datastore"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchCategoryToResourceMap(t *testing.T) {
	for _, searchCategory := range GetAllSearchableCategories() {
		_, ok := searchCategoryToResource[searchCategory]
		// This is a programming error. If you see this, add the new category you've added to the
		// searchCategoryToResource map!
		assert.True(t, ok, "Please add category %s to the searchCategoryToResource map used by the authorizer", searchCategory.String())
	}
}

func TestSearchFuncs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	s := New(alertMocks.NewMockDataStore(mockCtrl), deploymentMocks.NewMockDataStore(mockCtrl), imageMocks.NewMockDataStore(mockCtrl), policyMocks.NewMockDataStore(mockCtrl), secretMocks.NewMockDataStore(mockCtrl))
	searchFuncMap := s.(*serviceImpl).getSearchFuncs()
	for _, searchCategory := range GetAllSearchableCategories() {
		_, ok := searchFuncMap[searchCategory]
		// This is a programming error. If you see this, add the new category you've added to the
		// searchCategoryToResource map!
		assert.True(t, ok, "Please add category %s to the map in getSearchFuncs()", searchCategory.String())
	}
}

func TestAutocomplete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Create Deployment Indexer
	idx, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)
	deploymentIndexer := index.New(idx)

	deploymentNameOneOff := fixtures.GetDeployment()
	require.NoError(t, deploymentIndexer.AddDeployment(deploymentNameOneOff))

	deploymentName1 := fixtures.GetDeployment()
	deploymentName1.Id = "name1"
	deploymentName1.Name = "name1"
	require.NoError(t, deploymentIndexer.AddDeployment(deploymentName1))

	deploymentName1Duplicate := fixtures.GetDeployment()
	deploymentName1Duplicate.Id = "name1Dup"
	deploymentName1Duplicate.Name = "name1"
	require.NoError(t, deploymentIndexer.AddDeployment(deploymentName1Duplicate))

	deploymentName2 := fixtures.GetDeployment()
	deploymentName2.Id = "name12"
	deploymentName2.Name = "name12"
	require.NoError(t, deploymentIndexer.AddDeployment(deploymentName2))

	ds := datastore.New(nil, deploymentIndexer, nil, nil)

	service := New(
		alertMocks.NewMockDataStore(mockCtrl),
		ds,
		imageMocks.NewMockDataStore(mockCtrl),
		policyMocks.NewMockDataStore(mockCtrl),
		secretMocks.NewMockDataStore(mockCtrl),
	).(*serviceImpl)

	q := search.NewQueryBuilder().AddStrings(search.DeploymentName, deploymentNameOneOff.Name).Query()
	results, err := service.autocomplete(q, []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS})
	require.NoError(t, err)
	assert.Equal(t, []string{deploymentNameOneOff.Name}, results)

	q = search.NewQueryBuilder().AddStrings(search.DeploymentName, "name").Query()
	results, err = service.autocomplete(q, []v1.SearchCategory{v1.SearchCategory_DEPLOYMENTS})
	require.NoError(t, err)
	// This is odd, but this is correct. Bleve scores name12 higher than name1
	assert.Equal(t, []string{"name12", "name1"}, results)
}
