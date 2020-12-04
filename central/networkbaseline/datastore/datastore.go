package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// DataStore stores network baselines for all deployments.
//go:generate mockgen-wrapper
type DataStore interface {
	Exists(ctx context.Context, deploymentID string) (bool, error)
	GetNetworkBaseline(ctx context.Context, deploymentID string) (*storage.NetworkBaseline, bool, error)

	UpsertNetworkBaseline(ctx context.Context, baseline *storage.NetworkBaseline) error
	DeleteNetworkBaseline(ctx context.Context, deploymentID string) error
}
