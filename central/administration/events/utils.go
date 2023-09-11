package events

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

var rootNamespaceUUID = uuid.FromStringOrNil("d4dcc3d8-fcdf-4621-8386-0be1372ecbba")

// GenerateEventID returns the dedup ID for an administration event.
func GenerateEventID(event *storage.AdministrationEvent) string {
	dedupKey := strings.Join([]string{
		event.GetDomain(),
		event.GetMessage(),
		event.GetResourceId(),
		event.GetResourceType(),
		event.GetType().String(),
	}, ",")
	return uuid.NewV5(rootNamespaceUUID, dedupKey).String()
}
