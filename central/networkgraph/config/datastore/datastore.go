package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// DataStore provides functionality to interact with network graph configuration.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetNetworkGraphConfig(ctx context.Context) (*storage.NetworkGraphConfig, error)
	UpdateNetworkGraphConfig(ctx context.Context, config *storage.NetworkGraphConfig) error
}
