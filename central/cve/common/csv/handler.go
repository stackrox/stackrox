package csv

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/graphql/resolvers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
)

var (
	// DeploymentOnlyOptionsMap is OptionsMap containing deployment only fields
	DeploymentOnlyOptionsMap search.OptionsMap
	// ImageOnlyOptionsMap is OptionsMap containing image only fields
	ImageOnlyOptionsMap search.OptionsMap
	// NodeOnlyOptionsMap is OptionsMap containing node only fields
	NodeOnlyOptionsMap search.OptionsMap
	// NamespaceOnlyOptionsMap is OptionsMap namespace only fields
	NamespaceOnlyOptionsMap search.OptionsMap
)

func init() {
	NamespaceOnlyOptionsMap = search.Difference(schema.NamespacesSchema.OptionsMap, schema.ClustersSchema.OptionsMap)
	DeploymentOnlyOptionsMap = search.Difference(
		schema.DeploymentsSchema.OptionsMap,
		search.CombineOptionsMaps(
			schema.ClustersSchema.OptionsMap,
			schema.NamespacesSchema.OptionsMap,
			schema.ImagesSchema.OptionsMap,
		),
	)
	ImageOnlyOptionsMap = search.Difference(
		schema.ImagesSchema.OptionsMap,
		search.CombineOptionsMaps(
			schema.ImageComponentEdgesSchema.OptionsMap,
			schema.ImageComponentsSchema.OptionsMap,
			schema.ImageComponentCveEdgesSchema.OptionsMap,
			schema.ImageCvesSchema.OptionsMap,
		),
	)
	NodeOnlyOptionsMap = search.Difference(
		schema.NodesSchema.OptionsMap,
		search.CombineOptionsMaps(
			schema.NodeComponentEdgesSchema.OptionsMap,
			schema.NodeComponentsSchema.OptionsMap,
			schema.NodeComponentsCvesEdgesSchema.OptionsMap,
			schema.NodeCvesSchema.OptionsMap,
		),
	)
}

// SearchWrapper is used to extract scope from a cve csv export query
type SearchWrapper struct {
	category   v1.SearchCategory
	optionsMap search.OptionsMap
	searcher   search.Searcher
}

// NewSearchWrapper creates SearchWrapper instance
func NewSearchWrapper(category v1.SearchCategory, optionsMap search.OptionsMap, searcher search.Searcher) *SearchWrapper {
	return &SearchWrapper{
		category:   category,
		optionsMap: optionsMap,
		searcher:   searcher,
	}
}

// HandlerImpl represents a handler for csv export
type HandlerImpl struct {
	resolver       *resolvers.Resolver
	searchWrappers []*SearchWrapper
}

// NewCSVHandler creates HandlerImpl instance
func NewCSVHandler(resolver *resolvers.Resolver, searchWrappers []*SearchWrapper) *HandlerImpl {
	return &HandlerImpl{
		resolver:       resolver,
		searchWrappers: searchWrappers,
	}
}

// GetResolver returns the root graphQL resolver in the handler
func (h *HandlerImpl) GetResolver() *resolvers.Resolver {
	if h == nil {
		return nil
	}
	return h.resolver
}

// GetSearchWrappers returns the search wrappers in the handler
func (h *HandlerImpl) GetSearchWrappers() []*SearchWrapper {
	if h == nil {
		return nil
	}
	return h.searchWrappers
}

// GetScopeContext returns the context containing exactly matching scope from the given query
func (h *HandlerImpl) GetScopeContext(ctx context.Context, query *v1.Query) (context.Context, error) {
	if h == nil {
		return nil, errors.New("Handler for CSV export is nil")
	}
	if _, ok := scoped.GetScope(ctx); ok {
		return ctx, nil
	}

	cloned := query.Clone()
	// Remove pagination since we are only determining the resource category which should scope the query.
	cloned.Pagination = nil
	for _, searchWrapper := range h.searchWrappers {
		// Filter the query by resource categories to determine the category that should scope the query.
		// Note that the resource categories are ordered from COMPONENTS to CLUSTERS.
		filteredQ, _ := search.FilterQueryWithMap(cloned, searchWrapper.optionsMap)
		if filteredQ == nil {
			continue
		}

		result, err := searchWrapper.searcher.Search(ctx, filteredQ)
		if err != nil {
			return nil, err
		}

		if len(result) == 0 {
			continue
		}

		// Add searchWrapper only if we get exactly one match. Currently only scoping by one resource is supported in search.
		if len(result) == 1 {
			return scoped.Context(ctx, scoped.Scope{Level: searchWrapper.category, ID: result[0].ID}), nil
		}
	}
	return ctx, nil
}
