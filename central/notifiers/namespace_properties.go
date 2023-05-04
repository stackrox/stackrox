package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// NamespaceProperties provides functions for getting information about namespace metadata.
type NamespaceProperties interface {
	GetAnnotationValue(ctx context.Context, alert *storage.Alert, annotationKey, defaultValue string) string
}
