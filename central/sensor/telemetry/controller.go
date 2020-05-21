package telemetry

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
)

// KubernetesInfoChunkCallback is a callback function that handles a single chunk of Kubernetes info returned from the sensor.
type KubernetesInfoChunkCallback func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload_KubernetesInfo) error

// ClusterInfoCallback is a callback function that handles a single chunk of ClusterInfo returned from the sensor
type ClusterInfoCallback func(ctx concurrency.ErrorWaitable, sensorInfo *central.TelemetryResponsePayload_ClusterInfo) error

// Controller handles requesting telemetry data from remote clusters.
type Controller interface {
	PullKubernetesInfo(ctx context.Context, cb KubernetesInfoChunkCallback) error
	PullClusterInfo(ctx context.Context, cb ClusterInfoCallback) error
	ProcessTelemetryDataResponse(resp *central.PullTelemetryDataResponse) error
}

// NewController creates and returns a new controller for telemetry data.
func NewController(capabilities centralsensor.SensorCapabilitySet, injector common.MessageInjector, stopSig concurrency.ReadOnlyErrorSignal) Controller {
	return newController(capabilities, injector, stopSig)
}
