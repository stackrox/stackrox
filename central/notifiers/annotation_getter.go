package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// AnnotationGetter provides functionality for getting information about annotation values.
type AnnotationGetter interface {
	GetAnnotationValue(ctx context.Context, alert *storage.Alert, annotationKey, defaultValue string) string
}
