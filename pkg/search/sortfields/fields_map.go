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
				{
					Field:    search.ImageRegistry.String(),
					Reversed: option.GetReversed(),
				},
				{
					Field:    search.ImageRemote.String(),
					Reversed: option.GetReversed(),
				},
				{
					Field:    search.ImageTag.String(),
					Reversed: option.GetReversed(),
				},
			}
		},
		search.Component: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				{
					Field:    search.Component.String(),
					Reversed: option.GetReversed(),
				},
				{
					Field:    search.ComponentVersion.String(),
					Reversed: option.GetReversed(),
				},
			}
		},
		search.NodePriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				{
					Field:    search.NodeRiskScore.String(),
					Reversed: !option.GetReversed(),
				},
			}
		},
		search.DeploymentPriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				{
					Field:    search.DeploymentRiskScore.String(),
					Reversed: !option.GetReversed(),
				},
			}
		},
		search.ImagePriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				{
					Field:    search.ImageRiskScore.String(),
					Reversed: !option.GetReversed(),
				},
			}
		},
		search.ComponentPriority: func(option *v1.QuerySortOption) []*v1.QuerySortOption {
			return []*v1.QuerySortOption{
				{
					Field:    search.ComponentRiskScore.String(),
					Reversed: !option.GetReversed(),
				},
			}
		},
	}
)
