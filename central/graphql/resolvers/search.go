package resolvers

import (
	"context"
	"sort"

	"github.com/stackrox/rox/central/graphql/schema"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/search/options"
	searchService "github.com/stackrox/rox/central/search/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/search"
)

func init() {
	schema.AddQuery("searchOptions(categories: [SearchCategory!]): [String!]!")
	schema.AddQuery("globalSearch(categories: [SearchCategory!], query: String!): [SearchResult!]!")
}

type rawQuery struct {
	Query *string
}

func (r rawQuery) AsV1Query() (*v1.Query, error) {
	if r.Query == nil {
		return nil, nil
	}
	return search.ParseRawQuery(*r.Query)
}

type categoryData struct {
	searchFunc func(query *v1.Query) ([]*v1.SearchResult, error)
	resource   permissions.Resource
}

func (resolver *Resolver) getCategoryData() map[v1.SearchCategory]categoryData {
	return map[v1.SearchCategory]categoryData{
		v1.SearchCategory_ALERTS: {
			searchFunc: resolver.ViolationsDataStore.SearchAlerts,
			resource:   resources.Alert,
		},
		v1.SearchCategory_IMAGES: {
			searchFunc: resolver.ImageDataStore.SearchImages,
			resource:   resources.Image,
		},
		v1.SearchCategory_POLICIES: {
			searchFunc: resolver.PolicyDataStore.SearchPolicies,
			resource:   resources.Policy,
		},
		v1.SearchCategory_DEPLOYMENTS: {
			searchFunc: resolver.DeploymentDataStore.SearchDeployments,
			resource:   resources.Deployment,
		},
		v1.SearchCategory_SECRETS: {
			searchFunc: resolver.SecretsDataStore.SearchSecrets,
			resource:   resources.Secret,
		},
	}
}

// SearchOptions gets all search options available for the listed categories
func (resolver *Resolver) SearchOptions(args struct{ Categories *[]string }) ([]string, error) {
	var categories []v1.SearchCategory
	if args.Categories == nil {
		categories = searchService.GetAllSearchableCategories()
	} else {
		categories = toSearchCategories(args.Categories)
	}
	return options.GetOptions(categories), nil
}

type searchRequest struct {
	Query      string
	Categories *[]string
}

// GlobalSearch returns search results for the given categories and query.
// Note: there is not currently a way to request the underlying object from SearchResult; it might be nice to have
// this in the future.
func (resolver *Resolver) GlobalSearch(ctx context.Context, args searchRequest) ([]*searchResultResolver, error) {
	q, err := search.ParseRawQuery(args.Query)
	if err != nil {
		return nil, err
	}
	data := resolver.getCategoryData()
	categories := searchService.GetAllSearchableCategories()
	if args.Categories != nil {
		categories = toSearchCategories(args.Categories)
	}
	var allResults []*searchResultResolver
	for _, c := range categories {
		cdata, ok := data[c]
		if !ok {
			continue
		}
		if user.With(permissions.View(cdata.resource)).Authorized(ctx, "graphql") == nil {
			result, err := resolver.wrapSearchResults(cdata.searchFunc(q))
			if err != nil {
				return nil, err
			}
			allResults = append(allResults, result...)
		}
	}
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score() > allResults[j].Score()
	})
	return allResults, nil
}
