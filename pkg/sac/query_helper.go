package sac

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
)

var (
	clusterIDField = search.ClusterID
	namespaceField = search.Namespace
)

// BuildClusterLevelSACQueryFilter builds a Scoped Access Control query filter that can be
// injected in search queries for resource types that have direct cluster scope level.
func BuildClusterLevelSACQueryFilter(root *effectiveaccessscope.ScopeTree) (*v1.Query, error) {
	if root == nil {
		return getMatchNoneQuery(), nil
	}
	if root.State == effectiveaccessscope.Included {
		return nil, nil
	}
	if root.State == effectiveaccessscope.Excluded {
		return getMatchNoneQuery(), nil
	}
	clusterIDs := root.GetClusterIDs()
	clusterFilters := make([]string, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		clusterAccessScope := root.GetClusterByID(clusterID)
		// Partial clusters have at least one included namespace, possibly via
		// the default of an exclusion map.
		if clusterAccessScope == nil ||
			clusterAccessScope.State == effectiveaccessscope.Excluded ||
			(clusterAccessScope.State == effectiveaccessscope.Partial && !clusterAccessScope.HasNamespaceAccess()) {
			continue
		}
		clusterFilters = append(clusterFilters, clusterID)
	}
	switch len(clusterFilters) {
	case 0:
		return getMatchNoneQuery(), nil
	default:
		return getClusterMatchQuery(clusterFilters...), nil
	}
}

// BuildClusterNamespaceLevelSACQueryFilter builds a Scoped Access Control query filter that can be
// injected in search queries for resource types that have direct namespace scope level.
func BuildClusterNamespaceLevelSACQueryFilter(root *effectiveaccessscope.ScopeTree) (*v1.Query, error) {
	if root == nil {
		return getMatchNoneQuery(), nil
	}
	if root.State == effectiveaccessscope.Excluded {
		return getMatchNoneQuery(), nil
	}
	if root.State == effectiveaccessscope.Included {
		return nil, nil
	}
	clusterIDs := root.GetClusterIDs()
	clusterFilters := make([]*v1.Query, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		clusterAccessScope := root.GetClusterByID(clusterID)
		if clusterAccessScope == nil {
			continue
		}

		clusterQuery := getClusterMatchQuery(clusterID)

		switch clusterAccessScope.State {
		case effectiveaccessscope.Included:
			clusterFilters = append(clusterFilters, clusterQuery)
		case effectiveaccessscope.Partial:
			namespaceFilters, exclusions := clusterAccessScope.NamespaceFilterNames()
			if len(namespaceFilters) > 0 {
				namespaceSubQuery := getNamespaceMatchQuery(namespaceFilters...)
				clusterFilter := search.ConjunctionQuery(clusterQuery, namespaceSubQuery)
				if len(exclusions) > 0 {
					clusterFilter = search.NewBooleanQuery(
						&v1.ConjunctionQuery{Queries: []*v1.Query{clusterQuery, namespaceSubQuery}},
						&v1.DisjunctionQuery{Queries: []*v1.Query{getNamespaceMatchQuery(exclusions...)}})
				}
				clusterFilters = append(clusterFilters, clusterFilter)
			}
		case effectiveaccessscope.Excluded:
			continue
		}
	}
	switch len(clusterFilters) {
	case 0:
		return getMatchNoneQuery(), nil
	case 1:
		return clusterFilters[0], nil
	default:
		return search.DisjunctionQuery(clusterFilters...), nil
	}
}

func getMatchNoneQuery() *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchNoneQuery{
					MatchNoneQuery: &v1.MatchNoneQuery{},
				},
			},
		},
	}
}

func getClusterMatchQuery(clusterID ...string) *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(clusterIDField, clusterID...).ProtoQuery()
}

func getNamespaceMatchQuery(namespace ...string) *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(namespaceField, namespace...).ProtoQuery()
}
