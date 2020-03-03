package fields

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/builders"
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

	// DisallowedImageLabelQueryBuilder is a KeyValue query builder for disallowed image labels.
	DisallowedImageLabelQueryBuilder = builders.DisallowedMapValueRegexKeyQueryBuilder{
		FieldName:  "image label",
		FieldLabel: search.ImageLabel,
		GetKeyValuePolicy: func(fields *storage.PolicyFields) *storage.KeyValuePolicy {
			return fields.GetDisallowedImageLabel()
		},
	}
)
