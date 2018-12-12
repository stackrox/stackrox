package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var imageNameQueryBuilder = builders.RegexQueryBuilder{
	RegexFields: []builders.RegexField{
		{FieldLabel: search.ImageTag, FieldHumanName: "Image tag", RetrieveFieldValue: func(fields *storage.PolicyFields) string {
			return fields.GetImageName().GetTag()
		}},
		{FieldLabel: search.ImageRemote, FieldHumanName: "Image remote", AllowSubstrings: true, RetrieveFieldValue: func(fields *storage.PolicyFields) string {
			return fields.GetImageName().GetRemote()
		}},
		{FieldLabel: search.ImageRegistry, FieldHumanName: "Image registry", RetrieveFieldValue: func(fields *storage.PolicyFields) string {
			return fields.GetImageName().GetRegistry()
		}},
	},
}
