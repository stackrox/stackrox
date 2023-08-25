package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// ReadOnlyDataStore is a sub-interface of datastore with the read-only methods.
type ReadOnlyDataStore interface {
	GetNetworkBaseline(ctx context.Context, deploymentID string) (*storage.NetworkBaseline, bool, error)
	Walk(ctx context.Context, f func(baseline *storage.NetworkBaseline) error) error
}

// DataStore stores network baselines for all deployments.
//
//go:generate mockgen-wrapper
type DataStore interface {
	ReadOnlyDataStore

	// The below methods mutate the contents of the datastore.
	// ALL PRODUCTION METHODS MUST NOT CALL THEM DIRECTLY, THEY MUST GO THROUGH THE MANAGER.
	UpsertNetworkBaselines(ctx context.Context, baselines []*storage.NetworkBaseline) error
	DeleteNetworkBaseline(ctx context.Context, deploymentID string) error
	DeleteNetworkBaselines(ctx context.Context, deploymentIDs []string) error
}
