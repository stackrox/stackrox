package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

type virtualMachineDispatcher struct {
	clusterID string
	store     *VirtualMachineStore
}

func newVirtualMachineDispatcher(clusterID string, store *VirtualMachineStore) *virtualMachineDispatcher {
	return &virtualMachineDispatcher{
		clusterID: clusterID,
		store:     store,
	}
}

func (d *virtualMachineDispatcher) ProcessEvent(
	obj interface{},
	_ interface{},
	action central.ResourceAction,
) *component.ResourceEvent {
	virtualMachine := obj.(*kubeVirtV1.VirtualMachine)
	vmWrap := &virtualMachineWrap{
		vm:       virtualMachine,
		original: obj,
	}
	if action == central.ResourceAction_REMOVE_RESOURCE {
		d.store.removeVirtualMachine(vmWrap)
	} else {
		d.store.addOrUpdateVirtualMachine(vmWrap)
	}
	return component.NewEvent(&central.SensorEvent{
		Id:     string(vmWrap.vm.GetUID()),
		Action: action,
		Resource: &central.SensorEvent_VirtualMachine{
			VirtualMachine: &virtualMachineV1.VirtualMachine{
				Id:        string(vmWrap.vm.GetUID()),
				Namespace: vmWrap.vm.GetNamespace(),
				Name:      vmWrap.vm.GetName(),
				ClusterId: d.clusterID,
				Facts:     make(map[string]string),
			},
		},
	})
}
