package search

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
)

// GetGlobalSearchCategories returns a set of search categories
func GetGlobalSearchCategories() set.Set[v1.SearchCategory] {
	// globalSearchCategories is exposed for e2e options test
	globalSearchCategories := set.NewSet(
		v1.SearchCategory_ALERTS,
		v1.SearchCategory_CLUSTERS,
		v1.SearchCategory_DEPLOYMENTS,
		v1.SearchCategory_IMAGES,
		v1.SearchCategory_NODES,
		v1.SearchCategory_NAMESPACES,
		v1.SearchCategory_POLICIES,
		v1.SearchCategory_SECRETS,
		v1.SearchCategory_SERVICE_ACCOUNTS,
		v1.SearchCategory_ROLES,
		v1.SearchCategory_ROLEBINDINGS,
		v1.SearchCategory_SUBJECTS,
		v1.SearchCategory_IMAGE_INTEGRATIONS,
		v1.SearchCategory_POLICY_CATEGORIES,
	)
	return globalSearchCategories
}
