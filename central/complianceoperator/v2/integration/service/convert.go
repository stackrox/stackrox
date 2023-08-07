package service

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
)

/*
storage type to apiV2 type conversions
*/

func convertStorateIntegrationToV2(ctx context.Context, integration *storage.ComplianceIntegration, clusterStore datastore.DataStore) (*v2.ComplianceIntegration, error) {
	if integration == nil {
		return nil, nil
	}

	clusterName, _, err := clusterStore.GetClusterName(ctx, integration.GetClusterId())
	if err != nil {
		return nil, err
	}

	return &v2.ComplianceIntegration{
		Id:           integration.GetId(),
		Version:      integration.GetVersion(),
		ClusterId:    integration.GetClusterId(),
		ClusterName:  clusterName,
		Namespace:    integration.GetNamespace(),
		StatusErrors: integration.GetStatusErrors(),
	}, nil
}

func convertStorageProtos(ctx context.Context, integrations []*storage.ComplianceIntegration, clusterStore datastore.DataStore) ([]*v2.ComplianceIntegration, error) {
	if integrations == nil {
		return nil, nil
	}

	apiIntegrations := make([]*v2.ComplianceIntegration, 0, len(integrations))

	for _, integration := range integrations {
		converted, err := convertStorateIntegrationToV2(ctx, integration, clusterStore)
		if err != nil {
			return nil, errors.Wrapf(err, "Error converting storage compliance operator integration with id %s to response", integration.GetId())
		}
		apiIntegrations = append(apiIntegrations, converted)
	}

	return apiIntegrations, nil
}
