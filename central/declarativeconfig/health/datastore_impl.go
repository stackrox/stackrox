package health

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/declarativeconfig/health/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

type datastoreImpl struct {
	store store.Store
}

func (ds *datastoreImpl) GetDeclarativeConfigs(ctx context.Context) ([]*storage.DeclarativeConfigHealth, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, errors.Errorf("failed to retrieve health for declarative configurations: %v", err)
	} else if !ok {
		return nil, nil
	}

	var declarativeConfigHealths []*storage.DeclarativeConfigHealth
	walkFn := func() error {
		declarativeConfigHealths = declarativeConfigHealths[:0]
		return ds.store.Walk(ctx, func(obj *storage.DeclarativeConfigHealth) error {
			declarativeConfigHealths = append(declarativeConfigHealths, obj)

			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return declarativeConfigHealths, nil
}

func (ds *datastoreImpl) UpsertDeclarativeConfig(ctx context.Context, configHealth *storage.DeclarativeConfigHealth) error {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrapf(err, "failed to update health for declarative config %s", configHealth.GetId())
	}

	return ds.store.Upsert(ctx, configHealth)
}

func (ds *datastoreImpl) RemoveDeclarativeConfig(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return errors.Wrapf(err, "failed to remove health for declarative config %s", id)
	}
	_, exists, err := ds.GetDeclarativeConfig(ctx, id)
	if err != nil {
		return errors.Errorf("failed to retrieve config health %q", id)
	}
	if !exists {
		return errox.NotFound.Newf("unable to find config health for declarative config %q", id)
	}

	return ds.store.Delete(ctx, id)
}

func (ds *datastoreImpl) GetDeclarativeConfig(ctx context.Context, id string) (*storage.DeclarativeConfigHealth, bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); err != nil {
		return nil, false, errors.Errorf("Failed to get health for declarative config %s: %v", id, err)
	} else if !ok {
		return nil, false, nil
	}
	return ds.store.Get(ctx, id)
}
