package resolvers

import (
	"fmt"

	"github.com/stackrox/rox/central/graphql/schema"
	"github.com/stackrox/rox/central/search/options"
	searchService "github.com/stackrox/rox/central/search/service"
	"github.com/stackrox/rox/generated/api/v1"
)

func init() {
	schema.AddQuery("searchOptions(categories: [SearchCategory!]): [String!]!")
}

// SearchOptions gets all search options available for the listed categories
func (resolver *Resolver) SearchOptions(args struct{ Categories *[]string }) ([]string, error) {
	categories := make([]v1.SearchCategory, 0, len(v1.SearchCategory_value))
	if args.Categories == nil || *args.Categories == nil {
		categories = searchService.GetAllSearchableCategories()
	} else {
		for _, s := range *args.Categories {
			val, ok := v1.SearchCategory_value[s]
			if !ok {
				return nil, fmt.Errorf("invalid search category: %q", s)
			}
			categories = append(categories, v1.SearchCategory(val))
		}
	}
	return options.GetOptions(categories), nil
}

// todo: add actual search. The permissions are going to suck.
// perhaps we should use the search service here.
