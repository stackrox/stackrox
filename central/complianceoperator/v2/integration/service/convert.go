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

func convertStorageIntegrationToV2(ctx context.Context, integration *storage.ComplianceIntegration, clusterStore datastore.DataStore) (*v2.ComplianceIntegration, bool, error) {
	if integration == nil {
		return nil, false, nil
	}

	clusterName, clusterFound, err := clusterStore.GetClusterName(ctx, integration.GetClusterId())
	if err != nil {
		return nil, false, err
	}

	return &v2.ComplianceIntegration{
		Id:           integration.GetId(),
		Version:      integration.GetVersion(),
		ClusterId:    integration.GetClusterId(),
		ClusterName:  clusterName,
		Namespace:    integration.GetComplianceNamespace(),
		StatusErrors: integration.GetStatusErrors(),
		ReadOnly:     integration.GetReadOnly(),
	}, clusterFound, nil
}

func convertStorageProtos(ctx context.Context, integrations []*storage.ComplianceIntegration, clusterStore datastore.DataStore) ([]*v2.ComplianceIntegration, error) {
	if integrations == nil {
		return nil, nil
	}

	apiIntegrations := make([]*v2.ComplianceIntegration, 0, len(integrations))

	for _, integration := range integrations {
		converted, clusterFound, err := convertStorageIntegrationToV2(ctx, integration, clusterStore)
		if err != nil {
			return nil, errors.Wrapf(err, "Error converting storage compliance operator integration with id %s to response", integration.GetId())
		}
		// If the cluster cannot be found that means it was removed, so we should not
		// return this as a valid integration
		if !clusterFound {
			continue
		}
		apiIntegrations = append(apiIntegrations, converted)
	}

	return apiIntegrations, nil
}
