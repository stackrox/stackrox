package sensor

import (
	"context"
	"io"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sensor/queue"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/fake"
)

// CreateOptions represents the custom configuration that can be provided when creating sensor
// using CreateSensor.
type CreateOptions struct {
	workloadManager                    *fake.WorkloadManager
	centralConnFactory                 centralclient.CentralConnectionFactory
	certLoader                         centralclient.CertLoader
	localSensor                        bool
	k8sClient                          client.Interface
	introspectionK8sClient             client.Interface
	traceWriter                        io.Writer
	eventPipelineQueueSize             int
	networkFlowServiceAuthFuncOverride func(context.Context, string) (context.Context, error)
	signalServiceAuthFuncOverride      func(context.Context, string) (context.Context, error)
	networkFlowWriter                  io.Writer
	processIndicatorWriter             io.Writer
}

// ConfigWithDefaults creates a new config object with default properties.
// CentralConnectionFactory is set to nil because the current constructor
// requires an environment variable, and it starts an HTTP connection.
// In order to add a default connection factory here, first the real
// implementation should be refactored to not start an HTTP connection
// before running CreateSensor.
func ConfigWithDefaults() *CreateOptions {
	return &CreateOptions{
		workloadManager:                    nil,
		centralConnFactory:                 nil,
		certLoader:                         centralclient.EmptyCertLoader(),
		k8sClient:                          nil,
		introspectionK8sClient:             nil,
		localSensor:                        false,
		traceWriter:                        nil,
		eventPipelineQueueSize:             queue.ScaleSizeOnNonDefault(env.EventPipelineQueueSize),
		networkFlowServiceAuthFuncOverride: nil,
		signalServiceAuthFuncOverride:      nil,
		networkFlowWriter:                  nil,
		processIndicatorWriter:             nil,
	}
}

// WithK8sClient sets the k8s client.
// Default: nil
func (cfg *CreateOptions) WithK8sClient(k8s client.Interface) *CreateOptions {
	cfg.k8sClient = k8s
	return cfg
}

// WithIntrospectionK8sClient sets the introspection k8s client.
// This is necessary if we want to use the fake-workloads with a CRS installation.
// Default: nil
func (cfg *CreateOptions) WithIntrospectionK8sClient(k8s client.Interface) *CreateOptions {
	cfg.introspectionK8sClient = k8s
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

func (cfg *CreateOptions) WithCertLoader(certLoader centralclient.CertLoader) *CreateOptions {
	cfg.certLoader = certLoader
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

// WithNetworkFlowServiceAuthFuncOverride sets the AuthFuncOverride for the NetworkFlow service.
// Default: nil
func (cfg *CreateOptions) WithNetworkFlowServiceAuthFuncOverride(fn func(context.Context, string) (context.Context, error)) *CreateOptions {
	cfg.networkFlowServiceAuthFuncOverride = fn
	return cfg
}

// WithSignalServiceAuthFuncOverride sets the AuthFuncOverride for the Signal service.
// Default: nil
func (cfg *CreateOptions) WithSignalServiceAuthFuncOverride(fn func(context.Context, string) (context.Context, error)) *CreateOptions {
	cfg.signalServiceAuthFuncOverride = fn
	return cfg
}

// WithNetworkFlowTraceWriter sets the network flows trace writer.
// Default: nil
func (cfg *CreateOptions) WithNetworkFlowTraceWriter(writer io.Writer) *CreateOptions {
	cfg.networkFlowWriter = writer
	return cfg
}

// WithProcessIndicatorTraceWriter sets the network flows trace writer.
// Default: nil
func (cfg *CreateOptions) WithProcessIndicatorTraceWriter(writer io.Writer) *CreateOptions {
	cfg.processIndicatorWriter = writer
	return cfg
}
