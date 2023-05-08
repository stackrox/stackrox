package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// MetadataGetter provides functionality for getting metadata information for alerts
type MetadataGetter interface {
	GetAnnotationValue(ctx context.Context, alert *storage.Alert, annotationKey, defaultValue string) string
}
