package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	k8sUtils "github.com/stackrox/rox/sensor/kubernetes/utils"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

var (
	log = logging.LoggerForModule()
)

//go:generate mockgen-wrapper
type virtualMachineStore interface {
	Has(id virtualmachine.VMID) bool
	Get(id virtualmachine.VMID) *virtualmachine.Info
	AddOrUpdate(vm *virtualmachine.Info) *virtualmachine.Info
	Remove(id virtualmachine.VMID)
	UpdateStateOrCreate(vm *virtualmachine.Info)
	ClearState(id virtualmachine.VMID)
}

type VirtualMachineInstanceDispatcher struct {
	clusterID string
	store     virtualMachineStore
}

func NewVirtualMachineInstanceDispatcher(clusterID string, store virtualMachineStore) *VirtualMachineInstanceDispatcher {
	return &VirtualMachineInstanceDispatcher{
		clusterID: clusterID,
		store:     store,
	}
}

func (d *VirtualMachineInstanceDispatcher) ProcessEvent(
	obj interface{},
	_ interface{},
	action central.ResourceAction,
) *component.ResourceEvent {
	virtualMachineInstance := &kubeVirtV1.VirtualMachineInstance{}
	if err := k8sUtils.FromUnstructuredToSpecificTypePointer(obj, virtualMachineInstance); err != nil {
		log.Errorf("unable to convert 'Unstructured' to 'VirtualMachineInstance': %v", err)
		return nil
	}
	if virtualMachineInstance.GetUID() == "" {
		log.Errorf("conversion from 'Unstructured' to '%T' failed: %v", virtualMachineInstance, obj)
		return nil
	}
	vmUID := virtualMachineInstance.GetUID()
	vmName := virtualMachineInstance.GetName()
	namespace := virtualMachineInstance.GetNamespace()
	vmReference, handled := getVirtualMachineOwnerReference(virtualMachineInstance.GetOwnerReferences())
	// If this is instance is handled by a VirtualMachine
	// then we track the OwnerReference
	if handled {
		vmUID = vmReference.UID
		vmName = vmReference.Name
	}
	vm := &virtualmachine.Info{
		ID:        virtualmachine.VMID(vmUID),
		Name:      vmName,
		Namespace: namespace,
		Running:   virtualMachineInstance.Status.Phase == kubeVirtV1.Running,
		VSOCKCID:  virtualMachineInstance.Status.VSOCKCID,
	}
	// If the instance is NOT handled by a VirtualMachine
	// Process the instance as a VirtualMachine
	if !handled {
		return processVirtualMachine(vm, action, d.clusterID, d.store)
	}

	// This is an instance that is handled by a VirtualMachine

	// We need to check whether the parent is already in the store here
	// because UpdateStateOrCreate will create the entry if it is not present
	ownerReceived := d.store.Has(vm.ID)
	if action == central.ResourceAction_REMOVE_RESOURCE {
		d.store.ClearState(vm.ID)
	} else {
		// This will create an entry for this VirtualMachine if it is not present
		d.store.UpdateStateOrCreate(vm)
	}

	// Do not send events if we are syncing resources
	// and the instance is handled by a VirtualMachine
	if action == central.ResourceAction_SYNC_RESOURCE {
		return nil
	}

	// We should not send any Update events,
	// if we have not received the VirtualMachine yet
	if !ownerReceived {
		return nil
	}

	// Send an Update event for the VirtualMachine that handles this instance
	return component.NewEvent(createEvent(central.ResourceAction_UPDATE_RESOURCE, d.clusterID, vm))
}
