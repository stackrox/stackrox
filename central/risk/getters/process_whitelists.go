package getters

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ProcessWhitelists encapsulates the sub-interface of the process whitelists datastore required for risk.
type ProcessWhitelists interface {
	GetProcessWhitelist(*storage.ProcessWhitelistKey) (*storage.ProcessWhitelist, error)
}

// ProcessIndicators encapulates the sub-interface of the process indicator datastore required for risk.
type ProcessIndicators interface {
	SearchRawProcessIndicators(q *v1.Query) ([]*storage.ProcessIndicator, error)
}
