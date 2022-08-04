package tests

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testNSResource = permissions.ResourceMetadata{
		Resource: "test-resource",
		Scope:    permissions.NamespaceScope,
	}
	testClusterResource = permissions.ResourceMetadata{
		Resource: "test-resource",
		Scope:    permissions.ClusterScope,
	}
)

func fakeResult(id, cluster, namespace string) search.Result {
	return search.Result{
		ID: id,
		Fields: map[string]interface{}{
			"cluster_id": cluster,
			"namespace":  namespace,
		},
	}
}

func TestSearchHelper_TestApply_WithFilter(t *testing.T) {
	options := search.OptionsMapFromMap(v1.SearchCategory_DEPLOYMENTS, map[search.FieldLabel]*search.Field{
		search.ClusterID: {
			FieldPath: "cluster_id",
			Store:     true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
		},
		search.Namespace: {
			FieldPath: "namespace",
			Store:     true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
		},
	})

	mockSearchFunc := func(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
		return []search.Result{
			fakeResult("1", "cluster1", "nsA"),
			fakeResult("2", "cluster1", "nsB"),
			fakeResult("3", "cluster2", "nsA"),
			fakeResult("4", "cluster2", "nsB"),
			fakeResult("5", "cluster3", "nsA"),
			fakeResult("6", "cluster3", "nsB"),
		}, nil
	}

	h, err := sac.NewSearchHelper(testNSResource, options, sac.ForResource(testNSResource).ScopeChecker)
	require.NoError(t, err)

	scc := sac.TestScopeCheckerCoreFromFullScopeMap(t,
		sac.TestScopeMap{
			storage.Access_READ_ACCESS: {
				testNSResource.GetResource(): &sac.TestResourceScope{
					Clusters: map[string]*sac.TestClusterScope{
						"cluster1": {Included: true},
						"cluster2": {Namespaces: []string{"nsA"}},
					},
				},
			},
		})

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

	searchResults, err := h.Apply(mockSearchFunc)(ctx, search.EmptyQuery())
	require.NoError(t, err)

	resultIDs := search.ResultsToIDs(searchResults)
	assert.ElementsMatch(t, resultIDs, []string{"1", "2", "3"})
}

func TestSearchHelper_TestApply_WithAllAccess(t *testing.T) {
	options := search.OptionsMapFromMap(v1.SearchCategory_DEPLOYMENTS, map[search.FieldLabel]*search.Field{
		search.ClusterID: {
			FieldPath: "cluster_id",
			Store:     true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
		},
		search.Namespace: {
			FieldPath: "namespace",
			Store:     true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
		},
	})

	mockSearchFunc := func(q *v1.Query, options ...blevesearch.SearchOption) ([]search.Result, error) {
		return []search.Result{
			fakeResult("1", "cluster1", "nsA"),
			fakeResult("2", "cluster1", "nsB"),
			fakeResult("3", "cluster2", "nsA"),
			fakeResult("4", "cluster2", "nsB"),
			fakeResult("5", "cluster3", "nsA"),
			fakeResult("6", "cluster3", "nsB"),
		}, nil
	}

	h, err := sac.NewSearchHelper(testNSResource, options, sac.ForResource(testNSResource).ScopeChecker)
	require.NoError(t, err)

	scc := sac.AllowAllAccessScopeChecker()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), scc)

	searchResults, err := h.Apply(mockSearchFunc)(ctx, search.EmptyQuery())
	require.NoError(t, err)
	resultIDs := search.ResultsToIDs(searchResults)
	assert.ElementsMatch(t, resultIDs, []string{"1", "2", "3", "4", "5", "6"})
}

func TestSearchHelper_TestNew_WithMissingClusterIDField(t *testing.T) {
	options := search.OptionsMapFromMap(v1.SearchCategory_DEPLOYMENTS, map[search.FieldLabel]*search.Field{
		search.Namespace: {
			FieldPath: "namespace",
			Store:     true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
		},
	})

	_, err := sac.NewSearchHelper(testClusterResource, options, sac.ForResource(testClusterResource).ScopeChecker)
	assert.Error(t, err)
}

func TestSearchHelper_TestNew_WithFieldNotStored(t *testing.T) {
	options := search.OptionsMapFromMap(v1.SearchCategory_CLUSTERS, map[search.FieldLabel]*search.Field{
		search.ClusterID: {
			FieldPath: "cluster_id",
			Store:     false,
			Category:  v1.SearchCategory_CLUSTERS,
		},
	})

	_, err := sac.NewSearchHelper(testClusterResource, options, sac.ForResource(testClusterResource).ScopeChecker)
	assert.Error(t, err)
}

func TestSearchHelper_TestNew_WithMissingNSField_NotScoped(t *testing.T) {
	options := search.OptionsMapFromMap(v1.SearchCategory_CLUSTERS, map[search.FieldLabel]*search.Field{
		search.ClusterID: {
			FieldPath: "cluster_id",
			Store:     true,
			Category:  v1.SearchCategory_CLUSTERS,
		},
	})

	_, err := sac.NewSearchHelper(testClusterResource, options, sac.ForResource(testClusterResource).ScopeChecker)
	assert.NoError(t, err)
}

func TestSearchHelper_TestNew_WithMissingNSField_Scoped(t *testing.T) {
	options := search.OptionsMapFromMap(v1.SearchCategory_DEPLOYMENTS, map[search.FieldLabel]*search.Field{
		search.ClusterID: {
			FieldPath: "cluster_id",
			Store:     true,
			Category:  v1.SearchCategory_DEPLOYMENTS,
		},
	})

	_, err := sac.NewSearchHelper(testNSResource, options, sac.ForResource(testNSResource).ScopeChecker)
	assert.Error(t, err)
}
