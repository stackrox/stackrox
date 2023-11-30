package sensor

import (
	"io"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
)

// CreateOptions represents the custom configuration that can be provided when creating sensor
// using CreateSensor.
type CreateOptions struct {
	workloadManager        *fake.WorkloadManager
	centralConnFactory     centralclient.CentralConnectionFactory
	localSensor            bool
	resyncPeriod           time.Duration
	k8sClient              client.Interface
	traceWriter            io.Writer
	eventPipelineQueueSize int
}

// ConfigWithDefaults creates a new config object with default properties.
// CentralConnectionFactory is set to nil because the current constructor
// requires an environment variable, and it starts an HTTP connection.
// In order to add a default connection factory here, first the real
// implementation should be refactored to not start an HTTP connection
// before running CreateSensor.
func ConfigWithDefaults() *CreateOptions {
	return &CreateOptions{
		workloadManager:        nil,
		centralConnFactory:     nil,
		k8sClient:              nil,
		localSensor:            false,
		resyncPeriod:           1 * time.Minute,
		traceWriter:            nil,
		eventPipelineQueueSize: env.EventPipelineQueueSize.IntegerSetting(),
	}
}

// WithK8sClient sets the k8s client.
// Default: nil
func (cfg *CreateOptions) WithK8sClient(k8s client.Interface) *CreateOptions {
	cfg.k8sClient = k8s
	return cfg
}

// WithWorkloadManager sets workload manager.
// Default: nil
func (cfg *CreateOptions) WithWorkloadManager(manager *fake.WorkloadManager) *CreateOptions {
	cfg.workloadManager = manager
	return cfg
}

// WithCentralConnectionFactory sets central connection factory.
// Default: nil
func (cfg *CreateOptions) WithCentralConnectionFactory(centralConnFactory centralclient.CentralConnectionFactory) *CreateOptions {
	cfg.centralConnFactory = centralConnFactory
	return cfg
}

// WithLocalSensor sets if sensor is running locally (local sensor or in tests) or if it's running
// on a cluster.
// Default: false
func (cfg *CreateOptions) WithLocalSensor(flag bool) *CreateOptions {
	cfg.localSensor = flag
	return cfg
}

// WithEventPipelineQueueSize sets the size of the eventPipeline's queue.
// Default: 1000
func (cfg *CreateOptions) WithEventPipelineQueueSize(size int) *CreateOptions {
	cfg.eventPipelineQueueSize = size
	return cfg
}

// WithTraceWriter sets the trace writer.
// Default: nil
func (cfg *CreateOptions) WithTraceWriter(trWriter io.Writer) *CreateOptions {
	cfg.traceWriter = trWriter
	return cfg
}
