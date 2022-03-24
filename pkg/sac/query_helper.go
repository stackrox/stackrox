package sac

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
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
	clusterFilters := make([]*v1.Query, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		clusterAccessScope := root.GetClusterByID(clusterID)
		if clusterAccessScope == nil {
			continue
		}
		if clusterAccessScope.State == effectiveaccessscope.Included {
			clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID)
			clusterFilters = append(clusterFilters, clusterQuery.ProtoQuery())
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
		if clusterAccessScope.State == effectiveaccessscope.Included {
			clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID)
			clusterFilters = append(clusterFilters, clusterQuery.ProtoQuery())
		} else if clusterAccessScope.State == effectiveaccessscope.Partial {
			clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID)
			namespaces := clusterAccessScope.Namespaces
			namespaceFilters := make([]*v1.Query, 0, len(namespaces))
			for namespaceName, namespaceAccessScope := range namespaces {
				if namespaceAccessScope.State == effectiveaccessscope.Included {
					namespaceSubQuery := search.NewQueryBuilder().AddExactMatches(search.Namespace, namespaceName)
					namespaceFilters = append(namespaceFilters, namespaceSubQuery.ProtoQuery())
				}
			}
			if len(namespaceFilters) > 0 {
				namespaceSubQuery := search.DisjunctionQuery(namespaceFilters...)
				clusterFilter := search.ConjunctionQuery(clusterQuery.ProtoQuery(), namespaceSubQuery)
				clusterFilters = append(clusterFilters, clusterFilter)
			}
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
