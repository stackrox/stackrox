package datastore

import (
	"context"

	store "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore defines the possible interactions with compliance operator check results
//
//go:generate mockgen-wrapper
type DataStore interface {
	// UpsertResults adds the results to the database
	UpsertResults(ctx context.Context, result *storage.ComplianceOperatorCheckResultV2) error

	// DeleteResults removes a result from the database
	DeleteResults(ctx context.Context, id string) error

	// SearchCheckResults retrieves the scan results specified by query
	SearchCheckResults(ctx context.Context, query *v1.Query) ([]*storage.ComplianceOperatorCheckResultV2, error)
}

// New returns the datastore wrapper for compliance operator check results
func New(store store.Store) DataStore {
	return &datastoreImpl{store: store}
}
