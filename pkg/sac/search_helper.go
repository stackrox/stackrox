package sac

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
)

// SearchHelper facilitates applying scoped access control to search operations.
type SearchHelper struct {
	resource permissions.Resource

	clusterIDField *v1.SearchField
	namespaceField *v1.SearchField
}

// NewSearchHelper returns a new search helper for the given resource.
func NewSearchHelper(resource permissions.Resource, optionsMap search.OptionsMap, namespaceScoped bool) (*SearchHelper, error) {
	clusterIDField := optionsMap.Original()[search.ClusterID]
	if clusterIDField == nil {
		return nil, errors.Errorf("resource %v used with search helper does not have a %v field", resource, search.ClusterID)
	}

	var nsField *v1.SearchField
	if namespaceScoped {
		nsField = optionsMap.Original()[search.Namespace]
		if nsField == nil {
			return nil, errors.Errorf("resource %v used with search helper is supposed to be namespace-scoped, but does not have a %v field", resource, search.Namespace)
		}
	}

	return &SearchHelper{
		resource:       resource,
		clusterIDField: clusterIDField,
		namespaceField: nsField,
	}, nil
}

// Apply takes in a context-less search function, and returns a search function taking in a context and applying
// scoped access control checks for result filtering.
func (h *SearchHelper) Apply(rawSearchFunc func(*v1.Query) ([]search.Result, error)) func(context.Context, *v1.Query) ([]search.Result, error) {
	return func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		return h.execute(ctx, q, rawSearchFunc)
	}
}

func (h *SearchHelper) execute(ctx context.Context, q *v1.Query, rawSearchFunc func(*v1.Query) ([]search.Result, error)) ([]search.Result, error) {
	scopeChecker := GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(h.resource)
	if ok, err := scopeChecker.Allowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return rawSearchFunc(q)
	}

	// Make sure the cluster and perhaps namespace fields are part of the returned fields.
	fieldQB := search.NewQueryBuilder()
	fieldQB = fieldQB.AddStringsHighlighted(search.ClusterID, search.WildcardString)
	if h.namespaceField != nil {
		fieldQB = fieldQB.AddStringsHighlighted(search.Namespace, search.WildcardString)
	}
	queryWithFields := search.NewConjunctionQuery(q, fieldQB.ProtoQuery())

	results, err := rawSearchFunc(queryWithFields)
	if err != nil {
		return nil, err
	}

	return h.filterResults(ctx, scopeChecker, results)
}

func (h *SearchHelper) objectSubScopeKey(result search.Result) []ScopeKey {
	key := make([]ScopeKey, 0, 2)
	clusterID, _ := result.Fields[h.clusterIDField.GetFieldPath()].(string)
	key = append(key, ClusterScopeKey(clusterID))
	if h.namespaceField != nil {
		namespace, _ := result.Fields[h.namespaceField.GetFieldPath()].(string)
		key = append(key, NamespaceScopeKey(namespace))
	}
	return key
}

func (h *SearchHelper) filterResultsOnce(resourceScopeChecker ScopeChecker, results []search.Result) (allowed []search.Result, maybe []search.Result) {
	for _, result := range results {
		objKey := h.objectSubScopeKey(result)
		if res := resourceScopeChecker.TryAllowed(objKey...); res == Allow {
			allowed = append(allowed, result)
		} else if res == Unknown {
			maybe = append(maybe, result)
		}
	}
	return
}

func (h *SearchHelper) filterResults(ctx context.Context, resourceScopeChecker ScopeChecker, results []search.Result) ([]search.Result, error) {
	allowed, maybe := h.filterResultsOnce(resourceScopeChecker, results)
	if len(maybe) > 0 {
		if err := resourceScopeChecker.PerformChecks(ctx); err != nil {
			return nil, err
		}
		extraAllowed, maybe := h.filterResultsOnce(resourceScopeChecker, maybe)
		if len(maybe) > 0 {
			errorhelpers.PanicOnDevelopmentf("still %d maybe results after PerformChecks", len(maybe))
		}
		allowed = append(allowed, extraAllowed...)
	}

	return allowed, nil
}
