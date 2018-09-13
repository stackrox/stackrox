package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var imageNameQueryBuilder = builders.NewConjunctionQueryBuilder(
	builders.NewRegexQueryBuilder(search.ImageTag, "Image tag", func(fields *v1.PolicyFields) string {
		return fields.GetImageName().GetTag()
	}),
	builders.NewRegexQueryBuilderWithSubstrings(search.ImageRemote, "Image remote", func(fields *v1.PolicyFields) string {
		return fields.GetImageName().GetRemote()
	}),
	builders.NewRegexQueryBuilder(search.ImageRegistry, "Image registry", func(fields *v1.PolicyFields) string {
		return fields.GetImageName().GetRegistry()
	}),
)
