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
	return buildClusterLevelSACQueryFilter(root, true)
}

// BuildNonVerboseClusterLevelSACQueryFilter builds a Scoped Access Control query filter that can be
// injected in search queries for resource types that have direct cluster scope level.
func BuildNonVerboseClusterLevelSACQueryFilter(root *effectiveaccessscope.ScopeTree) (*v1.Query, error) {
	return buildClusterLevelSACQueryFilter(root, false)
}

func buildClusterLevelSACQueryFilter(root *effectiveaccessscope.ScopeTree, verbose bool) (*v1.Query, error) {
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
			clusterFilters = append(clusterFilters, getClusterMatchQuery(clusterID, verbose))
		} else if clusterAccessScope.State == effectiveaccessscope.Partial &&
			len(clusterAccessScope.Namespaces) > 0 {
			clusterFilters = append(clusterFilters, getClusterMatchQuery(clusterID, verbose))
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
	return buildClusterNamespaceLevelSACQueryFilter(root, true)
}

// BuildNonVerboseClusterNamespaceLevelSACQueryFilter builds a Scoped Access Control query filter that can be
// injected in search queries for resource types that have direct namespace scope level.
func BuildNonVerboseClusterNamespaceLevelSACQueryFilter(root *effectiveaccessscope.ScopeTree) (*v1.Query, error) {
	return buildClusterNamespaceLevelSACQueryFilter(root, false)
}

func buildClusterNamespaceLevelSACQueryFilter(root *effectiveaccessscope.ScopeTree, verbose bool) (*v1.Query, error) {
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
			var clusterQuery *v1.Query
			if verbose {
				clusterQuery = search.ConjunctionQuery(getClusterMatchQuery(clusterID, verbose),
					getAnyNamespaceMatchQuery())
			} else {
				clusterQuery = getClusterMatchQuery(clusterID, verbose)
			}
			clusterFilters = append(clusterFilters, clusterQuery)
		} else if clusterAccessScope.State == effectiveaccessscope.Partial {
			clusterQuery := getClusterMatchQuery(clusterID, verbose)
			namespaces := clusterAccessScope.Namespaces
			namespaceFilters := make([]*v1.Query, 0, len(namespaces))
			for namespaceName, namespaceAccessScope := range namespaces {
				if namespaceAccessScope.State == effectiveaccessscope.Included {
					namespaceFilters = append(namespaceFilters, getNamespaceMatchQuery(namespaceName, verbose))
				}
			}
			if len(namespaceFilters) > 0 {
				namespaceSubQuery := search.DisjunctionQuery(namespaceFilters...)
				clusterFilter := search.ConjunctionQuery(clusterQuery, namespaceSubQuery)
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

func getClusterMatchQuery(clusterID string, verbose bool) *v1.Query {
	if verbose {
		return search.NewQueryBuilder().AddExactMatches(clusterIDField, clusterID).MarkHighlighted(clusterIDField).ProtoQuery()
	}
	return search.NewQueryBuilder().AddExactMatches(clusterIDField, clusterID).ProtoQuery()
}

func getNamespaceMatchQuery(namespace string, verbose bool) *v1.Query {
	if verbose {
		return search.NewQueryBuilder().AddExactMatches(namespaceField, namespace).MarkHighlighted(namespaceField).ProtoQuery()
	}
	return search.NewQueryBuilder().AddExactMatches(namespaceField, namespace).ProtoQuery()
}

func getAnyNamespaceMatchQuery() *v1.Query {
	return search.NewQueryBuilder().AddStringsHighlighted(namespaceField, search.WildcardString).ProtoQuery()
}
