package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// AnnotationGetter provides functions for getting information about namespace metadata.
type AnnotationGetter interface {
	GetAnnotationValue(ctx context.Context, alert *storage.Alert, annotationKey, defaultValue string) string
}
