package compliancemanager

import "context"

// Manager provides functionality to manage compliance requests.
// Use Manager to process compliance requests and forward them to Sensor.
type Manager interface {
	// Sync reconclies the compliance scan configurations stored in Central with all Sensors.
	// Use it to sync scan configurations upon Central start, Sensor start, and ad-hoc sync requests.
	Sync(ctx context.Context)

	// TODO: update interface{} type to exact struct once API modeling is complete.

	// ProcessScanRequest processes a request to apply a compliance scan configuration to one or more Sensors.
	ProcessScanRequest(ctx context.Context, scanRequest interface{}) error
	// ProcessRescanRequest processes a request to rerun an existing compliance scan configuration.
	ProcessRescanRequest(ctx context.Context, rescanRequest interface{}) error
	// DeleteScan processes a request to delete an existing compliance scan configuration.
	DeleteScan(ctx context.Context, deleteScanRequest interface{}) error
}
