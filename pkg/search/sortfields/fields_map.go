package sortfields

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// SortFieldMapper represents helper function that returns an array of query sort options to fulfill sorting by incoming sort option.
type SortFieldMapper func(option *v1.QuerySortOption) []*v1.QuerySortOption

var (
	// SortFieldsMap represents the mapping from searchable fields to sort field helper function
	SortFieldsMap = map[search.FieldLabel]SortFieldMapper{
		search.ImageName: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.ImageRegistry.String(),
					Reversed: option.GetReversed(),
				}.Build(),
				v1.QuerySortOption_builder{
					Field:    search.ImageRemote.String(),
					Reversed: option.GetReversed(),
				}.Build(),
				v1.QuerySortOption_builder{
					Field:    search.ImageTag.String(),
					Reversed: option.GetReversed(),
				}.Build(),
			}
		},
		search.Component: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.Component.String(),
					Reversed: option.GetReversed(),
				}.Build(),
				v1.QuerySortOption_builder{
					Field:    search.ComponentVersion.String(),
					Reversed: option.GetReversed(),
				}.Build(),
			}
		},
		search.NodePriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.NodeRiskScore.String(),
					Reversed: !option.GetReversed(),
				}.Build(),
			}
		},
		search.DeploymentPriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.DeploymentRiskScore.String(),
					Reversed: !option.GetReversed(),
				}.Build(),
			}
		},
		search.ImagePriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.ImageRiskScore.String(),
					Reversed: !option.GetReversed(),
				}.Build(),
			}
		},
		search.ComponentPriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.ComponentRiskScore.String(),
					Reversed: !option.GetReversed(),
				}.Build(),
			}
		},
	}
)
