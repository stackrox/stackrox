package datastore

import (
	"github.com/stackrox/rox/generated/storage"
)

// IntegrationDetails represents integration along with cluster metadata
type IntegrationDetails struct {
	ID                                string                       `db:"compliance_operator_integration_id"`
	Version                           string                       `db:"compliance_operator_version"`
	OperatorInstalled                 bool                         `db:"compliance_operator_installed"`
	OperatorStatus                    storage.COStatus             `db:"compliance_operator_status"`
	ClusterID                         string                       `db:"cluster_id"`
	ClusterName                       string                       `db:"cluster"`
	Type                              storage.ClusterType          `db:"cluster_platform_type"`
	StatusProviderMetadataClusterType storage.ClusterMetadata_Type `db:"cluster_type"`
}
