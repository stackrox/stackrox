package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
)

type clusterIDGetter interface {
	GetNoWait() string
}

// Handler provides functionality to send virtual machine index reports to Central.
// It embeds ComplianceComponent (which itself embeds SensorComponent) so that
// compliance channel wiring is part of the compile-time contract.
type Handler interface {
	common.ComplianceComponent

	Send(ctx context.Context, indexReport *v1.IndexReport, discoveredData *v1.DiscoveredData) error
}

// VirtualMachineStore interface to the VirtualMachine store
//
//go:generate mockgen-wrapper
type VirtualMachineStore interface {
	Get(id virtualmachine.VMID) *virtualmachine.Info
	GetFromCID(cid uint32) *virtualmachine.Info
	AddOrUpdate(vm *virtualmachine.Info) *virtualmachine.Info
}

// NewHandler returns the virtual machine component for Sensor to use.
func NewHandler(clusterIDGetter clusterIDGetter, store VirtualMachineStore) Handler {
	return &handlerImpl{
		clusterID:    clusterIDGetter,
		centralReady: concurrency.NewSignal(),
		lock:         &sync.RWMutex{},
		stopper:      concurrency.NewStopper(),
		store:        store,
	}
}
