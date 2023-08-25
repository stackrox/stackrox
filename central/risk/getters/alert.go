package getters

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AlertSearcher provides the required access to alerts for risk scoring.
type AlertSearcher interface {
	SearchListAlerts(ctx context.Context, q *v1.Query) ([]*storage.ListAlert, error)
}
