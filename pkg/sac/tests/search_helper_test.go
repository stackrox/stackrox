package tests

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	. "github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testResource = permissions.Resource("test-resource")
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
	options := search.OptionsMapFromMap(map[search.FieldLabel]*v1.SearchField{
		search.ClusterID: {
			FieldPath: "cluster_id",
		},
		search.Namespace: {
			FieldPath: "namespace",
		},
	})

	mockSearchFunc := func(q *v1.Query) ([]search.Result, error) {
		return []search.Result{
			fakeResult("1", "cluster1", "nsA"),
			fakeResult("2", "cluster1", "nsB"),
			fakeResult("3", "cluster2", "nsA"),
			fakeResult("4", "cluster2", "nsB"),
			fakeResult("5", "cluster3", "nsA"),
			fakeResult("6", "cluster3", "nsB"),
		}, nil
	}

	h, err := NewSearchHelper(testResource, options, true)
	require.NoError(t, err)

	scc := OneStepSCC{
		AccessModeScopeKey(storage.Access_READ_ACCESS): OneStepSCC{
			ResourceScopeKey(testResource): OneStepSCC{
				ClusterScopeKey("cluster1"): AllowAllAccessScopeChecker(),
				ClusterScopeKey("cluster2"): OneStepSCC{
					NamespaceScopeKey("nsA"): AllowAllAccessScopeChecker(),
				},
			},
		},
	}

	ctx := WithGlobalAccessScopeChecker(context.Background(), scc)

	searchResults, err := h.Apply(mockSearchFunc)(ctx, search.EmptyQuery())
	require.NoError(t, err)
	resultIDs := search.ResultsToIDs(searchResults)
	assert.ElementsMatch(t, resultIDs, []string{"1", "2", "3"})
}

func TestSearchHelper_TestApply_WithAllAccess(t *testing.T) {
	options := search.OptionsMapFromMap(map[search.FieldLabel]*v1.SearchField{
		search.ClusterID: {
			FieldPath: "cluster_id",
		},
		search.Namespace: {
			FieldPath: "namespace",
		},
	})

	mockSearchFunc := func(q *v1.Query) ([]search.Result, error) {
		return []search.Result{
			fakeResult("1", "cluster1", "nsA"),
			fakeResult("2", "cluster1", "nsB"),
			fakeResult("3", "cluster2", "nsA"),
			fakeResult("4", "cluster2", "nsB"),
			fakeResult("5", "cluster3", "nsA"),
			fakeResult("6", "cluster3", "nsB"),
		}, nil
	}

	h, err := NewSearchHelper(testResource, options, true)
	require.NoError(t, err)

	scc := AllowAllAccessScopeChecker()

	ctx := WithGlobalAccessScopeChecker(context.Background(), scc)

	searchResults, err := h.Apply(mockSearchFunc)(ctx, search.EmptyQuery())
	require.NoError(t, err)
	resultIDs := search.ResultsToIDs(searchResults)
	assert.ElementsMatch(t, resultIDs, []string{"1", "2", "3", "4", "5", "6"})
}

func TestSearchHelper_TestNew_WithMissingClusterIDField(t *testing.T) {
	options := search.OptionsMapFromMap(map[search.FieldLabel]*v1.SearchField{
		search.Namespace: {
			FieldPath: "namespace",
		},
	})

	_, err := NewSearchHelper(testResource, options, false)
	assert.Error(t, err)
}

func TestSearchHelper_TestNew_WithMissingNSField_NotScoped(t *testing.T) {
	options := search.OptionsMapFromMap(map[search.FieldLabel]*v1.SearchField{
		search.ClusterID: {
			FieldPath: "cluster_id",
		},
	})

	_, err := NewSearchHelper(testResource, options, false)
	assert.NoError(t, err)
}

func TestSearchHelper_TestNew_WithMissingNSField_Scoped(t *testing.T) {
	options := search.OptionsMapFromMap(map[search.FieldLabel]*v1.SearchField{
		search.ClusterID: {
			FieldPath: "cluster_id",
		},
	})

	_, err := NewSearchHelper(testResource, options, true)
	assert.Error(t, err)
}
