package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var (
	requiredLabelQueryBuilder = builders.RequiredMapValueQueryBuilder{
		FieldName:  "label",
		FieldLabel: search.Label,
		GetKeyValuePolicy: func(fields *v1.PolicyFields) *v1.KeyValuePolicy {
			return fields.GetRequiredLabel()
		},
	}

	requiredAnnotationQueryBuilder = builders.RequiredMapValueQueryBuilder{
		FieldName:  "annotation",
		FieldLabel: search.Annotation,
		GetKeyValuePolicy: func(fields *v1.PolicyFields) *v1.KeyValuePolicy {
			return fields.GetRequiredAnnotation()
		},
	}
)
