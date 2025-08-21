package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/store"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("not of type 'Unstructured': %T", obj)
		return nil
	}
	virtualMachine := &kubeVirtV1.VirtualMachine{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, virtualMachine); err != nil {
		log.Errorf("unable to convert 'Unstructured' to 'VirtualMachine': %v", err)
		return nil
	}
	if virtualMachine.GetUID() == "" {
		log.Errorf("convertion from unstructured failed: %v", obj)
		return nil
	}
	isRunning := virtualMachine.Status.PrintableStatus == kubeVirtV1.VirtualMachineStatusRunning
	vm := &store.VirtualMachineInfo{
		UID:       string(virtualMachine.GetUID()),
		Name:      virtualMachine.GetName(),
		Namespace: virtualMachine.GetNamespace(),
		Running:   isRunning,
	}
	if action == central.ResourceAction_REMOVE_RESOURCE {
		d.store.RemoveVirtualMachine(vm.UID)
	} else {
		d.store.AddOrUpdateVirtualMachine(vm)
	}
	return component.NewEvent(&central.SensorEvent{
		Id:     vm.UID,
		Action: action,
		Resource: &central.SensorEvent_VirtualMachine{
			VirtualMachine: &virtualMachineV1.VirtualMachine{
				Id:        vm.UID,
				Namespace: vm.Namespace,
				Name:      vm.Name,
				ClusterId: d.clusterID,
			},
		},
	})
}
