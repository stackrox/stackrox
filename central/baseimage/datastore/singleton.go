package datastore

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = logging.LoggerForModule()

	once sync.Once
	ds   DataStore
)

// Singleton returns the global datastore instance.
func Singleton() DataStore {
	once.Do(func() {
		if !devbuild.IsEnabled() {
			log.Panic("Base image watcher mock datastore only available in dev builds, set ROX_DEVELOPMENT_BUILD=true")
		}
		log.Info("Using mock datastore for base image detection (dev mode)")
		// TODO: ROX-31919 - Replace with real PostgreSQL datastore
		ds = newMockDataStore()
	})
	return ds
}

// newMockDataStore creates a mock datastore for development.
// This will be removed once ROX-31919 completes.
func newMockDataStore() DataStore {
	return &inMemoryDataStore{
		repositories: sampleRepositories(),
	}
}

func sampleRepositories() []*storage.BaseImageRepository {
	return []*storage.BaseImageRepository{
		{
			Id:             "pattern-1",
			RepositoryPath: "registry.access.redhat.com/ubi8/ubi",
			TagPattern:     "8.10-*",
			LastPollAt:     timestamppb.Now(),
			FailureCount:   0,
			HealthStatus:   storage.BaseImageRepository_HEALTHY,
		},
		{
			Id:             "pattern-2",
			RepositoryPath: "docker.io/library/ubuntu",
			TagPattern:     "focal-*",
			LastPollAt:     nil,
			FailureCount:   0,
			HealthStatus:   storage.BaseImageRepository_HEALTHY,
		},
	}
}

type inMemoryDataStore struct {
	repositories []*storage.BaseImageRepository
}

func (ds *inMemoryDataStore) ListRepositories(ctx context.Context) ([]*storage.BaseImageRepository, error) {
	return ds.repositories, nil
}
