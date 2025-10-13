package notifiers

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// MetadataGetter provides functionality for getting metadata information for alerts
//
//go:generate mockgen-wrapper MetadataGetter
type MetadataGetter interface {
	GetAnnotationValue(ctx context.Context, alert *storage.Alert, annotationKey, defaultValue string) string
	GetNamespaceLabels(ctx context.Context, alert *storage.Alert) map[string]string
}
