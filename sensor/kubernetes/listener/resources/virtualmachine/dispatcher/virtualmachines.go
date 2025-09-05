package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/store"
	"github.com/stackrox/rox/sensor/kubernetes/utils"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

type VirtualMachineDispatcher struct {
	clusterID string
	store     virtualMachineStore
}

func NewVirtualMachineDispatcher(clusterID string, store virtualMachineStore) *VirtualMachineDispatcher {
	return &VirtualMachineDispatcher{
		clusterID: clusterID,
		store:     store,
	}
}

func (d *VirtualMachineDispatcher) ProcessEvent(
	obj interface{},
	_ interface{},
	action central.ResourceAction,
) *component.ResourceEvent {
	virtualMachine := &kubeVirtV1.VirtualMachine{}
	if err := utils.FromUnstructuredToSpecificTypePointer(obj, virtualMachine); err != nil {
		log.Errorf("unable to convert 'Unstructured' to 'VirtualMachine': %v", err)
		return nil
	}
	if virtualMachine.GetUID() == "" {
		log.Errorf("conversion from 'Unstructured' to '%T' failed: %v", virtualMachine, obj)
		return nil
	}
	isRunning := virtualMachine.Status.PrintableStatus == kubeVirtV1.VirtualMachineStatusRunning
	vm := &store.VirtualMachineInfo{
		ID:        store.VMID(virtualMachine.GetUID()),
		Name:      virtualMachine.GetName(),
		Namespace: virtualMachine.GetNamespace(),
		Running:   isRunning,
	}
	return processVirtualMachine(vm, action, d.clusterID, d.store)
}

func processVirtualMachine(vm *store.VirtualMachineInfo, action central.ResourceAction, clusterID string, store virtualMachineStore) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		store.Remove(vm.ID)
	} else {
		store.AddOrUpdate(vm)
	}
	return component.NewEvent(&central.SensorEvent{
		Id:     string(vm.ID),
		Action: action,
		Resource: &central.SensorEvent_VirtualMachine{
			VirtualMachine: &virtualMachineV1.VirtualMachine{
				Id:        string(vm.ID),
				Namespace: vm.Namespace,
				Name:      vm.Name,
				ClusterId: clusterID,
			},
		},
	})
}
