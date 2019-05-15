package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// DisallowedAnnotationQueryBuilder is a KeyValue query builder for disallowed annotations on deployments.
	DisallowedAnnotationQueryBuilder = builders.DisallowedMapValueQueryBuilder{
		FieldName:  "annotation",
		FieldLabel: search.Annotation,
		GetKeyValuePolicy: func(fields *storage.PolicyFields) *storage.KeyValuePolicy {
			return fields.GetDisallowedAnnotation()
		},
	}
)
