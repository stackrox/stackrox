package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// RequiredLabelQueryBuilder is a query builder for required labels on a deployment.
	RequiredLabelQueryBuilder = builders.RequiredMapValueQueryBuilder{
		FieldName:  "label",
		FieldLabel: search.Label,
		GetKeyValuePolicy: func(fields *storage.PolicyFields) *storage.KeyValuePolicy {
			return fields.GetRequiredLabel()
		},
	}

	// RequiredAnnotationQueryBuilder is a query builder for required annotations on a deployment.
	RequiredAnnotationQueryBuilder = builders.RequiredMapValueQueryBuilder{
		FieldName:  "annotation",
		FieldLabel: search.Annotation,
		GetKeyValuePolicy: func(fields *storage.PolicyFields) *storage.KeyValuePolicy {
			return fields.GetRequiredAnnotation()
		},
	}
)
