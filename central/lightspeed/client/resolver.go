package client

import (
	"context"

	"github.com/stackrox/rox/central/aiintegration/datastore"
	"github.com/stackrox/rox/pkg/sac"
)

type integrationResolver struct {
	ds datastore.DataStore
}

// NewIntegrationResolver returns an EndpointResolver that reads the service URL
// from the first stored AI integration.
func NewIntegrationResolver(ds datastore.DataStore) EndpointResolver {
	if ds == nil {
		return nil
	}
	return &integrationResolver{ds: ds}
}

func (r *integrationResolver) GetEndpoint(ctx context.Context) (string, bool, error) {
	elevatedCtx := sac.WithAllAccess(ctx)
	integrations, err := r.ds.GetAll(elevatedCtx)
	if err != nil {
		return "", false, err
	}
	if len(integrations) == 0 {
		return "", false, nil
	}
	return integrations[0].GetServiceUrl(), true, nil
}
