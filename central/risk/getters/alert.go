package getters

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// PolicyNameAndSeverity is a lightweight projection of alert data containing only
// the policy name and severity. Used by risk scoring to avoid deserializing full
// alert protobuf blobs when only these two fields are needed.
type PolicyNameAndSeverity struct {
	PolicyName string `db:"policy"`
	Severity   int    `db:"severity"`
}

// AlertSearcher provides the required access to alerts for risk scoring.
type AlertSearcher interface {
	SearchListAlerts(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*storage.ListAlert, error)
	SearchAlertPolicyNamesAndSeverities(ctx context.Context, q *v1.Query, excludeResolved bool) ([]*PolicyNameAndSeverity, error)
}
