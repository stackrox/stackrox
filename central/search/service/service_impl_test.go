package service

import (
	"testing"

	alertMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	policyMocks "github.com/stackrox/rox/central/policy/datastore/mocks"
	secretMocks "github.com/stackrox/rox/central/secret/datastore/mocks"
	"github.com/stretchr/testify/assert"
)

func TestSearchCategoryToResourceMap(t *testing.T) {
	for _, searchCategory := range getAllSearchableCategories() {
		_, ok := searchCategoryToResource[searchCategory]
		// This is a programming error. If you see this, add the new category you've added to the
		// searchCategoryToResource map!
		assert.True(t, ok, "Please add category %s to the searchCategoryToResource map used by the authorizer", searchCategory.String())
	}
}

func TestSearchFuncs(t *testing.T) {
	s := New(&alertMocks.DataStore{}, &deploymentMocks.DataStore{}, &imageMocks.DataStore{}, &policyMocks.DataStore{}, &secretMocks.DataStore{})
	searchFuncMap := s.(*serviceImpl).getSearchFuncs()
	for _, searchCategory := range getAllSearchableCategories() {
		_, ok := searchFuncMap[searchCategory]
		// This is a programming error. If you see this, add the new category you've added to the
		// searchCategoryToResource map!
		assert.True(t, ok, "Please add category %s to the map in getSearchFuncs()", searchCategory.String())
	}
}
