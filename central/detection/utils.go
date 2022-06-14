package detection

import (
	"strings"

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
		lowercaseCategory := strings.ToLower(category)
		allowedCategorySet.Add(lowercaseCategory)
		unusedCategorySet.Add(lowercaseCategory)
	}

	filterOption := func(policy *storage.Policy) bool {
		if allowedCategorySet.Cardinality() == 0 {
			return true
		}

		foundAllowedCategory := false
		for _, category := range policy.GetCategories() {
			lowercaseCategory := strings.ToLower(category)
			if allowedCategorySet.Contains(lowercaseCategory) {
				unusedCategorySet.Remove(lowercaseCategory)
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
