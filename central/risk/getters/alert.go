package getters

import (
	"context"

	alertviews "github.com/stackrox/rox/central/alert/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// AlertSearcher provides the required access to alerts for risk scoring.
type AlertSearcher interface {
	SearchListAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.ListAlert, error)
	SearchAlertPolicyNamesAndSeverities(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*alertviews.PolicyNameAndSeverity, error)
}
