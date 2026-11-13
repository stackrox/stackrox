package administrationevents

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// EventView provides administration event reads using the search framework.
//
//go:generate mockgen-wrapper
type EventView interface {
	Count(ctx context.Context, q *v1.Query) (int, error)
	Get(ctx context.Context, q *v1.Query) ([]*storage.AdministrationEvent, error)
}
