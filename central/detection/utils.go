package detection

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/set"
)

// MakeCategoryFilter takes a list of category names and returns two functions:
// a FilterOption which will match any policy with any of the category names, ignoring case
// a function which will return all category names which have not been passed to the FilterOption
func MakeCategoryFilter(filterForCategories []string) (detection.FilterOption, func() []string) {
	allowedCategorySet := set.NewStringSet()
	unusedCategorySet := set.NewStringSet()
	for _, category := range filterForCategories {
		allowedCategorySet.Add(category)
		unusedCategorySet.Add(category)
	}

	filterOption := func(policy *storage.Policy) bool {
		if allowedCategorySet.Cardinality() == 0 {
			return true
		}

		foundAllowedCategory := false
		for _, category := range policy.GetCategories() {
			if allowedCategorySet.Contains(category) {
				unusedCategorySet.Remove(category)
				foundAllowedCategory = true
			}
		}

		return foundAllowedCategory
	}

	getUnused := func() []string {
		return unusedCategorySet.AsSlice()
	}

	return filterOption, getUnused
}
