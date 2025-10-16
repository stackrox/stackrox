package service

import (
	"context"

	"github.com/pkg/errors"
	complianceDS "github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

/*
storage type to apiV2 type conversions
*/

func convertStorageIntegrationToV2(ctx context.Context, integration *complianceDS.IntegrationDetails, complianceStore complianceDS.DataStore) (*v2.ComplianceIntegration, bool, error) {
	if integration == nil {
		return nil, false, nil
	}

	integrationDetail, found, err := complianceStore.GetComplianceIntegration(ctx, integration.ID)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, errors.Errorf("unable to get details for compliance integration %q", integrationDetail.GetId())
	}

	opStatus := v2.COStatus_UNHEALTHY
	if integration.GetOperatorStatus() == storage.COStatus_HEALTHY {
		opStatus = v2.COStatus_HEALTHY
	}

	opInstalled := false
	if integration.GetOperatorInstalled() {
		opInstalled = true
	}

	ci := &v2.ComplianceIntegration{}
	ci.SetId(integration.ID)
	ci.SetVersion(integration.Version)
	ci.SetClusterId(integration.ClusterID)
	ci.SetClusterName(integration.ClusterName)
	ci.SetNamespace(integrationDetail.GetComplianceNamespace())
	ci.SetStatusErrors(integrationDetail.GetStatusErrors())
	ci.SetOperatorInstalled(opInstalled)
	ci.SetStatus(opStatus)
	ci.SetClusterPlatformType(convertPlatformType(integration.GetType()))
	ci.SetClusterProviderType(convertProviderType(integration.GetStatusProviderMetadataClusterType()))
	return ci, true, nil
}

func convertStorageProtos(ctx context.Context, integrations []*complianceDS.IntegrationDetails, complianceStore complianceDS.DataStore) ([]*v2.ComplianceIntegration, error) {
	if integrations == nil {
		return nil, nil
	}

	apiIntegrations := make([]*v2.ComplianceIntegration, 0, len(integrations))

	for _, integration := range integrations {
		converted, clusterFound, err := convertStorageIntegrationToV2(ctx, integration, complianceStore)
		if err != nil {
			return nil, errors.Wrapf(err, "Error converting storage compliance operator integration with id %s to response", integration.ID)
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

func convertPlatformType(platformType storage.ClusterType) v2.ClusterPlatformType {
	switch platformType {
	case storage.ClusterType_GENERIC_CLUSTER:
		return v2.ClusterPlatformType_GENERIC_CLUSTER
	case storage.ClusterType_KUBERNETES_CLUSTER:
		return v2.ClusterPlatformType_KUBERNETES_CLUSTER
	case storage.ClusterType_OPENSHIFT_CLUSTER:
		return v2.ClusterPlatformType_OPENSHIFT_CLUSTER
	case storage.ClusterType_OPENSHIFT4_CLUSTER:
		return v2.ClusterPlatformType_OPENSHIFT4_CLUSTER
	default:
		utils.Should(errors.Errorf("unhandled cluster platform type encountered %s", platformType))
		return v2.ClusterPlatformType_GENERIC_CLUSTER
	}
}

func convertProviderType(providerType storage.ClusterMetadata_Type) v2.ClusterProviderType {
	switch providerType {
	case storage.ClusterMetadata_UNSPECIFIED:
		return v2.ClusterProviderType_UNSPECIFIED
	case storage.ClusterMetadata_AKS:
		return v2.ClusterProviderType_AKS
	case storage.ClusterMetadata_ARO:
		return v2.ClusterProviderType_ARO
	case storage.ClusterMetadata_EKS:
		return v2.ClusterProviderType_EKS
	case storage.ClusterMetadata_GKE:
		return v2.ClusterProviderType_GKE
	case storage.ClusterMetadata_OCP:
		return v2.ClusterProviderType_OCP
	case storage.ClusterMetadata_OSD:
		return v2.ClusterProviderType_OSD
	case storage.ClusterMetadata_ROSA:
		return v2.ClusterProviderType_ROSA
	default:
		utils.Should(errors.Errorf("unhandled cluster platform type encountered %s", providerType))
		return v2.ClusterProviderType_UNSPECIFIED
	}
}
