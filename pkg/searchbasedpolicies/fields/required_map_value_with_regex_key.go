package fields

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/builders"
)

var (
	// RequiredImageLabelQueryBuilder is a query builder for required labels on an image
	RequiredImageLabelQueryBuilder = builders.RequiredMapValueWithRegexKeyQueryBuilder{
		FieldName:  "image label",
		FieldLabel: search.ImageLabel,
		GetKeyValuePolicy: func(fields *storage.PolicyFields) *storage.KeyValuePolicy {
			return fields.GetRequiredImageLabel()
		},
	}
)
