package compliancemanager

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Manager provides functionality to manage compliance requests.
// Use Manager to process compliance requests and forward them to Sensor.
//
//go:generate mockgen-wrapper
type Manager interface {
	// Sync reconclies the compliance scan configurations stored in Central with all Sensors.
	// Use it to sync scan configurations upon Central start, Sensor start, and ad-hoc sync requests.
	Sync(ctx context.Context)

	// ProcessComplianceOperatorInfo processes and stores the compliance operator metadata coming from sensor
	ProcessComplianceOperatorInfo(ctx context.Context, complianceIntegration *storage.ComplianceIntegration) error

	// TODO: update interface{} type to exact struct once API modeling is complete.

	// ProcessScanRequest processes a request to apply a compliance scan configuration to one or more Sensors.
	ProcessScanRequest(ctx context.Context, scanRequest *storage.ComplianceOperatorScanConfigurationV2, clusters []string) (*storage.ComplianceOperatorScanConfigurationV2, error)
	// HandleScanRequestResponse processes response of compliance scan configuration from a sensor.
	HandleScanRequestResponse(ctx context.Context, requestID string, clusterID string, responsePayload string) error

	// ProcessRescanRequest processes a request to rerun an existing compliance scan configuration.
	ProcessRescanRequest(ctx context.Context, rescanRequest interface{}) error
	// DeleteScan processes a request to delete an existing compliance scan configuration.
	// TODO(ROX-19540)
	DeleteScan(ctx context.Context, deleteScanRequest interface{}) error
}
