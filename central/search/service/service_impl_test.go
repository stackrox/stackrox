package service

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestSearchCategoryToResourceMap(t *testing.T) {
	// Iterating over all the values of search category in the proto to make sure that programmers don't
	// forget to add an entry to searchCategoryToResource after they add a new category.
	for id, name := range v1.SearchCategory_name {
		searchCategory := v1.SearchCategory(id)
		if searchCategory == v1.SearchCategory_SEARCH_UNSET {
			continue
		}
		_, ok := searchCategoryToResource[searchCategory]
		// This is a programming error. If you see this, add the new category you've added to the
		// searchCategoryToResource map!
		assert.True(t, ok, "Please add category %s to the searchCategoryToResource map used by the authorizer", name)
	}
}
