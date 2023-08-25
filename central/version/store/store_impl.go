package store

import (
	"context"

	pgStore "github.com/stackrox/rox/central/version/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

type storeImpl struct {
	pgStore pgStore.Store
}

func (s *storeImpl) GetVersion() (*storage.Version, error) {
	ctx := sac.WithAllAccess(context.Background())
	version, exists, err := s.pgStore.Get(ctx)
	if err != nil || !exists {
		return nil, err
	}
	return version, nil
}

func (s *storeImpl) UpdateVersion(version *storage.Version) error {
	ctx := sac.WithAllAccess(context.Background())
	return s.pgStore.Upsert(ctx, version)
}

// GetPreviousVersion returns the version found in central_previous.
// TODO(ROX-18005) -- remove this.  During transition away from serialized version, UpgradeStatus will make this call against
// the older database.  In that case we will need to process the serialized data.
func (s *storeImpl) GetPreviousVersion() (*storage.Version, error) {
	ctx := sac.WithAllAccess(context.Background())
	version, exists, err := s.pgStore.GetPrevious(ctx)
	if err != nil || !exists {
		return nil, err
	}
	return version, nil
}
