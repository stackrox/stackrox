package repository

import (
	"context"
	"time"

	repoStore "github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// FailureCountOp specifies the operation to perform on the failure count.
type FailureCountOp int

const (
	// FailureCountNoOp leaves the failure count unchanged.
	FailureCountNoOp FailureCountOp = iota
	// FailureCountReset sets the failure count to zero.
	FailureCountReset
	// FailureCountIncrement increments the failure count by one.
	FailureCountIncrement
)

// ConfigUpdate contains user-configurable fields.
type ConfigUpdate struct {
	RepositoryPath *string
	TagPattern     *string
}

// StatusUpdate contains fields for internal lifecycle updates.
type StatusUpdate struct {
	Status             storage.BaseImageRepository_Status
	LastPolledAt       *time.Time
	LastFailureMessage *string
	FailureCountOp     FailureCountOp

	// OnlyIfStatus makes the update conditional: it only applies if the current
	// status matches one of the specified values. If empty, the update is unconditional.
	OnlyIfStatus []storage.BaseImageRepository_Status
}

// DataStore provides access to base image repositories.
//
//go:generate mockgen-wrapper
type DataStore interface {
	// GetRepository retrieves a base image repository by its ID.
	// Returns the repository, a boolean indicating if it was found, and an error if something went wrong.
	GetRepository(ctx context.Context, id string) (*storage.BaseImageRepository, bool, error)

	// ListRepositories returns all configured base image repositories.
	// Returns empty slice if no repositories configured.
	// Returns error only for system failures (database connection, etc.).
	ListRepositories(ctx context.Context) ([]*storage.BaseImageRepository, error)

	// UpsertRepository inserts or updates the given base image repository.
	// Returns the updated repository or an error if the operation failed.
	UpsertRepository(ctx context.Context, repo *storage.BaseImageRepository) (*storage.BaseImageRepository, error)

	// DeleteRepository removes the base image repository with the specified ID.
	// Returns an error if deletion fails.
	DeleteRepository(ctx context.Context, id string) error

	// UpdateConfiguration updates user-configurable fields of an existing repository.
	// Returns NotFound if the repository does not exist.
	// Returns ErrScanInProgress if the repository is being scanned as per its status.
	UpdateConfiguration(ctx context.Context, id string, update ConfigUpdate) (*storage.BaseImageRepository, error)

	// UpdateStatus updates internal lifecycle fields of a repository.
	// Returns (nil, nil) if the repository does not exist or if conditions are not met.
	UpdateStatus(ctx context.Context, id string, update StatusUpdate) (*storage.BaseImageRepository, error)
}

// New returns a base image repository DataStore.
func New(s repoStore.Store, wf concurrency.KeyFence) DataStore {
	return &datastoreImpl{
		store:      s,
		writeFence: wf,
	}
}
