package datastore

import (
	"github.com/stackrox/rox/generated/storage"
)

// IntegrationDetails represents integration along with cluster metadata
type IntegrationDetails struct {
	ID                                string                        `db:"compliance_operator_integration_id"`
	Version                           string                        `db:"compliance_operator_version"`
	OperatorInstalled                 *bool                         `db:"compliance_operator_installed"`
	OperatorStatus                    *storage.COStatus             `db:"compliance_operator_status"`
	ClusterID                         string                        `db:"cluster_id"`
	ClusterName                       string                        `db:"cluster"`
	Type                              *storage.ClusterType          `db:"cluster_platform_type"`
	StatusProviderMetadataClusterType *storage.ClusterMetadata_Type `db:"cluster_type"`
}

func (i *IntegrationDetails) GetOperatorInstalled() bool {
	if i.OperatorInstalled == nil {
		return false
	}
	return *i.OperatorInstalled
}

func (i *IntegrationDetails) GetOperatorStatus() storage.COStatus {
	if i.OperatorStatus == nil {
		return storage.COStatus_UNHEALTHY
	}

	return *i.OperatorStatus
}

func (i *IntegrationDetails) GetType() storage.ClusterType {
	if i.Type == nil {
		return storage.ClusterType_GENERIC_CLUSTER
	}

	return *i.Type
}

func (i *IntegrationDetails) GetStatusProviderMetadataClusterType() storage.ClusterMetadata_Type {
	if i.StatusProviderMetadataClusterType == nil {
		return storage.ClusterMetadata_UNSPECIFIED
	}

	return *i.StatusProviderMetadataClusterType
}
