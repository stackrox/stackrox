package getters

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// AlertSearcher provides the required access to alert results for risk scoring.
type AlertSearcher interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
}
