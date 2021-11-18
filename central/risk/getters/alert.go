package getters

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// AlertGetter provides the required access to alerts for risk scoring.
type AlertGetter interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
}
