//go:build !fakeworkloads

package fake

import (
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/signal"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	vmStore "github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/store"
)

// WorkloadManager is a stub when fakeworkloads build tag is not set.
type WorkloadManager struct{}

// WorkloadManagerConfig is a stub when fakeworkloads build tag is not set.
type WorkloadManagerConfig struct{}

// NewWorkloadManager returns nil when fakeworkloads build tag is not set.
func NewWorkloadManager(_ *WorkloadManagerConfig) *WorkloadManager {
	return nil
}

// ConfigDefaults returns a stub config when fakeworkloads build tag is not set.
func ConfigDefaults() *WorkloadManagerConfig {
	return nil
}

// WithWorkloadFile is a no-op stub.
func (c *WorkloadManagerConfig) WithWorkloadFile(_ string) *WorkloadManagerConfig {
	return c
}

// Client satisfies the interface but should never be called (manager is always nil).
func (w *WorkloadManager) Client() client.Interface {
	return nil
}

// SetSignalHandlers is a no-op stub.
func (w *WorkloadManager) SetSignalHandlers(_ signal.Pipeline, _ manager.Manager) {}

// SetVMIndexReportHandler is a no-op stub.
func (w *WorkloadManager) SetVMIndexReportHandler(_ index.Handler) {}

// SetVMStore is a no-op stub.
func (w *WorkloadManager) SetVMStore(_ *vmStore.VirtualMachineStore) {}

// Notify satisfies the common.Notifiable interface.
func (w *WorkloadManager) Notify(_ common.SensorComponentEvent) {}

// Stop is a no-op stub.
func (w *WorkloadManager) Stop() {}
