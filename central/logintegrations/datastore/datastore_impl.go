package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/logintegrations/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	logIntegrationSAC = sac.ForResource(resources.LogIntegration)
)

type datastoreImpl struct {
	storage store.Store
}

func (ds *datastoreImpl) GetLogIntegration(ctx context.Context, id string) (*storage.LogIntegration, bool, error) {
	if ok, err := logIntegrationSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	return ds.storage.Get(id)
}

func (ds *datastoreImpl) GetLogIntegrations(ctx context.Context) ([]*storage.LogIntegration, error) {
	if ok, err := logIntegrationSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	var integrations []*storage.LogIntegration
	if err := ds.storage.Walk(func(integration *storage.LogIntegration) error {
		integrations = append(integrations, integration)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "fetching log integrations")
	}
	return integrations, nil
}

func (ds *datastoreImpl) CreateLogIntegration(ctx context.Context, integration *storage.LogIntegration) error {
	if ok, err := logIntegrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}

	integration.CreatedAt = types.TimestampNow()
	if err := validateLogIntegration(integration); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, errors.Wrap(err, "creating log integration").Error())
	}
	return ds.storage.Upsert(integration)
}

func (ds *datastoreImpl) UpdateLogIntegration(ctx context.Context, integration *storage.LogIntegration) error {
	if ok, err := logIntegrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}

	if err := validateLogIntegration(integration); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, errors.Wrap(err, "updating log integration").Error())
	}
	return ds.storage.Upsert(integration)
}

func (ds *datastoreImpl) DeleteLogIntegration(ctx context.Context, id string) error {
	if ok, err := logIntegrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}

	return ds.storage.Delete(id)
}

func validateLogIntegration(obj *storage.LogIntegration) error {
	if obj.GetId() == "" {
		return errors.New("log integration ID must be provided")
	}

	if obj.GetName() == "" {
		return errors.New("log integration name must be provided")
	}

	if obj.GetConfig() == nil {
		return errors.New("log integration configuration must be provided")
	}
	return nil
}
