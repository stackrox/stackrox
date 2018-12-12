package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	requiredLabelQueryBuilder = builders.RequiredMapValueQueryBuilder{
		FieldName:  "label",
		FieldLabel: search.Label,
		GetKeyValuePolicy: func(fields *storage.PolicyFields) *storage.KeyValuePolicy {
			return fields.GetRequiredLabel()
		},
	}

	requiredAnnotationQueryBuilder = builders.RequiredMapValueQueryBuilder{
		FieldName:  "annotation",
		FieldLabel: search.Annotation,
		GetKeyValuePolicy: func(fields *storage.PolicyFields) *storage.KeyValuePolicy {
			return fields.GetRequiredAnnotation()
		},
	}
)
